package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/janevala/home_be/api"
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

	log.Println("Server listening on " + serverPort)
	log.Fatal(server.ListenAndServe())
}

// Client needs these endpoints
func init() {
	r := mux.NewRouter()
	r.HandleFunc("/auth", api.AuthHandler).Methods(http.MethodPost, http.MethodOptions)
	r.HandleFunc("/sites", api.RssHandler).Methods(http.MethodGet, http.MethodOptions)
	r.HandleFunc("/aggregate", api.AggregateHandler).Methods(http.MethodGet, http.MethodOptions)
	r.HandleFunc("/test", TestHandler).Methods(http.MethodGet, http.MethodOptions)
	http.Handle("/", r)
}

func TestHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Category: %v\n", vars["category"])
}
