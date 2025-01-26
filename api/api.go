package api

import (
	"database/sql"
	"log"
	"net/http"
	"strings"
	"time"

	"bytes"
	"encoding/json"
	"io"
	"sort"

	_ "github.com/lib/pq"
	"github.com/rifaideen/talkative"
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

type QuestionItem struct {
	Question string `json:"question,omitempty"`
}

type AnswerItem struct {
	Answer string `json:"answer,omitempty"`
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
			w.Write(responseJson)
			w.WriteHeader(http.StatusOK)
		}
	}
}

func Explain() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.Header().Set("Access-Control-Allow-Origin", "*")

			w.WriteHeader(http.StatusOK)
		} else if r.Method == http.MethodPost {
			if !strings.Contains(r.URL.RawQuery, "code=123") {
				log.Println("Invalid URI")
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte("Invalid URI"))
				return
			}

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

			var questionObject QuestionItem
			var jsonString bytes.Buffer

			if len(bodyBytes) > 0 {
				if err = json.Indent(&jsonString, bodyBytes, "", "\t"); err != nil {
					log.Println("JSON parse error")
					return
				}
				err := json.Unmarshal(bodyBytes, &questionObject)
				if err != nil {
					log.Println("JSON Unmarshal error")
					return
				}
			} else {
				log.Printf("Body: No Body Supplied\n")
			}

			answerItem := queryAI(questionObject)

			responseJson, _ := json.Marshal(answerItem)
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Write(responseJson)
			w.WriteHeader(http.StatusOK)
		}
	}
}

func queryAI(q QuestionItem) AnswerItem {
	var question string = q.Question

	client, err := talkative.New("http://127.0.0.1:11434")

	if err != nil {
		panic("Failed to create talkative client")
	}

	model := "mistral:7b"
	//model := "qwen2.5-coder:14b"

	responseAnswer := talkative.ChatResponse{}
	callback := func(cr *talkative.ChatResponse, err error) {
		if err != nil {
			log.Println("Error in callback")
			return
		}

		responseAnswer = *cr
	}

	message := talkative.ChatMessage{
		Role:    talkative.USER,
		Content: question,
	}

	done, err := client.Chat(model, callback, &talkative.ChatParams{
		Stream: pointFalse(),
	}, message)

	if err != nil {
		panic(err)
	}

	<-done

	answerItem := AnswerItem{Answer: responseAnswer.Message.Content}

	return answerItem
}

// fix this hack
func pointFalse() *bool {
	b := false
	return &b
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
			w.Write(responseJson)
			w.WriteHeader(http.StatusOK)
		}
	}
}
