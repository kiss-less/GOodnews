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
	"time"

	"goodnews/scraping"

	_ "github.com/mattn/go-sqlite3"
)

var dryRun bool
var debug bool

func main() {
	flag.BoolVar(&dryRun, "dry-run", false, "Perform a dry run. We are still going to scrape the sources, but no write actions will be done to DB and message won't be sent to external source")
	flag.BoolVar(&debug, "debug", false, "Output more information during the run")
	flag.Parse()
	urls := []string{"https://positivnews.ru/", "https://ntdtv.ru/c/pozitivnye-novosti"}
	userAgent := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/116.0.0.0 Safari/537.36 Edg/116.0.1938.62"
	s := scraping.NewScraper(userAgent, urls, 500)
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

	var newsItems []scraping.NewsItem

	newsUrls := s.ScrapNewsUrlsFromSources()
	err, newsItems = s.ScrapNewsFromNewsUrls(newsUrls)

	if err != nil {
		log.Printf("Error from s.ScrapNewsFromNewsUrls: %v", err)
	}

	log.Printf("Total number of news received: %d. Processing the news...", len(newsUrls))

	if debug {
		for i := 0; i < len(newsUrls); i++ {
			log.Printf("DEBUG: Item # %d: %s", i, newsUrls[i])
		}
	}

	for _, item := range newsItems {
		if debug {
			log.Printf("DEBUG: %s, %s, %s, %s, %s, p1:%s\n\ntext(elements: %d):%v\n\ntext[0]:%s", item.Url, item.Category, item.Posted, item.Title, item.Image, item.P1, len(item.Text), item.Text, item.Text[0])
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

func checkAndInsertItem(db *sql.DB, item scraping.NewsItem) error {
	if !dryRun {
		query := "SELECT COUNT(*) FROM news_items WHERE url = ?"
		var count int
		err := db.QueryRow(query, item.Url).Scan(&count)
		if err != nil {
			return err
		}

		currentTime := time.Now()

		postedTime, err := time.Parse("02-01-2006 15:04:05", item.Posted)
		if err != nil {
			return err
		}

		daysDiff := currentTime.Sub(postedTime).Hours() / 24

		if count > 0 {
			log.Printf("Item with URL '%s' already exists in the database.", item.Url)
			return nil
		}

		if count == 0 && daysDiff <= 14 {
			textJSON, err := json.Marshal(item.Text)
			if err != nil {
				return err
			}

			insertQuery := `
		INSERT INTO news_items (url, category, posted, title, image, text, p1, item_was_sent)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`
			_, err = db.Exec(insertQuery, item.Url, item.Category, item.Posted, item.Title, item.Image, textJSON, item.Text[0], false)
			if err != nil {
				return err
			}

			log.Printf("Item with URL '%s' inserted into the database.", item.Url)
		}
	} else {
		log.Printf("DRY-RUN: Item with URL '%s' inserted into the database.", item.Url)
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

		var unsentItems []scraping.NewsItem

		for rows.Next() {
			var id int
			var category, posted, title, image, p1 string

			if err := rows.Scan(&id, &category, &posted, &title, &image, &p1); err != nil {
				return err
			}

			item := scraping.NewsItem{
				Id:       id,
				Category: category,
				Posted:   posted,
				Title:    title,
				Image:    image,
				P1:       p1,
			}

			unsentItems = append(unsentItems, item)
		}

		if len(unsentItems) == 0 {
			log.Println("No unsent items found...")
			return nil
		}

		sort.Slice(unsentItems, func(i, j int) bool {
			timeI, _ := time.Parse("02-01-2006 15:04:05", unsentItems[i].Posted)
			timeJ, _ := time.Parse("02-01-2006 15:04:05", unsentItems[j].Posted)
			return timeI.Before(timeJ)
		})

		for _, item := range unsentItems {
			err := sendToExternalService(item)
			if err != nil {
				log.Printf("Error sending item with ID %d to external service: %v", item.Id, err)
				continue
			}

			updateQuery := "UPDATE news_items SET item_was_sent = true WHERE id = ?"
			_, err = tx.Exec(updateQuery, item.Id)
			if err != nil {
				log.Printf("Error updating item_was_sent flag for item with ID %d: %v", item.Id, err)
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

func sendToExternalService(item scraping.NewsItem) error {
	apiKey := os.Getenv("API_KEY")
	chatId := os.Getenv("CHAT_ID")

	messageEnding := pickRandomMessageEnding()
	caption := url.QueryEscape(fmt.Sprintf("%s: %s\n\n%s\n\n%s\n\n%s", item.Category, item.Title, item.P1, item.Posted, messageEnding))

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendPhoto", apiKey)

	resp, err := http.Get(fmt.Sprintf("%s?chat_id=%s&photo=%s&caption=%s&parse_mode=Markdown", url, chatId, item.Image, caption))
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		log.Printf("Request: %s?chat_id=%s&photo=%s&caption=%s&parse_mode=Markdown", url, chatId, item.Image, caption)
		log.Printf("Response body:\n%s", body)
		return fmt.Errorf("sendToExternalService request for item id: %d. StatusCode: %d", item.Id, resp.StatusCode)
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
