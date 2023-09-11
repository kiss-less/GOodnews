package scraping

import (
	"fmt"
	"log"
	"math/rand"
	"net/url"
	"strings"
	"time"

	"github.com/gocolly/colly"
)

type NewsItem struct {
	Id                                      int
	Url, Category, Posted, Title, Image, P1 string
	Text                                    []string
}

type ScrapeEntity struct {
	SourceUrl              string
	ScrapeNewsUrlsElements ScrapeNewsURL
	ScrapeNewsHTMLElements ScrapeNewsHTML
}

type ScrapeNewsURL struct {
	UrlElements []string
}

type ScrapeNewsHTML struct {
	TextTxt, CategoryTxt, PostedFormat, TitleTxt string
	PostedAttr, ImageAttr                        []string
}

type Scraper struct {
	UserAgent      string
	Collector      *colly.Collector
	ReqSleepMs     int
	DebugFlag      bool
	ScrapeEntities []ScrapeEntity
}

func NewScraper(userAgent string, ScrapeEntities []ScrapeEntity, reqSleepMs int, debug bool) *Scraper {
	c := colly.NewCollector()
	c.UserAgent = userAgent

	return &Scraper{
		UserAgent:      userAgent,
		Collector:      c,
		ReqSleepMs:     reqSleepMs,
		DebugFlag:      debug,
		ScrapeEntities: ScrapeEntities,
	}
}

func (s *Scraper) ScrapeNewsUrlsFromSources() []string {
	var newsUrls []string

	for i := 0; i < len(s.ScrapeEntities); i++ {

		s.Collector.OnError(func(_ *colly.Response, err error) {
			log.Println("Something went wrong: ", err)
		})

		for _, htmlElement := range s.ScrapeEntities[i].ScrapeNewsUrlsElements.UrlElements {
			s.Collector.OnHTML(htmlElement, func(e *colly.HTMLElement) {
				newsUrls = append(newsUrls, e.ChildAttr("a", "href"))
			})
			s.Collector.Visit(s.ScrapeEntities[i].SourceUrl)
			time.Sleep(time.Duration(s.ReqSleepMs) * time.Millisecond)
		}
	}

	newsUrls = s.removeDuplicatesAndRootSite(newsUrls)

	if s.DebugFlag {
		log.Printf("DEBUG: newsUrls: %v", newsUrls)
	}

	return newsUrls
}

func (s *Scraper) ScrapeNewsFromNewsUrls(newsUrls []string) ([]NewsItem, error) {
	if len(newsUrls) == 0 {
		return nil, fmt.Errorf("the NewsUrls is empty. Please make sure to run scraper.ScrapeNewsUrlsFromSources() first")
	} else {
		var newsItems []NewsItem

		for i := 0; i < len(newsUrls); i++ {
			var newsItem NewsItem
			newsItem.Url = newsUrls[i]

			for j := 0; j < len(s.ScrapeEntities); j++ {
				u, err := url.Parse(s.ScrapeEntities[j].SourceUrl)
				if err != nil {
					log.Printf("Error! the url %s can't be parsed! Proceeding without it", s.ScrapeEntities[j].SourceUrl)
					break
				}

				if strings.Contains(newsUrls[i], u.Scheme+"://"+u.Host) {
					s.Collector.OnError(func(_ *colly.Response, err error) {
						log.Println("Something went wrong: ", err)
					})

					s.Collector.OnHTML(s.ScrapeEntities[j].ScrapeNewsHTMLElements.TextTxt, func(e *colly.HTMLElement) {
						// Workaround for the articles without the heading <p> element in the div with the post
						text := strings.TrimSpace(e.Text)
						if text != "" {
							newsItem.Text = append(newsItem.Text, text)
						}
					})

					s.Collector.OnHTML(s.ScrapeEntities[j].ScrapeNewsHTMLElements.CategoryTxt, func(e *colly.HTMLElement) {
						newsItem.Category = e.Text
					})

					s.Collector.OnHTML(s.ScrapeEntities[j].ScrapeNewsHTMLElements.TitleTxt, func(e *colly.HTMLElement) {
						newsItem.Title = e.Text
					})

					s.Collector.OnHTML(s.ScrapeEntities[j].ScrapeNewsHTMLElements.PostedAttr[0], func(e *colly.HTMLElement) {
						newsItem.Posted = formatTime(e.Attr(s.ScrapeEntities[j].ScrapeNewsHTMLElements.PostedAttr[1]), s.ScrapeEntities[j].ScrapeNewsHTMLElements.PostedFormat)
					})

					s.Collector.OnHTML(s.ScrapeEntities[j].ScrapeNewsHTMLElements.ImageAttr[0], func(e *colly.HTMLElement) {
						newsItem.Image = e.Attr(s.ScrapeEntities[j].ScrapeNewsHTMLElements.ImageAttr[1])
					})
					s.Collector.Visit(newsUrls[i])

					if s.DebugFlag {
						log.Printf("DEBUG: i = %d, processing url: %s len(newsUrls): %d\n", i, newsUrls[i], len(newsUrls))
					}
					newsItem.P1 = newsItem.Text[0] + " " + newsItem.Text[1]
					time.Sleep(time.Duration(s.ReqSleepMs) * time.Millisecond)

					newsItems = append(newsItems, newsItem)
					break
				}
			}
		}
		return newsItems, nil
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

	for i := 0; i < len(s.ScrapeEntities); i++ {
		uniqueMap[s.ScrapeEntities[i].SourceUrl] = true
	}

	for _, url := range newsUrls {
		if url != "" && !uniqueMap[url] {
			uniqueMap[url] = true
			result = append(result, url)
		}
	}
	return result
}

func PickRandomUserAgent() string {
	userAgents := []string{
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/116.0.0.0 Safari/537.36",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 13_5_1) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/116.0.0.0 Safari/537.36",
		"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/116.0.0.0 Safari/537.36",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/116.0.0.0 Safari/537.36 Edg/116.0.1938.76",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/116.0.0.0 Safari/537.36 Edg/116.0.1938.62",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 13_5_1) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/116.0.0.0 Safari/537.36 Edg/116.0.1938.76",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/116.0.0.0 Safari/537.36 OPR/102.0.0.0",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 13_5_1) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/116.0.0.0 Safari/537.36 OPR/102.0.0.0",
		"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/116.0.0.0 Safari/537.36 OPR/102.0.0.0",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:109.0) Gecko/20100101 Firefox/117.0",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 13.5; rv:109.0) Gecko/20100101 Firefox/117.0",
		"Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:109.0) Gecko/20100101 Firefox/117.0",
	}

	rand.New(rand.NewSource(time.Now().UnixNano()))
	randomIndex := rand.Intn(len(userAgents))

	return userAgents[randomIndex]
}
