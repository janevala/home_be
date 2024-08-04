package api

import (
	"database/sql"
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

type ExtendedItem struct {
	Title           string     `json:"title,omitempty"`
	Description     string     `json:"description,omitempty"`
	Content         string     `json:"content,omitempty"`
	Link            string     `json:"link,omitempty"`
	Updated         string     `json:"updated,omitempty"`
	Published       string     `json:"published,omitempty"`
	PublishedParsed *time.Time `json:"publishedParsed,omitempty"`
	LinkImage       string     `json:"linkImage,omitempty"`
	GUID            string     `json:"guid,omitempty"`
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
					if feed.Image != nil {
						for j := 0; j < len(feed.Items); j++ {
							feed.Items[j].Image = &*feed.Image
						}
					} else {
						for j := 0; j < len(feed.Items); j++ {
							feed.Items[j].Image = &gofeed.Image{
								URL: "https://www.google.com",
							}
						}
					}

					for j := 0; j < len(feed.Items); j++ {
						feed.Items[j].Updated = sites.Sites[i].Title // reusing for another purpose because lazyness TODO
					}

					items = append(items, feed.Items...)
				}
			}

			for i := 0; i < len(items); i++ {
				items[i].Description = EllipticalTruncate(items[i].Description, 990)
				guidString := base64.StdEncoding.EncodeToString([]byte(EllipticalTruncate(items[i].Title, 50)))
				items[i].GUID = guidString
			}

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

			rows, err2 := db.Query("SELECT title, description, link, published, published_parsed, source, thumbnail, guid FROM feed_items")
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
			var linkImage string
			var guid string

			items := []ExtendedItem{}
			for rows.Next() {
				err3 := rows.Scan(&title, &description, &link, &published, &published_parsed, &source, &linkImage, &guid)
				if err3 != nil {
					log.Fatal(err3)
				}

				items = append(items, ExtendedItem{Title: title, Description: description, Link: link, Published: published, PublishedParsed: published_parsed, Updated: source, LinkImage: linkImage, GUID: guid})
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
