package main

import (
	"log"
	"net/http"
	"time"

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

// / Client needs these endpoints
func init() {
	http.HandleFunc("/auth", api.AuthApi)
	http.HandleFunc("/sites", api.RssApi)
	http.HandleFunc("/aggregate", api.AggregateApi)
}
