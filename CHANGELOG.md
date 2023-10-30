# 2.1.0

* Changed formatting to html instead of markdown to support more options
* Moved source to the title
* Adjusted `external` package to handle 2 edge-cases: remove asterisk to avoid 400 response; separate P1 in case it is too long

# 2.0.2

* Fixed a bug when text[i] items of the news were not considered for a caption

# 2.0.1

* Fixed caption indentation
* Removed datetime in caption by default 
* Added emojis before the link

# 2.0.0

* Breaking change! Introduced `scraping.ScrapeEntity` struct which is now a part of the `NewScraper` constructor. This change allows to define additional news sources without adjusting `scraping` funcs
* Added `scraping.PickRandomUserAgent` func to randomize user agent for each run
* Introduced `database.CheckIfRecordWithUrlExists` func to avoid scraping news that already exist in DB
* Added a new source
* Reduced the value of newsAgeDays to 2 days instead of 14

# 1.2.0

* Improved the project structure for better readability: separated structs and functions
* Introduced `external.assembleCaption` message formatting func to respect Telegram API limitations
* Adjusted signatures of `database.CheckAndInsertItem` and `database.ProcessUnsentItems` to include additional args
* Added more items to `external.pickRandomMessageEnding`

# 1.1.1

* Introduced 0.5sec delay between colly requests to avoid potential rate-limiting

# 1.1.0

* Added README and CHANGELOG
* Refactored `GetNewsUrls`, `GetNews` functions to support multiple sources
* Refactored `formatTime` to support dynamic layouts
* Added `https://ntdtv.ru/c/pozitivnye-novosti` as a secondary news source
* Added `pickRandomMessageEnding` function to randomize message endings
* Added dry-run and debug modes available with the flags `--dry-run` and `--debug`

# 1.0.0

* Initial version
