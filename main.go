package main

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mmcdole/gofeed"
)

type LoginObject struct {
	Username     string `json:"username"`
	Password     string `json:"password"`
	GrantType    string `json:"grant_type"`
	ClientId     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

type LoggerHandler struct {
	handler http.Handler
	logger  *log.Logger
}

func (h *LoggerHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.logger.Printf("Received request: %s %s", r.Method, r.URL.Path)
	h.handler.ServeHTTP(w, r)
}

func AuthApi(w http.ResponseWriter, r *http.Request) {
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
			w.Write([]byte("Logged in as " + loginObject.Username))
		} else {
			log.Printf("Invalid credentials for %s\n", loginObject.Username)

			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("Invalid Credentials"))
		}
	} else {
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func RssAggregateApi(w http.ResponseWriter, r *http.Request) {
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

		yleFeed, _ := feedParser.ParseURL("https://feeds.yle.fi/uutiset/v1/recent.rss?publisherIds=YLE_UUTISET")
		kalevaFeed, _ := feedParser.ParseURL("https://www.kaleva.fi/feedit/rss/managed-listing/rss-uusimmat/")
		talousElamaFeed, _ := feedParser.ParseURL("https://www.talouselama.fi/rss.xml")
		suomenUutisetFeed, _ := feedParser.ParseURL("https://www.suomenuutiset.fi/feed/")
		kansanUutisetFeed, _ := feedParser.ParseURL("https://www.ku.fi/feed")

		var allRssSlice []*gofeed.Item = []*gofeed.Item{}
		allRssSlice = append(suomenUutisetFeed.Items, kansanUutisetFeed.Items...)
		allRssSlice = append(allRssSlice, yleFeed.Items...)
		allRssSlice = append(allRssSlice, kalevaFeed.Items...)
		allRssSlice = append(allRssSlice, talousElamaFeed.Items...)

		var isSorted bool = sort.SliceIsSorted(allRssSlice, func(i, j int) bool {
			return allRssSlice[i].PublishedParsed.After(*allRssSlice[j].PublishedParsed)
		})

		if !isSorted {
			sort.Slice(allRssSlice, func(i, j int) bool {
				return allRssSlice[i].PublishedParsed.After(*allRssSlice[j].PublishedParsed)
			})
		}

		jsonArray, err := json.Marshal(allRssSlice)
		if err != nil {
			log.Println("JSON Marshal error")
		}

		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.WriteHeader(http.StatusOK)
		w.Write(jsonArray)
	} else {
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func RssApi(w http.ResponseWriter, r *http.Request) {
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

		timestamp := strconv.FormatInt(time.Now().UTC().UnixMilli(), 10)

		jsonArray := []byte(`{ "time": "` + timestamp + `",
			"title": "RSS Feeds",
			"sites": [
			  {
				"uuid": "` + uuid.NewString() + `",
				"title": "Ilta-Sanomat",
				"url": "https://www.is.fi/rss/tuoreimmat.xml"
			  },
			  {
				"uuid": "` + uuid.NewString() + `",
				"title": "Helsingin Sanomat",
				"url": "https://www.hs.fi/rss/tuoreimmat.xml"
			  },
			  {
				"uuid": "` + uuid.NewString() + `",
				"title": "Yle",
				"url": "https://feeds.yle.fi/uutiset/v1/recent.rss?publisherIds=YLE_UUTISET"
			  },
			  {
				"uuid": "` + uuid.NewString() + `",
				"title": "Iltalehti",
				"url": "https://www.iltalehti.fi/rss/uutiset.xml"
			  },
			  {
				"uuid": "` + uuid.NewString() + `",
				"title": "Talouselämä",
				"url": "https://www.talouselama.fi/rss.xml"
			  },
			  {
				"uuid": "` + uuid.NewString() + `",
				"title": "Kaleva",
				"url": "https://www.kaleva.fi/feedit/rss/managed-listing/rss-uusimmat/"
			  },
			  {
				"uuid": "` + uuid.NewString() + `",
				"title": "Kauppalehti",
				"url": "https://feeds.kauppalehti.fi/rss/main"
			  },
			  {
				"uuid": "` + uuid.NewString() + `",
				"title": "Suomen Uutiset",
				"url": "https://www.suomenuutiset.fi/feed/"
			  },
			  {
				"uuid": "` + uuid.NewString() + `",
				"title": "Kansan Uutiset",
				"url": "https://www.ku.fi/feed"
			  },
			  {
				"uuid": "` + uuid.NewString() + `",
				"title": "Talouselämä",
				"url": "https://www.talouselama.fi/rss.xml"
			  }
			]
		  }`)

		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.WriteHeader(http.StatusOK)
		w.Write(jsonArray)
	} else {
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func main() {
	logger := log.New(log.Writer(), "[HTTP] ", log.LstdFlags)

	serverPort := ":8091"

	server := http.Server{
		Addr:         serverPort,
		Handler:      &LoggerHandler{handler: http.DefaultServeMux, logger: logger},
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	log.Println("Server listening on " + serverPort)
	log.Fatal(server.ListenAndServe())
}

func init() {
	http.HandleFunc("/auth", AuthApi)
	http.HandleFunc("/rss_sites", RssApi)
	http.HandleFunc("/rss_aggregate", RssAggregateApi)
}
