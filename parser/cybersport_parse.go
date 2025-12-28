package parser

import (
	"fmt"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

func ParseCybersportArticleFromHTML(html string, sourceURL string) (*Article, error) {
	if strings.TrimSpace(html) == "" {
		return nil, fmt.Errorf("empty html")
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, err
	}

	title := strings.TrimSpace(doc.Find("h1").First().Text())
	var contentBuilder strings.Builder
	doc.Find(".post-content p").Each(func(i int, s *goquery.Selection) {
		text := strings.TrimSpace(s.Text())
		if text != "" {
			contentBuilder.WriteString(text)
			contentBuilder.WriteString("\n\n")
		}
	})

	article := &Article{
		ID:      "",
		URL:     sourceURL,
		Title:   title,
		Content: contentBuilder.String(),
	}

	return article, nil
}
