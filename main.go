package main

import (
	"context"
	"flag"
	"log"
	"strings"

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

	tests, err := secureframe.DashboardTests(context.Background(), strings.Split(*keysFlag, ","), *companyIDFlag, *bearerTokenFlag)
	if err != nil {
		log.Panicf("error: %v", err)
	}

	for x, t := range tests {
		log.Printf("Found open test #%d: %s: %s", x, t.ID, t.Description)
	}
}
