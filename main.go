package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/graphql-go/graphql"
	"github.com/graphql-go/graphql/examples/todo/schema"

	Api "github.com/janevala/home_be/api"
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

	sites := Api.Sites{}
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

	database := Api.Database{}
	json.Unmarshal(databaseFile, &database)
	databaseString, err := json.MarshalIndent(database, "", "\t")
	if err != nil {
		panic(err)
	} else {
		log.Println(string(databaseString))
	}

	r := mux.NewRouter()
	r.HandleFunc("/auth", Api.AuthHandler).Methods(http.MethodPost, http.MethodOptions)
	r.HandleFunc("/sites", Api.RssHandler(sites)).Methods(http.MethodGet, http.MethodOptions)
	r.HandleFunc("/archive", Api.ArchiveHandler(database)).Methods(http.MethodGet, http.MethodOptions)
	r.HandleFunc("/explain", Api.Explain()).Methods(http.MethodPost, http.MethodOptions)
	r.HandleFunc("/graphql", handleQuery("ASDF")).Methods(http.MethodPost, http.MethodOptions)
	r.HandleFunc("/", homeHandler).Methods(http.MethodGet)
	http.Handle("/", r)
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Hello, World!"))
}

type postData struct {
	Query     string                 `json:"query"`
	Operation string                 `json:"operationName"`
	Variables map[string]interface{} `json:"variables"`
}

func handleQuery(q string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.Header().Set("Access-Control-Allow-Origin", "*")

			w.WriteHeader(http.StatusOK)
		} else if r.Method == http.MethodPost {

			var p postData
			if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
				w.WriteHeader(400)
				return
			}
			result := graphql.Do(graphql.Params{
				Context:        r.Context(),
				Schema:         schema.TodoSchema,
				RequestString:  p.Query,
				VariableValues: p.Variables,
				OperationName:  p.Operation,
			})

			if err := json.NewEncoder(w).Encode(result); err != nil {
				log.Printf("could not write result to response: %s", err)
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
				log.Fatalf("failed to create new schema, error: %v", err)
			}

			query := `
				{
					hello
				}
			`
			params := graphql.Params{Schema: schema, RequestString: query}
			r := graphql.Do(params)
			if len(r.Errors) > 0 {
				log.Fatalf("failed to execute graphql operation, errors: %+v", r.Errors)
			}
			json, _ := json.Marshal(r)

			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Write(json)
			w.WriteHeader(http.StatusOK)
		}
	}
}
