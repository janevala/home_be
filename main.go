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

	API "github.com/janevala/home_be/api"
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

func init() {
	sitesFile, err := os.ReadFile("sites.json")
	if err != nil {
		panic(err)
	}

	sites := API.Sites{}
	json.Unmarshal(sitesFile, &sites)
	sitesString, err := json.MarshalIndent(sites, "", "\t")
	if err != nil {
		panic(err)
	} else {
		sites.Time = int(time.Now().UTC().UnixMilli())
		for i := 0; i < len(sites.Sites); i++ {
			sites.Sites[i].Uuid = uuid.NewString()
		}

		log.Println(string(sitesString))
	}

	databaseFile, err := os.ReadFile("database.json")
	if err != nil {
		panic(err)
	}

	database := API.Database{}
	json.Unmarshal(databaseFile, &database)
	databaseString, err := json.MarshalIndent(database, "", "\t")
	if err != nil {
		panic(err)
	} else {
		log.Println(string(databaseString))
	}

	r := mux.NewRouter()
	r.HandleFunc("/auth", API.AuthHandler).Methods(http.MethodPost, http.MethodOptions)
	r.HandleFunc("/sites", API.RssHandler(sites)).Methods(http.MethodGet, http.MethodOptions)
	r.HandleFunc("/aggregate", API.AggregateHandler(sites, database)).Methods(http.MethodGet, http.MethodOptions)
	r.HandleFunc("/archive", API.ArchiveHandler(database)).Methods(http.MethodGet, http.MethodOptions)
	r.HandleFunc("/test", TestHandler).Methods(http.MethodGet, http.MethodOptions)
	http.Handle("/", r)
}

func TestHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Category: %v\n", vars["category"])
}
