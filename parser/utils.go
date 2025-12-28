package parser

import (
	"fmt"
	"math/rand"
	"regexp"
	"strings"
	"time"
)

// SanitizeFilename removes invalid filename characters
func SanitizeFilename(name string) string {
	re := regexp.MustCompile(`[<>:"/\\|?*]`)
	name = re.ReplaceAllString(name, "_")
	if len(name) > 200 {
		name = name[:200]
	}
	return name
}

// SleepWithJitter sleeps with exponential backoff and jitter
func SleepWithJitter(base time.Duration, attempt int) {
	wait := base * time.Duration(1<<uint(attempt))
	if wait > 30*time.Second {
		wait = 30 * time.Second
	}
	wait += time.Duration(rand.Intn(300)) * time.Millisecond
	time.Sleep(wait)
}

// ExtractDomain extracts domain from URL
func ExtractDomain(urlStr string) string {
	if strings.Contains(urlStr, "hltv.org") {
		return "hltv.org"
	}
	if strings.Contains(urlStr, "cybersport.ru") {
		return "cybersport.ru"
	}
	return ""
}

// IsEmptyHLTVArticle checks if article is valid
func IsEmptyHLTVArticle(article *Article) error {
	if article == nil {
		return fmt.Errorf("article is nil")
	}

	title := strings.TrimSpace(article.Title)
	content := strings.TrimSpace(article.Content)

	if title == "" {
		return fmt.Errorf("empty title")
	}

	if len(content) < 15 {
		return fmt.Errorf("content too short (%d chars)", len(content))
	}

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
