package main

import (
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"runtime"
	"strconv"
	"time"

	"github.com/graphql-go/graphql"
	"github.com/graphql-go/graphql/examples/todo/schema"

	Ai "github.com/janevala/home_be/ai"
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
	llog.Fatal(server.ListenAndServe())
}

func init() {
	var err error
	cfg, err = config.LoadConfig("config.json")
	if err != nil {
		llog.Err(err)
		panic(err)
	}

	llog.Out("Server port: " + cfg.Server.Port)

	httpRouter := http.NewServeMux()
	httpRouter.HandleFunc("POST /auth", Api.AuthHandler)
	httpRouter.HandleFunc("OPTIONS /auth", Api.AuthHandler)
	httpRouter.HandleFunc("GET /sites", Api.SitesHandler(cfg.Sites))
	httpRouter.HandleFunc("OPTIONS /sites", Api.SitesHandler(cfg.Sites))
	httpRouter.HandleFunc("GET /archive", Api.ArchiveHandler(cfg.Database))
	httpRouter.HandleFunc("OPTIONS /archive", Api.ArchiveHandler(cfg.Database))
	httpRouter.HandleFunc("POST /explain", Ai.ExplainHandler())
	httpRouter.HandleFunc("OPTIONS /explain", Ai.ExplainHandler())
	httpRouter.HandleFunc("POST /graphql", graphQlHandler("Hello, GraphQL!"))
	httpRouter.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		tmpl, err := template.ParseFiles("index.html")
		if err != nil {
			llog.Err(err)
			http.Error(w, "Could not load template", http.StatusInternalServerError)
			return
		}
		data := map[string]interface{}{
			"BuildTime":    time.Now().Format(time.RFC3339),
			"GoVersion":    runtime.Version(),
			"NumCPU":       runtime.NumCPU(),
			"NumGoroutine": runtime.NumGoroutine(),
			"Database":     cfg.Database.Postgres,
			"Sites":        cfg.Sites,
		}

		if err := tmpl.Execute(w, data); err != nil {
			llog.Err(err)
			http.Error(w, "Could not execute template", http.StatusInternalServerError)
			return
		}
		llog.Out("Request served: %s %s", r.Method, r.URL.Path)
	})

	http.Handle("/auth", httpRouter)
	http.Handle("/sites", httpRouter)
	http.Handle("/archive", httpRouter)
	http.Handle("/explain", httpRouter)
	http.Handle("/graphql", httpRouter)
	http.Handle("/", httpRouter)
}

type postData struct {
	Query     string                 `json:"query"`
	Operation string                 `json:"operationName"`
	Variables map[string]interface{} `json:"variables"`
}

func graphQlHandler(q string) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		var p postData
		if err := json.NewDecoder(req.Body).Decode(&p); err != nil {
			w.WriteHeader(400)
			return
		}
		result := graphql.Do(graphql.Params{
			Context:        req.Context(),
			Schema:         schema.TodoSchema,
			RequestString:  p.Query,
			VariableValues: p.Variables,
			OperationName:  p.Operation,
		})

		if err := json.NewEncoder(w).Encode(result); err != nil {
			llog.Err(err)
		}

		//// INCOMPLETE DABBLING

		fields := graphql.Fields{
			"hello": &graphql.Field{
				Type: graphql.String,
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					return q, nil
				},
			},
		}

		rootQuery := graphql.ObjectConfig{Name: "RootQuery", Fields: fields}
		schemaConfig := graphql.SchemaConfig{Query: graphql.NewObject(rootQuery)}
		schema, err := graphql.NewSchema(schemaConfig)
		if err != nil {
			llog.Err(err)
		}

		query := `
				{
					hello
				}
			`
		params := graphql.Params{Schema: schema, RequestString: query}
		res := graphql.Do(params)
		if len(res.Errors) > 0 {
			llog.Out("failed to execute graphql operation, errors: %+v", res.Errors)
		}
		json, _ := json.Marshal(res)

		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Write(json)
		w.WriteHeader(http.StatusOK)
	}
}
