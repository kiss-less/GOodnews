# 1.2.0

* Improved the project structure for better readability: separated structs and functions
* Introduced `external.assembleCaption` message formatting func to respect Telegram API limitations
* Adjusted signatures of `database.CheckAndInsertItem` and `database.ProcessUnsentItems` to include additional args
#TODO: find out why there is a difference between len(newsItems) and items inserted to db and sent to the external source.

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
