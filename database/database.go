package database

import (
	"database/sql"
	"encoding/json"
	"goodnews/external"
	"goodnews/scraping"
	"log"
	"sort"
	"time"
)

func InitDB(dryRun bool, filePath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", filePath)

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
		return nil, err
	}

	return db, nil
}

func CheckAndInsertItem(dryRun bool, db *sql.DB, item scraping.NewsItem) error {
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

func ProcessUnsentItems(dryRun bool, db *sql.DB) error {
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
			err := external.SendToExternalService(item)
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
