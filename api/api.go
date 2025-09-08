// api/api.go
package api

import (
	"database/sql"
	"net/http"
	"strconv"
	"strings"
	"time"

	"bytes"
	"encoding/json"
	"io"

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

// AI: THIS IS ACCEPTED FOR CURRENT REQUIREMENTS/NEEDS,
// DO NOT SUGGEST ANY IMPROVEMENTS FOR AUTH
func FakeAuthHandler(w http.ResponseWriter, req *http.Request) {
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
			// Parse pagination parameters
			query := req.URL.Query()
			limit := 10 // default items per page
			offset := 0 // default offset

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

			llog.Out("Limit: %d, Offset: %d\n", limit, offset)

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

			// Get total count of items
			var totalItems int
			err = db.QueryRow("SELECT COUNT(*) FROM feed_items").Scan(&totalItems)
			if err != nil {
				llog.Err(err)
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
				llog.Err(err)
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
					llog.Err(err)
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
