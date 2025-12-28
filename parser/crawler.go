package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/cheggaaa/pb/v3"
)

type CrawlerConfig struct {
	Database      *Database
	CorpusDir     string
	DelayMs       int
	ReCrawl       bool
	ReCrawlInt    int
	ResumeFromURL string
}

func DownloadHLTVArticlesWithDB(articles []map[string]string, cfg *CrawlerConfig, bar *pb.ProgressBar, stats *Statistics, mu *sync.Mutex, workers int) {
	jobsChan := make(chan map[string]string, len(articles))
	var wg sync.WaitGroup

	numWorkers := workers
	if numWorkers <= 0 {
		numWorkers = 2
	}

	resumeMode := cfg.ResumeFromURL != ""
	skipUntilFound := resumeMode

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for articleInfo := range jobsChan {
				articleID := articleInfo["id"]
				articleSlug := articleInfo["slug"]
				url := BuildHLTVURL(articleID, articleSlug)

				normalizedURL, err := NormalizeURL(url)
				if err != nil {
					fmt.Printf("[HLTV] Failed to normalize URL %s: %v\n", url, err)
					bar.Increment()
					continue
				}

				// Resume logic: skip until we find the resume point
				if skipUntilFound {
					if normalizedURL == cfg.ResumeFromURL {
						skipUntilFound = false
					} else {
						bar.Increment()
						continue
					}
				}

				hltvDir := filepath.Join(cfg.CorpusDir, "hltv/raw")
				safeID := SanitizeFilename(articleID)
				htmlFilename := safeID + ".html"
				htmlPath := filepath.Join(hltvDir, htmlFilename)

				var html string
				if _, err := os.Stat(htmlPath); err == nil {
					// File exists, read it
					htmlBytes, err := os.ReadFile(htmlPath)
					if err == nil {
						html = string(htmlBytes)
						fmt.Printf("[HLTV] Using existing file: %s\n", articleID)
					}
				}

				// Download if not found in file
				if html == "" {
					html, err = FetchURLHTML(url)
					if err != nil {
						fmt.Printf("[HLTV] Failed to download %s: %v\n", articleID, err)
						bar.Increment()
						time.Sleep(time.Duration(cfg.DelayMs) * time.Millisecond)
						continue
					}

					if err := IsBlockedHTML(html); err != nil {
						fmt.Printf("[HLTV] Blocked (anti-bot) %s: %v\n", articleID, err)
						bar.Increment()
						time.Sleep(time.Duration(cfg.DelayMs) * time.Millisecond)
						continue
					}

					if err := SaveRawHTML(html, hltvDir, htmlFilename); err != nil {
						fmt.Printf("[HLTV] Failed to save raw html %s: %v\n", articleID, err)
					}
				}

				if cfg.Database != nil {
					exists, err := cfg.Database.DocumentExists(normalizedURL)
					if err == nil && exists {
						changed, err := cfg.Database.HasDocumentChanged(normalizedURL, html)
						if err == nil {
							if changed {
								if err := cfg.Database.SaveDocument(normalizedURL, html, "hltv"); err != nil {
									fmt.Printf("[HLTV] Failed to update in DB %s: %v\n", articleID, err)
								} else {
									fmt.Printf("[HLTV] Updated in DB (changed): %s\n", articleID)
								}
							} else {
								cfg.Database.UpdateLastChecked(normalizedURL)
								fmt.Printf("[HLTV] Document unchanged, updated timestamp: %s\n", articleID)
							}
						} else {
							fmt.Printf("[HLTV] Error checking document change: %v\n", err)
						}
					} else {
						if err := cfg.Database.SaveDocument(normalizedURL, html, "hltv"); err != nil {
							fmt.Printf("[HLTV] Failed to save to DB %s: %v\n", articleID, err)
						} else {
							fmt.Printf("[HLTV] Saved to DB: %s\n", articleID)
						}
					}
				}

				mu.Lock()
				stats.HLTVArticles++
				stats.TotalArticles++
				stats.TotalSize += int64(len(html))
				mu.Unlock()

				bar.Increment()
				time.Sleep(time.Duration(cfg.DelayMs) * time.Millisecond)
			}
		}()
	}

	for _, articleInfo := range articles {
		jobsChan <- articleInfo
	}
	close(jobsChan)

	wg.Wait()
}

func DownloadCybersportArticlesWithDB(articles []map[string]string, cfg *CrawlerConfig, bar *pb.ProgressBar, stats *Statistics, mu *sync.Mutex, workers int) {
	jobsChan := make(chan map[string]string, len(articles))
	var wg sync.WaitGroup

	numWorkers := workers
	if numWorkers <= 0 {
		numWorkers = 2
	}

	resumeMode := cfg.ResumeFromURL != ""
	skipUntilFound := resumeMode

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for articleInfo := range jobsChan {
				tag := articleInfo["tag"]
				slug := articleInfo["slug"]
				url := BuildCybersportURL(tag, slug)

				normalizedURL, err := NormalizeURL(url)
				if err != nil {
					fmt.Printf("[Cybersport] Failed to normalize URL %s: %v\n", url, err)
					bar.Increment()
					continue
				}

				if skipUntilFound {
					if normalizedURL == cfg.ResumeFromURL {
						skipUntilFound = false
					} else {
						bar.Increment()
						continue
					}
				}

				csDir := filepath.Join(cfg.CorpusDir, "cybersport/raw")
				safeName := SanitizeFilename(tag + "__" + slug)
				htmlFilename := safeName + ".html"
				htmlPath := filepath.Join(csDir, htmlFilename)

				var html string
				if _, err := os.Stat(htmlPath); err == nil {
					// File exists, read it
					htmlBytes, err := os.ReadFile(htmlPath)
					if err == nil {
						html = string(htmlBytes)
						fmt.Printf("[Cybersport] Using existing file: %s/%s\n", tag, slug)
					}
				}

				// Download if not found in file
				if html == "" {
					html, err = FetchURLHTML(url)
					if err != nil {
						fmt.Printf("[Cybersport] Failed to download %s/%s: %v\n", tag, slug, err)
						bar.Increment()
						time.Sleep(time.Duration(cfg.DelayMs) * time.Millisecond)
						continue
					}

					if err := IsBlockedHTML(html); err != nil {
						fmt.Printf("[Cybersport] Blocked (anti-bot) %s/%s: %v\n", tag, slug, err)
						bar.Increment()
						_ = SaveRawHTML(html, filepath.Join(csDir, "blocked"), htmlFilename)
						time.Sleep(time.Duration(cfg.DelayMs) * time.Millisecond)
						continue
					}

					if err := SaveRawHTML(html, csDir, htmlFilename); err != nil {
						fmt.Printf("[Cybersport] Failed to save raw html %s/%s: %v\n", tag, slug, err)
					}
				}

				if cfg.Database != nil {
					exists, err := cfg.Database.DocumentExists(normalizedURL)
					if err == nil && exists {
						changed, err := cfg.Database.HasDocumentChanged(normalizedURL, html)
						if err == nil {
							if changed {
								if err := cfg.Database.SaveDocument(normalizedURL, html, "cybersport"); err != nil {
									fmt.Printf("[Cybersport] Failed to update in DB %s/%s: %v\n", tag, slug, err)
								} else {
									fmt.Printf("[Cybersport] Updated in DB (changed): %s/%s\n", tag, slug)
								}
							} else {
								cfg.Database.UpdateLastChecked(normalizedURL)
								fmt.Printf("[Cybersport] Document unchanged, updated timestamp: %s/%s\n", tag, slug)
							}
						} else {
							fmt.Printf("[Cybersport] Error checking document change: %v\n", err)
						}
					} else {
						if err := cfg.Database.SaveDocument(normalizedURL, html, "cybersport"); err != nil {
							fmt.Printf("[Cybersport] Failed to save to DB %s/%s: %v\n", tag, slug, err)
						} else {
							fmt.Printf("[Cybersport] Saved to DB: %s/%s\n", tag, slug)
						}
					}
				}

				mu.Lock()
				stats.CybersportArticles++
				stats.TotalArticles++
				stats.TotalSize += int64(len(html))
				mu.Unlock()

				bar.Increment()
				time.Sleep(time.Duration(cfg.DelayMs) * time.Millisecond)
			}
		}()
	}

	for _, articleInfo := range articles {
		jobsChan <- articleInfo
	}
	close(jobsChan)

	wg.Wait()
}

func AddExistingPagesToDB(corpusDir string, db *Database, source string) error {
	var baseDir string
	var urlBuilder func(map[string]string) string

	if source == "hltv" {
		baseDir = filepath.Join(corpusDir, "hltv/raw")
		urlBuilder = func(info map[string]string) string {
			return BuildHLTVURL(info["id"], info["slug"])
		}
	} else if source == "cybersport" {
		baseDir = filepath.Join(corpusDir, "cybersport/raw")
		urlBuilder = func(info map[string]string) string {
			return BuildCybersportURL(info["tag"], info["slug"])
		}
	} else {
		return fmt.Errorf("unknown source: %s", source)
	}

	var articles []map[string]string
	var err error
	if source == "hltv" {
		csvPath := filepath.Join(corpusDir, "hltv_links.csv")
		articles, err = ReadHLTVCSV(csvPath)
		if err != nil {
			return fmt.Errorf("failed to read HLTV CSV: %w", err)
		}
	} else {
		csvPath := filepath.Join(corpusDir, "cybersport_links.csv")
		articles, err = ReadCybersportCSV(csvPath)
		if err != nil {
			return fmt.Errorf("failed to read Cybersport CSV: %w", err)
		}
	}

	added := 0
	skipped := 0

	for _, articleInfo := range articles {
		url := urlBuilder(articleInfo)
		normalizedURL, err := NormalizeURL(url)
		if err != nil {
			fmt.Printf("[%s] Failed to normalize URL %s: %v\n", source, url, err)
			continue
		}

		exists, err := db.DocumentExists(normalizedURL)
		if err != nil {
			fmt.Printf("[%s] Error checking existence: %v\n", source, err)
			continue
		}
		if exists {
			skipped++
			continue
		}

		var htmlPath string
		if source == "hltv" {
			safeID := SanitizeFilename(articleInfo["id"])
			htmlPath = filepath.Join(baseDir, safeID+".html")
		} else {
			safeName := SanitizeFilename(articleInfo["tag"] + "__" + articleInfo["slug"])
			htmlPath = filepath.Join(baseDir, safeName+".html")
		}

		htmlBytes, err := os.ReadFile(htmlPath)
		if err != nil {
			fmt.Printf("[%s] File not found: %s\n", source, htmlPath)
			continue
		}

		html := string(htmlBytes)

		if err := db.SaveDocument(normalizedURL, html, source); err != nil {
			fmt.Printf("[%s] Failed to save to DB: %v\n", source, err)
			continue
		}

		added++
		fmt.Printf("[%s] Added to DB: %s\n", source, normalizedURL)
	}

	fmt.Printf("[%s] Added %d pages, skipped %d (already in DB)\n", source, added, skipped)
	return nil
}

