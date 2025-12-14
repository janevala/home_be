package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"runtime"
	"strconv"
	"time"

	Api "github.com/janevala/home_be/api"
	"github.com/janevala/home_be/config"
	"github.com/janevala/home_be/llog"
)

type LoggerHandler struct {
	handler http.Handler
	logger  *log.Logger
}

func (h *LoggerHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.logger.Printf("Received request: %s %s", r.Method, r.URL.Path)
	h.handler.ServeHTTP(w, r)
}

var cfg *config.Config

func main() {
	logger := log.New(log.Writer(), "[HTTP] ", log.LstdFlags)

	if cfg == nil {
		panic("Config is nil")
	}

	server := http.Server{
		Addr:         cfg.Server.Port,
		Handler:      &LoggerHandler{handler: http.DefaultServeMux, logger: logger},
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	llog.Out("Number of CPUs: " + strconv.Itoa(runtime.NumCPU()))
	llog.Out("Number of Goroutines: " + strconv.Itoa(runtime.NumGoroutine()))
	llog.Out("Server listening on: " + cfg.Server.Port)

	llog.Out("Starting with configuration:")
	llog.Out("Server: " + fmt.Sprintf("%#v", cfg.Server))
	llog.Out("Sites: " + fmt.Sprintf("%#v", cfg.Sites))
	llog.Out("Database: " + fmt.Sprintf("%#v", cfg.Database))
	llog.Out("Ollama: " + fmt.Sprintf("%#v", cfg.Ollama))

	llog.Fatal(server.ListenAndServe())
}

func init() {
	var err error
	cfg, err = config.LoadConfig("config.json")
	if err != nil {
		llog.Fatal(err)
		panic(err)
	}

	connStr := cfg.Database.Postgres
	db, err := sql.Open("postgres", connStr)

	if err != nil {
		llog.Fatal(err)
		panic(err)
	}

	if err = db.Ping(); err != nil {
		llog.Fatal(err)
		panic(err)
	}

	llog.Out("Server port: " + cfg.Server.Port)

	httpRouter := http.NewServeMux()

	/// WEB FRONTEND
	httpRouter.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		tmpl, err := template.ParseFiles("index.html")
		if err != nil {
			llog.Err(err)
			http.Error(w, "Could not load template", http.StatusInternalServerError)
			return
		}
		data := map[string]interface{}{
			"BuildTime":     time.Now().Format(time.RFC3339),
			"GoVersion":     runtime.Version(),
			"NumCPU":        runtime.NumCPU(),
			"NumGoroutine":  runtime.NumGoroutine(),
			"NumGOMAXPROCS": runtime.GOMAXPROCS(0),
			"NumCgoCall":    runtime.NumCgoCall(),
			"Server":        fmt.Sprintf("%#v", cfg.Server),
			"Database":      fmt.Sprintf("%#v", cfg.Database.Postgres),
			"Sites":         fmt.Sprintf("%#v", cfg.Sites),
			"Ollama":        fmt.Sprintf("%#v", cfg.Ollama), // TODO: sleeper for now
		}

		if err := tmpl.Execute(w, data); err != nil {
			llog.Err(err)
			http.Error(w, "Could not execute template", http.StatusInternalServerError)
			return
		}
		llog.Out("Request served: %s %s", r.Method, r.URL.Path)
	})
	http.Handle("/", httpRouter)

	/// API
	httpRouter.HandleFunc("POST /auth", Api.FakeAuthHandler)
	httpRouter.HandleFunc("OPTIONS /auth", Api.FakeAuthHandler)
	httpRouter.HandleFunc("GET /sites", Api.SitesHandler(cfg.Sites))
	httpRouter.HandleFunc("OPTIONS /sites", Api.SitesHandler(cfg.Sites))
	httpRouter.HandleFunc("GET /archive", Api.ArchiveHandler(cfg.Database))
	httpRouter.HandleFunc("OPTIONS /archive", Api.ArchiveHandler(cfg.Database))
	httpRouter.HandleFunc("OPTIONS /q", Api.SearchHandler(cfg.Database))
	httpRouter.HandleFunc("GET /q", Api.SearchHandler(cfg.Database))
	httpRouter.HandleFunc("OPTIONS /health", Api.HealthCheckHandler(cfg.Database))
	httpRouter.HandleFunc("GET /health", Api.HealthCheckHandler(cfg.Database))
	// httpRouter.HandleFunc("POST /explain", Ai.ExplainHandler(cfg.Ollama))
	// httpRouter.HandleFunc("OPTIONS /explain", Ai.ExplainHandler(cfg.Ollama))

	// if we reach here, use Api.NotFoundHandler

	http.Handle("/auth", httpRouter)
	http.Handle("/sites", httpRouter)
	http.Handle("/archive", httpRouter)
	http.Handle("/q", httpRouter)
	http.Handle("/health", httpRouter)
	// http.Handle("/explain", httpRouter)

}
