package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/cheggaaa/pb/v3"
)

func ParseHLTVArticleFromHTML(html string, sourceURL string) (*Article, error) {
	if strings.TrimSpace(html) == "" {
		return nil, fmt.Errorf("empty html")
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, err
	}

	title := strings.TrimSpace(doc.Find("h1").First().Text())
	var contentBuilder strings.Builder
	doc.Find(".article-content p").Each(func(i int, s *goquery.Selection) {
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

	if err := IsEmptyHLTVArticle(article); err != nil {
		return nil, err
	}

	return article, nil
}

func ProcessHLTVRawFiles(corpusDir string) error {
	rawDir := filepath.Join(corpusDir, "hltv/raw")
	parsedDir := filepath.Join(corpusDir, "hltv/parsed")

	os.MkdirAll(parsedDir, 0755)

	entries, err := os.ReadDir(rawDir)
	if err != nil {
		return fmt.Errorf("failed to read raw directory: %w", err)
	}

	htmlFiles := 0
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".html") {
			htmlFiles++
		}
	}

	if htmlFiles == 0 {
		return fmt.Errorf("no html files found in %s", rawDir)
	}

	bar := pb.New(htmlFiles)
	bar.SetTemplateString(`[{{counters . }}] {{bar . }} {{percent . }} | {{etime . }}`)
	bar.Start()

	processed := 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".html") {
			continue
		}

		rawPath := filepath.Join(rawDir, entry.Name())
		htmlBytes, err := os.ReadFile(rawPath)
		if err != nil {
			bar.Increment()
			continue
		}

		html := string(htmlBytes)
		article, err := ParseHLTVArticleFromHTML(html, "")
		if err != nil {
			bar.Increment()
			continue
		}

		textFileName := strings.TrimSuffix(entry.Name(), ".html") + ".txt"
		textPath := filepath.Join(parsedDir, textFileName)

		content := fmt.Sprintf("Title: %s\n\n%s", article.Title, article.Content)
		os.WriteFile(textPath, []byte(content), 0644)

		processed++
		bar.Increment()
	}

	bar.Finish()
	fmt.Printf("Processed: %d/%d files\n\n", processed, htmlFiles)

	return nil
}
