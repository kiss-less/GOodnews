## Goodnews

## Disclaimer: Proper Attribution of News Sources for Scraped Content

*This news scraping application is designed to aggregate and present news articles from various sources across the internet. While we strive to provide accurate and up-to-date information, it is important to note that the responsibility for proper attribution of news sources lies with each individual user of the application.*

*By using this application, you acknowledge that it is your responsibility to always mention and attribute the original source of any news content that is obtained through the scraping process. This includes providing appropriate credit to the authors, publications, and websites from which the news articles originate.*

*We encourage all users to respect the intellectual property rights of news publishers and content creators. Failure to attribute sources correctly may infringe upon copyright and ethical guidelines. Additionally, inaccurate or misleading attribution can undermine the credibility and integrity of the news dissemination process.*

This GO package is scrapping websites with good news, saving the news from the last 2 days (in case they were not already saved) to the SQLite3 DB (by default `./data/news_items.db`) and sending them to the external service, specified in the `external.SendToExternalService` function (Telegram by default), sorted by the posted date. Once the item is sent, it is marked accordingly in the db and won't be sent again.

### Pre-req and installation

To run/build the project, you need GO 1.20+ and the packages mentioned in `go.mod`. In all the below commands, do not forget to replace `<YOUR_TELEGRAM_BOT_API_KEY>` and `<ID_OF_THE_CHAT_TO_SEND_NEWS_TO>` placeholders with the actual values.

#### To run the project:

`mkdir data && API_KEY=<YOUR_TELEGRAM_BOT_API_KEY> CHAT_ID=<ID_OF_THE_CHAT_TO_SEND_NEWS_TO> go run main.go`

#### To build and run the project:

```
go build -o goodnews
chmod +x ./goodnews
export API_KEY=<YOUR_TELEGRAM_BOT_API_KEY>
export CHAT_ID=<ID_OF_THE_CHAT_TO_SEND_NEWS_TO>
mkdir data
./goodnews
```

#### To build and run with Docker:

```
docker build -t goodnews:<VERSION> .
docker volume create goodnews_data
docker run --rm -v goodnews_data:/app/data -e API_KEY=<YOUR_TELEGRAM_BOT_API_KEY> -e CHAT_ID=<ID_OF_THE_CHAT_TO_SEND_NEWS_TO> goodnews:<VERSION>
```

If you already have a SQLite3 db file, and you'd like to use it with the newly created docker container, you can run the following command from the directory with the file to copy the file to the volume before you run the container (make sure that you have jq installed in your system):

`cp ./news_items.db $(docker volume inspect goodnews_data | jq -r '.[0].Mountpoint')/news_items.db`

#### Additional information

Since version 1.1.0 you can use the following flags (both of them are turned off by default):

* `--dry-run` to perform a dry run. We are still going to scrape the sources, but no write actions will be done to DB and message won't be sent to external source
* `--debug` to output more information during the run

Usage example:

`go run main.go --dry-run --debug`
