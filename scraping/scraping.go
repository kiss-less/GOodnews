package scraping

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/gocolly/colly"
)

type NewsItem struct {
	Id                                      int
	Url, Category, Posted, Title, Image, P1 string
	Text                                    []string
}

type Scraper struct {
	UserAgent  string
	Sources    []string
	Collector  *colly.Collector
	ReqSleepMs int
}

func NewScraper(userAgent string, sources []string, reqSleepMs int) *Scraper {
	c := colly.NewCollector()
	c.UserAgent = userAgent

	return &Scraper{
		UserAgent:  userAgent,
		Sources:    sources,
		Collector:  c,
		ReqSleepMs: reqSleepMs,
	}
}

func (s *Scraper) ScrapNewsUrlsFromSources() []string {
	var newsUrls []string
	s.Collector.OnError(func(_ *colly.Response, err error) {
		log.Println("Something went wrong: ", err)
	})

	// TODO: Adjust the struct to include mapping between url and HTML elements to be able to pass both to the constructor instead of using switch statement
	for i := 0; i < len(s.Sources); i++ {
		switch s.Sources[i] {
		case "https://positivnews.ru/":
			s.Collector.OnHTML("div.digital-newspaper-container", func(e *colly.HTMLElement) {
				newsUrls = append(newsUrls, e.ChildAttr("a", "href"))
			})

			s.Collector.OnHTML("article.post", func(e *colly.HTMLElement) {
				newsUrls = append(newsUrls, e.ChildAttr("a", "href"))
			})
		case "https://ntdtv.ru/c/pozitivnye-novosti":
			s.Collector.OnHTML("div.entry-image", func(e *colly.HTMLElement) {
				newsUrls = append(newsUrls, e.ChildAttr("a", "href"))
			})
		default:
			log.Panic("The news source passed to scraper.ScrapNewsUrlsFromSources func is not found!")
		}

		s.Collector.Visit(s.Sources[i])
		time.Sleep(time.Duration(s.ReqSleepMs) * time.Millisecond)
	}
	newsUrls = s.removeDuplicatesAndRootSite(newsUrls)

	return newsUrls
}

func (s *Scraper) ScrapNewsFromNewsUrls(newsUrls []string) (error, []NewsItem) {
	if len(newsUrls) == 0 {
		return fmt.Errorf("The NewsUrls is empty. Please make sure to run scraper.ScrapNewsUrlsFromSources() first!"), nil
	} else {
		var newsItems []NewsItem

		for i := 0; i < len(newsUrls); i++ {
			var newsItem NewsItem
			newsItem.Url = newsUrls[i]

			s.Collector.OnError(func(_ *colly.Response, err error) {
				log.Println("Something went wrong: ", err)
			})

			// TODO: Adjust the struct to include mapping between url and HTML elements to be able to pass both to the constructor instead of using if\else condition
			if strings.Contains(newsUrls[i], "https://positivnews.ru/") {
				s.Collector.OnHTML(".entry-content p", func(e *colly.HTMLElement) {
					newsItem.Text = append(newsItem.Text, e.Text)
				})

				s.Collector.OnHTML(".post-categories a", func(e *colly.HTMLElement) {
					newsItem.Category = e.Text
				})

				s.Collector.OnHTML(".entry-title", func(e *colly.HTMLElement) {
					newsItem.Title = e.Text
				})

				s.Collector.OnHTML(".entry-meta time.updated", func(e *colly.HTMLElement) {
					newsItem.Posted = formatTime(e.Attr("datetime"), "2006-01-02T15:04:05-07:00")
				})

				s.Collector.OnHTML("div.post-inner div.post-thumbnail img.wp-post-image", func(e *colly.HTMLElement) {
					newsItem.Image = e.Attr("src")
				})
			} else if strings.Contains(newsUrls[i], "https://ntdtv.ru/") {
				s.Collector.OnHTML("div[id=cont_post] p", func(e *colly.HTMLElement) {
					// Workaround for the articles without the heading <p> element in the cont_post div
					text := strings.TrimSpace(e.Text)
					if text != "" {
						newsItem.Text = append(newsItem.Text, text)
					}
				})

				s.Collector.OnHTML("header.entry-header h1", func(e *colly.HTMLElement) {
					newsItem.Title = e.Text
				})

				s.Collector.OnHTML("span.entry-date time", func(e *colly.HTMLElement) {
					newsItem.Posted = formatTime(e.Attr("datetime"), "2006-01-02 15:04:05")
				})

				s.Collector.OnHTML("link[itemprop=thumbnailUrl]", func(e *colly.HTMLElement) {
					newsItem.Image = e.Attr("href")
				})
				newsItem.Category = "Мировые новости"
			} else {
				log.Panic("The news source passed to scraper.ScrapNewsFromNewsUrls func is not found!")
			}
			fmt.Printf("i = %d, processing url: %s len(newsUrls): %d\n", i, newsUrls[i], len(newsUrls))
			s.Collector.Visit(newsUrls[i])
			newsItem.P1 = newsItem.Text[0] + " " + newsItem.Text[1]
			time.Sleep(time.Duration(s.ReqSleepMs) * time.Millisecond)

			newsItems = append(newsItems, newsItem)
		}
		return nil, newsItems
	}
}

func formatTime(src, inputLayout string) string {

	t, err := time.Parse(inputLayout, src)
	if err != nil {
		fmt.Println("Error parsing timestamp:", err)
		return ""
	}

	return t.Format("02-01-2006 15:04:05")
}

func (s *Scraper) removeDuplicatesAndRootSite(newsUrls []string) []string {
	uniqueMap := make(map[string]bool)
	result := []string{}

	for _, source := range s.Sources {
		uniqueMap[source] = true
	}

	for _, url := range newsUrls {
		if url != "" && !uniqueMap[url] {
			uniqueMap[url] = true
			result = append(result, url)
		}
	}

	filteredResult := []string{}
	for _, url := range result {
		if !contains(s.Sources, url) {
			filteredResult = append(filteredResult, url)
		}
	}

	return filteredResult
}

func contains(urls []string, url string) bool {
	for _, u := range urls {
		if u == url {
			return true
		}
	}
	return false
}
