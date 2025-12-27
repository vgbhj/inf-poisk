package parser

import (
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
)

// InitBrowser initializes a new browser instance
func InitBrowser(show, debug bool) (*rod.Browser, error) {
	launch := launcher.New().
		Headless(!show).
		Set("disable-blink-features", "AutomationControlled").
		UserDataDir("browser_data")

	if debug {
		launch = launch.Devtools(true)
	}

	url := launch.MustLaunch()
	b := rod.New().ControlURL(url).MustConnect()
	// b := rod.New().ControlURL(url).MustConnect().Trace(true)

	return b, nil
}

// ExtractCookiesFromPage extracts Cloudflare cookies from page
func ExtractCookiesFromPage(page *rod.Page, domain string) {
	cookies, err := page.Cookies([]string{})
	if err != nil {
		return
	}

	CookiesLock.Lock()
	defer CookiesLock.Unlock()

	for _, cookie := range cookies {
		if cookie.Name == "cf_clearance" {
			CFCookies[domain] = cookie.Value
			break
		}
	}
}

// FetchPage fetches a page using either browser or HTTP client
func FetchPage(url string) (*goquery.Document, error) {
	if Browser != nil {
		return BrowserFetchPage(url)
	}
	return HTTPFetchPage(url)
}

// HTTPFetchPage fetches page using HTTP client
func HTTPFetchPage(url string) (*goquery.Document, error) {
	maxRetries := 6
	baseDelay := 800 * time.Millisecond

	for attempt := 0; attempt < maxRetries; attempt++ {
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, err
		}

		// Set headers
		req.Header.Set("User-Agent", UserAgent)
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

		// Add cached cookies
		CookiesLock.RLock()
		for domain, cookie := range CFCookies {
			if strings.Contains(url, domain) {
				req.Header.Set("Cookie", "cf_clearance="+cookie)
				break
			}
		}
		CookiesLock.RUnlock()

		resp, err := HTTPClient.Do(req)
		if err != nil {
			SleepWithJitter(baseDelay, attempt)
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
			SleepWithJitter(baseDelay, attempt)
			continue
		}

		doc, err := goquery.NewDocumentFromReader(resp.Body)
		resp.Body.Close()
		if err != nil {
			SleepWithJitter(baseDelay, attempt)
			continue
		}

		html, _ := doc.Html()
		if !strings.Contains(html, "challenge-form") &&
			!strings.Contains(html, "Cloudflare") &&
			!strings.Contains(html, "cf-browser-verification") {
			for _, cookie := range resp.Cookies() {
				if cookie.Name == "cf_clearance" {
					domain := resp.Request.URL.Hostname()
					CookiesLock.Lock()
					CFCookies[domain] = cookie.Value
					CookiesLock.Unlock()
					break
				}
			}
		}

		return doc, nil
	}

	return nil, fmt.Errorf("failed to fetch %s after %d attempts", url, maxRetries)
}

// BrowserFetchPage fetches page using browser
func BrowserFetchPage(url string) (*goquery.Document, error) {
	if Browser == nil {
		return nil, fmt.Errorf("browser not initialized")
	}

	page := Browser.MustPage("")
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

	domain := ExtractDomain(url)
	ExtractCookiesFromPage(page, domain)

	html, err := page.HTML()
	if err != nil {
		return nil, fmt.Errorf("failed to get HTML: %w", err)
	}

	return goquery.NewDocumentFromReader(strings.NewReader(html))
}
