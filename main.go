package main

import (
	"flag"
	"log"

	"goodnews/database"
	"goodnews/scraping"
)

var dryRun bool
var debug bool
var newsItems []scraping.NewsItem

func main() {
	flag.BoolVar(&dryRun, "dry-run", false, "Perform a dry run. We are still going to scrape the sources, but no write actions will be done to DB and message won't be sent to external source")
	flag.BoolVar(&debug, "debug", false, "Output more information during the run")
	flag.Parse()
	userAgent := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/116.0.0.0 Safari/537.36 Edg/116.0.1938.62"
	db, err := database.InitDB(dryRun, "data/news_items.db")

	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	scrapedEntities := []scraping.ScrapedEntity{
		{
			SourceUrl: "https://positivnews.ru/",
			ScrapeNewsUrlsElements: scraping.ScrapedNewsURL{
				UrlElements: []string{"div.digital-newspaper-container", "article.post"},
			},
			ScrapeNewsHTMLElement: scraping.ScrapedNewsElement{
				TextTxt:      ".entry-content p",
				CategoryTxt:  ".post-categories a",
				PostedAttr:   []string{".entry-meta time.updated", "datetime"},
				PostedFormat: "2006-01-02T15:04:05-07:00",
				TitleTxt:     ".entry-title",
				ImageAttr:    []string{"div.post-inner div.post-thumbnail img.wp-post-image", "src"},
			},
		},
		{
			SourceUrl: "https://ntdtv.ru/c/pozitivnye-novosti",
			ScrapeNewsUrlsElements: scraping.ScrapedNewsURL{
				UrlElements: []string{"div.entry-image"},
			},
			ScrapeNewsHTMLElement: scraping.ScrapedNewsElement{
				TextTxt:      "div[id=cont_post] p",
				CategoryTxt:  "div.entry-meta a[href=\"https://ntdtv.ru/\"]",
				PostedAttr:   []string{"span.entry-date time", "datetime"},
				PostedFormat: "2006-01-02 15:04:05",
				TitleTxt:     "header.entry-header h1",
				ImageAttr:    []string{"link[itemprop=thumbnailUrl]", "href"},
			},
		},
	}

	s := scraping.NewScraper(userAgent, scrapedEntities, 500, debug)

	newsUrls := s.ScrapeNewsUrlsFromSources()
	newsItems, err = s.ScrapeNewsFromNewsUrls(newsUrls)

	if err != nil {
		log.Printf("Error from s.ScrapeNewsFromNewsUrls: %v", err)
	}

	if debug {
		log.Printf("DEBUG: Total number of news received: %d. Processing the news...", len(newsUrls))
		for i := 0; i < len(newsUrls); i++ {
			log.Printf("DEBUG: Item # %d: %s", i, newsUrls[i])
		}
	}

	for _, item := range newsItems {
		if debug {
			log.Printf("DEBUG: %s, %s, %s, %s, %s, p1:%s\n\ntext(elements: %d):%v\n\ntext[0]:%s", item.Url, item.Category, item.Posted, item.Title, item.Image, item.P1, len(item.Text), item.Text, item.Text[0])
		}
		err := database.CheckAndInsertItem(dryRun, db, item, 14)
		if err != nil {
			log.Printf("Error processing item: %v", err)
		}
	}

	log.Println("Running processUnsentItems...")

	err = database.ProcessUnsentItems(dryRun, db, 500)
	if err != nil {
		log.Printf("Error processing unsent items: %v", err)
	}
}
