package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/cheggaaa/pb/v3"
)

// DownloadHLTVArticles downloads HLTV articles in parallel
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

				hltvDir := filepath.Join(corpusDir, "hltv")
				safeID := SanitizeFilename(articleID)
				filePath := filepath.Join(hltvDir, safeID+".txt")

				if _, err := os.Stat(filePath); err == nil {
					fmt.Printf("[HLTV] Skipped (already downloaded): %s\n", articleID)
					bar.Increment()
					continue
				}

				article, err := ParseHLTVArticle(articleID, articleSlug)
				if err != nil {
					fmt.Printf("[HLTV] Failed to parse %s: %v\n", articleID, err)
					bar.Increment()
					time.Sleep(500 * time.Millisecond)
					continue
				}

				if err := IsEmptyHLTVArticle(article); err != nil {
					fmt.Printf(
						"[HLTV] Skipped empty article %s (%s): %v\n",
						articleID,
						article.URL,
						err,
					)
					bar.Increment()
					time.Sleep(300 * time.Millisecond)
					continue
				}

				err = SaveArticle(article, corpusDir)
				if err != nil {
					fmt.Printf("[HLTV] Failed to save %s: %v\n", articleID, err)
					bar.Increment()
					time.Sleep(500 * time.Millisecond)
					continue
				}

				mu.Lock()
				stats.HLTVArticles++
				stats.TotalArticles++
				stats.TotalSize += int64(len(article.Content))
				mu.Unlock()

				fmt.Printf("[HLTV] Downloaded: %s - %s\n", articleID, article.Title)
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

// DownloadCybersportArticles downloads Cybersport articles in parallel
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
				article, err := ParseCybersportArticle(articleInfo["tag"], articleInfo["slug"])
				if err != nil {
					fmt.Printf("[Cybersport] Failed to parse %s/%s: %v\n", articleInfo["tag"], articleInfo["slug"], err)
					bar.Increment()
					continue
				}

				err = SaveArticle(article, corpusDir)
				if err != nil {
					fmt.Printf("[Cybersport] Failed to save %s: %v\n", article.ID, err)
					bar.Increment()
					continue
				}

				mu.Lock()
				stats.CybersportArticles++
				stats.TotalArticles++
				stats.TotalSize += int64(len(article.Content))
				mu.Unlock()

				bar.Increment()
			}
		}()
	}

	for _, articleInfo := range articles {
		jobsChan <- articleInfo
	}
	close(jobsChan)

	wg.Wait()
}
