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

	Ai "github.com/janevala/home_be/ai"
	Api "github.com/janevala/home_be/api"
	B "github.com/janevala/home_be/build"
	Conf "github.com/janevala/home_be/config"
)

type LoggerHandler struct {
	handler http.Handler
	logger  *log.Logger
}

func (h *LoggerHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.logger.Printf("Received request: %s %s %s", r.Method, r.URL.Path, r.URL.RawQuery)
	h.handler.ServeHTTP(w, r)
}

var cfg *Conf.Config

func main() {
	if cfg == nil {
		B.LogFatal("Config is nil")
	}

	logger := log.New(log.Writer(), "[HTTP] ", log.LstdFlags)
	B.SetLogger(logger)

	server := http.Server{
		Addr:         cfg.Server.Port,
		Handler:      &LoggerHandler{handler: http.DefaultServeMux, logger: logger},
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	B.LogOut("Number of CPUs: " + strconv.Itoa(runtime.NumCPU()))
	B.LogOut("Number of Goroutines: " + strconv.Itoa(runtime.NumGoroutine()))
	B.LogOut("Server listening on: " + cfg.Server.Port)

	B.LogOut("Starting with configuration:")
	B.LogOut("Server: " + fmt.Sprintf("%#v", cfg.Server))
	B.LogOut("Sites: " + fmt.Sprintf("%#v", cfg.Sites))
	B.LogOut("Database: " + fmt.Sprintf("%#v", cfg.Database))
	B.LogOut("Ollama: " + fmt.Sprintf("%#v", cfg.Ollama))

	B.LogFatal(server.ListenAndServe())
}

func init() {
	var err error
	cfg, err = Conf.LoadConfig("config.json")
	if err != nil {
		B.LogFatal(err)
	}

	connStr := cfg.Database.Postgres
	db, err := sql.Open("postgres", connStr)

	if err != nil {
		if B.IsProduction() {
			B.LogFatal(err)
		} else {
			B.LogErr(err)
		}
	}

	if err = db.Ping(); err != nil {
		if B.IsProduction() {
			B.LogFatal(err)
		} else {
			B.LogErr(err)
		}
	}

	B.LogOut("Server port: " + cfg.Server.Port)

	httpRouter := http.NewServeMux()

	/// WEB FRONTEND
	httpRouter.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		tmpl, err := template.ParseFiles("index.html")
		if err != nil {
			B.LogErr(err)
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
			B.LogErr(err)
			http.Error(w, "Could not execute template", http.StatusInternalServerError)
			return
		}

		B.LogOut("Request served: %s %s", r.Method, r.URL.Path)
	})
	http.Handle("/", httpRouter)

	/// API
	httpRouter.HandleFunc("POST /auth", Api.FakeAuthHandler(cfg.Database))
	httpRouter.HandleFunc("OPTIONS /auth", Api.FakeAuthHandler(cfg.Database))
	httpRouter.HandleFunc("GET /sites", Api.SitesHandler(cfg.Sites))
	httpRouter.HandleFunc("OPTIONS /sites", Api.SitesHandler(cfg.Sites))
	httpRouter.HandleFunc("GET /archive", Api.ArchiveHandler(cfg.Database))
	httpRouter.HandleFunc("OPTIONS /archive", Api.ArchiveHandler(cfg.Database))
	httpRouter.HandleFunc("OPTIONS /search", Api.SearchHandler(cfg.Database))
	httpRouter.HandleFunc("GET /search", Api.SearchHandler(cfg.Database))
	httpRouter.HandleFunc("OPTIONS /health", Api.HealthCheckHandler(cfg.Database))
	httpRouter.HandleFunc("GET /health", Api.HealthCheckHandler(cfg.Database))
	httpRouter.HandleFunc("OPTIONS /refresh", Api.ArchiveRefreshHandler(cfg.Sites, cfg.Database))
	httpRouter.HandleFunc("GET /refresh", Api.ArchiveRefreshHandler(cfg.Sites, cfg.Database))
	httpRouter.HandleFunc("POST /translate", Ai.ExplainHandler(cfg.Ollama))
	httpRouter.HandleFunc("OPTIONS /translate", Ai.ExplainHandler(cfg.Ollama))

	// TODO: if we reach here, use Api.NotFoundHandler
	http.Handle("/auth", httpRouter)
	http.Handle("/sites", httpRouter)
	http.Handle("/archive", httpRouter)
	http.Handle("/search", httpRouter)
	http.Handle("/health", httpRouter)
	http.Handle("/refresh", httpRouter)
	http.Handle("/translate", httpRouter)
}
