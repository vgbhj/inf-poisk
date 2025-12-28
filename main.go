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
	if len(os.Args) > 1 {
		firstArg := os.Args[1]

		if firstArg == "add-to-db" {
			runAddToDB()
			return
		}

		if strings.HasSuffix(firstArg, ".yaml") || strings.HasSuffix(firstArg, ".yml") {
			if _, err := os.Stat(firstArg); err == nil {
				runYAMLMode(firstArg)
				return
			}
		}
	}

	runLegacyMode()
}

func runYAMLMode(configPath string) {
	cfg, err := parser.LoadYAMLConfig(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	rand.Seed(time.Now().UnixNano())

	db, err := parser.NewDatabase(cfg.DB.URI, cfg.DB.Database, cfg.DB.Collection)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	fmt.Println("Connected to MongoDB successfully")

	corpusDir := "corpus"
	os.MkdirAll(corpusDir, 0755)

	hltvCSVPath := filepath.Join(corpusDir, "hltv_links.csv")
	cybersportCSVPath := filepath.Join(corpusDir, "cybersport_links.csv")

	stats := &parser.Statistics{
		CorpusPath:  corpusDir,
		BrowserMode: cfg.Browser.UseBrowser,
	}
	startTime := time.Now()

	if cfg.Browser.UseBrowser {
		fmt.Println("Initializing browser...")
		var err error
		parser.Browser, err = parser.InitBrowser(cfg.Browser.ShowBrowser, cfg.Browser.BrowserDebug)
		if err != nil {
			fmt.Printf("Failed to initialize browser: %v\n", err)
			fmt.Println("Falling back to HTTP client...")
			cfg.Browser.UseBrowser = false
			stats.BrowserMode = false
		} else {
			defer parser.Browser.MustClose()
		}
	}

	resumeURL := ""
	lastURL, err := db.GetLastProcessedURL()
	if err == nil && lastURL != "" {
		fmt.Printf("Found last processed URL: %s\n", lastURL)
		fmt.Print("Resume from last position? (y/n): ")
		var response string
		fmt.Scanln(&response)
		if strings.ToLower(response) == "y" {
			resumeURL = lastURL
			fmt.Println("Resuming from last position...")
		}
	}

	reCrawlEnabled := cfg.Logic.ReCrawlInterval > 0
	if reCrawlEnabled {
		fmt.Printf("Re-crawl enabled: checking documents older than %d seconds...\n", cfg.Logic.ReCrawlInterval)
		docsToReCrawl, err := db.GetDocumentsForReCrawl(cfg.Logic.ReCrawlInterval)
		if err == nil && len(docsToReCrawl) > 0 {
			fmt.Printf("Found %d documents to re-crawl\n", len(docsToReCrawl))
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

	fmt.Println("Collecting article lists...")
	if cfg.Site == "hltv" || cfg.Site == "both" {
		fmt.Println("HLTV.org...")
		if _, err := os.Stat(hltvCSVPath); err == nil {
			h, err := parser.ReadHLTVCSV(hltvCSVPath)
			if err == nil {
				hltvArticles = h
				fmt.Printf("Read %d HLTV articles from CSV\n", len(hltvArticles))
			} else {
				fmt.Printf("Failed to read HLTV CSV, collecting new...\n")
				h, _ := parser.GetHLTVNewsIDs()
				hltvArticles = h
				parser.WriteHLTVCSV(hltvCSVPath, hltvArticles)
			}
		} else {
			h, _ := parser.GetHLTVNewsIDs()
			hltvArticles = h
			fmt.Printf("Found %d HLTV articles\n", len(hltvArticles))
			parser.WriteHLTVCSV(hltvCSVPath, hltvArticles)
		}
	}

	if cfg.Site == "cybersport" || cfg.Site == "both" {
		fmt.Println("Cybersport.ru...")
		if _, err := os.Stat(cybersportCSVPath); err == nil {
			c, err := parser.ReadCybersportCSV(cybersportCSVPath)
			if err == nil {
				cybersportArticles = c
				fmt.Printf("Read %d Cybersport articles from CSV\n", len(cybersportArticles))
			} else {
				fmt.Printf("Failed to read Cybersport CSV, collecting new...\n")
				c, _ := parser.GetCybersportArticles()
				cybersportArticles = c
				parser.WriteCybersportCSV(cybersportCSVPath, cybersportArticles)
			}
		} else {
			c, _ := parser.GetCybersportArticles()
			cybersportArticles = c
			fmt.Printf("Found %d Cybersport articles\n", len(cybersportArticles))
			parser.WriteCybersportCSV(cybersportCSVPath, cybersportArticles)
		}
	}

	total := 0
	if cfg.Site == "hltv" || cfg.Site == "both" {
		total += len(hltvArticles)
	}
	if cfg.Site == "cybersport" || cfg.Site == "both" {
		total += len(cybersportArticles)
	}

	if total == 0 {
		fmt.Println("No articles to process")
		return
	}

	fmt.Printf("Total articles to process: %d\n", total)
	fmt.Printf("Delay between pages: %d ms\n", cfg.Logic.DelayBetweenPages)

	bar := pb.New(total)
	bar.SetTemplateString(`{{counters . }} {{bar . }} {{percent . }} {{etime . }}`)
	bar.Start()

	var mu sync.Mutex
	var wg sync.WaitGroup

	crawlerCfg := &parser.CrawlerConfig{
		Database:      db,
		CorpusDir:     corpusDir,
		DelayMs:       cfg.Logic.DelayBetweenPages,
		ReCrawl:       reCrawlEnabled,
		ReCrawlInt:    cfg.Logic.ReCrawlInterval,
		ResumeFromURL: resumeURL,
	}

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

	if cfg.Site == "hltv" || cfg.Site == "both" {
		go func() {
			defer wg.Done()
			parser.DownloadHLTVArticlesWithDB(hltvArticles, crawlerCfg, bar, stats, &mu, cfg.Workers)
		}()
	}
	if cfg.Site == "cybersport" || cfg.Site == "both" {
		go func() {
			defer wg.Done()
			parser.DownloadCybersportArticlesWithDB(cybersportArticles, crawlerCfg, bar, stats, &mu, cfg.Workers)
		}()
	}

	wg.Wait()
	bar.Finish()

	stats.DownloadTime = time.Since(startTime).String()

	statsJSON, _ := json.MarshalIndent(stats, "", "  ")
	os.WriteFile(filepath.Join(corpusDir, "statistics.json"), statsJSON, 0644)

	fmt.Printf("\nCompleted. Downloaded articles: %d\n", stats.TotalArticles)
	fmt.Printf("Statistics saved to %s/statistics.json\n", corpusDir)
}

func runLegacyMode() {
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

	bar := pb.New(total)
	bar.SetTemplateString(`{{counters . }} {{bar . }} {{percent . }} {{etime . }}`)
	bar.Start()

	var mu sync.Mutex
	var wg sync.WaitGroup

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

	statsJSON, _ := json.MarshalIndent(stats, "", "  ")
	os.WriteFile(filepath.Join(corpusDir, "statistics.json"), statsJSON, 0644)

	fmt.Printf("\nCompleted. Downloaded articles: %d\n", stats.TotalArticles)
	fmt.Printf("Statistics saved to %s/statistics.json\n", corpusDir)
}

func runAddToDB() {
	var configPath string
	var source string

	flagSet := flag.NewFlagSet("add-to-db", flag.ExitOnError)
	flagSet.StringVar(&configPath, "config", "config.yaml", "Path to YAML config file")
	flagSet.StringVar(&source, "source", "", "Source to add: hltv or cybersport (required)")
	flagSet.Parse(os.Args[2:])

	if source == "" {
		fmt.Fprintf(os.Stderr, "Error: -source is required (hltv or cybersport)\n")
		flagSet.Usage()
		os.Exit(1)
	}

	if source != "hltv" && source != "cybersport" {
		fmt.Fprintf(os.Stderr, "Error: source must be 'hltv' or 'cybersport'\n")
		os.Exit(1)
	}

	cfg, err := parser.LoadYAMLConfig(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	db, err := parser.NewDatabase(cfg.DB.URI, cfg.DB.Database, cfg.DB.Collection)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	fmt.Printf("Connected to MongoDB\n\n")

	corpusDir := "corpus"
	if err := parser.AddExistingPagesToDB(corpusDir, db, source); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
