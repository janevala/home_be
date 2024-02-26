package api

import (
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"bytes"
	"encoding/json"
	"io"
	"sort"

	"github.com/mmcdole/gofeed"
)

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
			w.Write([]byte("Logged in as " + loginObject.Username))
		} else {
			log.Printf("Invalid credentials for %s\n", loginObject.Username)

			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("Invalid Credentials"))
		}
	}
}

func AggregateHandler(w http.ResponseWriter, r *http.Request) {
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

		kalevaFeed, _ := feedParser.ParseURL("https://www.kaleva.fi/feedit/rss/managed-listing/rss-uusimmat/")
		talousElamaFeed, _ := feedParser.ParseURL("https://www.talouselama.fi/rss.xml")
		kauppalehtiFeed, _ := feedParser.ParseURL("https://feeds.kauppalehti.fi/rss/main")
		iltaLehtiFeed, _ := feedParser.ParseURL("https://www.iltalehti.fi/rss/uutiset.xml")
		suomenUutisetFeed, _ := feedParser.ParseURL("https://www.suomenuutiset.fi/feed/")
		kansanUutisetFeed, _ := feedParser.ParseURL("https://www.ku.fi/feed")

		var combinedFeed []*gofeed.Item = []*gofeed.Item{}
		combinedFeed = append(combinedFeed, kalevaFeed.Items...)
		combinedFeed = append(combinedFeed, talousElamaFeed.Items...)
		combinedFeed = append(combinedFeed, kauppalehtiFeed.Items...)
		combinedFeed = append(combinedFeed, iltaLehtiFeed.Items...)
		combinedFeed = append(combinedFeed, kansanUutisetFeed.Items...)
		combinedFeed = append(combinedFeed, suomenUutisetFeed.Items...)

		var isSorted bool = sort.SliceIsSorted(combinedFeed, func(i, j int) bool {
			return combinedFeed[i].PublishedParsed.After(*combinedFeed[j].PublishedParsed)
		})

		if !isSorted {
			sort.Slice(combinedFeed, func(i, j int) bool {
				return combinedFeed[i].PublishedParsed.After(*combinedFeed[j].PublishedParsed)
			})
		}

		jsonArray, err := json.Marshal(combinedFeed)
		if err != nil {
			log.Println("JSON Marshal error")
		}

		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.WriteHeader(http.StatusOK)
		w.Write(jsonArray)
	}
}

type sites struct {
	Time  int    `json:"time"`
	Title string `json:"title"`
	Sites []site `json:"sites"`
}

type site struct {
	Uuid  string `json:"uuid"`
	Title string `json:"title"`
	Url   string `json:"url"`
}

func RssHandler(w http.ResponseWriter, r *http.Request) {
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

		sites := sites{
			Time:  int(time.Now().UTC().UnixMilli()),
			Title: "RSS Feeds",
			Sites: []site{
				{
					Uuid:  uuid.NewString(),
					Title: "Ilta-Sanomat",
					Url:   "https://www.is.fi/rss/tuoreimmat.xml",
				},
				{
					Uuid:  uuid.NewString(),
					Title: "Helsingin Sanomat",
					Url:   "https://www.hs.fi/rss/tuoreimmat.xml",
				},
				{
					Uuid:  uuid.NewString(),
					Title: "Yle",
					Url:   "https://feeds.yle.fi/uutiset/v1/recent.rss?publisherIds=YLE_UUTISET",
				},
				{
					Uuid:  uuid.NewString(),
					Title: "Iltalehti",
					Url:   "https://www.iltalehti.fi/rss/uutiset.xml",
				},
				{
					Uuid:  uuid.NewString(),
					Title: "Talouselämä",
					Url:   "https://www.talouselama.fi/rss.xml",
				},
				{
					Uuid:  uuid.NewString(),
					Title: "Kaleva",
					Url:   "https://www.kaleva.fi/feedit/rss/managed-listing/rss-uusimmat/",
				},
				{
					Uuid:  uuid.NewString(),
					Title: "Kauppalehti",
					Url:   "https://feeds.kauppalehti.fi/rss/main",
				},
				{
					Uuid:  uuid.NewString(),
					Title: "Suomen Uutiset",
					Url:   "https://www.suomenuutiset.fi/feed/",
				},
				{
					Uuid:  uuid.NewString(),
					Title: "Kansan Uutiset",
					Url:   "https://www.ku.fi/feed",
				},
				{
					Uuid:  uuid.NewString(),
					Title: "Talouselämä",
					Url:   "https://www.talouselama.fi/rss.xml",
				},
			},
		}

		finalJson, err := json.Marshal(sites)
		if err != nil {
			log.Println("JSON Marshal error")
		} else {
			log.Println(string(finalJson))
		}

		finalJsonIndent, err := json.MarshalIndent(sites, "", "\t")
		if err != nil {
			log.Println("JSON Marshal error")
		} else {
			log.Println(string(finalJsonIndent))
		}

		responseBytes := []byte(finalJson)

		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.WriteHeader(http.StatusOK)
		w.Write(responseBytes)
	}
}
