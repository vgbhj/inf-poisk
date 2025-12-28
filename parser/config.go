package parser

import (
	"net/http"
	"sync"
	"time"

	"github.com/go-rod/rod"
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

type Config struct {
	UseBrowser   bool
	ShowBrowser  bool
	BrowserDebug bool
	CollectOnly  bool
	DownloadOnly bool
	Workers      int
	Site         string
}

var (
	HTTPClient = &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     30 * time.Second,
		},
	}

	UserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"

	Browser *rod.Browser

	CFCookies   = make(map[string]string)
	CookiesLock sync.RWMutex
)
