package issue

import (
	"bytes"
	_ "embed"
	"fmt"
	"html"
	"text/template"

	"github.com/chainguard-dev/secureframe-github-sync/pkg/secureframe"
)

//go:embed issue.tmpl
var issueTmpl string

type IssueForm struct {
	Title string
	Body  string
	Tags  []string
}

func FromTest(t secureframe.Test) (IssueForm, error) {
	i := IssueForm{
		Title: fmt.Sprintf("%s: %s", t.Key, t.Title),
		Tags:  t.ReportKeys,
	}
	tmpl, err := template.New("issue").Funcs(template.FuncMap{
		"Unescape":   html.UnescapeString,
		"AssertWork": secureframe.AssertWork,
	}).Parse(issueTmpl)
	if err != nil {
		return i, fmt.Errorf("parse: %v", err)
	}

	data := struct {
		Test secureframe.Test
	}{
		Test: t,
	}

	var tpl bytes.Buffer
	if err = tmpl.Execute(&tpl, data); err != nil {
		return i, fmt.Errorf("exec: %w", err)
	}

	i.Body = tpl.String()
	return i, nil
}

func Create(i IssueForm) error {
	return nil
}
