package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/gocolly/colly"
	_ "github.com/mattn/go-sqlite3"
)

type NewsItem struct {
	id                                      int
	url, category, posted, title, image, p1 string
	text                                    []string
}

var dryRun bool
var debug bool

func main() {
	flag.BoolVar(&dryRun, "dry-run", false, "Perform a dry run. We are still going to scrape the sources, but no write actions will be done to DB and message won't be sent to external source")
	flag.BoolVar(&debug, "debug", false, "Output more information during the run")
	flag.Parse()
	var news []NewsItem
	urls := []string{"https://positivnews.ru/", "https://ntdtv.ru/c/pozitivnye-novosti"}
	userAgent := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/116.0.0.0 Safari/537.36 Edg/116.0.1938.62"
	db, err := sql.Open("sqlite3", "data/news_items.db")

	if !dryRun {
		createTableSQL := `
		CREATE TABLE IF NOT EXISTS news_items (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			url TEXT,
			category TEXT,
			posted TEXT,
			title TEXT,
			image TEXT,
			text TEXT,
			p1 TEXT,
			item_was_sent BOOLEAN
		);
	`

		_, err = db.Exec(createTableSQL)
		if err != nil {
			log.Fatal(err)
		}
		log.Println("Database setup completed")
	}

	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	var newsUrl []string

	for _, url := range urls {
		newsUrl = append(newsUrl, removeDuplicatesAndRootSite(GetNewsUrls(url, userAgent), url)...)
	}

	log.Printf("Total number of news received: %d. Processing the news...", len(newsUrl))

	for i := 0; i < len(newsUrl); i++ {
		if debug {
			log.Printf("DEBUG: Item # %d: %s", i, newsUrl[i])
		}
		news = append(news, GetNews(newsUrl[i], userAgent))
	}

	for _, item := range news {
		if debug {
			log.Printf("DEBUG: %s, %s, %s, %s, %s, p1:%s\n\ntext(elements: %d):%v\n\ntext[0]:%s", item.url, item.category, item.posted, item.title, item.image, item.p1, len(item.text), item.text, item.text[0])
		}
		err := checkAndInsertItem(db, item)
		if err != nil {
			log.Printf("Error processing item: %v", err)
		}
	}

	fmt.Println("Running processUnsentItems...")

	err = processUnsentItems(db)
	if err != nil {
		log.Printf("Error processing unsent items: %v", err)
	}
}

func GetNewsUrls(url, ua string) []string {
	c := colly.NewCollector()
	c.UserAgent = ua
	urls := []string{}

	c.OnError(func(_ *colly.Response, err error) {
		log.Println("Something went wrong: ", err)
	})

	switch url {
	case "https://positivnews.ru/":
		c.OnHTML("div.digital-newspaper-container", func(e *colly.HTMLElement) {
			urls = append(urls, e.ChildAttr("a", "href"))
		})

		c.OnHTML("article.post", func(e *colly.HTMLElement) {
			urls = append(urls, e.ChildAttr("a", "href"))
		})
	case "https://ntdtv.ru/c/pozitivnye-novosti":
		c.OnHTML("div.entry-image", func(e *colly.HTMLElement) {
			urls = append(urls, e.ChildAttr("a", "href"))
		})
	default:
		log.Panic("The news source passed to GetNewsUrls func is not found!")
	}

	c.Visit(url)

	return urls
}

func GetNews(url, ua string) NewsItem {
	news := NewsItem{}
	news.url = url
	c := colly.NewCollector()
	c.UserAgent = ua

	c.OnError(func(_ *colly.Response, err error) {
		log.Println("Something went wrong: ", err)
	})

	if strings.Contains(url, "https://positivnews.ru/") {
		c.OnHTML(".entry-content p", func(e *colly.HTMLElement) {
			news.text = append(news.text, e.Text)
		})

		c.OnHTML(".post-categories a", func(e *colly.HTMLElement) {
			news.category = e.Text
		})

		c.OnHTML(".entry-title", func(e *colly.HTMLElement) {
			news.title = e.Text
		})

		c.OnHTML(".entry-meta time.updated", func(e *colly.HTMLElement) {
			news.posted = formatTime(e.Attr("datetime"), "2006-01-02T15:04:05-07:00")
		})

		c.OnHTML("div.post-inner div.post-thumbnail img.wp-post-image", func(e *colly.HTMLElement) {
			news.image = e.Attr("src")
		})
	} else if strings.Contains(url, "https://ntdtv.ru/") {
		c.OnHTML("div[id=cont_post] p", func(e *colly.HTMLElement) {
			// Workaround for the articles without the heading <p> element in the cont_post div
			text := strings.TrimSpace(e.Text)
			if text != "" {
				news.text = append(news.text, text)
			}
		})

		c.OnHTML("header.entry-header h1", func(e *colly.HTMLElement) {
			news.title = e.Text
		})

		c.OnHTML("span.entry-date time", func(e *colly.HTMLElement) {
			news.posted = formatTime(e.Attr("datetime"), "2006-01-02 15:04:05")
		})

		c.OnHTML("link[itemprop=thumbnailUrl]", func(e *colly.HTMLElement) {
			news.image = e.Attr("href")
		})
		news.category = "Мировые новости"
	} else {
		log.Panic("The news source passed to GetNewsUrls func is not found!")
	}

	c.Visit(url)
	news.p1 = news.text[0] + " " + news.text[1]
	time.Sleep(500 * time.Millisecond)

	return news
}

func formatTime(src, inputLayout string) string {

	t, err := time.Parse(inputLayout, src)
	if err != nil {
		fmt.Println("Error parsing timestamp:", err)
		return ""
	}

	return t.Format("02-01-2006 15:04:05")
}

func removeDuplicatesAndRootSite(input []string, rootUrl string) []string {
	uniqueMap := make(map[string]bool)
	result := []string{}

	for _, s := range input {
		if !uniqueMap[s] && s != rootUrl && s != "" {
			uniqueMap[s] = true
			result = append(result, s)
		}
	}

	return result
}

func checkAndInsertItem(db *sql.DB, item NewsItem) error {
	if !dryRun {
		query := "SELECT COUNT(*) FROM news_items WHERE url = ?"
		var count int
		err := db.QueryRow(query, item.url).Scan(&count)
		if err != nil {
			return err
		}

		currentTime := time.Now()

		postedTime, err := time.Parse("02-01-2006 15:04:05", item.posted)
		if err != nil {
			return err
		}

		daysDiff := currentTime.Sub(postedTime).Hours() / 24

		if count > 0 {
			log.Printf("Item with URL '%s' already exists in the database.", item.url)
			return nil
		}

		if count == 0 && daysDiff <= 14 {
			textJSON, err := json.Marshal(item.text)
			if err != nil {
				return err
			}

			insertQuery := `
		INSERT INTO news_items (url, category, posted, title, image, text, p1, item_was_sent)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`
			_, err = db.Exec(insertQuery, item.url, item.category, item.posted, item.title, item.image, textJSON, item.text[0], false)
			if err != nil {
				return err
			}

			log.Printf("Item with URL '%s' inserted into the database.", item.url)
		}
	} else {
		log.Printf("DRY-RUN: Item with URL '%s' inserted into the database.", item.url)
	}
	return nil
}

func processUnsentItems(db *sql.DB) error {
	if !dryRun {
		tx, err := db.Begin()

		if err != nil {
			return err
		}
		defer tx.Rollback()

		query := "SELECT id, category, posted, title, image, p1 FROM news_items WHERE item_was_sent = false"
		rows, err := tx.Query(query)
		if err != nil {
			return err
		}
		defer rows.Close()

		var unsentItems []NewsItem

		for rows.Next() {
			var id int
			var category, posted, title, image, p1 string

			if err := rows.Scan(&id, &category, &posted, &title, &image, &p1); err != nil {
				return err
			}

			item := NewsItem{
				id:       id,
				category: category,
				posted:   posted,
				title:    title,
				image:    image,
				p1:       p1,
			}

			unsentItems = append(unsentItems, item)
		}

		if len(unsentItems) == 0 {
			log.Println("No unsent items found...")
			return nil
		}

		sort.Slice(unsentItems, func(i, j int) bool {
			timeI, _ := time.Parse("02-01-2006 15:04:05", unsentItems[i].posted)
			timeJ, _ := time.Parse("02-01-2006 15:04:05", unsentItems[j].posted)
			return timeI.Before(timeJ)
		})

		for _, item := range unsentItems {
			err := sendToExternalService(item)
			if err != nil {
				log.Printf("Error sending item with ID %d to external service: %v", item.id, err)
				continue
			}

			updateQuery := "UPDATE news_items SET item_was_sent = true WHERE id = ?"
			_, err = tx.Exec(updateQuery, item.id)
			if err != nil {
				log.Printf("Error updating item_was_sent flag for item with ID %d: %v", item.id, err)
				tx.Rollback()
				continue
			}
		}

		if err := tx.Commit(); err != nil {
			return err
		}
	} else {
		log.Printf("DRY-RUN: Finished processUnsentItems")
	}

	return nil
}

func sendToExternalService(item NewsItem) error {
	apiKey := os.Getenv("API_KEY")
	chatId := os.Getenv("CHAT_ID")

	messageEnding := pickRandomMessageEnding()
	caption := url.QueryEscape(fmt.Sprintf("%s: %s\n\n%s\n\n%s\n\n%s", item.category, item.title, item.p1, item.posted, messageEnding))

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendPhoto", apiKey)

	resp, err := http.Get(fmt.Sprintf("%s?chat_id=%s&photo=%s&caption=%s&parse_mode=Markdown", url, chatId, item.image, caption))
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		log.Printf("Request: %s?chat_id=%s&photo=%s&caption=%s&parse_mode=Markdown", url, chatId, item.image, caption)
		log.Printf("Response body:\n%s", body)
		return fmt.Errorf("sendToExternalService request for item id: %d. StatusCode: %d", item.id, resp.StatusCode)
	}
	defer resp.Body.Close()

	return nil
}

func pickRandomMessageEnding() string {
	messageEndings := []string{
		"[Подписывайся! У нас только хорошие новости!](t.me/nomoredoomscrolling)",
		"[Жми сюда, если надоел Doom Scrolling](t.me/nomoredoomscrolling)",
		"[Если понравилось, заходи. У нас есть ещё!](t.me/nomoredoomscrolling)",
		"[Ждём тебя на нашем канале!](t.me/nomoredoomscrolling)",
		"[Не упусти свою порцию позитива!](t.me/nomoredoomscrolling)",
		"[Брось Doom Scrolling и присоединяйся!](t.me/nomoredoomscrolling)",
		"[У нас всегда только светлая сторона новостей!](t.me/nomoredoomscrolling)",
		"[Забудь о плохих новостях на нашем канале!](t.me/nomoredoomscrolling)",
		"[Больше хороших новостей на нашем канале!](t.me/nomoredoomscrolling)",
		"[С нами ты всегда найдёшь причину улыбнуться!](t.me/nomoredoomscrolling)",
		"[Подними себе настроение на нашем канале!](t.me/nomoredoomscrolling)",
		"[Позитивные истории ждут тебя! Присоединяйся!](t.me/nomoredoomscrolling)",
	}

	rand.New(rand.NewSource(time.Now().UnixNano()))
	randomIndex := rand.Intn(len(messageEndings))

	return messageEndings[randomIndex]
}
