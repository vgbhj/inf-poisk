package parser

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/go-rod/rod"
)

func ParseCybersportArticle(tag string, slug string) (*Article, error) {
	url := fmt.Sprintf("https://www.cybersport.ru/tags/%s/%s", tag, slug)
	doc, err := FetchPage(url)
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

func GetCybersportArticles() ([]map[string]string, error) {
	var articles []map[string]string
	seen := make(map[string]bool)
	// tags := []string{"dota-2", "cs2"}
	tags := []string{"cs2"}

	for _, tag := range tags {
		fmt.Printf("Tag: %s\n", tag)
		if Browser == nil {
			fmt.Println("Browser error")
			break
		}

		page := Browser.MustPage("")
		defer page.MustClose()

		url := fmt.Sprintf("https://www.cybersport.ru/tags/%s", tag)
		err := rod.Try(func() {
			page.MustNavigate(url)
			page.MustWaitLoad()
		})
		if err != nil {
			continue
		}

		noNewAttempts := 0
		flag := true
		for {
			countBefore := len(articles)

			page.MustEval(`() => {
				const overlays = document.querySelectorAll('.accept-cookies-text, [class*="Header_sticky"], [class*="Cookie"]');
				overlays.forEach(el => el.remove());
				document.body.style.pointerEvents = 'auto';
				document.body.style.overflow = 'auto';
				document.documentElement.style.overflow = 'auto';
			}`)

			btn, err := page.Element(`button[class*="button_+fnen"]`)
			if err == nil && btn != nil && flag {
				_ = rod.Try(func() {
					btn.MustScrollIntoView()
					page.Mouse.MustScroll(0, -200)
					time.Sleep(500 * time.Millisecond)

					if errClick := rod.Try(func() { btn.MustClick() }); errClick != nil {
						btn.MustEval(`el => {
							el.focus();
							el.dispatchEvent(new MouseEvent('mousedown', {bubbles: true}));
							el.dispatchEvent(new MouseEvent('mouseup', {bubbles: true}));
							el.click();
						}`)
					}
					time.Sleep(2 * time.Second)
					page.MustWaitRequestIdle()
				})
				flag = false
			} else {
				page.Mouse.MustScroll(0, 3000)
				time.Sleep(1500 * time.Millisecond)
			}

			html := page.MustHTML()
			doc, _ := goquery.NewDocumentFromReader(strings.NewReader(html))

			doc.Find("a[href*='/tags/" + tag + "/']").Each(func(i int, s *goquery.Selection) {
				href, exists := s.Attr("href")
				if !exists {
					return
				}

				re := regexp.MustCompile(`/tags/` + tag + `/([^?#]+)`)
				matches := re.FindStringSubmatch(href)
				if len(matches) == 2 {
					slug := strings.Trim(matches[1], "/")
					if slug != "" && slug != tag && !strings.Contains(slug, "page") {
						key := tag + "/" + slug
						if !seen[key] {
							seen[key] = true
							articles = append(articles, map[string]string{"tag": tag, "slug": slug})
						}
					}
				}
			})

			countAfter := len(articles)

			if countAfter > countBefore {
				noNewAttempts = 0
				fmt.Printf("  %s: +%d (Total: %d)\n", tag, countAfter-countBefore, countAfter)
			} else {
				noNewAttempts++
				if noNewAttempts >= 6 {
					fmt.Printf("  %s: Stopped. No more content.\n", tag)
					break
				}
				page.Mouse.MustScroll(0, 1000)
			}

			if countAfter >= 20000 {
				fmt.Printf("  %s: Limit reached.\n", tag)
				break
			}
		}
	}
	return articles, nil
}
