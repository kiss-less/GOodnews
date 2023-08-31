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
