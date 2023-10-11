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
	"strings"
	"time"
)

func SendToExternalService(item scraping.NewsItem) error {
	apiKey := os.Getenv("API_KEY")
	chatId := os.Getenv("CHAT_ID")

	// As of now, Telegram API allows sendPhoto's caption parameter to contain up to 1024 charactes: https://core.telegram.org/bots/api#sendphoto
	caption := assembleCaption(item, 900, false)
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
	emoji := []string{
		"\xF0\x9F\x98\x8A",
		"\xF0\x9F\x98\x8F",
		"\xF0\x9F\x91\x8C",
		"\xF0\x9F\x91\x8D",
		"\xF0\x9F\x91\x80",
		"\xF0\x9F\x98\xB8",
		"\xF0\x9F\x98\x81",
		"\xF0\x9F\x98\x83",
		"\xF0\x9F\x98\x87",
		"\xF0\x9F\x98\x8E",
		"\xF0\x9F\x9A\x80",
		"\xE2\x9C\x8C",
		"\xF0\x9F\x99\x8C",
	}

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
		"[Подписывайся на лучший канал новостей!](t.me/nomoredoomscrolling)",
		"[Ищешь хорошие новости? Тебе сюда!](t.me/nomoredoomscrolling)",
		"[Присоединяйся и делай мир ярче вместе с нами!](t.me/nomoredoomscrolling)",
		"[Вместе мы сделаем этот мир лучше!](t.me/nomoredoomscrolling)",
		"[Лучшие новости каждый день - только у нас!](t.me/nomoredoomscrolling)",
		"[Поделись позитивом с друзьями!](t.me/nomoredoomscrolling)",
		"[Жми на кнопку подписки и получай дозу счастья!](t.me/nomoredoomscrolling)",
		"[Не упусти возможность улучшить свой день!](t.me/nomoredoomscrolling)",
		"[Подписка на счастье всего в одном клике!](t.me/nomoredoomscrolling)",
		"[Новости, которые поднимут настроение!](t.me/nomoredoomscrolling)",
	}

	rand.New(rand.NewSource(time.Now().UnixNano()))
	randomIndexMsg := rand.Intn(len(messageEndings))
	randomIndexEmj := rand.Intn(len(emoji))

	result := fmt.Sprintf("%v %v", emoji[randomIndexEmj], messageEndings[randomIndexMsg])

	return result
}

func assembleCaption(item scraping.NewsItem, maxLength int, postDatetime bool) string {
	resultText := ""
	caption := ""

	if len(item.Text) == 1 {
		item.Text = strings.Split(item.Text[0], "\n")
	}

	for i := 0; i < len(item.Text); i++ {
		elementLength := len(item.Text[i])
		if len(resultText)+elementLength <= maxLength {
			resultText += fmt.Sprintf("%v\n", item.Text[i])
		} else {
			break
		}
	}
	if resultText == "" {
		resultText = item.P1
	}

	var dateTime string

	if postDatetime {
		dateTime = fmt.Sprintf("%s\n\n", item.Posted)
	}

	if len(resultText) > 0 && resultText[len(resultText)-1] != '\n' {
		caption = fmt.Sprintf("*%s: %s*\n\n%s\n\n%s%s", item.Category, item.Title, item.P1, dateTime, pickRandomMessageEnding())
	} else {
		caption = fmt.Sprintf("*%s: %s*\n\n%s\n\n%s%s", item.Category, item.Title, resultText, dateTime, pickRandomMessageEnding())
	}

	caption = strings.ReplaceAll(caption, "\n\n\n", "\n\n")

	return url.QueryEscape(caption)
}
