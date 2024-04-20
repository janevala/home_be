package api

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"strings"

	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"
	"sort"

	"github.com/mmcdole/gofeed"

	_ "github.com/lib/pq"
)

type Database struct {
	Postgress string `json:"postgress"`
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

			var combinedFeed []*gofeed.Item = []*gofeed.Item{}
			for i := 0; i < len(sites.Sites); i++ {
				feed, err := feedParser.ParseURL(sites.Sites[i].Url)
				if err != nil {
					panic(err)
				} else {
					combinedFeed = append(combinedFeed, feed.Items...)
				}
			}

			for i := 0; i < len(combinedFeed); i++ {
				titleb64 := base64.StdEncoding.EncodeToString([]byte(combinedFeed[i].Title))
				combinedFeed[i].GUID = titleb64
			}

			connStr := database.Postgress
			db, err := sql.Open("postgres", connStr) // check _ import, https://www.youtube.com/watch?v=Y7a0sNKdoQk

			if err != nil {
				log.Fatal(err)
			}

			if err = db.Ping(); err != nil {
				log.Fatal(err)
			}

			createTableIfNeeded(db)

			var pkAccumulated int
			for i := 0; i < len(combinedFeed); i++ {
				var pk = insertItem(db, combinedFeed[i])
				if pk <= pkAccumulated {
					log.Fatal(fmt.Errorf("PK ERROR"))
				} else {
					pkAccumulated = pk
				}
			}

			defer db.Close()

			var isSorted bool = sort.SliceIsSorted(combinedFeed, func(i, j int) bool {
				return combinedFeed[i].PublishedParsed.After(*combinedFeed[j].PublishedParsed)
			})

			if !isSorted {
				sort.Slice(combinedFeed, func(i, j int) bool {
					return combinedFeed[i].PublishedParsed.After(*combinedFeed[j].PublishedParsed)
				})
			}

			indentJson, err := json.MarshalIndent(combinedFeed, "", "\t")
			if err != nil {
				log.Println("JSON Marshal error")
			} else {
				log.Println(string(indentJson))
			}

			responseJson, _ := json.Marshal(combinedFeed)
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.WriteHeader(http.StatusOK)
			w.Write(responseJson)
		}
	}
}

func createTableIfNeeded(db *sql.DB) {
	query := `CREATE TABLE IF NOT EXISTS feed_items (
		id SERIAL PRIMARY KEY,
		title VARCHAR(200) NOT NULL,
		description VARCHAR(500) NOT NULL,
		link VARCHAR(200) NOT NULL,
		published timestamp NOT NULL,
		published_parsed timestamp NOT NULL,
		guid VARCHAR(100) NOT NULL,
		created timestamp DEFAULT NOW()
	)`

	_, err := db.Exec(query)
	if err != nil {
		log.Fatal(err)
	}
}

func insertItem(db *sql.DB, item *gofeed.Item) int {
	query := `INSERT INTO feed_items (title, description, link, published, published_parsed, guid) VALUES ($1, $2, $3, $4 $5, $6) RETURNING id`

	var pk int
	err := db.QueryRow(query, item.Title, item.Description, item.Link, item.Published, item.PublishedParsed).Scan(&pk)

	if err != nil {
		log.Fatal(err)
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
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("INCOMPLETE"))
		}
	}
}
