package main

import (
	"context"
	_ "embed"
	"flag"
	"log"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/chainguard-dev/secureframe-issue-sync/pkg/issue"
	"github.com/chainguard-dev/secureframe-issue-sync/pkg/secureframe"
	"github.com/danott/envflag"
	"golang.org/x/oauth2"

	"github.com/google/go-github/v44/github"
)

var (
	githubTokenPathFlag = flag.String("github-token-path", "", "path to github token file")
	githubTokenFlag     = flag.String("github-token", "", "github token")
	dryRunFlag          = flag.Bool("dry-run", false, "dry-run mode")
	sfTokenFlag         = flag.String("secureframe-token", "", "Secureframe bearer token")
	reportKeyFlag       = flag.String("report-key", "soc2_alpha", "report key to filter by")
	companyIDFlag       = flag.String("company", "079b854c-c53a-4c71-bfb8-f9e87b13b6c4", "secureframe company user ID")
	githubRepoFlag      = flag.String("github-repo", "chainguard-dev/secureframe", "github repo to open issues against")
	githubLabelFlag     = flag.String("github-label", "", "additional github label to apply")

	idRE         = regexp.MustCompile(`Secureframe ID: ([\w-]+)`)
	sleepMS      = 250
	maxSleepTime = 5 * time.Second
)

func main() {
	flag.Parse()
	envflag.Parse()

	// Also available in the environment as GITHUB_TOKEN
	ghToken := *githubTokenFlag

	if *githubTokenPathFlag != "" {
		bs, err := os.ReadFile(*githubTokenPathFlag)
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
	tests, err := secureframe.GetTests(context.Background(), *companyIDFlag, *sfTokenFlag, *reportKeyFlag)
	if err != nil {
		log.Panicf("Secureframe test query failed: %v", err)
	}

	log.Printf("%d Secureframe tests found", len(tests))
	if len(tests) == 0 {
		os.Exit(0)
	}

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

	// issue by test ID
	issuesByID := map[string]*github.Issue{}
	for _, i := range issues {
		id := ""
		match := idRE.FindStringSubmatch(i.GetBody())
		if len(match) > 0 {
			// log.Printf("found match: %v", match)
			id = match[1]
			issuesByID[id] = i
		} else {
			log.Printf("no test ID found in issue[%s]: %+v", id, i.GetTitle())
		}
		log.Printf("issue[%s]: %+v", id, i.GetTitle())
	}

	log.Printf("%d synced issues found", len(issues))

	testsByID := map[string]secureframe.Test{}
	lastWasMod := false
	updates := 0

	created := 0
	reopened := 0
	closed := 0
	updated := 0

	log.Printf("syncing %d tests ...", len(tests))
	for _, t := range tests {
		// Avoid Github API DoS attack
		if lastWasMod && !*dryRunFlag {
			log.Printf("Sleeping ...")
			sleepTime := time.Duration(updates*sleepMS) * time.Millisecond
			if sleepTime > maxSleepTime {
				sleepTime = maxSleepTime
			}

			time.Sleep(sleepTime)
			updates++
			lastWasMod = false
		}

		testsByID[t.ID] = t
		// log.Printf("Creating issue template from test: %+v", t)
		ft, err := issue.FromTest(t, *githubLabelFlag, *reportKeyFlag)
		if err != nil {
			log.Panicf("issue: %v", err)
		}

		i := issuesByID[t.ID]

		// Test does not exist in Github
		if i == nil {
			if t.Pass || !t.Enabled {
				// log.Printf("Skipping passing test: %s", t.Description)
				continue
			}

			log.Printf("Creating: %s", ft.Title)
			created++
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
				closed++
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
				closed++
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
				updated++
				log.Printf("Updating #%d: %s", i.GetNumber(), ft.Title)
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
				reopened++
				log.Printf("Reopening #%d (%s) ...", i.GetNumber(), i.GetTitle())
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

	// Close Github issues that are no longer being tracked by Secureframe
	for id, i := range issuesByID {
		_, ok := testsByID[id]
		if ok {
			continue
		}
		if i.GetState() == "closed" {
			continue
		}
		log.Printf("Closing #%d (%s) as it is no longer tracked by Secureframe...", i.GetNumber(), i.GetTitle())
		closed++
		if !*dryRunFlag {
			if err := issue.Close(ctx, gc, org, project, i, issue.DisabledLabel); err != nil {
				log.Panicf("close: %v", err)
			}
			time.Sleep(250 * time.Millisecond)
		}
		continue
	}

	log.Printf("%d issues created", created)
	log.Printf("%d issues updated", updated)
	log.Printf("%d issues closed", closed)
	log.Printf("%d issues reopened", reopened)
}
