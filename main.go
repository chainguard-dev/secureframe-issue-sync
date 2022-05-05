package main

import (
	"context"
	_ "embed"
	"flag"
	"log"
	"strings"

	"github.com/chainguard-dev/secureframe-github-sync/pkg/github"
	"github.com/chainguard-dev/secureframe-github-sync/pkg/secureframe"
)

var (
	githubTokenFlag = flag.String("github-token-file", "", "path to github token file")
	dryRunFlag      = flag.Bool("dry-run", false, "dry-run mode")
	bearerTokenFlag = flag.String("bearer-token", "", "secureframe bearer token")
	keysFlag        = flag.String("report-keys", "soc2_alpha", "comma-delimited list of report keys to use")
	companyIDFlag   = flag.String("company-id", "079b854c-c53a-4c71-bfb8-f9e87b13b6c4", "secureframe company user ID")
	githubRepoFlag  = flag.String("github-repo", "", "github repo to open issues against")
)

func main() {
	flag.Parse()

	// TODO: Use OAuth instead of passing bearer tokens around (HELP WANTED

	tests, err := secureframe.GetDashboardTests(context.Background(), *companyIDFlag, *bearerTokenFlag, strings.Split(*keysFlag, ","))
	if err != nil {
		log.Panicf("error: %v", err)
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
		i, err := github.IssueFromTest(t)
		if err != nil {
			log.Panicf("issue: %v", err)
		}

		log.Printf("issue: %+v", i)
	}
}
