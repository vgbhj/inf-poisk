package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// ParseHLTVArticle parses a single HLTV article
func ParseHLTVArticle(id string, slug string) (*Article, error) {
	url := fmt.Sprintf("https://www.hltv.org/news/%s/%s", id, slug)
	doc, err := FetchPage(url)
	if err != nil {
		return nil, err
	}

	title := doc.Find("h1").First().Text()
	title = strings.TrimSpace(title)

	var contentBuilder strings.Builder
	var foundContent bool

	// HLTV stores news in specific structure
	contentSelectors := []string{
		".news-content",
		".article-content",
		".standard-box .bodyshot-team",
		"article.newspost",
		".newstext",
	}

	for _, selector := range contentSelectors {
		doc.Find(selector).Each(func(i int, container *goquery.Selection) {
			container.Find("p, blockquote").Each(func(j int, s *goquery.Selection) {
				text := strings.TrimSpace(s.Text())
				if text == "" || len(text) < 15 {
					return
				}

				if goquery.NodeName(s) == "blockquote" {
					contentBuilder.WriteString("> ")
				}
				contentBuilder.WriteString(text)
				contentBuilder.WriteString("\n\n")
				foundContent = true
			})
		})

		if foundContent {
			break
		}
	}

	// Fallback method if no content found
	if !foundContent {
		doc.Find("p, blockquote").Each(func(i int, s *goquery.Selection) {
			// Check if element is not inside script/style/nav
			if s.Closest("script, style, noscript, nav, header, footer").Length() > 0 {
				return
			}

			className, _ := s.Parent().Attr("class")
			if strings.Contains(className, "comment") ||
				strings.Contains(className, "forum") ||
				strings.Contains(className, "navigation") {
				return
			}

			text := strings.TrimSpace(s.Text())
			if text == "" || len(text) < 15 {
				return
			}

			// Strict filtering of JS/CSS content
			if strings.Contains(text, "{") && strings.Contains(text, "}") {
				return
			}
			if strings.Contains(text, "window.") || strings.Contains(text, "document.") {
				return
			}
			if strings.Contains(text, "appendChild") || strings.Contains(text, "createElement") {
				return
			}

			if goquery.NodeName(s) == "blockquote" {
				contentBuilder.WriteString("> ")
			}
			contentBuilder.WriteString(text)
			contentBuilder.WriteString("\n\n")
		})
	}

	content := strings.TrimSpace(contentBuilder.String())

	return &Article{
		ID:      id,
		URL:     url,
		Title:   title,
		Content: content,
		Source:  "hltv",
	}, nil
}

// GetHLTVNewsIDs collects all HLTV news article IDs and slugs
func GetHLTVNewsIDs() ([]map[string]string, error) {
	var articles []map[string]string
	seen := make(map[string]bool)

	re := regexp.MustCompile(`/news/(\d+)/(?:[^/]+)`)

	months := []string{
		"january", "february", "march", "april",
		"may", "june", "july", "august",
		"september", "october", "november", "december",
	}

	currentYear := time.Now().Year()
	consecutiveErrors := 0

	for year := 2006; year <= currentYear; year++ {
		for _, month := range months {
			url := fmt.Sprintf("https://www.hltv.org/news/archive/%d/%s", year, month)

			doc, err := FetchPage(url)
			if err != nil {
				fmt.Printf("  %s %d: load error - %v\n", month, year, err)
				consecutiveErrors++
				if consecutiveErrors >= 3 {
					fmt.Println("multiple 429s detected — sleeping 5 minutes")
					consecutiveErrors = 0
					time.Sleep(5 * time.Minute)
				}
				continue
			}

			consecutiveErrors = 0

			monthCount := 0
			doc.Find("a[href*='/news/']").Each(func(_ int, s *goquery.Selection) {
				href, _ := s.Attr("href")
				m := re.FindStringSubmatch(href)
				if len(m) >= 2 {
					r2 := regexp.MustCompile(`/news/(\d+)/([^/]+)`)
					m2 := r2.FindStringSubmatch(href)
					if len(m2) == 3 {
						key := m2[1] + "/" + m2[2]
						if !seen[key] {
							seen[key] = true
							articles = append(articles, map[string]string{
								"id":   m2[1],
								"slug": m2[2],
							})
							monthCount++
						}
					}
				}
			})

			if monthCount > 0 {
				fmt.Printf("  %s %d: found %d articles\n", month, year, monthCount)
			}

			time.Sleep(1500 * time.Millisecond)
		}
	}

	return articles, nil
}

// SaveArticle saves article to file
func SaveArticle(article *Article, baseDir string) error {
	var dir string
	if article.Source == "hltv" {
		dir = filepath.Join(baseDir, "hltv")
	} else {
		dir = filepath.Join(baseDir, "cybersport", article.Tag)
	}

	err := os.MkdirAll(dir, 0755)
	if err != nil {
		return err
	}

	safeID := SanitizeFilename(article.ID)
	filename := filepath.Join(dir, safeID+".txt")
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = fmt.Fprintf(file, "Title: %s\nURL: %s\nSource: %s\n\n%s", article.Title, article.URL, article.Source, article.Content)
	return err
}
