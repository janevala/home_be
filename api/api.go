// api/api.go
package api

import (
	"database/sql"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"

	"github.com/google/uuid"
	B "github.com/janevala/home_be/build"
	Conf "github.com/janevala/home_be/config"
	_ "github.com/lib/pq"
	"github.com/mmcdole/gofeed"
)

type LoginObject struct {
	Username     string `json:"username"`
	Password     string `json:"password"`
	GrantType    string `json:"grant_type"`
	ClientId     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

type NewsItem struct {
	Source          string     `json:"source,omitempty"`
	Title           string     `json:"title,omitempty"`
	Description     string     `json:"description,omitempty"`
	Content         string     `json:"content,omitempty"`
	Link            string     `json:"link,omitempty"`
	Published       string     `json:"published,omitempty"`
	PublishedParsed *time.Time `json:"publishedParsed,omitempty"`
	LinkImage       string     `json:"linkImage,omitempty"`
	Uuid            string     `json:"uuid,omitempty"`
	Llm             string     `json:"llm,omitempty"`
	Language        string     `json:"language,omitempty"`
}

type NewsItems struct {
	Items      []NewsItem `json:"items"`
	TotalItems int        `json:"totalItems"`
	Limit      int        `json:"limit"`
	Offset     int        `json:"offset"`
}

type ArchiveRefreshResponse struct {
	Status string `json:"status"`
	Count  int    `json:"count"`
	Oldest string `json:"oldest"`
}

func SitesHandler(sites Conf.SitesConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		switch req.Method {
		case http.MethodGet:
			if !strings.Contains(req.URL.RawQuery, "code=123") {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte("Invalid"))
				return
			}

			responseJson, _ := json.Marshal(sites)
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.WriteHeader(http.StatusOK)
			w.Write(responseJson)
		}
	}
}

func ArchiveRefreshHandler(sites Conf.SitesConfig, db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		switch req.Method {
		case http.MethodGet:
			if !strings.Contains(req.URL.RawQuery, "code=123") {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte("Invalid"))
				return
			}

			row, err := db.Query("SELECT created FROM feed_items ORDER BY created DESC LIMIT 1")

			if err != nil {
				B.LogErr(err)
				http.Error(w, "Database refresh error", http.StatusInternalServerError)
				return
			}

			defer row.Close()

			if row != nil {
				for row.Next() {
					var now = time.Now()
					var lastCreated time.Time
					err := row.Scan(&lastCreated)

					if err != nil {
						B.LogErr(err)
						http.Error(w, "Database scan error", http.StatusInternalServerError)
						return
					}

					if now.Sub(lastCreated) > 2*time.Hour {
						B.LogOut("Starting archive refresh...")
						B.LogOut("Last refresh was at: " + lastCreated.String())
						B.LogOut("Current time is: " + now.String())

						var wg sync.WaitGroup
						wg.Add(1)

						go func() {
							defer wg.Done()
							defer B.LogOut("Crawling completed")

							crawl(sites, db)

							var records int
							err := db.QueryRow("SELECT COUNT(*) FROM feed_items").Scan(&records)
							if err != nil {
								http.Error(w, "Database scan error", http.StatusInternalServerError)
								return
							}

							var oldest time.Time
							err = db.QueryRow("SELECT published_parsed FROM feed_items ORDER BY published_parsed ASC LIMIT 1").Scan(&oldest)
							if err != nil {
								http.Error(w, "Database scan error", http.StatusInternalServerError)
								return
							}

							archiveRefreshResponse := ArchiveRefreshResponse{
								Status: "Refreshed",
								Count:  records,
								Oldest: oldest.String(),
							}

							responseJson, _ := json.Marshal(archiveRefreshResponse)
							w.Header().Set("Content-Type", "application/json")
							w.WriteHeader(http.StatusOK)
							w.Write(responseJson)
						}()

						wg.Wait()
					} else {
						var records int
						err := db.QueryRow("SELECT COUNT(*) FROM feed_items").Scan(&records)
						if err != nil {
							http.Error(w, "Database scan error", http.StatusInternalServerError)
							return
						}

						var oldest time.Time
						err = db.QueryRow("SELECT published_parsed FROM feed_items ORDER BY published_parsed ASC LIMIT 1").Scan(&oldest)
						if err != nil {
							http.Error(w, "Database scan error", http.StatusInternalServerError)
							return
						}

						archiveRefreshResponse := ArchiveRefreshResponse{
							Status: "Not needed",
							Count:  records,
							Oldest: oldest.String(),
						}

						responseJson, _ := json.Marshal(archiveRefreshResponse)
						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(http.StatusOK)
						w.Write(responseJson)
					}
				}
			}
		}
	}
}

func ArchiveHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		switch req.Method {
		case http.MethodGet:
			if !strings.Contains(req.URL.RawQuery, "code=123") {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte("Invalid"))
				return
			}

			query := req.URL.Query()

			limit := 10
			offset := 0
			language := "en"

			if l := query.Get("limit"); l != "" {
				if l, err := strconv.Atoi(l); err == nil && l > 0 && l < 1000 {
					limit = l
				}
			}

			if o := query.Get("offset"); o != "" {
				if o, err := strconv.Atoi(o); err == nil && o >= 0 && o < 1000000 {
					// if we have over million news items, thats a positive problem
					offset = o
				}
			}

			if L := query.Get("lang"); L != "" {
				if L == "en" || L == "de" || L == "fi" || L == "th" {
					language = L
				}
			}

			// Get total count of items
			// var totalItems int
			// err := db.QueryRow("SELECT COUNT(*) FROM feed_items").Scan(&totalItems)
			// if err != nil {
			// 	B.LogErr(err)
			// 	http.Error(w, "Database count error", http.StatusInternalServerError)
			// 	return
			// }

			if language == "en" {
				rows, err := db.Query(
					`SELECT title, description, link, published, published_parsed, source, thumbnail, uuid 
					FROM feed_items 
					ORDER BY published_parsed DESC 
					LIMIT $1 OFFSET $2`,
					limit, offset)
				if err != nil {
					B.LogErr(err)
					http.Error(w, "Database query error", http.StatusInternalServerError)
					return
				}

				var source string
				var title string
				var description string
				var link string
				var published string
				var published_parsed *time.Time
				var thumbnail string
				var uuid string
				var llm string = "original"

				items := []NewsItem{}
				for rows.Next() {
					err := rows.Scan(&title, &description, &link, &published, &published_parsed, &source, &thumbnail, &uuid)
					if err != nil {
						B.LogErr(err)
						http.Error(w, "Database scan error", http.StatusInternalServerError)
						return
					}

					items = append(items, NewsItem{
						Source:          source,
						Title:           title,
						Description:     description,
						Link:            link,
						Published:       published,
						PublishedParsed: published_parsed,
						LinkImage:       thumbnail,
						Uuid:            uuid,
						Llm:             llm,
						Language:        language,
					})
				}

				defer rows.Close()

				newsItems := NewsItems{
					Items:      items,
					TotalItems: 0,
					Limit:      limit,
					Offset:     offset,
				}

				responseJson, _ := json.Marshal(newsItems)
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("Access-Control-Allow-Origin", "*")
				w.WriteHeader(http.StatusOK)
				w.Write(responseJson)
			} else {
				rows, err := db.Query(`SELECT fi.link, fi.published, fi.source, fi.thumbnail, fi.uuid,
					ft.published_parsed, ft.language, ft.title, ft.description, ft.llm
					FROM feed_translations ft
					JOIN feed_items fi ON fi.id = ft.item_id
					WHERE ft.language = $3
					ORDER BY ft.published_parsed DESC 
					LIMIT $1 OFFSET $2`, limit, offset, language)

				if err != nil {
					B.LogErr(err)
					http.Error(w, "Database query error", http.StatusInternalServerError)
					return
				}

				var fiLink string
				var fiPublished string
				var fiSource string
				var fiThumbnail string
				var fiUuid string

				var ftPublishedParsed *time.Time
				var ftLanguage string
				var ftTitle string
				var ftDescription string
				var ftLlm string

				items := []NewsItem{}
				for rows.Next() {
					err := rows.Scan(&fiLink, &fiPublished, &fiSource, &fiThumbnail, &fiUuid, &ftPublishedParsed, &ftLanguage, &ftTitle, &ftDescription, &ftLlm)
					if err != nil {
						B.LogErr(err)
						http.Error(w, "Database scan error", http.StatusInternalServerError)
						return
					}

					items = append(items, NewsItem{
						Source:          fiSource,
						Title:           ftTitle,
						Description:     ftDescription,
						Link:            fiLink,
						Published:       fiPublished,
						PublishedParsed: ftPublishedParsed,
						LinkImage:       fiThumbnail,
						Uuid:            fiUuid,
						Llm:             ftLlm,
						Language:        ftLanguage,
					})
				}

				defer rows.Close()

				newsItems := NewsItems{
					Items:      items,
					TotalItems: totalItems,
					Limit:      limit,
					Offset:     offset,
				}

				responseJson, _ := json.Marshal(newsItems)
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("Access-Control-Allow-Origin", "*")
				w.WriteHeader(http.StatusOK)
				w.Write(responseJson)
			}
		}
	}
}

func SearchHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		switch req.Method {
		case http.MethodGet:
			if !strings.Contains(req.URL.RawQuery, "code=123") {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte("Invalid"))
				return
			}

			query := req.URL.Query()

			searchQuery := query.Get("q")

			if searchQuery == "" || stringLength(searchQuery) > 20 {
				B.LogOut("Search query invalid")
				http.Error(w, "Search query invalid", http.StatusBadRequest)
				return
			}

			language := "en"

			if L := query.Get("lang"); L != "" {
				if L == "en" || L == "de" || L == "fi" || L == "th" {
					language = L
				}
			}

			if language == "en" {
				rows, err := db.Query(
					`SELECT title, description, link, published, published_parsed, source, thumbnail, uuid 
					FROM feed_items 
					WHERE title ILIKE '%' || $1 || '%' 
					OR description ILIKE '%' || $1 || '%'
					OR source ILIKE '%' || $1 || '%'
					ORDER BY published_parsed DESC
					LIMIT 50`, searchQuery)

				if err != nil {
					B.LogErr(err)
					http.Error(w, "Database query error", http.StatusInternalServerError)
					return
				}

				defer rows.Close()

				var source string
				var title string
				var description string
				var link string
				var published string
				var published_parsed *time.Time
				var thumbnail string
				var uuid string
				var llm string = "original"

				items := []NewsItem{}
				for rows.Next() {
					err := rows.Scan(&title, &description, &link, &published, &published_parsed, &source, &thumbnail, &uuid)
					if err != nil {
						B.LogErr(err)
						http.Error(w, "Database scan error", http.StatusInternalServerError)
						return
					}

					items = append(items, NewsItem{
						Source:          source,
						Title:           title,
						Description:     description,
						Link:            link,
						Published:       published,
						PublishedParsed: published_parsed,
						LinkImage:       thumbnail,
						Uuid:            uuid,
						Llm:             llm,
						Language:        language,
					})
				}

				defer rows.Close()

				newsItems := NewsItems{
					Items:      items,
					TotalItems: len(items),
					Limit:      0,
					Offset:     0,
				}

				responseJson, _ := json.Marshal(newsItems)
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("Access-Control-Allow-Origin", "*")
				w.WriteHeader(http.StatusOK)
				w.Write(responseJson)
			} else {
				rows, err := db.Query(
					`SELECT fi.link, fi.published, fi.source, fi.thumbnail, fi.uuid,
					ft.published_parsed, ft.language, ft.title, ft.description, ft.llm
					FROM feed_translations ft
					JOIN feed_items fi ON fi.id = ft.item_id
					WHERE ft.language = $2
					AND (ft.title ILIKE '%' || $1 || '%'
					OR ft.description ILIKE '%' || $1 || '%'
					OR fi.source ILIKE '%' || $1 || '%')
					ORDER BY ft.published_parsed DESC
					LIMIT 50`, searchQuery, language)

				if err != nil {
					B.LogErr(err)
					http.Error(w, "Database query error", http.StatusInternalServerError)
					return
				}

				var fiLink string
				var fiPublished string
				var fiSource string
				var fiThumbnail string
				var fiUuid string

				var ftPublishedParsed *time.Time
				var ftLanguage string
				var ftTitle string
				var ftDescription string
				var ftLlm string

				items := []NewsItem{}
				for rows.Next() {
					err := rows.Scan(&fiLink, &fiPublished, &fiSource, &fiThumbnail, &fiUuid, &ftPublishedParsed, &ftLanguage, &ftTitle, &ftDescription, &ftLlm)
					if err != nil {
						B.LogErr(err)
						http.Error(w, "Database scan error", http.StatusInternalServerError)
						return
					}

					items = append(items, NewsItem{
						Source:          fiSource,
						Title:           ftTitle,
						Description:     ftDescription,
						Link:            fiLink,
						Published:       fiPublished,
						PublishedParsed: ftPublishedParsed,
						LinkImage:       fiThumbnail,
						Uuid:            fiUuid,
						Llm:             ftLlm,
						Language:        ftLanguage,
					})
				}

				defer rows.Close()

				newsItems := NewsItems{
					Items:      items,
					TotalItems: len(items),
					Limit:      0,
					Offset:     0,
				}

				responseJson, _ := json.Marshal(newsItems)
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("Access-Control-Allow-Origin", "*")
				w.WriteHeader(http.StatusOK)
				w.Write(responseJson)
			}
		}
	}
}

func FakeAuthHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		switch req.Method {
		case http.MethodPost:
			var bodyBytes []byte
			var err error

			if req.Body != nil {
				bodyBytes, err = io.ReadAll(req.Body)
				if err != nil {
					B.LogErr(err)
					return
				}
				defer req.Body.Close()
			}

			var loginObject LoginObject
			var jsonString bytes.Buffer

			if len(bodyBytes) > 0 {
				if err = json.Indent(&jsonString, bodyBytes, "", "\t"); err != nil {
					B.LogErr(err)
					return
				}
				err := json.Unmarshal(bodyBytes, &loginObject)
				if err != nil {
					B.LogErr(err)
					return
				}
			} else {
				B.LogOut("Body: No Body Supplied")
			}

			if (loginObject.Username == "123") && (loginObject.Password == "123") {
				B.LogOut("Logged in as %s", loginObject.Username)

				w.Header().Set("Access-Control-Allow-Origin", "*")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(loginObject.Username))
			} else {
				B.LogOut("Invalid login attempt for user %s", loginObject.Username)

				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte("Invalid Credentials"))
			}
		}
	}
}

func crawl(sites Conf.SitesConfig, db *sql.DB) {
	feedParser := gofeed.NewParser()

	var combinedItems []*NewsItem = []*NewsItem{}
	for i := 0; i < len(sites.Sites); i++ {
		feed, err := feedParser.ParseURL(sites.Sites[i].Url)
		if err != nil {
			B.LogErr(err)
		} else {
			if feed.Image != nil {
				for j := 0; j < len(feed.Items); j++ {
					feed.Items[j].Image = feed.Image
				}
			} else {
				for j := 0; j < len(feed.Items); j++ {
					feed.Items[j].Image = &gofeed.Image{
						URL:   "https://github.com/janevala/home_be_crawler.git",
						Title: "N/A",
					}
				}
			}

			var items []*NewsItem = []*NewsItem{}
			for j := 0; j < len(feed.Items); j++ {
				NewsItem := &NewsItem{
					Source:          sites.Sites[i].Title,
					Title:           strings.TrimSpace(feed.Items[j].Title),
					Description:     feed.Items[j].Description,
					Content:         feed.Items[j].Content,
					Link:            feed.Items[j].Link,
					Published:       feed.Items[j].Published,
					PublishedParsed: feed.Items[j].PublishedParsed,
					LinkImage:       feed.Items[j].Image.URL,
					Uuid:            uuid.NewString(),
				}

				items = append(items, NewsItem)
			}

			combinedItems = append(combinedItems, items...)
		}
	}

	if len(combinedItems) > 0 {
		for i := 0; i < len(combinedItems); i++ {
			combinedItems[i].Description = ellipticalTruncate(combinedItems[i].Description, 950)

			// Hashing title to create unique ID, that serves as mechanism to prevent duplicates in DB
			uuidString := base64.StdEncoding.EncodeToString([]byte(ellipticalTruncate(combinedItems[i].Title, 35)))
			combinedItems[i].Uuid = uuidString
		}

		sort.Slice(combinedItems, func(i, j int) bool {
			return combinedItems[i].PublishedParsed.After(*combinedItems[j].PublishedParsed)
		})

		createTableIfNeeded(db)

		var pkAccumulated int
		for i := 0; i < len(combinedItems); i++ {
			var pk = insertItem(db, combinedItems[i])
			if pk == 0 {
				continue
			}

			if pk <= pkAccumulated {
				B.LogOut("PK minor error")
			} else {
				pkAccumulated = pk
			}
		}
	}
}

func createTableIfNeeded(db *sql.DB) {
	query := `CREATE TABLE IF NOT EXISTS feed_items (
		id SERIAL PRIMARY KEY,
		title VARCHAR(500) NOT NULL,
		description VARCHAR(1000) NOT NULL,
		link VARCHAR(500) NOT NULL,
		published timestamp NOT NULL,
		published_parsed timestamp NOT NULL,
		source VARCHAR(300) NOT NULL,
		thumbnail VARCHAR(500),
		uuid VARCHAR(300) NOT NULL,
		language VARCHAR(10),
		created timestamp DEFAULT NOW(),
		UNIQUE (uuid)
	)`

	_, err := db.Exec(query)
	if err != nil {
		B.LogErr(err)
		os.Exit(1)
	}
}

func insertItem(db *sql.DB, item *NewsItem) int {
	query := "INSERT INTO feed_items (title, description, link, published, published_parsed, source, thumbnail, uuid) VALUES ($1, $2, $3, $4, $5, $6, $7, $8) ON CONFLICT DO NOTHING RETURNING id"

	var pk int
	err := db.QueryRow(query, item.Title, item.Description, item.Link, item.Published, item.PublishedParsed, item.Source, item.LinkImage, item.Uuid).Scan(&pk)

	if err != nil {
		B.LogOut(err.Error() + " - duplicate uuid: " + item.Uuid)
	} else {
		B.LogOut("Inserted item (pk: " + strconv.Itoa(pk) + "): " + ellipticalTruncate(item.Title, 35))
	}

	return pk
}

// https://stackoverflow.com/a/73939904 find better way with AI if needed
func ellipticalTruncate(text string, maxLen int) string {
	lastSpaceIx := maxLen
	len := 0
	for i, r := range text {
		if unicode.IsSpace(r) {
			lastSpaceIx = i
		}
		len++
		if len > maxLen {
			return text[:lastSpaceIx] + "..."
		}
	}

	return text
}

func stringLength(str string) int {
	var length int
	for range str {
		length++
	}
	return length
}
