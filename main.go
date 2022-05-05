package main

import (
	"context"
	_ "embed"
	"flag"
	"io/ioutil"
	"log"
	"strings"

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

	issues := []*github.Issue{}
	if token != "" {
		parts := strings.Split(*githubRepoFlag, "/")
		org := parts[0]
		project := parts[1]
		issues, err = issue.Synced(ctx, gc, org, project)
		if err != nil {
			log.Panicf("synced: %v", err)
		}
	}

	for _, i := range issues {
		log.Printf("issue: %+v", i)
	}

	for x, t := range tests {
		log.Printf("Found open test #%d: %s: %s", x, t.ID, t.Description)

		t, err := secureframe.GetTest(context.Background(), *companyIDFlag, *bearerTokenFlag, t.ID)
		if err != nil {
			log.Panicf("error: %v", err)
		}

		if !t.Enabled {
			log.Printf("skipping: %s", t.DisabledJustification)
			continue
		}

		for _, r := range t.AssertionResults.Collection {
			log.Printf("assertion: %+v", r)
			if r.Resourceable != nil {
				log.Printf("resourceable: %+v", r.Resourceable)
			}
		}
		i, err := issue.FromTest(t)
		if err != nil {
			log.Panicf("issue: %v", err)
		}

		if !*dryRunFlag {
			if err := issue.Create(i); err != nil {
				log.Panicf("create: %v", err)
			}
		}

		log.Printf("issue: %+v", i)
	}
}
