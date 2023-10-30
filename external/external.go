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

	resp, err := http.Get(fmt.Sprintf("%s?chat_id=%s&photo=%s&caption=%s&parse_mode=html", url, chatId, item.Image, caption))
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		log.Printf("Request: %s?chat_id=%s&photo=%s&caption=%s&parse_mode=html", url, chatId, item.Image, caption)
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
		"\xF0\x9F\x98\x82",
		"\xF0\x9F\x98\x84",
		"\xF0\x9F\x98\x85",
		"\xF0\x9F\x98\x89",
		"\xF0\x9F\x98\x8B",
		"\xF0\x9F\x98\x8D",
		"\xF0\x9F\x98\x9C",
		"\xF0\x9F\x99\x8F",
		"\xE2\x9A\xA1",
	}

	messageEndings := []string{
		"Подписывайся! У нас только хорошие новости!",
		"Жми сюда, если надоел Doom Scrolling",
		"Если понравилось, заходи. У нас есть ещё!",
		"Ждём тебя на нашем канале!",
		"Не упусти свою порцию позитива!",
		"Брось Doom Scrolling и присоединяйся!",
		"У нас всегда только светлая сторона новостей!",
		"Забудь о плохих новостях на нашем канале!",
		"Больше хороших новостей на нашем канале!",
		"С нами ты всегда найдёшь причину улыбнуться!",
		"Подними себе настроение на нашем канале!",
		"Позитивные истории ждут тебя! Присоединяйся!",
		"Подписывайся на лучший канал новостей!",
		"Ищешь хорошие новости? Тебе сюда!",
		"Присоединяйся и делай мир ярче вместе с нами!",
		"Вместе мы сделаем этот мир лучше!",
		"Лучшие новости каждый день - только у нас!",
		"Поделись позитивом с друзьями!",
		"Жми на кнопку подписки и получай дозу счастья!",
		"Не упусти возможность улучшить свой день!",
		"Подписка на счастье всего в одном клике!",
		"Новости, которые поднимут настроение!",
		"Улучши свое настроение с нашими новостями!",
		"Жми на кнопку подписки и окунись в мир позитива!",
		"Подними себе настроение каждый день!",
		"Делай мир ярче, присоединяйся к нам!",
		"С нами каждый день - праздник добрых новостей!",
	}

	rand.New(rand.NewSource(time.Now().UnixNano()))
	randomIndexMsg := rand.Intn(len(messageEndings))
	randomIndexEmj := rand.Intn(len(emoji))

	result := fmt.Sprintf("%v <a href='t.me/nomoredoomscrolling'>%v</a>", emoji[randomIndexEmj], messageEndings[randomIndexMsg])

	return result
}

func assembleCaption(item scraping.NewsItem, maxLength int, postDatetime bool) string {
	resultText := ""
	caption := ""
	newsText := ""

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
		if len(item.P1) >= maxLength {
			newsText = strings.Split(item.P1, "\n")[0]
		} else {
			newsText = item.P1
		}
	} else {
		newsText = resultText
	}

	caption = fmt.Sprintf("<a href='%v'>%s: %s</a>\n\n%s\n\n%s%s", item.Url, item.Category, item.Title, newsText, dateTime, pickRandomMessageEnding())
	caption = strings.ReplaceAll(caption, "\n\n\n", "\n\n")
	caption = strings.ReplaceAll(caption, "*", "")

	return url.QueryEscape(caption)
}
