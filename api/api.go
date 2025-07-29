package api

import (
	"database/sql"
	"net/http"
	"strings"
	"time"

	"bytes"
	"encoding/json"
	"io"
	"sort"

	"github.com/janevala/home_be/config"
	"github.com/janevala/home_be/llog"
	_ "github.com/lib/pq"
)

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

func AuthHandler(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodOptions:
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		w.WriteHeader(http.StatusOK)
	case http.MethodPost:
		var bodyBytes []byte
		var err error

		if req.Body != nil {
			bodyBytes, err = io.ReadAll(req.Body)
			if err != nil {
				llog.Err(err)
				return
			}
			defer req.Body.Close()
		}

		var loginObject LoginObject
		var jsonString bytes.Buffer

		if len(bodyBytes) > 0 {
			if err = json.Indent(&jsonString, bodyBytes, "", "\t"); err != nil {
				llog.Err(err)
				return
			}
			err := json.Unmarshal(bodyBytes, &loginObject)
			if err != nil {
				llog.Err(err)
				return
			}
		} else {
			llog.Out("Body: No Body Supplied\n")
		}

		if (loginObject.Username == "123") && (loginObject.Password == "123") {
			llog.Out("Logged in as %s\n", loginObject.Username)

			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(loginObject.Username))
		} else {
			llog.Out("Invalid credentials for %s\n", loginObject.Username)

			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("Invalid Credentials"))
		}
	}
}

func SitesHandler(sites config.SitesConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		switch req.Method {
		case http.MethodOptions:
			w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.Header().Set("Access-Control-Allow-Origin", "*")

			w.WriteHeader(http.StatusOK)
		case http.MethodGet:
			if !strings.Contains(req.URL.RawQuery, "code=123") {
				llog.Out("Invalid URI")
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte("Invalid URI"))
				return
			}

			responseJson, _ := json.Marshal(sites)
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Write(responseJson)
			w.WriteHeader(http.StatusOK)
		}
	}
}

func ArchiveHandler(database config.Database) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		switch req.Method {
		case http.MethodOptions:
			w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.Header().Set("Access-Control-Allow-Origin", "*")

			w.WriteHeader(http.StatusOK)
		case http.MethodGet:
			connStr := database.Postgres
			db, err := sql.Open("postgres", connStr)

			if err != nil {
				llog.Err(err)
			}

			if err = db.Ping(); err != nil {
				llog.Err(err)
				http.Error(w, "Database connection error", http.StatusInternalServerError)
				return
			}

			rows, err := db.Query("SELECT title, description, link, published, published_parsed, source, thumbnail, guid FROM feed_items")
			if err != nil {
				llog.Err(err)
				http.Error(w, "Database query error", http.StatusInternalServerError)
				return
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
				err := rows.Scan(&title, &description, &link, &published, &published_parsed, &source, &linkImage, &guid)
				if err != nil {
					llog.Err(err)
					http.Error(w, "Database scan error", http.StatusInternalServerError)
					return
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
			w.Write(responseJson)
			w.WriteHeader(http.StatusOK)
		}
	}
}
