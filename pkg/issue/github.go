package issue

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/google/go-github/v44/github"
)

var (
	SyncLabel     = "sframe"
	DisabledLabel = "disabled"
	PassingLabel  = "passing"

	open   = "open"
	closed = "closed"
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

func SyncLabels(ctx context.Context, gc *github.Client, org string, project string, labels []string) error {
	desc := "Added by secureframe-issue-sync"

	for _, l := range labels {
		_, _, err := gc.Issues.GetLabel(ctx, org, project, l)
		// not there?
		if err != nil {
			label := &github.Label{
				Name:        &l,
				Description: &desc,
			}
			_, _, err := gc.Issues.CreateLabel(ctx, org, project, label)
			if err != nil {
				return fmt.Errorf("create %+v: %w", label, err)
			}
		}
	}
	return nil
}

// Create creates an issue
func Create(ctx context.Context, gc *github.Client, org string, project string, ft IssueForm) error {
	i := &github.IssueRequest{
		Title:  &ft.Title,
		Body:   &ft.Body,
		Labels: &ft.Labels,
		State:  &open,
	}
	_, _, err := gc.Issues.Create(ctx, org, project, i)
	return err
}

// Update creates an issue
func Update(ctx context.Context, gc *github.Client, org string, project string, id int, ft IssueForm) error {
	i := &github.IssueRequest{
		Title:  &ft.Title,
		Body:   &ft.Body,
		Labels: &ft.Labels,
		State:  &open,
	}
	_, _, err := gc.Issues.Edit(ctx, org, project, id, i)
	return err
}

// Close closes an issue
func Close(ctx context.Context, gc *github.Client, org string, project string, i *github.Issue, label string) error {
	title := i.GetTitle()
	body := i.GetBody()
	labels := []string{}
	for _, l := range i.Labels {
		labels = append(labels, l.GetName())
	}
	labels = append(labels, label)

	ir := &github.IssueRequest{
		Title:  &title,
		Body:   &body,
		State:  &closed,
		Labels: &labels,
	}
	_, _, err := gc.Issues.Edit(ctx, org, project, i.GetNumber(), ir)
	return err
}
