package api

import (
	"log"
	"net/http"
	"os"
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

		slashdotFeed, _ := feedParser.ParseURL("https://rss.slashdot.org/Slashdot/slashdotMain")
		tomsHardwareFeed, _ := feedParser.ParseURL("https://www.tomshardware.com/feeds/all")
		techCrunchFeed, _ := feedParser.ParseURL("https://techcrunch.com/feed/")
		phoronixFeed, _ := feedParser.ParseURL("https://www.phoronix.com/rss.php")

		var combinedFeed []*gofeed.Item = []*gofeed.Item{}
		combinedFeed = append(combinedFeed, slashdotFeed.Items...)
		combinedFeed = append(combinedFeed, tomsHardwareFeed.Items...)
		combinedFeed = append(combinedFeed, techCrunchFeed.Items...)
		combinedFeed = append(combinedFeed, phoronixFeed.Items...)

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

		fileBytes, err := os.ReadFile("sites.json")
		if err != nil {
			panic(err)
		}

		sitesModel := Sites{}
		json.Unmarshal(fileBytes, &sitesModel)
		sitesString, err := json.MarshalIndent(sitesModel, "", "\t")
		if err != nil {
			panic(err)
		} else {
			log.Println(string(sitesString))
		}

		sites := Sites{
			Time:  int(time.Now().UTC().UnixMilli()),
			Title: "RSS Feeds",
			Sites: []Site{
				{
					Uuid:  uuid.NewString(),
					Title: "Phoronix",
					Url:   "https://www.phoronix.com/rss.php",
				},
				{
					Uuid:  uuid.NewString(),
					Title: "Slashdot",
					Url:   "https://rss.slashdot.org/Slashdot/slashdotMain",
				},
				{
					Uuid:  uuid.NewString(),
					Title: "Tom's Hardware",
					Url:   "https://www.tomshardware.com/feeds/all",
				},
				{
					Uuid:  uuid.NewString(),
					Title: "TechCrunch",
					Url:   "https://techcrunch.com/feed/",
				},
			},
		}

		indentJson, err := json.MarshalIndent(sites, "", "\t")
		if err != nil {
			log.Println("JSON Marshal error")
		} else {
			log.Println(string(indentJson))
		}

		responseJson, _ := json.Marshal(sites)
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.WriteHeader(http.StatusOK)
		w.Write(responseJson)
	}
}
