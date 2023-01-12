package issue

import (
	"bytes"
	_ "embed"
	"fmt"
	"html"
	"log"
	"strings"
	"text/template"

	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/chainguard-dev/secureframe-issue-sync/pkg/secureframe"
)

//go:embed issue.tmpl
var issueTmpl string

type IssueForm struct {
	Title  string
	Body   string
	Labels []string
}

func assertWork(a secureframe.AssertionResult) string {
	work := strings.TrimSpace(a.FailMessage)
	// TODO: Don't hardcode this
	url := "https://app.secureframe.com/dashboard/incomplete-tests/soc2-beta"

	if work == "" {
		work = fmt.Sprintf("Upload evidence for %s", a.Data.Type)
	}

	if a.Resourceable != nil {
		return fmt.Sprintf("%s: %s", secureframe.ResourceID(*a.Resourceable), work)
	}

	if strings.HasPrefix(work, "Upload") {
		work := strings.TrimRight(work, ".")
		return work + " to " + url
	}

	if strings.HasPrefix(work, "Select") && strings.HasSuffix(work, "Policy") {
		return work + " at " + url
	}

	return work
}

func makeMarkdown(html string) string {
	converter := md.NewConverter("", true, nil)
	markdown, err := converter.ConvertString(html)
	if err != nil {
		log.Printf("markdown conversion failed: %v", err)
		return fmt.Sprintf("markdown conversion failed: %v\nraw html: %s", err, html)
	}
	return markdown
}

func FromTest(t secureframe.Test, additionalLabel string, reportKey string) (IssueForm, error) {
	reportLabel, _, _ := strings.Cut(reportKey, "_")
	labels := []string{SyncLabel, reportLabel}
	if additionalLabel != "" {
		labels = append(labels, additionalLabel)
	}

	i := IssueForm{
		Title:  fmt.Sprintf("%s: %s", t.V2.Key, t.V2.Title),
		Labels: labels,
	}

	tmpl, err := template.New("issue").Funcs(template.FuncMap{
		"Unescape":   html.UnescapeString,
		"AssertWork": assertWork,
		"Markdown":   makeMarkdown,
	}).Parse(issueTmpl)
	if err != nil {
		return i, fmt.Errorf("parse: %v", err)
	}

	data := struct {
		Test      secureframe.Test
		ReportKey string
	}{
		Test:      t,
		ReportKey: reportKey,
	}

	var tpl bytes.Buffer
	if err = tmpl.Execute(&tpl, data); err != nil {
		return i, fmt.Errorf("exec: %w", err)
	}

	i.Body = tpl.String()
	return i, nil
}
