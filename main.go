package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"sync"
	"time"

	Ai "github.com/janevala/home_be/ai"
	Api "github.com/janevala/home_be/api"
	B "github.com/janevala/home_be/build"
	Conf "github.com/janevala/home_be/config"
	"github.com/joho/godotenv"
)

var (
	startupTime time.Time = time.Now()
	version     string    = "dev"
	cfg         *Conf.Config
	db          *sql.DB
	httpStats   *HTTPStats
)

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

type HTTPStats struct {
	mu                sync.RWMutex
	TotalRequests     int
	RequestsByMethod  map[string]int
	RequestsByPath    map[string]int
	ResponseCodeCount map[int]int
	TotalResponseTime time.Duration
}

func NewHTTPStats() *HTTPStats {
	return &HTTPStats{
		RequestsByMethod:  make(map[string]int),
		RequestsByPath:    make(map[string]int),
		ResponseCodeCount: make(map[int]int),
	}
}

func (s *HTTPStats) Record(method, path string, statusCode int, duration time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.TotalRequests++
	s.RequestsByMethod[method]++
	s.RequestsByPath[path]++
	s.ResponseCodeCount[statusCode]++
	s.TotalResponseTime += duration
}

func (s *HTTPStats) GetJsonSnapshot() string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	avgResponseTime := time.Duration(0)
	if s.TotalRequests > 0 {
		avgResponseTime = s.TotalResponseTime / time.Duration(s.TotalRequests)
	}

	jsonStruct := map[string]interface{}{
		"TotalRequests":     s.TotalRequests,
		"TotalResponseTime": s.TotalResponseTime.String(),
		"RequestsByMethod":  s.RequestsByMethod,
		"ResponseCodeCount": s.ResponseCodeCount,
		"AvgResponseTime":   avgResponseTime.String(),
	}

	jsonData, err := json.Marshal(jsonStruct)
	if err != nil {
		B.LogErr(err)
		return "{}"
	}

	return string(jsonData)
}

type HttpHandler struct {
	handler http.Handler
	logger  *log.Logger
	stats   *HTTPStats
}

func (h *HttpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	h.logger.Printf("Received request: %s %s %s", r.Method, r.URL.Path, r.URL.RawQuery)

	sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
	h.handler.ServeHTTP(sw, r)

	duration := time.Since(start)
	h.stats.Record(r.Method, r.URL.Path, sw.status, duration)
}

func dbStatsToJson(db *sql.DB) string {
	dbStats := db.Stats()
	statsStruct := map[string]interface{}{
		"MaxOpenConnections": dbStats.MaxOpenConnections,
		"OpenConnections":    dbStats.OpenConnections,
		"InUse":              dbStats.InUse,
		"Idle":               dbStats.Idle,
		"WaitCount":          dbStats.WaitCount,
		"WaitDuration":       dbStats.WaitDuration.String(),
		"MaxIdleClosed":      dbStats.MaxIdleClosed,
		"MaxIdleTimeClosed":  dbStats.MaxIdleTimeClosed,
		"MaxLifetimeClosed":  dbStats.MaxLifetimeClosed,
	}

	statJson, _ := json.Marshal(statsStruct)
	databaseStats := string(statJson)
	return databaseStats
}

func main() {
	logger := log.New(log.Writer(), "[HTTP] ", log.LstdFlags)
	B.SetLogger(logger)

	B.LogOut("In main()...")

	if cfg == nil {
		B.LogFatal("Config is nil")
	}

	defer db.Close()

	server := http.Server{
		Addr:         cfg.Server.Port,
		Handler:      &HttpHandler{handler: http.DefaultServeMux, logger: logger, stats: httpStats},
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	B.LogOut("Startup time: " + startupTime.Format(time.RFC3339))
	B.LogOut("Version: " + version)
	B.LogOut("Go Version: " + runtime.Version())
	B.LogOut("Number of CPUs: " + strconv.Itoa(runtime.NumCPU()))
	B.LogOut("Number of Goroutines: " + strconv.Itoa(runtime.NumGoroutine()))
	B.LogOut("Server listening on: " + cfg.Server.Port)
	B.LogOut("Server: " + fmt.Sprintf("%#v", cfg.Server))
	B.LogOut("Sites: " + fmt.Sprintf("%#v", cfg.Sites))
	B.LogOut("Ollama: " + fmt.Sprintf("%#v", cfg.Ollama))

	B.LogFatal(server.ListenAndServe())
}

func init() {
	// TODO: logger is not set yet, check main
	fmt.Println("In init()...")
	var err error
	cfg, err = Conf.LoadConfig("config.json")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if B.IsProduction() {
		err = godotenv.Load(".env")
	} else {
		err = godotenv.Load(".env.example")
	}

	if err != nil {
		fmt.Println("No .env file found")
		os.Exit(1)
	}

	databaseUrl := os.Getenv("DATABASE_URL")
	db, err = sql.Open("postgres", databaseUrl)

	fmt.Println("Testing database...")
	if err = db.Ping(); err != nil {
		// TODO: if host exists, but is unavail, this ping starts waiting forever
		fmt.Println(err)
		os.Exit(1)
	}

	db.SetMaxOpenConns(30)
	db.SetMaxIdleConns(20)
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetConnMaxIdleTime(5 * time.Minute)

	fmt.Println("Server port: " + cfg.Server.Port)

	httpStats = NewHTTPStats()

	httpRouter := http.NewServeMux()

	httpRouter.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		tmpl, err := template.ParseFiles("index.html")
		if err != nil {
			fmt.Println(err)
			http.Error(w, "Could not load template", http.StatusInternalServerError)
			return
		}

		startupMilliseconds := time.Since(startupTime).Milliseconds()
		processUptime := strconv.FormatInt(startupMilliseconds, 10)

		json := `{"uptime": "` + processUptime + `", "os": "` + runtime.GOOS + `", "arch": "` + runtime.GOARCH + `", "version": "` + version + `", "go_version": "` + runtime.Version() + `", "num_cpu": ` + strconv.Itoa(runtime.NumCPU()) + `, "num_goroutine": ` + strconv.Itoa(runtime.NumGoroutine()) + `, "num_gomaxprocs": ` + strconv.Itoa(runtime.GOMAXPROCS(0)) + `, "num_cgo_call": ` + strconv.FormatInt(runtime.NumCgoCall(), 10) + `, "db_stats": ` + dbStatsToJson(db) + `, "http_stats": ` + httpStats.GetJsonSnapshot() + `}`

		data := map[string]interface{}{
			"StatsJSON": json,
		}

		if err := tmpl.Execute(w, data); err != nil {
			fmt.Println(err)
			http.Error(w, "Could not execute template", http.StatusInternalServerError)
			return
		}

		fmt.Printf("Request served: %s %s\n", r.Method, r.URL.Path)
	})

	httpRouter.HandleFunc("GET /jq", func(w http.ResponseWriter, r *http.Request) {
		startupMilliseconds := time.Since(startupTime).Milliseconds()
		processUptime := strconv.FormatInt(startupMilliseconds, 10)

		json := `{"uptime": "` + processUptime + `", "os": "` + runtime.GOOS + `", "arch": "` + runtime.GOARCH + `", "version": "` + version + `", "go_version": "` + runtime.Version() + `", "num_cpu": ` + strconv.Itoa(runtime.NumCPU()) + `, "num_goroutine": ` + strconv.Itoa(runtime.NumGoroutine()) + `, "num_gomaxprocs": ` + strconv.Itoa(runtime.GOMAXPROCS(0)) + `, "num_cgo_call": ` + strconv.FormatInt(runtime.NumCgoCall(), 10) + `, "db_stats": ` + dbStatsToJson(db) + `, "http_stats": ` + httpStats.GetJsonSnapshot() + `}`
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(json))
	})

	httpRouter.HandleFunc("POST /auth", Api.FakeAuthHandler(db))
	httpRouter.HandleFunc("OPTIONS /auth", Api.FakeAuthHandler(db))
	httpRouter.HandleFunc("GET /sites", Api.SitesHandler(cfg.Sites))
	httpRouter.HandleFunc("OPTIONS /sites", Api.SitesHandler(cfg.Sites))
	httpRouter.HandleFunc("GET /archive", Api.ArchiveHandler(db))
	httpRouter.HandleFunc("OPTIONS /archive", Api.ArchiveHandler(db))
	httpRouter.HandleFunc("OPTIONS /search", Api.SearchHandler(db))
	httpRouter.HandleFunc("GET /search", Api.SearchHandler(db))
	httpRouter.HandleFunc("OPTIONS /refresh", Api.ArchiveRefreshHandler(cfg.Sites, db))
	httpRouter.HandleFunc("GET /refresh", Api.ArchiveRefreshHandler(cfg.Sites, db))
	httpRouter.HandleFunc("POST /translate", Ai.ExplainHandler(cfg.Ollama))
	httpRouter.HandleFunc("OPTIONS /translate", Ai.ExplainHandler(cfg.Ollama))

	http.Handle("/", httpRouter)
	http.Handle("/auth", httpRouter)
	http.Handle("/sites", httpRouter)
	http.Handle("/archive", httpRouter)
	http.Handle("/search", httpRouter)
	http.Handle("/refresh", httpRouter)
	http.Handle("/translate", httpRouter)
}
