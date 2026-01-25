// api/api.go
package api

import (
	"database/sql"
	"net/http"
	"sort"
	"strconv"
	"strings"
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
}

type NewsItems struct {
	Items      []NewsItem `json:"items"`
	TotalItems int        `json:"totalItems"`
	Limit      int        `json:"limit"`
	Offset     int        `json:"offset"`
}

func SitesHandler(sites Conf.SitesConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		switch req.Method {
		case http.MethodOptions:
			HandleMethodOptions(w, req, "GET, OPTIONS")
		case http.MethodGet:
			if !strings.Contains(req.URL.RawQuery, "code=123") {
				B.LogOut("Invalid URI")
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte("Invalid URI"))
				return
			}

			responseJson, _ := json.Marshal(sites)
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.WriteHeader(http.StatusOK)
			w.Write(responseJson)
		}
	}
}

func ArchiveHandler(database Conf.Database) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		switch req.Method {
		case http.MethodOptions:
			HandleMethodOptions(w, req, "GET, OPTIONS")
		case http.MethodGet:
			query := req.URL.Query()

			limit := 10
			offset := 0

			if l := query.Get("limit"); l != "" {
				if l, err := strconv.Atoi(l); err == nil && l > 0 {
					limit = l
				}
			}

			if o := query.Get("offset"); o != "" {
				if o, err := strconv.Atoi(o); err == nil && o >= 0 {
					offset = o
				}
			}

			connStr := database.Postgres
			db, err := sql.Open("postgres", connStr)

			if err != nil {
				B.LogErr(err)
				http.Error(w, "Database connection error", http.StatusInternalServerError)
				return
			}

			if err = db.Ping(); err != nil {
				B.LogErr(err)
				http.Error(w, "Database connection error", http.StatusInternalServerError)
				return
			}

			// Get total count of items
			var totalItems int
			err = db.QueryRow("SELECT COUNT(*) FROM feed_items").Scan(&totalItems)
			if err != nil {
				B.LogErr(err)
				http.Error(w, "Database count error", http.StatusInternalServerError)
				return
			}

			// Get paginated items
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

			defer rows.Close()

			var source string
			var title string
			var description string
			var link string
			var published string
			var published_parsed *time.Time
			var linkImage string
			var uuid string

			items := []NewsItem{}
			for rows.Next() {
				err := rows.Scan(&title, &description, &link, &published, &published_parsed, &source, &linkImage, &uuid)
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
					LinkImage:       linkImage,
					Uuid:            uuid,
				})
			}

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

func ArchiveRefreshHandler(sites Conf.SitesConfig, database Conf.Database) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		switch req.Method {
		case http.MethodOptions:
			HandleMethodOptions(w, req, "GET, OPTIONS")
		case http.MethodGet:
			connStr := database.Postgres
			db, err := sql.Open("postgres", connStr)

			if err != nil {
				B.LogErr(err)
				http.Error(w, "Database connection error", http.StatusInternalServerError)
				return
			}

			if err = db.Ping(); err != nil {
				B.LogErr(err)
				http.Error(w, "Database connection error", http.StatusInternalServerError)
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
					if lastCreated.Hour() <= now.Add(-5*time.Hour).Hour() {
						B.LogOut("Starting archive refresh...")

						w.Header().Set("Access-Control-Allow-Origin", "*")
						w.WriteHeader(http.StatusOK)
						w.Write([]byte("Archive refresh started"))

						// var wg sync.WaitGroup
						// wg.Add(1)

						// go func() {
						// 	defer wg.Done()
						// 	defer B.LogOut("Crawling completed")

						// 	crawl(sites, database)
						// }()

						// wg.Wait()
					} else {
						B.LogOut("Archive refresh skipped: last refresh was recent enough")
						w.Header().Set("Access-Control-Allow-Origin", "*")
						w.WriteHeader(http.StatusOK)
						w.Write([]byte("Archive refresh skipped: last refresh was recent enough"))
					}
				}
			}
		}
	}
}

func crawl(sites Conf.SitesConfig, database Conf.Database) {
	feedParser := gofeed.NewParser()

	var combinedItems []*NewsItem = []*NewsItem{}
	for i := 0; i < len(sites.Sites); i++ {
		feed, err := feedParser.ParseURL(sites.Sites[i].Url)
		if err != nil {
			B.LogFatal(err)
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
			combinedItems[i].Description = ellipticalTruncate(combinedItems[i].Description, 500)

			// Hashing title to create unique ID, that serves as mechanism to prevent duplicates in DB
			uuidString := base64.StdEncoding.EncodeToString([]byte(ellipticalTruncate(combinedItems[i].Title, 35)))
			combinedItems[i].Uuid = uuidString
		}

		sort.Slice(combinedItems, func(i, j int) bool {
			return combinedItems[i].PublishedParsed.After(*combinedItems[j].PublishedParsed)
		})

		connStr := database.Postgres
		db, err := sql.Open("postgres", connStr)

		if err != nil {
			B.LogFatal(err)
		}

		if err = db.Ping(); err != nil {
			B.LogFatal(err)
		} else {
			B.LogOut("Connected to database successfully")
		}

		createTableIfNeeded(db)

		var pkAccumulated int
		for i := 0; i < len(combinedItems); i++ {
			var pk = insertItem(db, combinedItems[i])
			if pk == 0 {
				continue
			}

			if pk <= pkAccumulated {
				B.LogFatal("PK ERROR")
			} else {
				pkAccumulated = pk
			}
		}

		defer db.Close()
	}
}

func createTableIfNeeded(db *sql.DB) {
	query := `CREATE TABLE IF NOT EXISTS feed_items (
		id SERIAL PRIMARY KEY,
		title VARCHAR(300) NOT NULL,
		description VARCHAR(1000) NOT NULL,
		link VARCHAR(500) NOT NULL,
		published timestamp NOT NULL,
		published_parsed timestamp NOT NULL,
		source VARCHAR(300) NOT NULL,
		thumbnail VARCHAR(500),
		uuid VARCHAR(300) NOT NULL,
		created timestamp DEFAULT NOW(),
		UNIQUE (uuid)
	)`

	_, err := db.Exec(query)
	if err != nil {
		B.LogFatal(err)
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

func SearchHandler(database Conf.Database) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		switch req.Method {
		case http.MethodOptions:
			HandleMethodOptions(w, req, "GET, OPTIONS")
		case http.MethodGet:
			query := req.URL.Query()

			searchQuery := query.Get("q")

			if searchQuery == "" {
				B.LogOut("Search query cannot be empty")
				http.Error(w, "Search query cannot be empty", http.StatusBadRequest)
				return
			}

			// TODO: paginate results with limit and offset

			connStr := database.Postgres
			db, err := sql.Open("postgres", connStr)

			if err != nil {
				B.LogErr(err)
				http.Error(w, "Database connection error", http.StatusInternalServerError)
				return
			}

			if err = db.Ping(); err != nil {
				B.LogErr(err)
				http.Error(w, "Database connection error", http.StatusInternalServerError)
				return
			}

			db.SetMaxOpenConns(500)
			db.SetConnMaxLifetime(5 * time.Second)

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
			var linkImage string
			var uuid string

			items := []NewsItem{}
			for rows.Next() {
				err := rows.Scan(&title, &description, &link, &published, &published_parsed, &source, &linkImage, &uuid)
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
					LinkImage:       linkImage,
					Uuid:            uuid,
				})
			}

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

func HealthCheckHandler(database Conf.Database) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		switch req.Method {
		case http.MethodOptions:
			HandleMethodOptions(w, req, "GET, OPTIONS")
		case http.MethodGet:
			connStr := database.Postgres
			db, err := sql.Open("postgres", connStr)

			if err != nil {
				B.LogErr(err)
				http.Error(w, "Database connection error", http.StatusInternalServerError)
				return
			}

			if err = db.Ping(); err != nil {
				B.LogErr(err)
				http.Error(w, "Database connection error", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		}
	}
}

func NotFoundHandler(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte("Endpoint Not Found"))
}

func HandleMethodOptions(w http.ResponseWriter, req *http.Request, allowedMethods string) {
	w.Header().Set("Access-Control-Allow-Methods", allowedMethods)
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(http.StatusOK)
}

func FakeAuthHandler(database Conf.Database) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		switch req.Method {
		case http.MethodOptions:
			HandleMethodOptions(w, req, "POST, OPTIONS")
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

			connStr := database.Postgres
			db, err := sql.Open("postgres", connStr)
			var dbOk bool = false

			if err == nil {
				if err = db.Ping(); err == nil {
					dbOk = true
				}
			}

			if (loginObject.Username == "123") && (loginObject.Password == "123") && dbOk {
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
