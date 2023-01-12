package issue

import (
	"bytes"
	_ "embed"
	"fmt"
	"html"
	"strings"
	"text/template"

	"github.com/chainguard-dev/secureframe-github-sync/pkg/secureframe"
)

//go:embed issue.tmpl
var issueTmpl string

type IssueForm struct {
	Title  string
	Body   string
	Labels []string
}

func FromTest(t secureframe.Test, label string, reportKey string) (IssueForm, error) {
	reportLabel, _, _ := strings.Cut(reportKey, "_")

	i := IssueForm{
		Title:  fmt.Sprintf("%s: %s", t.V2.Key, t.V2.Title),
		Labels: []string{SyncLabel, label, reportLabel},
	}

	tmpl, err := template.New("issue").Funcs(template.FuncMap{
		"Unescape":   html.UnescapeString,
		"AssertWork": secureframe.AssertWork,
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
