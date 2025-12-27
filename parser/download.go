package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/cheggaaa/pb/v3"
)

func IsBlockedHTML(content string) error {
	badMarkers := []string{
		"Verify you are human",
		"Checking your browser",
		"Enable JavaScript",
		"Access denied",
	}
	for _, m := range badMarkers {
		if strings.Contains(content, m) {
			return fmt.Errorf("blocked page marker detected: %s", m)
		}
	}
	return nil
}

func SaveRawHTML(html string, dir, filename string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	fpath := filepath.Join(dir, filename)
	return os.WriteFile(fpath, []byte(html), 0o644)
}

func FetchURLHTML(url string) (string, error) {
	doc, err := FetchPage(url)
	if err != nil {
		return "", err
	}
	html, err := doc.Html()
	if err != nil {
		return "", err
	}
	return html, nil
}

func DownloadHLTVArticles(articles []map[string]string, corpusDir string, bar *pb.ProgressBar, stats *Statistics, mu *sync.Mutex, workers int) {
	jobsChan := make(chan map[string]string, len(articles))
	var wg sync.WaitGroup

	numWorkers := workers
	if numWorkers <= 0 {
		numWorkers = 1
	}

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for articleInfo := range jobsChan {
				articleID := articleInfo["id"]
				articleSlug := articleInfo["slug"]

				hltvDir := filepath.Join(corpusDir, "hltv/raw")
				safeID := SanitizeFilename(articleID)
				htmlFilename := safeID + ".html"
				htmlPath := filepath.Join(hltvDir, htmlFilename)

				if _, err := os.Stat(htmlPath); err == nil {
					fmt.Printf("[HLTV] Skipped (raw html exists): %s\n", articleID)
					bar.Increment()
					continue
				}

				url := BuildHLTVURL(articleID, articleSlug)

				html, err := FetchURLHTML(url)
				if err != nil {
					fmt.Printf("[HLTV] Failed to download %s: %v\n", articleID, err)
					bar.Increment()
					time.Sleep(500 * time.Millisecond)
					continue
				}

				if err := IsBlockedHTML(html); err != nil {
					fmt.Printf("[HLTV] Blocked (anti-bot) %s: %v\n", articleID, err)
					bar.Increment()
					_ = SaveRawHTML(html, filepath.Join(hltvDir, "blocked"), htmlFilename)
					time.Sleep(300 * time.Millisecond)
					continue
				}

				if err := SaveRawHTML(html, hltvDir, htmlFilename); err != nil {
					fmt.Printf("[HLTV] Failed to save raw html %s: %v\n", articleID, err)
					bar.Increment()
					time.Sleep(500 * time.Millisecond)
					continue
				}

				mu.Lock()
				stats.HLTVArticles++
				stats.TotalArticles++
				stats.TotalSize += int64(len(html))
				mu.Unlock()

				fmt.Printf("[HLTV] Downloaded raw HTML: %s\n", articleID)
				bar.Increment()
				time.Sleep(300 * time.Millisecond)
			}
		}()
	}

	for _, articleInfo := range articles {
		jobsChan <- articleInfo
	}
	close(jobsChan)

	wg.Wait()
}

func DownloadCybersportArticles(articles []map[string]string, corpusDir string, bar *pb.ProgressBar, stats *Statistics, mu *sync.Mutex, workers int) {
	jobsChan := make(chan map[string]string, len(articles))
	var wg sync.WaitGroup

	numWorkers := workers
	if numWorkers <= 0 {
		numWorkers = 2
	}

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for articleInfo := range jobsChan {
				tag := articleInfo["tag"]
				slug := articleInfo["slug"]
				safeName := SanitizeFilename(tag + "__" + slug)

				csDir := filepath.Join(corpusDir, "cybersport/raw")
				htmlFilename := safeName + ".html"
				htmlPath := filepath.Join(csDir, htmlFilename)

				if _, err := os.Stat(htmlPath); err == nil {
					fmt.Printf("[Cybersport] Skipped (raw html exists): %s/%s\n", tag, slug)
					bar.Increment()
					continue
				}

				url := BuildCybersportURL(tag, slug)

				html, err := FetchURLHTML(url)
				if err != nil {
					fmt.Printf("[Cybersport] Failed to download %s/%s: %v\n", tag, slug, err)
					bar.Increment()
					time.Sleep(500 * time.Millisecond)
					continue
				}

				if err := IsBlockedHTML(html); err != nil {
					fmt.Printf("[Cybersport] Blocked (anti-bot) %s/%s: %v\n", tag, slug, err)
					bar.Increment()
					_ = SaveRawHTML(html, filepath.Join(csDir, "blocked"), htmlFilename)
					time.Sleep(300 * time.Millisecond)
					continue
				}

				if err := SaveRawHTML(html, csDir, htmlFilename); err != nil {
					fmt.Printf("[Cybersport] Failed to save raw html %s/%s: %v\n", tag, slug, err)
					bar.Increment()
					time.Sleep(500 * time.Millisecond)
					continue
				}

				mu.Lock()
				stats.CybersportArticles++
				stats.TotalArticles++
				stats.TotalSize += int64(len(html))
				mu.Unlock()

				fmt.Printf("[Cybersport] Downloaded raw HTML: %s/%s\n", tag, slug)
				bar.Increment()
				time.Sleep(300 * time.Millisecond)
			}
		}()
	}

	for _, articleInfo := range articles {
		jobsChan <- articleInfo
	}
	close(jobsChan)

	wg.Wait()
}

func BuildHLTVURL(articleID, slug string) string {
	return fmt.Sprintf("https://www.hltv.org/news/%s/%s", articleID, slug)
}

func BuildCybersportURL(tag, slug string) string {
	return fmt.Sprintf("https://www.cybersport.ru/tags/%s/%s", tag, slug)
}
