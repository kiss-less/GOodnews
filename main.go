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
	db, err := database.InitDB(dryRun, "data/news_items.db")

	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	scrapeEntities := []scraping.ScrapeEntity{
		{
			SourceUrl: "https://positivnews.ru/",
			ScrapeNewsUrlsElements: scraping.ScrapeNewsURL{
				UrlElements: []string{"div.digital-newspaper-container", "article.post"},
			},
			ScrapeNewsHTMLElements: scraping.ScrapeNewsHTML{
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
			ScrapeNewsUrlsElements: scraping.ScrapeNewsURL{
				UrlElements: []string{"div.entry-image"},
			},
			ScrapeNewsHTMLElements: scraping.ScrapeNewsHTML{
				TextTxt:      "div[id=cont_post] p",
				CategoryTxt:  "div.entry-meta a[href=\"https://ntdtv.ru/\"]",
				PostedAttr:   []string{"span.entry-date time", "datetime"},
				PostedFormat: "2006-01-02 15:04:05",
				TitleTxt:     "header.entry-header h1",
				ImageAttr:    []string{"link[itemprop=thumbnailUrl]", "href"},
			},
		},
		{
			SourceUrl: "https://allpozitive.ru/",
			ScrapeNewsUrlsElements: scraping.ScrapeNewsURL{
				UrlElements: []string{"div.col-1-2.mq-sidebar div.sb-widget ul.cp-widget.row.clearfix li.cp-wrap.clearfix div.cp-data p.cp-widget-title"},
			},
			ScrapeNewsHTMLElements: scraping.ScrapeNewsHTML{
				TextTxt:      "div.entry.clearfix",
				CategoryTxt:  "header.post-header p.meta.post-meta a[rel=\"category tag\"]",
				PostedAttr:   []string{"p.meta.post-meta", "datetime"},
				PostedFormat: "2006-01-02T15:04:05-07:00",
				TitleTxt:     "h1.post-title",
				ImageAttr:    []string{"div.post-thumbnail img", "src"},
				PostedTextToParse: scraping.TextToParse{
					Regex:  `\d{2}\.\d{2}\.\d{4}`,
					Layout: "02.01.2006",
				},
			},
		},
	}

	s := scraping.NewScraper(scraping.PickRandomUserAgent(), scrapeEntities, 500, debug)
	newsUrls := s.ScrapeNewsUrlsFromSources()
	var newsUrlsNotAlreadyInDB []string

	for i := 0; i < len(newsUrls); i++ {
		urlExists, err := database.CheckIfRecordWithUrlExists(dryRun, debug, db, newsUrls[i])
		if err != nil {
			log.Printf("Error checking the url %s in the db: %v\n", newsUrls[i], err)
		}

		if !urlExists {
			newsUrlsNotAlreadyInDB = append(newsUrlsNotAlreadyInDB, newsUrls[i])
		}
	}
	if debug {
		log.Printf("DEBUG: len(newsUrlsNotAlreadyInDB): %d", len(newsUrlsNotAlreadyInDB))
		log.Printf("DEBUG: newsUrlsNotAlreadyInDB: %v", newsUrlsNotAlreadyInDB)
	}

	newsItems, err = s.ScrapeNewsFromNewsUrls(newsUrlsNotAlreadyInDB)

	if err != nil {
		log.Printf("Error from s.ScrapeNewsFromNewsUrls: %v\n", err)
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
		err := database.CheckAndInsertItem(dryRun, db, item, 2)
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
