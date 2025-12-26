package main

import (
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/cheggaaa/pb/v3"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
)

var (
	useBrowser   bool
	showBrowser  bool
	browserDebug bool

	collectOnly  bool
	downloadOnly bool
	workers      int
	site         string
)

type Article struct {
	ID      string
	URL     string
	Title   string
	Content string
	Source  string
	Tag     string
}

type Statistics struct {
	TotalArticles      int    `json:"total_articles"`
	TotalSize          int64  `json:"total_size_bytes"`
	HLTVArticles       int    `json:"hltv_articles"`
	CybersportArticles int    `json:"cybersport_articles"`
	DownloadTime       string `json:"download_time"`
	CorpusPath         string `json:"corpus_path"`
	BrowserMode        bool   `json:"browser_mode"`
}

var (
	client = &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     30 * time.Second,
		},
	}
	userAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
	browser   *rod.Browser

	cfCookies   = make(map[string]string)
	cookiesLock sync.RWMutex
)

func initBrowser(show, debug bool) (*rod.Browser, error) {
	launch := launcher.New().
		Headless(!show).
		Set("disable-blink-features", "AutomationControlled").
		UserDataDir("browser_data")

	if debug {
		launch = launch.Devtools(true)
	}

	url := launch.MustLaunch()
	b := rod.New().ControlURL(url).MustConnect()
	b.MustIgnoreCertErrors(true)

	return b, nil
}

func extractCookiesFromPage(page *rod.Page, domain string) {
	cookies, err := page.Cookies([]string{})
	if err != nil {
		return
	}

	cookiesLock.Lock()
	defer cookiesLock.Unlock()

	for _, cookie := range cookies {
		if cookie.Name == "cf_clearance" {
			cfCookies[domain] = cookie.Value
			break
		}
	}
}

func sleepWithJitter(base time.Duration, attempt int) {
	wait := base * time.Duration(1<<uint(attempt))
	if wait > 30*time.Second {
		wait = 30 * time.Second
	}
	wait += time.Duration(rand.Intn(300)) * time.Millisecond
	time.Sleep(wait)
}

func fetchPage(url string) (*goquery.Document, error) {
	if useBrowser && browser != nil {
		return browserFetchPage(url)
	}
	return httpFetchPage(url)
}

func httpFetchPage(url string) (*goquery.Document, error) {
	maxRetries := 6
	baseDelay := 800 * time.Millisecond

	for attempt := 0; attempt < maxRetries; attempt++ {
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, err
		}

		req.Header.Set("User-Agent", userAgent)
		req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
		req.Header.Set("Accept-Language", "en-US,en;q=0.5")
		req.Header.Set("Accept-Encoding", "gzip, deflate, br")
		req.Header.Set("DNT", "1")
		req.Header.Set("Connection", "keep-alive")
		req.Header.Set("Upgrade-Insecure-Requests", "1")
		req.Header.Set("Sec-Fetch-Dest", "document")
		req.Header.Set("Sec-Fetch-Mode", "navigate")
		req.Header.Set("Sec-Fetch-Site", "none")
		req.Header.Set("Sec-Fetch-User", "?1")
		req.Header.Set("Cache-Control", "max-age=0")

		cookiesLock.RLock()
		for domain, cookie := range cfCookies {
			if strings.Contains(url, domain) {
				req.Header.Set("Cookie", "cf_clearance="+cookie)
				break
			}
		}
		cookiesLock.RUnlock()

		resp, err := client.Do(req)
		if err != nil {
			sleepWithJitter(baseDelay, attempt)
			continue
		}

		if resp.StatusCode == 429 {
			resp.Body.Close()
			wait := time.Duration(2<<uint(attempt)) * time.Second
			if wait > 60*time.Second {
				wait = 60 * time.Second
			}
			wait += time.Duration(rand.Intn(500)) * time.Millisecond
			fmt.Printf("got 429 from %s; sleeping %v (attempt %d)\n", url, wait, attempt+1)
			time.Sleep(wait)
			continue
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			sleepWithJitter(baseDelay, attempt)
			continue
		}

		doc, err := goquery.NewDocumentFromReader(resp.Body)
		resp.Body.Close()
		if err != nil {
			sleepWithJitter(baseDelay, attempt)
			continue
		}

		html, _ := doc.Html()
		if !strings.Contains(html, "challenge-form") &&
			!strings.Contains(html, "Cloudflare") &&
			!strings.Contains(html, "cf-browser-verification") {
			for _, cookie := range resp.Cookies() {
				if cookie.Name == "cf_clearance" {
					domain := resp.Request.URL.Hostname()
					cookiesLock.Lock()
					cfCookies[domain] = cookie.Value
					cookiesLock.Unlock()
					break
				}
			}
		}

		return doc, nil
	}

	return nil, fmt.Errorf("failed to fetch %s after %d attempts", url, maxRetries)
}

func browserFetchPage(url string) (*goquery.Document, error) {
	if browser == nil {
		return nil, fmt.Errorf("browser not initialized")
	}

	page := browser.MustPage("")
	defer page.MustClose()

	err := rod.Try(func() {
		page.MustNavigate(url)
		page.MustWaitLoad()
		time.Sleep(2 * time.Second)

		html, _ := page.HTML()
		if strings.Contains(html, "challenge-form") ||
			strings.Contains(html, "Cloudflare") ||
			strings.Contains(html, "cf-browser-verification") {
			time.Sleep(3 * time.Second)
		}

		page.MustWaitIdle()
		time.Sleep(1 * time.Second)
	})

	if err != nil {
		return nil, fmt.Errorf("browser navigation failed: %w", err)
	}

	domain := extractDomain(url)
	extractCookiesFromPage(page, domain)

	html, err := page.HTML()
	if err != nil {
		return nil, fmt.Errorf("failed to get HTML: %w", err)
	}

	return goquery.NewDocumentFromReader(strings.NewReader(html))
}

func extractDomain(urlStr string) string {
	if strings.Contains(urlStr, "hltv.org") {
		return "hltv.org"
	}
	if strings.Contains(urlStr, "cybersport.ru") {
		return "cybersport.ru"
	}
	return ""
}

func parseHLTVArticle(id string, slug string) (*Article, error) {
	url := fmt.Sprintf("https://www.hltv.org/news/%s/%s", id, slug)
	doc, err := fetchPage(url)
	if err != nil {
		return nil, err
	}

	title := doc.Find("h1").First().Text()
	title = strings.TrimSpace(title)

	var contentBuilder strings.Builder
	doc.Find(".news-block, article, .standard-box").Each(func(i int, s *goquery.Selection) {
		s.Find("p").Each(func(j int, p *goquery.Selection) {
			text := strings.TrimSpace(p.Text())
			if text != "" && len(text) > 20 {
				contentBuilder.WriteString(text)
				contentBuilder.WriteString("\n\n")
			}
		})
	})

	if contentBuilder.Len() < 500 {
		doc.Find("p").Each(func(i int, s *goquery.Selection) {
			text := strings.TrimSpace(s.Text())
			if text != "" && len(text) > 20 {
				contentBuilder.WriteString(text)
				contentBuilder.WriteString("\n\n")
			}
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

func getHLTVNewsIDs() ([]map[string]string, error) {
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

			doc, err := fetchPage(url)
			if err != nil {
				fmt.Printf("  %s %d: load error - %v\n", month, year, err)
				consecutiveErrors++
				if consecutiveErrors >= 3 {
					fmt.Println("multiple 429s detected — sleeping 5 minutes")
					consecutiveErrors = 0
				}
				continue
			}

			consecutiveErrors = 0

			monthCount := 0
			doc.Find("a[href*='/news/']").Each(func(_ int, s *goquery.Selection) {
				href, _ := s.Attr("href")
				m := re.FindStringSubmatch(href)
				if len(m) >= 2 {
					// try to extract full id and slug by matching the full pattern
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

func parseCybersportArticle(tag string, slug string) (*Article, error) {
	url := fmt.Sprintf("https://www.cybersport.ru/tags/%s/%s", tag, slug)
	doc, err := fetchPage(url)
	if err != nil {
		return nil, err
	}

	title := doc.Find("h1").First().Text()
	title = strings.TrimSpace(title)

	var contentBuilder strings.Builder
	doc.Find(".article-content, .content, article, .post-content").Each(func(i int, s *goquery.Selection) {
		s.Find("p").Each(func(j int, p *goquery.Selection) {
			text := strings.TrimSpace(p.Text())
			if text != "" && len(text) > 20 {
				contentBuilder.WriteString(text)
				contentBuilder.WriteString("\n\n")
			}
		})
	})

	if contentBuilder.Len() < 500 {
		doc.Find("p").Each(func(i int, s *goquery.Selection) {
			text := strings.TrimSpace(s.Text())
			if text != "" && len(text) > 20 {
				contentBuilder.WriteString(text)
				contentBuilder.WriteString("\n\n")
			}
		})
	}

	content := strings.TrimSpace(contentBuilder.String())

	return &Article{
		ID:      slug,
		URL:     url,
		Title:   title,
		Content: content,
		Source:  "cybersport",
		Tag:     tag,
	}, nil
}

func getCybersportArticles() ([]map[string]string, error) {
	var articles []map[string]string
	seen := make(map[string]bool)
	tags := []string{"dota-2", "cs2", "valorant", "league-of-legends", "overwatch", "apex-legends", "rainbow-six", "rocket-league"}

	for _, tag := range tags {
		fmt.Printf("Processing tag: %s\n", tag)

		if !useBrowser || browser == nil {
			fmt.Printf("Warning: browser not available, skipping %s (requires browser for dynamic loading)\n", tag)
			continue
		}

		page := browser.MustPage("")
		defer page.MustClose()

		url := fmt.Sprintf("https://www.cybersport.ru/tags/%s", tag)

		err := rod.Try(func() {
			page.MustNavigate(url)
			page.MustWaitLoad()
			time.Sleep(2 * time.Second)
		})

		if err != nil {
			fmt.Printf("Failed to load %s: %v\n", tag, err)
			continue
		}

		extractCookiesFromPage(page, "cybersport.ru")

		for len(articles) < 25000 {
			html, err := page.HTML()
			if err != nil {
				fmt.Printf("Failed to get HTML for %s: %v\n", tag, err)
				break
			}

			doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
			if err != nil {
				fmt.Printf("Failed to parse HTML for %s: %v\n", tag, err)
				break
			}

			found := false
			doc.Find("a[href*='/tags/" + tag + "/']").Each(func(i int, s *goquery.Selection) {
				href, exists := s.Attr("href")
				if !exists {
					return
				}

				re := regexp.MustCompile(`/tags/` + tag + `/([^?]+)`)
				matches := re.FindStringSubmatch(href)
				if len(matches) == 2 {
					slug := strings.Trim(matches[1], "/")
					if slug != "" && slug != tag {
						key := tag + "/" + slug
						if !seen[key] {
							seen[key] = true
							articles = append(articles, map[string]string{
								"tag":  tag,
								"slug": slug,
							})
							found = true
						}
					}
				}
			})

			if !found {
				fmt.Printf("  %s: no new articles found, stopping\n", tag)
				break
			}

			fmt.Printf("  %s: found %d articles so far\n", tag, len(articles))

			btn := page.MustElements(`button.button_+fnen`)
			if len(btn) == 0 {
				fmt.Printf("  %s: 'Show more' button not found, stopping\n", tag)
				break
			}

			err = rod.Try(func() {
				btn[0].MustClick()
				time.Sleep(2 * time.Second)
				page.MustWaitIdle()
			})

			if err != nil {
				fmt.Printf("  %s: failed to click 'Show more' button: %v\n", tag, err)
				break
			}

			time.Sleep(1000 * time.Millisecond)
		}

		fmt.Printf("  %s: collected %d articles total\n", tag, len(articles))
	}

	return articles, nil
}

func sanitizeFilename(name string) string {
	re := regexp.MustCompile(`[<>:"/\\|?*]`)
	name = re.ReplaceAllString(name, "_")
	if len(name) > 200 {
		name = name[:200]
	}
	return name
}

func saveArticle(article *Article, baseDir string) error {
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

	safeID := sanitizeFilename(article.ID)
	filename := filepath.Join(dir, safeID+".txt")
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = fmt.Fprintf(file, "Title: %s\nURL: %s\nSource: %s\n\n%s", article.Title, article.URL, article.Source, article.Content)
	return err
}

func downloadHLTVArticles(articles []map[string]string, corpusDir string, bar *pb.ProgressBar, stats *Statistics, mu *sync.Mutex) {
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
				safeID := sanitizeFilename(articleID)
				filePath := filepath.Join(hltvDir, safeID+".txt")

				if _, err := os.Stat(filePath); err == nil {
					fmt.Printf("[HLTV] Skipped (already downloaded): %s\n", articleID)
					bar.Increment()
					continue
				}

				article, err := parseHLTVArticle(articleID, articleSlug)
				if err != nil {
					fmt.Printf("[HLTV] Failed to parse %s: %v\n", articleID, err)
					bar.Increment()
					time.Sleep(500 * time.Millisecond)
					continue
				}

				if strings.Contains(article.Content, "Verify you are human by completing the action below") {
					fmt.Printf("[HLTV] Skipped (Cloudflare block): %s\n", articleID)
					bar.Increment()
					continue
				}

				err = saveArticle(article, corpusDir)
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

func downloadCybersportArticles(articles []map[string]string, corpusDir string, bar *pb.ProgressBar, stats *Statistics, mu *sync.Mutex) {
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
				article, err := parseCybersportArticle(articleInfo["tag"], articleInfo["slug"])
				if err != nil {
					fmt.Printf("[Cybersport] Failed to parse %s/%s: %v\n", articleInfo["tag"], articleInfo["slug"], err)
					bar.Increment()
					continue
				}

				err = saveArticle(article, corpusDir)
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

func writeHLTVCSV(path string, articles []map[string]string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	if err := w.Write([]string{"id", "slug"}); err != nil {
		return err
	}

	for _, a := range articles {
		if err := w.Write([]string{a["id"], a["slug"]}); err != nil {
			return err
		}
	}
	return nil
}

func readHLTVCSV(path string) ([]map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	r := csv.NewReader(f)
	rows, err := r.ReadAll()
	if err != nil {
		return nil, err
	}
	var res []map[string]string
	for i, row := range rows {
		if i == 0 {
			continue
		}
		if len(row) < 2 {
			continue
		}
		res = append(res, map[string]string{"id": row[0], "slug": row[1]})
	}
	return res, nil
}

func writeCybersportCSV(path string, articles []map[string]string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	if err := w.Write([]string{"tag", "slug"}); err != nil {
		return err
	}

	for _, a := range articles {
		if err := w.Write([]string{a["tag"], a["slug"]}); err != nil {
			return err
		}
	}
	return nil
}

func readCybersportCSV(path string) ([]map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	r := csv.NewReader(f)
	rows, err := r.ReadAll()
	if err != nil {
		return nil, err
	}
	var res []map[string]string
	for i, row := range rows {
		if i == 0 {
			continue
		}
		if len(row) < 2 {
			continue
		}
		res = append(res, map[string]string{"tag": row[0], "slug": row[1]})
	}
	return res, nil
}

func main() {
	flag.BoolVar(&useBrowser, "b", false, "Use browser to bypass Cloudflare")
	flag.BoolVar(&showBrowser, "show", false, "Show browser window (only with -b)")
	flag.BoolVar(&browserDebug, "debug", false, "Enable browser debug mode")
	flag.BoolVar(&collectOnly, "collect-only", false, "Only collect article links and save to CSV")
	flag.BoolVar(&downloadOnly, "download-only", false, "Only download articles from CSV files (skip collection)")
	flag.IntVar(&workers, "workers", 4, "Number of parallel workers for downloading (default: 4)")
	flag.StringVar(&site, "site", "both", "Which site to process: hltv, cybersport, both")
	flag.Parse()

	rand.Seed(time.Now().UnixNano())

	corpusDir := "corpus"
	os.MkdirAll(corpusDir, 0755)

	hltvCSVPath := filepath.Join(corpusDir, "hltv_links.csv")
	cybersportCSVPath := filepath.Join(corpusDir, "cybersport_links.csv")

	stats := &Statistics{
		CorpusPath:  corpusDir,
		BrowserMode: useBrowser,
	}
	startTime := time.Now()

	if useBrowser {
		fmt.Println("Initializing browser...")
		var err error
		browser, err = initBrowser(showBrowser, browserDebug)
		if err != nil {
			fmt.Printf("Failed to initialize browser: %v\n", err)
			fmt.Println("Falling back to HTTP client...")
			useBrowser = false
		} else {
			defer browser.MustClose()
		}
	}

	var hltvArticles []map[string]string
	var cybersportArticles []map[string]string

	site = strings.ToLower(site)
	validSite := site == "hltv" || site == "cybersport" || site == "both"
	if !validSite {
		fmt.Printf("Invalid site '%s', defaulting to 'both'\n", site)
		site = "both"
	}

	if downloadOnly {
		fmt.Println("Download-only mode: reading lists from CSV...")
		if site == "hltv" || site == "both" {
			h, err := readHLTVCSV(hltvCSVPath)
			if err != nil {
				fmt.Printf("Failed to read HLTV CSV (%s): %v\n", hltvCSVPath, err)
				return
			}
			hltvArticles = h
		}
		if site == "cybersport" || site == "both" {
			c, err := readCybersportCSV(cybersportCSVPath)
			if err != nil {
				fmt.Printf("Failed to read Cybersport CSV (%s): %v\n", cybersportCSVPath, err)
				return
			}
			cybersportArticles = c
		}
		fmt.Printf("Read %d HLTV links and %d Cybersport links from CSV\n", len(hltvArticles), len(cybersportArticles))
	} else {
		fmt.Println("Collecting article lists...")
		if site == "hltv" || site == "both" {
			fmt.Println("HLTV.org...")
			h, _ := getHLTVNewsIDs()
			hltvArticles = h
			fmt.Printf("Found %d HLTV articles\n", len(hltvArticles))
		}

		if site == "cybersport" || site == "both" {
			fmt.Println("Cybersport.ru...")
			c, _ := getCybersportArticles()
			cybersportArticles = c
			fmt.Printf("Found %d Cybersport articles\n", len(cybersportArticles))
		}

		if collectOnly {
			fmt.Println("Collect-only mode: writing CSVs and exiting...")
			if site == "hltv" || site == "both" {
				if err := writeHLTVCSV(hltvCSVPath, hltvArticles); err != nil {
					fmt.Printf("Failed to write HLTV CSV: %v\n", err)
				} else {
					fmt.Printf("HLTV links written to %s\n", hltvCSVPath)
				}
			}
			if site == "cybersport" || site == "both" {
				if err := writeCybersportCSV(cybersportCSVPath, cybersportArticles); err != nil {
					fmt.Printf("Failed to write Cybersport CSV: %v\n", err)
				} else {
					fmt.Printf("Cybersport links written to %s\n", cybersportCSVPath)
				}
			}
			return
		}
	}

	// compute total based on selected sources
	total := 0
	if site == "hltv" || site == "both" {
		total += len(hltvArticles)
	}
	if site == "cybersport" || site == "both" {
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

	// start download goroutines only for selected sites
	workersCount := 0
	if site == "hltv" || site == "both" {
		workersCount++
	}
	if site == "cybersport" || site == "both" {
		workersCount++
	}

	if workersCount == 0 {
		fmt.Println("No sites selected to download")
		return
	}

	wg.Add(workersCount)

	if site == "hltv" || site == "both" {
		go func() {
			defer wg.Done()
			downloadHLTVArticles(hltvArticles, corpusDir, bar, stats, &mu)
		}()
	}
	if site == "cybersport" || site == "both" {
		go func() {
			defer wg.Done()
			downloadCybersportArticles(cybersportArticles, corpusDir, bar, stats, &mu)
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
