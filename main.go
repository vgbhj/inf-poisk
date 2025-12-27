package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"corpus_parser/parser"

	"github.com/cheggaaa/pb/v3"
)

func main() {
	cfg := &parser.Config{}

	flag.BoolVar(&cfg.UseBrowser, "b", false, "Use browser to bypass Cloudflare")
	flag.BoolVar(&cfg.ShowBrowser, "show", false, "Show browser window (only with -b)")
	flag.BoolVar(&cfg.BrowserDebug, "debug", false, "Enable browser debug mode")
	flag.BoolVar(&cfg.CollectOnly, "collect-only", false, "Only collect article links and save to CSV")
	flag.BoolVar(&cfg.DownloadOnly, "download-only", false, "Only download articles from CSV files (skip collection)")
	flag.IntVar(&cfg.Workers, "workers", 4, "Number of parallel workers for downloading (default: 4)")
	flag.StringVar(&cfg.Site, "site", "both", "Which site to process: hltv, cybersport, both")
	flag.Parse()

	rand.Seed(time.Now().UnixNano())

	corpusDir := "corpus"
	os.MkdirAll(corpusDir, 0755)

	hltvCSVPath := filepath.Join(corpusDir, "hltv_links.csv")
	cybersportCSVPath := filepath.Join(corpusDir, "cybersport_links.csv")

	stats := &parser.Statistics{
		CorpusPath:  corpusDir,
		BrowserMode: cfg.UseBrowser,
	}
	startTime := time.Now()

	// Initialize browser if needed
	if cfg.UseBrowser {
		fmt.Println("Initializing browser...")
		var err error
		parser.Browser, err = parser.InitBrowser(cfg.ShowBrowser, cfg.BrowserDebug)
		if err != nil {
			fmt.Printf("Failed to initialize browser: %v\n", err)
			fmt.Println("Falling back to HTTP client...")
			cfg.UseBrowser = false
			stats.BrowserMode = false
		} else {
			defer parser.Browser.MustClose()
		}
	}

	var hltvArticles []map[string]string
	var cybersportArticles []map[string]string

	cfg.Site = strings.ToLower(cfg.Site)
	validSite := cfg.Site == "hltv" || cfg.Site == "cybersport" || cfg.Site == "both"
	if !validSite {
		fmt.Printf("Invalid site '%s', defaulting to 'both'\n", cfg.Site)
		cfg.Site = "both"
	}

	// Collection or download mode
	if cfg.DownloadOnly {
		fmt.Println("Download-only mode: reading lists from CSV...")
		if cfg.Site == "hltv" || cfg.Site == "both" {
			h, err := parser.ReadHLTVCSV(hltvCSVPath)
			if err != nil {
				fmt.Printf("Failed to read HLTV CSV (%s): %v\n", hltvCSVPath, err)
				return
			}
			hltvArticles = h
		}
		if cfg.Site == "cybersport" || cfg.Site == "both" {
			c, err := parser.ReadCybersportCSV(cybersportCSVPath)
			if err != nil {
				fmt.Printf("Failed to read Cybersport CSV (%s): %v\n", cybersportCSVPath, err)
				return
			}
			cybersportArticles = c
		}
		fmt.Printf("Read %d HLTV links and %d Cybersport links from CSV\n", len(hltvArticles), len(cybersportArticles))
	} else {
		fmt.Println("Collecting article lists...")
		if cfg.Site == "hltv" || cfg.Site == "both" {
			fmt.Println("HLTV.org...")
			h, _ := parser.GetHLTVNewsIDs()
			hltvArticles = h
			fmt.Printf("Found %d HLTV articles\n", len(hltvArticles))
		}

		if cfg.Site == "cybersport" || cfg.Site == "both" {
			fmt.Println("Cybersport.ru...")
			c, _ := parser.GetCybersportArticles()
			cybersportArticles = c
			fmt.Printf("Found %d Cybersport articles\n", len(cybersportArticles))
		}

		if cfg.CollectOnly {
			fmt.Println("Collect-only mode: writing CSVs and exiting...")
			if cfg.Site == "hltv" || cfg.Site == "both" {
				if err := parser.WriteHLTVCSV(hltvCSVPath, hltvArticles); err != nil {
					fmt.Printf("Failed to write HLTV CSV: %v\n", err)
				} else {
					fmt.Printf("HLTV links written to %s\n", hltvCSVPath)
				}
			}
			if cfg.Site == "cybersport" || cfg.Site == "both" {
				if err := parser.WriteCybersportCSV(cybersportCSVPath, cybersportArticles); err != nil {
					fmt.Printf("Failed to write Cybersport CSV: %v\n", err)
				} else {
					fmt.Printf("Cybersport links written to %s\n", cybersportCSVPath)
				}
			}
			return
		}
	}

	// Compute total articles
	total := 0
	if cfg.Site == "hltv" || cfg.Site == "both" {
		total += len(hltvArticles)
	}
	if cfg.Site == "cybersport" || cfg.Site == "both" {
		total += len(cybersportArticles)
	}

	if total < 30000 {
		fmt.Printf("Warning: found only %d articles, minimum 30000 required\n", total)
	}

	// Progress bar
	bar := pb.New(total)
	bar.SetTemplateString(`{{counters . }} {{bar . }} {{percent . }} {{etime . }}`)
	bar.Start()

	var mu sync.Mutex
	var wg sync.WaitGroup

	// Count download goroutines
	workersCount := 0
	if cfg.Site == "hltv" || cfg.Site == "both" {
		workersCount++
	}
	if cfg.Site == "cybersport" || cfg.Site == "both" {
		workersCount++
	}

	if workersCount == 0 {
		fmt.Println("No sites selected to download")
		return
	}

	wg.Add(workersCount)

	// Start parallel downloads
	if cfg.Site == "hltv" || cfg.Site == "both" {
		go func() {
			defer wg.Done()
			parser.DownloadHLTVArticles(hltvArticles, corpusDir, bar, stats, &mu, cfg.Workers)
		}()
	}
	if cfg.Site == "cybersport" || cfg.Site == "both" {
		go func() {
			defer wg.Done()
			parser.DownloadCybersportArticles(cybersportArticles, corpusDir, bar, stats, &mu, cfg.Workers)
		}()
	}

	wg.Wait()
	bar.Finish()

	stats.DownloadTime = time.Since(startTime).String()

	// Save statistics
	statsJSON, _ := json.MarshalIndent(stats, "", "  ")
	os.WriteFile(filepath.Join(corpusDir, "statistics.json"), statsJSON, 0644)

	fmt.Printf("\nCompleted. Downloaded articles: %d\n", stats.TotalArticles)
	fmt.Printf("Statistics saved to %s/statistics.json\n", corpusDir)
}
