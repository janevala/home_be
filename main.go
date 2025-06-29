package main

import (
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/graphql-go/graphql"
	"github.com/graphql-go/graphql/examples/todo/schema"

	Ai "github.com/janevala/home_be/ai"
	Api "github.com/janevala/home_be/api"
	Log "github.com/janevala/home_be/llog"
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

	Log.Out("Number of CPUs: " + strconv.Itoa(runtime.NumCPU()))
	Log.Out("Number of Goroutines: " + strconv.Itoa(runtime.NumGoroutine()))
	Log.Out("Server listening on: " + serverPort)
	Log.Fatal(server.ListenAndServe())
}

func init() {
	sitesFile, err := os.ReadFile("sites.json")
	if err != nil {
		Log.Err(err)
		panic(err)
	}

	sites := Api.Sites{}
	json.Unmarshal(sitesFile, &sites)
	sitesString, err := json.MarshalIndent(sites, "", "\t")
	if err != nil {
		Log.Err(err)
		panic(err)
	} else {
		sites.Time = int(time.Now().UTC().UnixMilli())
		for i := 0; i < len(sites.Sites); i++ {
			sites.Sites[i].Uuid = uuid.NewString()
		}

		Log.Out(string(sitesString))
	}

	databaseFile, err := os.ReadFile("database.json")
	if err != nil {
		Log.Err(err)
		panic(err)
	}

	database := Api.Database{}
	json.Unmarshal(databaseFile, &database)
	databaseString, err := json.MarshalIndent(database, "", "\t")
	if err != nil {
		Log.Err(err)
		panic(err)
	} else {
		Log.Out(string(databaseString))
	}

	httpRouter := http.NewServeMux()
	httpRouter.HandleFunc("POST /auth", Api.AuthHandler)
	httpRouter.HandleFunc("OPTIONS /auth", Api.AuthHandler)
	httpRouter.HandleFunc("GET /sites", Api.SitesHandler(sites))
	httpRouter.HandleFunc("OPTIONS /sites", Api.SitesHandler(sites))
	httpRouter.HandleFunc("GET /archive", Api.ArchiveHandler(database))
	httpRouter.HandleFunc("OPTIONS /archive", Api.ArchiveHandler(database))
	httpRouter.HandleFunc("POST /explain", Ai.ExplainHandler())
	httpRouter.HandleFunc("OPTIONS /explain", Ai.ExplainHandler())
	httpRouter.HandleFunc("POST /graphql", graphQlHandler("Hello, GraphQL!"))
	httpRouter.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		tmpl, err := template.ParseFiles("index.html")
		if err != nil {
			Log.Err(err)
			http.Error(w, "Could not load template", http.StatusInternalServerError)
			return
		}
		data := map[string]interface{}{
			"BuildTime":    time.Now().Format(time.RFC3339),
			"GoVersion":    runtime.Version(),
			"NumCPU":       runtime.NumCPU(),
			"NumGoroutine": runtime.NumGoroutine(),
			"Database":     database,
			"Sites":        sites,
		}

		if err := tmpl.Execute(w, data); err != nil {
			Log.Err(err)
			http.Error(w, "Could not execute template", http.StatusInternalServerError)
			return
		}
		Log.Out("Request served: %s %s", r.Method, r.URL.Path)
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
			Log.Err(err)
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
			Log.Err(err)
		}

		query := `
				{
					hello
				}
			`
		params := graphql.Params{Schema: schema, RequestString: query}
		res := graphql.Do(params)
		if len(res.Errors) > 0 {
			Log.Out("failed to execute graphql operation, errors: %+v", res.Errors)
		}
		json, _ := json.Marshal(res)

		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Write(json)
		w.WriteHeader(http.StatusOK)
	}
}
