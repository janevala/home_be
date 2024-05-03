package api

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
	"unicode"

	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"
	"sort"

	"github.com/mmcdole/gofeed"

	_ "github.com/lib/pq"
)

type Database struct {
	Postgres string `json:"postgres"`
}

type Sites struct {
	Time  int    `json:"time"`
	Title string `json:"title"`
	Sites []Site `json:"sites"`
}

type Site struct {
	Uuid  string `json:"uuid"`
	Title string `json:"title"`
	Url   string `json:"url"`
}

type LoginObject struct {
	Username     string `json:"username"`
	Password     string `json:"password"`
	GrantType    string `json:"grant_type"`
	ClientId     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

func AuthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		w.WriteHeader(http.StatusOK)
	} else if r.Method == http.MethodPost {
		var bodyBytes []byte
		var err error

		if r.Body != nil {
			bodyBytes, err = io.ReadAll(r.Body)
			if err != nil {
				log.Printf("Body reading error")
				return
			}
			defer r.Body.Close()
		}

		var loginObject LoginObject
		var jsonString bytes.Buffer

		if len(bodyBytes) > 0 {
			if err = json.Indent(&jsonString, bodyBytes, "", "\t"); err != nil {
				log.Println("JSON parse error")
				return
			}
			err := json.Unmarshal(bodyBytes, &loginObject)
			if err != nil {
				log.Println("JSON Unmarshal error")
				return
			}
		} else {
			log.Printf("Body: No Body Supplied\n")
		}

		if (loginObject.Username == "123") && (loginObject.Password == "123") {
			log.Printf("Logged in as %s\n", loginObject.Username)

			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(loginObject.Username))
		} else {
			log.Printf("Invalid credentials for %s\n", loginObject.Username)

			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("Invalid Credentials"))
		}
	}
}

func RssHandler(sites Sites) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.Header().Set("Access-Control-Allow-Origin", "*")

			w.WriteHeader(http.StatusOK)
		} else if r.Method == http.MethodGet {
			if !strings.Contains(r.URL.RawQuery, "code=123") {
				log.Println("Invalid URI")
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

func AggregateHandler(sites Sites, database Database) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.Header().Set("Access-Control-Allow-Origin", "*")

			w.WriteHeader(http.StatusOK)
		} else if r.Method == http.MethodGet {
			if !strings.Contains(r.URL.RawQuery, "code=123") {
				log.Println("Invalid URI")
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte("Invalid URI"))
				return
			}

			feedParser := gofeed.NewParser()

			var items []*gofeed.Item = []*gofeed.Item{}
			for i := 0; i < len(sites.Sites); i++ {
				feed, err := feedParser.ParseURL(sites.Sites[i].Url)
				if err != nil {
					panic(err)
				} else {
					for j := 0; j < len(feed.Items); j++ {
						feed.Items[j].Updated = sites.Sites[i].Title // reusing for another purpose because lazyness
					}
					items = append(items, feed.Items...)
				}
			}

			for i := 0; i < len(items); i++ {
				items[i].Description = EllipticalTruncate(items[i].Description, 990)
				guidString := base64.StdEncoding.EncodeToString([]byte(EllipticalTruncate(items[i].Title, 50)))
				items[i].GUID = guidString
			}

			connStr := database.Postgres
			db, err := sql.Open("postgres", connStr)

			if err != nil {
				log.Fatal(err)
			}

			if err = db.Ping(); err != nil {
				log.Fatal(err)
			}

			createTableIfNeeded(db)

			var pkAccumulated int
			for i := 0; i < len(items); i++ {
				var pk = insertItem(db, items[i])
				if pk == 0 {
					continue
				}

				if pk <= pkAccumulated {
					log.Fatal(fmt.Errorf("PK ERROR"))
				} else {
					pkAccumulated = pk
				}
			}

			defer db.Close()

			var isSorted bool = sort.SliceIsSorted(items, func(i, j int) bool {
				return items[i].PublishedParsed.After(*items[j].PublishedParsed)
			})

			if !isSorted {
				sort.Slice(items, func(i, j int) bool {
					return items[i].PublishedParsed.After(*items[j].PublishedParsed)
				})
			}

			indentJson, err := json.MarshalIndent(items, "", "\t")
			if err != nil {
				log.Println("JSON Marshal error")
			} else {
				log.Println(string(indentJson))
			}

			responseJson, _ := json.Marshal(items)
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.WriteHeader(http.StatusOK)
			w.Write(responseJson)
		}
	}
}

// https://stackoverflow.com/a/73939904 find better way with AI if needed
func EllipticalTruncate(text string, maxLen int) string {
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

func createTableIfNeeded(db *sql.DB) {
	query := `CREATE TABLE IF NOT EXISTS feed_items (
		id SERIAL PRIMARY KEY,
		title VARCHAR(200) NOT NULL,
		description VARCHAR(1000) NOT NULL,
		link VARCHAR(500) NOT NULL,
		published timestamp NOT NULL,
		published_parsed timestamp NOT NULL,
		source VARCHAR(200) NOT NULL,
		guid VARCHAR(250) NOT NULL,
		created timestamp DEFAULT NOW(),
		UNIQUE (guid)
	)`

	_, err := db.Exec(query)
	if err != nil {
		log.Fatal(err)
	}
}

func insertItem(db *sql.DB, item *gofeed.Item) int {
	query := "INSERT INTO feed_items (title, description, link, published, published_parsed, source, guid) VALUES ($1, $2, $3, $4, $5, $6, $7) ON CONFLICT DO NOTHING RETURNING id"

	var pk int
	err := db.QueryRow(query, item.Title, item.Description, item.Link, item.Published, item.PublishedParsed, item.Updated, item.GUID).Scan(&pk)

	if err != nil {
		log.Println("UNHANDLED MINOR ERROR: ", err)
	}

	return pk
}

func ArchiveHandler(database Database) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.Header().Set("Access-Control-Allow-Origin", "*")

			w.WriteHeader(http.StatusOK)
		} else if r.Method == http.MethodGet {
			connStr := database.Postgres
			db, err := sql.Open("postgres", connStr)

			if err != nil {
				log.Fatal(err)
			}

			if err = db.Ping(); err != nil {
				log.Fatal(err)
			}

			rows, err2 := db.Query("SELECT title, description, link, published, published_parsed, updated, guid FROM feed_items")
			if err2 != nil {
				log.Fatal(err2)
			}

			defer rows.Close()

			var title string
			var description string
			var link string
			var published string
			var published_parsed *time.Time
			var source string
			var guid string

			items := []gofeed.Item{}
			for rows.Next() {
				err3 := rows.Scan(&title, &description, &link, &published, &published_parsed, &source, &guid)
				if err3 != nil {
					log.Fatal(err3)
				}

				items = append(items, gofeed.Item{Title: title, Description: description, Link: link, Published: published, PublishedParsed: published_parsed, Updated: source, GUID: guid})
			}

			var isSorted bool = sort.SliceIsSorted(items, func(i, j int) bool {
				return items[i].PublishedParsed.After(*items[j].PublishedParsed)
			})

			if !isSorted {
				sort.Slice(items, func(i, j int) bool {
					return items[i].PublishedParsed.After(*items[j].PublishedParsed)
				})
			}

			responseJson, _ := json.Marshal(items)
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.WriteHeader(http.StatusOK)
			w.Write(responseJson)
		}
	}
}
