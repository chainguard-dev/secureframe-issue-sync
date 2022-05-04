package secureframe

import (
	"bytes"
	"fmt"
	"log"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// ParseIncompleteTests parses the output of https://app.secureframe.com/dashboard/incomplete-tests/soc2-beta (unused)
func ParseIncompleteTests(bs []byte) ([]Test, error) {
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(bs))
	if err != nil {
		return nil, fmt.Errorf("document: %w", err)
	}

	ts := []Test{}

	doc.Find("div.ant-collapse-item").Each(func(i int, s *goquery.Selection) {
		t := Test{}

		s.Find("h5").Each(func(i int, s *goquery.Selection) {
			t.Title = strings.TrimSpace(s.Text())
			log.Printf("--- found title: %s", t.Title)
		})

		snippets := []string{}

		s.Find("div.ant-collapse-content-box").Each(func(i int, as *goquery.Selection) {
			as.Find("div").Each(func(i int, ds *goquery.Selection) {
				h, _ := ds.Html()
				log.Printf("found div: %s", h)
				if strings.TrimSpace(ds.Text()) != "" {
					snippets = append(snippets, strings.TrimSpace(ds.Text()))
				}
			})

			as.Find("span").Each(func(i int, ds *goquery.Selection) {
				h, _ := ds.Html()
				log.Printf("found span: %s (text=%s)", h, ds.Text())
				if strings.TrimSpace(ds.Text()) != "" {
					snippets = append(snippets, strings.TrimSpace(ds.Text()))
				}
			})
		})

		t.FailMessage = strings.Join(snippets, "\n")
		ts = append(ts, t)
	})

	return ts, nil
}
