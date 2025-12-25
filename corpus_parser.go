package main

import (
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

	html, err := page.HTML()
	if err != nil {
		return nil, fmt.Errorf("failed to get HTML: %w", err)
	}

	return goquery.NewDocumentFromReader(strings.NewReader(html))
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
	if len(content) < 500 {
		return nil, fmt.Errorf("article too short")
	}

	return &Article{
		ID:      id,
		URL:     url,
		Title:   title,
		Content: content,
		Source:  "hltv",
	}, nil
}

func getHLTVNewsIDs() ([]map[string]string, error) {
	type job struct {
		year  int
		month string
	}

	months := []string{
		"january", "february", "march", "april",
		"may", "june", "july", "august",
		"september", "october", "november", "december",
	}

	currentYear := time.Now().Year()
	maxWorkers := 5

	jobs := make(chan job)
	results := make(chan map[string]string)
	var wg sync.WaitGroup
	var seen sync.Map // key -> struct{}

	// helper to sanitize slug (remove params, fragments, trailing slashes)
	cleanSlug := func(raw string) string {
		if idx := strings.IndexAny(raw, "?#"); idx != -1 {
			raw = raw[:idx]
		}
		return strings.Trim(raw, "/")
	}

	// воркер
	worker := func() {
		defer wg.Done()
		re := regexp.MustCompile(`/news/(\d+)/([^/]+)`)
		for j := range jobs {
			fmt.Printf("start %s %d...\n", j.month, j.year)

			url := fmt.Sprintf("https://www.hltv.org/news/archive/%d/%s", j.year, j.month)
			doc, err := fetchPage(url)
			if err != nil {
				fmt.Printf("  %s %d: ERROR fetching page - %v\n", j.month, j.year, err)
				time.Sleep(300 * time.Millisecond)
				continue
			}

			foundThisMonth := 0
			doc.Find("a[href*='/news/']").Each(func(_ int, s *goquery.Selection) {
				href, ok := s.Attr("href")
				if !ok {
					return
				}
				matches := re.FindStringSubmatch(href)
				if len(matches) != 3 {
					return
				}
				id := matches[1]
				slug := cleanSlug(matches[2])
				if slug == "" {
					return
				}
				key := id + "/" + slug
				_, loaded := seen.LoadOrStore(key, struct{}{})
				if !loaded {
					results <- map[string]string{"id": id, "slug": slug}
					foundThisMonth++
				}
			})

			if foundThisMonth > 0 {
				fmt.Printf("  %s %d: processed %d new articles\n", j.month, j.year, foundThisMonth)
			} else {
				fmt.Printf("  %s %d: no articles found\n", j.month, j.year)
			}

			time.Sleep(300 * time.Millisecond) // polite delay
		}
	}

	// стартуем воркеров
	wg.Add(maxWorkers)
	for i := 0; i < maxWorkers; i++ {
		go worker()
	}

	// feeder: создаём jobs
	go func() {
		for year := 2006; year <= currentYear; year++ {
			for _, m := range months {
				jobs <- job{year: year, month: m}
			}
		}
		close(jobs)
	}()

	// собираем результаты в срез
	var articles []map[string]string
	collectDone := make(chan struct{})
	go func() {
		for r := range results {
			articles = append(articles, r)
		}
		close(collectDone)
	}()

	// ждём завершения воркеров
	wg.Wait()
	close(results)
	<-collectDone

	if len(articles) == 0 {
		fmt.Println("ERROR: no HLTV articles found!")
		return articles, fmt.Errorf("no HLTV articles found")
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
	if len(content) < 500 {
		return nil, fmt.Errorf("article too short")
	}

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
		page := 1
		for len(articles) < 25000 {
			url := fmt.Sprintf("https://www.cybersport.ru/tags/%s?page=%d", tag, page)
			doc, err := fetchPage(url)
			if err != nil {
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
				break
			}

			page++
			time.Sleep(500 * time.Millisecond)
		}
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
	for _, articleInfo := range articles {
		article, err := parseHLTVArticle(articleInfo["id"], articleInfo["slug"])
		if err != nil {
			bar.Increment()
			continue
		}

		err = saveArticle(article, corpusDir)
		if err != nil {
			bar.Increment()
			continue
		}

		mu.Lock()
		stats.HLTVArticles++
		stats.TotalArticles++
		stats.TotalSize += int64(len(article.Content))
		mu.Unlock()

		bar.Increment()
		time.Sleep(200 * time.Millisecond)
	}
}

func downloadCybersportArticles(articles []map[string]string, corpusDir string, bar *pb.ProgressBar, stats *Statistics, mu *sync.Mutex) {
	for _, articleInfo := range articles {
		article, err := parseCybersportArticle(articleInfo["tag"], articleInfo["slug"])
		if err != nil {
			bar.Increment()
			continue
		}

		err = saveArticle(article, corpusDir)
		if err != nil {
			bar.Increment()
			continue
		}

		mu.Lock()
		stats.CybersportArticles++
		stats.TotalArticles++
		stats.TotalSize += int64(len(article.Content))
		mu.Unlock()

		bar.Increment()
		time.Sleep(200 * time.Millisecond)
	}
}

func main() {
	flag.BoolVar(&useBrowser, "b", false, "Use browser to bypass Cloudflare")
	flag.BoolVar(&showBrowser, "show", false, "Show browser window (only with -b)")
	flag.BoolVar(&browserDebug, "debug", false, "Enable browser debug mode")
	flag.Parse()

	rand.Seed(time.Now().UnixNano())

	corpusDir := "corpus"
	os.MkdirAll(corpusDir, 0755)

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

	// go func() {
	// 	for range time.Tick(3 * time.Second) {
	// 		fmt.Printf("[STATS] goroutines=%d\n", runtime.NumGoroutine())
	// 	}
	// }()

	fmt.Println("Collecting article lists...")
	fmt.Println("HLTV.org...")
	hltvArticles, _ := getHLTVNewsIDs()
	fmt.Printf("Found %d HLTV articles\n", len(hltvArticles))

	fmt.Println("Cybersport.ru...")
	cybersportArticles, _ := getCybersportArticles()
	fmt.Printf("Found %d Cybersport articles\n", len(cybersportArticles))
	total := len(hltvArticles) + len(cybersportArticles)
	if total < 30000 {
		fmt.Printf("Warning: found only %d articles, minimum 30000 required\n", total)
	}

	bar := pb.New(total)
	bar.SetTemplateString(`{{counters . }} {{bar . }} {{percent . }} {{etime . }}`)
	bar.Start()

	var mu sync.Mutex
	var wg sync.WaitGroup

	wg.Add(2)
	go func() {
		defer wg.Done()
		downloadHLTVArticles(hltvArticles, corpusDir, bar, stats, &mu)
	}()
	go func() {
		defer wg.Done()
		downloadCybersportArticles(cybersportArticles, corpusDir, bar, stats, &mu)
	}()

	wg.Wait()
	bar.Finish()

	stats.DownloadTime = time.Since(startTime).String()

	statsJSON, _ := json.MarshalIndent(stats, "", "  ")
	os.WriteFile(filepath.Join(corpusDir, "statistics.json"), statsJSON, 0644)

	fmt.Printf("\nCompleted. Downloaded articles: %d\n", stats.TotalArticles)
	fmt.Printf("Statistics saved to %s/statistics.json\n", corpusDir)
}
