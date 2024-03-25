package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/janevala/home_be/api"

	S "github.com/janevala/home_be/api"
)

type LoggerHandler struct {
	handler http.Handler
	logger  *log.Logger
}

func (h *LoggerHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.logger.Printf("Received request: %s %s", r.Method, r.URL.Path)
	h.handler.ServeHTTP(w, r)
}

func main() {
	logger := log.New(log.Writer(), "[HTTP] ", log.LstdFlags)

	serverPort := ":8091"

	server := http.Server{
		Addr:         serverPort,
		Handler:      &LoggerHandler{handler: http.DefaultServeMux, logger: logger},
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	log.Println("Number of CPUs: ", runtime.NumCPU())
	log.Println("Number of Goroutines: ", runtime.NumGoroutine())
	log.Println("Server listening on: " + serverPort)
	log.Fatal(server.ListenAndServe())
}

// NEEDS TO MATCH WITH CLIENT
func init() {
	fileBytes, err := os.ReadFile("sites.json")
	if err != nil {
		panic(err)
	}

	sitesModel := S.Sites{}
	json.Unmarshal(fileBytes, &sitesModel)
	sitesString, err := json.MarshalIndent(sitesModel, "", "\t")
	if err != nil {
		panic(err)
	} else {
		sitesModel.Time = int(time.Now().UTC().UnixMilli())
		for i := 0; i < len(sitesModel.Sites); i++ {
			sitesModel.Sites[i].Uuid = uuid.NewString()
		}

		log.Println(string(sitesString))
	}

	r := mux.NewRouter()
	r.HandleFunc("/auth", api.AuthHandler).Methods(http.MethodPost, http.MethodOptions)
	r.HandleFunc("/sites", api.RssHandler(sitesModel)).Methods(http.MethodGet, http.MethodOptions)
	r.HandleFunc("/aggregate", api.AggregateHandler).Methods(http.MethodGet, http.MethodOptions)
	r.HandleFunc("/test", TestHandler).Methods(http.MethodGet, http.MethodOptions)
	http.Handle("/", r)
}

func TestHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Category: %v\n", vars["category"])
}

/* JSON WRITER BACKUP CODE

sites2 := Sites{
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

stringOut, _ := json.MarshalIndent(sites2, "", "\t")
file, err := os.Create("./sites.json")
if err != nil {
	panic(err)
}

lenghtOut, err := io.WriteString(file, string(stringOut))
if err != nil {
	panic(err)
}

log.Println(lenghtOut)

*/
