package issue

import (
	"context"
	"log"
	"net/http"

	"github.com/google/go-github/v44/github"
)

var (
	SyncLabel = "secureframe"
)

func NewClient(c *http.Client) *github.Client {
	return github.NewClient(c)
}

// Synced returns a list of synced issues within a project
func Synced(ctx context.Context, gc *github.Client, org string, project string) ([]*github.Issue, error) {
	result := []*github.Issue{}
	opts := &github.IssueListByRepoOptions{
		State:     "all",
		Sort:      "updated",
		Direction: "desc",
		Labels:    []string{SyncLabel},
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	}

	log.Printf("Gathering issues for %s/%s: %+v", org, project, opts)
	for page := 1; page != 0; {
		opts.ListOptions.Page = page
		issues, resp, err := gc.Issues.ListByRepo(ctx, org, project, opts)
		if err != nil {
			return result, err
		}

		if len(issues) == 0 {
			break
		}

		page = resp.NextPage

		for _, i := range issues {
			if i.IsPullRequest() {
				continue
			}

			result = append(result, i)
		}
	}

	return result, nil
}
