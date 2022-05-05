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
	"golang.org/x/oauth2"

	"github.com/google/go-github/v44/github"
)

var (
	githubTokenFlag = flag.String("github-token-file", "", "path to github token file")
	dryRunFlag      = flag.Bool("dry-run", false, "dry-run mode")
	// TODO: Replace with OAuth (help wanted)
	bearerTokenFlag = flag.String("bearer-token", "", "secureframe bearer token")
	keysFlag        = flag.String("report-keys", "soc2_alpha", "comma-delimited list of report keys to use")
	companyIDFlag   = flag.String("company-id", "079b854c-c53a-4c71-bfb8-f9e87b13b6c4", "secureframe company user ID")
	githubRepoFlag  = flag.String("github-repo", "chainguard-dev/secureframe-", "github repo to open issues against")
	githubLabelFlag = flag.String("github-label", "soc2", "github label to apply")
	idRE            = regexp.MustCompile(`Secureframe ID: ([\w-]+)`)

	sleepMS = 250
)

func main() {
	flag.Parse()
	token := ""

	if *githubTokenFlag != "" {
		bs, err := ioutil.ReadFile(*githubTokenFlag)
		if err != nil {
			log.Panicf("readfile: %v", err)
		}
		token = strings.TrimSpace(string(bs))
	} else {
		log.Printf("--github-token-file not provided: skipping github calls")
	}

	ctx := context.Background()
	tc := oauth2.NewClient(ctx, oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token}))
	gc := github.NewClient(tc)

	tests, err := secureframe.GetDashboardTests(context.Background(), *companyIDFlag, *bearerTokenFlag, strings.Split(*keysFlag, ","))
	if err != nil {
		log.Panicf("error: %v", err)
	}

	parts := strings.Split(*githubRepoFlag, "/")
	org := parts[0]
	project := parts[1]

	issues := []*github.Issue{}
	if token != "" {
		issues, err = issue.Synced(ctx, gc, org, project)
		if err != nil {
			log.Panicf("synced: %v", err)
		}
	}

	log.Printf("syncing labels ...")
	if !*dryRunFlag {
		if err := issue.SyncLabels(ctx, gc, org, project, []string{issue.SyncLabel, *githubLabelFlag}); err != nil {
			log.Panicf("sync labels: %v", err)
		}
	}

	issuesByID := map[string]*github.Issue{}
	for _, i := range issues {
		id := ""
		match := idRE.FindStringSubmatch(i.GetBody())
		if len(match) > 0 {
			log.Printf("found match: %v", match)
			id = match[1]
			issuesByID[id] = i
		}
		log.Printf("issue[%s]: %+v", id, i)
	}

	testsByID := map[string]secureframe.Test{}
	lastWasMod := false
	updates := 0

	for x, t := range tests {
		// Avoid Github API DoS attack
		if lastWasMod && !*dryRunFlag {
			log.Printf("Sleeping ...")
			time.Sleep(time.Duration(updates*sleepMS) * time.Millisecond)
			updates++
			lastWasMod = false
		}

		testsByID[t.ID] = t

		log.Printf("Found open test #%d: %s: %s", x, t.ID, t.Description)

		t, err := secureframe.GetTest(context.Background(), *companyIDFlag, *bearerTokenFlag, t.ID)
		if err != nil {
			log.Panicf("error: %v", err)
		}

		if !t.Enabled {
			log.Printf("skipping: %s", t.DisabledJustification)
			continue
		}

		ft, err := issue.FromTest(t, *githubLabelFlag)
		if err != nil {
			log.Panicf("issue: %v", err)
		}

		i := issuesByID[t.ID]

		// Test does not exist in Github
		if i == nil {
			if t.Pass {
				log.Printf("Skipping passing test: %s", t.Description)
				continue
			}

			log.Printf("Creating %+v", ft)
			if !*dryRunFlag {
				if err := issue.Create(ctx, gc, org, project, ft); err != nil {
					log.Panicf("create: %v", err)
				}
				lastWasMod = true
			}
			continue
		}

		// Test is in Github, and open
		if i.GetClosedAt().IsZero() {
			// Close passing or disabled tests
			if t.Pass || !t.Enabled {
				log.Printf("Closing #%d ...", i.GetNumber())
				if !*dryRunFlag {
					if err := issue.Close(ctx, gc, org, project, i); err != nil {
						log.Panicf("close: %v", err)
					}
					lastWasMod = true
				}
				continue
			}

			// Update failing tests
			if i.GetBody() != ft.Body || i.GetTitle() != ft.Title {
				log.Printf("Updating #%d ...", i.GetNumber())
				if !*dryRunFlag {
					if err := issue.Update(ctx, gc, org, project, i.GetNumber(), ft); err != nil {
						log.Panicf("update: %v", err)
					}
					lastWasMod = true
				}
			}
			continue
		}

		// Test is in Github, but closed
		if !t.Pass {
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
