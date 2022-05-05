package main

import (
	"context"
	_ "embed"
	"flag"
	"io/ioutil"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/chainguard-dev/secureframe-github-sync/pkg/issue"
	"github.com/chainguard-dev/secureframe-github-sync/pkg/secureframe"
	"github.com/danott/envflag"
	"golang.org/x/oauth2"

	"github.com/google/go-github/v44/github"
)

var (
	githubTokenPathFlag = flag.String("github-token-path", "", "path to github token file")
	githubTokenFlag     = flag.String("github-token", "", "github token")
	dryRunFlag          = flag.Bool("dry-run", false, "dry-run mode")
	sfTokenFlag         = flag.String("secureframe-token", "", "Secureframe bearer token")
	keysFlag            = flag.String("reports", "soc2_alpha", "comma-delimited list of report keys to use")
	companyIDFlag       = flag.String("company", "079b854c-c53a-4c71-bfb8-f9e87b13b6c4", "secureframe company user ID")
	githubRepoFlag      = flag.String("github-repo", "chainguard-dev/secureframe", "github repo to open issues against")
	githubLabelFlag     = flag.String("github-label", "soc2", "github label to apply")

	idRE    = regexp.MustCompile(`Secureframe ID: ([\w-]+)`)
	sleepMS = 250
)

func main() {
	flag.Parse()
	envflag.Parse()

	// Also available in the environment as GITHUB_TOKEN
	ghToken := *githubTokenFlag

	if *githubTokenPathFlag != "" {
		bs, err := ioutil.ReadFile(*githubTokenFlag)
		if err != nil {
			log.Panicf("readfile: %v", err)
		}
		ghToken = strings.TrimSpace(string(bs))
	}
	if ghToken == "" {
		log.Printf("github-token is empty: skipping github calls")
	}

	ctx := context.Background()
	tc := oauth2.NewClient(ctx, oauth2.StaticTokenSource(&oauth2.Token{AccessToken: ghToken}))
	gc := github.NewClient(tc)

	// NOTE: sfTokenFlag is also available in the environment as SECUREFRAME_TOKEN
	tests, err := secureframe.GetDashboardTests(context.Background(), *companyIDFlag, *sfTokenFlag, strings.Split(*keysFlag, ","))
	if err != nil {
		log.Panicf("error: %v", err)
	}

	log.Printf("%d Secureframe tests found", len(tests))

	parts := strings.Split(*githubRepoFlag, "/")
	org := parts[0]
	project := parts[1]

	issues := []*github.Issue{}
	if ghToken != "" {
		issues, err = issue.Synced(ctx, gc, org, project)
		if err != nil {
			log.Panicf("synced: %v", err)
		}
	}

	log.Printf("syncing labels ...")
	labels := []string{issue.SyncLabel, issue.DisabledLabel, issue.PassingLabel, *githubLabelFlag}
	if !*dryRunFlag {
		if err := issue.SyncLabels(ctx, gc, org, project, labels); err != nil {
			log.Panicf("sync labels: %v", err)
		}
	}

	issuesByID := map[string]*github.Issue{}
	for _, i := range issues {
		id := ""
		match := idRE.FindStringSubmatch(i.GetBody())
		if len(match) > 0 {
			//	log.Printf("found match: %v", match)
			id = match[1]
			issuesByID[id] = i
		}
		//	log.Printf("issue[%s]: %+v", id, i)
	}

	log.Printf("%d synced issues found", len(issues))

	testsByID := map[string]secureframe.Test{}
	lastWasMod := false
	updates := 0

	for _, t := range tests {
		// Avoid Github API DoS attack
		if lastWasMod && !*dryRunFlag {
			log.Printf("Sleeping ...")
			time.Sleep(time.Duration(updates*sleepMS) * time.Millisecond)
			updates++
			lastWasMod = false
		}

		testsByID[t.ID] = t

		// log.Printf("Test #%d: %s: %s (pass=%v, enabled=%v)", x, t.ID, t.Description, t.Pass, t.Enabled)

		t, err := secureframe.GetTest(context.Background(), *companyIDFlag, *sfTokenFlag, t.ID)
		if err != nil {
			log.Panicf("error: %v", err)
		}

		ft, err := issue.FromTest(t, *githubLabelFlag)
		if err != nil {
			log.Panicf("issue: %v", err)
		}

		i := issuesByID[t.ID]

		// Test does not exist in Github
		if i == nil {
			if t.Pass {
				// log.Printf("Skipping passing test: %s", t.Description)
				continue
			}

			log.Printf("Creating: %+v", ft)
			if !*dryRunFlag {
				if err := issue.Create(ctx, gc, org, project, ft); err != nil {
					log.Panicf("create: %v", err)
				}
				lastWasMod = true
			}
			continue
		}

		if i.GetState() == "open" {
			// Close passing or disabled tests
			if t.Pass {
				log.Printf("Closing #%d (%s) as it is passing...", i.GetNumber(), i.GetTitle())
				if !*dryRunFlag {
					if err := issue.Close(ctx, gc, org, project, i, issue.PassingLabel); err != nil {
						log.Panicf("close: %v", err)
					}
					lastWasMod = true
				}
				continue
			}

			if !t.Enabled {
				log.Printf("Closing #%d (%s) as it is disabled...", i.GetNumber(), i.GetTitle())
				if !*dryRunFlag {
					if err := issue.Close(ctx, gc, org, project, i, issue.DisabledLabel); err != nil {
						log.Panicf("close: %v", err)
					}
					lastWasMod = true
				}
				continue

			}

			// Update failing tests
			if i.GetBody() != ft.Body || i.GetTitle() != ft.Title {
				log.Printf("Updating #%d with: %+v", i.GetNumber(), ft)
				if !*dryRunFlag {
					if err := issue.Update(ctx, gc, org, project, i.GetNumber(), ft); err != nil {
						log.Panicf("update: %v", err)
					}
					lastWasMod = true
				}
			}
			continue
		}

		if i.GetState() == "closed" {
			if !t.Pass && t.Enabled {
				log.Printf("Reopening #%d ...", i.GetNumber())
				if !*dryRunFlag {
					if err := issue.Update(ctx, gc, org, project, i.GetNumber(), ft); err != nil {
						log.Panicf("update: %v", err)
					}
					lastWasMod = true
				}
				continue
			}
		}
	}
}
