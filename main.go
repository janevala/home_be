package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

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
	RequestCounts     map[string]map[string]map[int]int
	TotalResponseTime time.Duration
	DurationBuckets   map[string]float64
	InflightRequests  int
}

func NewHTTPStats() *HTTPStats {
	return &HTTPStats{
		RequestsByMethod:  make(map[string]int),
		RequestsByPath:    make(map[string]int),
		ResponseCodeCount: make(map[int]int),
		RequestCounts:     make(map[string]map[string]map[int]int),
		DurationBuckets:   make(map[string]float64),
	}
}

func (s *HTTPStats) IncrementInflight() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.InflightRequests++
}

func (s *HTTPStats) DecrementInflight() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.InflightRequests > 0 {
		s.InflightRequests--
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

	// Track histogram buckets
	durationSeconds := float64(duration.Nanoseconds()) / 1e9
	buckets := []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10}
	for _, bucket := range buckets {
		if durationSeconds <= bucket {
			bucketKey := fmt.Sprintf("le=\"%f\"", bucket)
			s.DurationBuckets[bucketKey]++
		}
	}
	// +Inf bucket (all requests)
	s.DurationBuckets["le=\"+Inf\""]++

	if s.RequestCounts[method] == nil {
		s.RequestCounts[method] = make(map[string]map[int]int)
	}
	if s.RequestCounts[method][path] == nil {
		s.RequestCounts[method][path] = make(map[int]int)
	}
	s.RequestCounts[method][path][statusCode]++
}

func (s *HTTPStats) GetJqSnapshot() string {
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

func (s *HTTPStats) GetPrometheusMetrics() string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var metrics []string

	metrics = append(metrics, fmt.Sprintf("http_inflight_requests %d", s.InflightRequests))

	for method, handlers := range s.RequestCounts {
		for handler, codes := range handlers {
			for code, count := range codes {
				metrics = append(metrics, fmt.Sprintf(`http_requests_total{method="%s",handler="%s",code="%d"} %d`, method, handler, code, count))
			}
		}
	}

	if s.TotalRequests > 0 {
		// Histogram buckets for http_request_duration_seconds
		buckets := []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10}
		for _, bucket := range buckets {
			bucketKey := fmt.Sprintf("le=\"%f\"", bucket)
			if count, exists := s.DurationBuckets[bucketKey]; exists {
				metrics = append(metrics, fmt.Sprintf(`http_request_duration_seconds_bucket{le="%f"} %f`, bucket, count))
			}
		}
		// +Inf bucket
		if count, exists := s.DurationBuckets["le=\"+Inf\""]; exists {
			metrics = append(metrics, fmt.Sprintf(`http_request_duration_seconds_bucket{le="+Inf"} %f`, count))
		}

		// Sum and count
		metrics = append(metrics, fmt.Sprintf("http_request_duration_seconds_sum %f", float64(s.TotalResponseTime.Nanoseconds())/1e9))
		metrics = append(metrics, fmt.Sprintf("http_request_duration_seconds_count %d", s.TotalRequests))
	}

	return strings.Join(metrics, "\n")
}

type HttpHandler struct {
	handler http.Handler
	logger  *log.Logger
	stats   *HTTPStats
}

func (h *HttpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.stats.IncrementInflight()
	defer h.stats.DecrementInflight()

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

func dbContentsToJson(db *sql.DB) string {
	type Result struct {
		TotalFeedItems        int
		TotalFeedTranslations int
		NewestFeedItem        time.Time
		OldestFeedItem        time.Time
		NewestFeedTranslation time.Time
		OldestFeedTranslation time.Time
	}

	var result Result

	err := db.QueryRow(`
		SELECT 
			(SELECT COUNT(*) FROM feed_items) AS total_feed_items,
			(SELECT COUNT(*) FROM feed_translations) AS total_feed_translations,
			(SELECT COALESCE(MAX(published_parsed), '1970-01-01') FROM feed_items) AS newest_feed_item,
			(SELECT COALESCE(MIN(published_parsed), '1970-01-01') FROM feed_items) AS oldest_feed_item,
			(SELECT COALESCE(MAX(published_parsed), '1970-01-01') FROM feed_translations) AS newest_feed_translation,
			(SELECT COALESCE(MIN(published_parsed), '1970-01-01') FROM feed_translations) AS oldest_feed_translation
	`).Scan(
		&result.TotalFeedItems,
		&result.TotalFeedTranslations,
		&result.NewestFeedItem,
		&result.OldestFeedItem,
		&result.NewestFeedTranslation,
		&result.OldestFeedTranslation,
	)

	if err != nil {
		B.LogErr(err)
		return "{}"
	}

	jsonData, _ := json.Marshal(result)
	return string(jsonData)
}

func memoryStatsToJson() string {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	memStats := map[string]interface{}{
		"Alloc":        m.Alloc,
		"TotalAlloc":   m.TotalAlloc,
		"Sys":          m.Sys,
		"NumGC":        m.NumGC,
		"PauseTotalNs": m.PauseTotalNs,
		"PauseNs":      m.PauseNs,
		"PauseEnd":     m.PauseEnd,
		"BySize":       m.BySize,
	}
	memStatsJson, _ := json.Marshal(memStats)
	return string(memStatsJson)
}

func prometheusMetrics() string {
	var metrics []string

	// HTTP metrics using existing data
	metrics = append(metrics, "# HELP http_requests_total Total HTTP requests")
	metrics = append(metrics, "# TYPE http_requests_total counter")
	metrics = append(metrics, httpStats.GetPrometheusMetrics())

	// Database metrics using existing data
	dbStats := db.Stats()
	metrics = append(metrics, "")
	metrics = append(metrics, "# HELP pg_connections Number of active connections")
	metrics = append(metrics, "# TYPE pg_connections gauge")
	metrics = append(metrics, fmt.Sprintf("pg_connections %d", dbStats.OpenConnections))
	metrics = append(metrics, "")
	metrics = append(metrics, "# HELP pg_connections_idle Number of idle connections")
	metrics = append(metrics, "# TYPE pg_connections_idle gauge")
	metrics = append(metrics, fmt.Sprintf("pg_connections_idle %d", dbStats.Idle))
	metrics = append(metrics, "")
	metrics = append(metrics, "# HELP pg_connections_in_use Number of connections in use")
	metrics = append(metrics, "# TYPE pg_connections_in_use gauge")
	metrics = append(metrics, fmt.Sprintf("pg_connections_in_use %d", dbStats.InUse))

	// Go runtime metrics
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	metrics = append(metrics, "")
	metrics = append(metrics, "# HELP go_goroutines Number of goroutines")
	metrics = append(metrics, "# TYPE go_goroutines gauge")
	metrics = append(metrics, fmt.Sprintf("go_goroutines %d", runtime.NumGoroutine()))
	metrics = append(metrics, "")
	metrics = append(metrics, "# HELP process_resident_memory_bytes Resident memory size")
	metrics = append(metrics, "# TYPE process_resident_memory_bytes gauge")
	metrics = append(metrics, fmt.Sprintf("process_resident_memory_bytes %d", m.Sys))
	metrics = append(metrics, "")
	metrics = append(metrics, "# HELP process_cpu_seconds_total CPU time")
	metrics = append(metrics, "# TYPE process_cpu_seconds_total counter")
	metrics = append(metrics, fmt.Sprintf("process_cpu_seconds_total %f", float64(time.Since(startupTime).Milliseconds())/1000))

	return strings.Join(metrics, "\n")
}

func main() {
	logger := log.New(log.Writer(), "[HTTP] ", log.LstdFlags)
	B.SetLogger(logger)

	B.LogOut("In main()...")

	if cfg == nil {
		B.LogOut("Config is nil")
	}

	defer db.Close()

	server := http.Server{
		Addr:         cfg.Server.Port,
		Handler:      &HttpHandler{handler: http.DefaultServeMux, logger: logger, stats: httpStats},
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	B.LogOut("Version: " + version)
	B.LogOut("Go Version: " + runtime.Version())
	B.LogOut("Server listening on: " + cfg.Server.Port)
	B.LogOut("Server: " + fmt.Sprintf("%#v", cfg.Server))
	B.LogOut("Sites: " + fmt.Sprintf("%#v", cfg.Sites))
	B.LogOut("Ollama: " + fmt.Sprintf("%#v", cfg.Ollama))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		B.LogOut("Server started...")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			B.LogErr(err)
		}
	}()

	<-ctx.Done()

	B.LogOut("Shutdown signal received")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		B.LogErr(err)
	}

	B.LogOut("Server exited properly")
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

	err = godotenv.Load(".env")

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
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	httpRouter.HandleFunc("GET /jq", func(w http.ResponseWriter, req *http.Request) {
		if !strings.Contains(req.URL.RawQuery, "code=123") {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Invalid"))
			return
		}

		startupMilliseconds := time.Since(startupTime).Milliseconds()
		processUptime := strconv.FormatInt(startupMilliseconds, 10)

		json := `{"uptime": "` + processUptime + `", "os": "` + runtime.GOOS + `", "arch": "` + runtime.GOARCH + `", "version": "` + version + `", "go_version": "` + runtime.Version() + `", "num_cpu": ` + strconv.Itoa(runtime.NumCPU()) + `, "num_goroutine": ` + strconv.Itoa(runtime.NumGoroutine()) + `, "num_gomaxprocs": ` + strconv.Itoa(runtime.GOMAXPROCS(0)) + `, "num_cgo_call": ` + strconv.FormatInt(runtime.NumCgoCall(), 10) + `, "memory_stats": ` + memoryStatsToJson() + `, "db_stats": ` + dbStatsToJson(db) + `, "db_contents": ` + dbContentsToJson(db) + `, "http_stats": ` + httpStats.GetJqSnapshot() + `}`
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(json))
	})

	httpRouter.HandleFunc("GET /metrics", func(w http.ResponseWriter, req *http.Request) {
		response := prometheusMetrics()

		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(response))
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
	// httpRouter.HandleFunc("POST /translate", Ai.ExplainHandler(cfg.Ollama))
	// httpRouter.HandleFunc("OPTIONS /translate", Ai.ExplainHandler(cfg.Ollama))

	corsRouter := corsMiddleware(httpRouter)

	http.Handle("/", corsRouter)
	http.Handle("/jq", corsRouter)
	http.Handle("/auth", corsRouter)
	http.Handle("/sites", corsRouter)
	http.Handle("/archive", corsRouter)
	http.Handle("/search", corsRouter)
	http.Handle("/refresh", corsRouter)
	// http.Handle("/translate", corsRouter)
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Allow-Origin", "https://techeavy.news")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
