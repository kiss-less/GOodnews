package external

import (
	"fmt"
	"goodnews/scraping"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"time"
)

func SendToExternalService(item scraping.NewsItem) error {
	apiKey := os.Getenv("API_KEY")
	chatId := os.Getenv("CHAT_ID")
	caption := assembleCaption(item, 900)
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

func assembleCaption(item scraping.NewsItem, maxLength int) string {
	resultText := ""
	caption := ""
	for i := 0; i < len(item.Text); i++ {
		elementLength := len(item.Text[i])
		if len(resultText)+elementLength <= maxLength {
			resultText += item.Text[i]
		} else {
			break
		}
	}
	if resultText == "" {
		resultText = item.P1
	}

	if len(resultText) > 0 && resultText[len(resultText)-1] != '.' {
		caption = fmt.Sprintf("%s: %s\n\n%s\n\n%s\n\n%s", item.Category, item.Title, item.P1, item.Posted, pickRandomMessageEnding())
	} else {
		caption = fmt.Sprintf("%s: %s\n\n%s\n\n%s\n\n%s", item.Category, item.Title, resultText, item.Posted, pickRandomMessageEnding())
	}

	return url.QueryEscape(caption)
}
