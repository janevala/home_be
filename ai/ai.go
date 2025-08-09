// ai/ai.go
package ai

import (
	"net/http"
	"strings"

	"encoding/json"

	"github.com/graphql-go/graphql"
	"github.com/janevala/home_be/config"
	"github.com/janevala/home_be/llog"
	_ "github.com/lib/pq"
	"github.com/rifaideen/talkative"
)

type QuestionItem struct {
	Question string `json:"question,omitempty"`
}

type AnswerItem struct {
	Answer string `json:"answer,omitempty"`
}

type QueryPost struct {
	Query     string                 `json:"query"`
	Operation string                 `json:"operationName"`
	Variables map[string]interface{} `json:"variables"`
}

func ExplainHandler(mcpServer config.McpServer) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		switch req.Method {
		case http.MethodOptions:
			w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.Header().Set("Access-Control-Allow-Origin", "*")

			w.WriteHeader(http.StatusOK)
		case http.MethodPost:
			if !strings.Contains(req.URL.RawQuery, "code=123") {
				llog.Out("Invalid request: missing or incorrect code parameter")

				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte("Invalid URI"))
				return
			}

			var q QueryPost
			if err := json.NewDecoder(req.Body).Decode(&q); err != nil {
				w.WriteHeader(400)
				return
			}

			fields := graphql.Fields{
				"query": &graphql.Field{
					Type: graphql.String,
					Resolve: func(p graphql.ResolveParams) (interface{}, error) {
						questionItem := QuestionItem{
							Question: p.Args["question"].(string),
						}

						answerItem := queryAI(questionItem)
						// answerItem := queryAI(questionItem, mcpServer)

						return answerItem.Answer, nil
					},
				},
			}

			rootQuery := graphql.ObjectConfig{Name: "RootQuery", Fields: fields}
			schemaConfig := graphql.SchemaConfig{Query: graphql.NewObject(rootQuery)}
			schema, err := graphql.NewSchema(schemaConfig)
			if err != nil {
				llog.Err(err)
			}

			params := graphql.Params{Schema: schema, RequestString: q.Query}
			r := graphql.Do(params)
			// if len(r.Errors) > 0 {
			// 	llog.Err(r.Errors[0].Error())
			// }

			responseJson, _ := json.Marshal(r)
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Write(responseJson)
			w.WriteHeader(http.StatusOK)
		}
	}
}

// / MCP
// func queryAI(q QuestionItem, mcpServer config.McpServer) AnswerItem {
// 	var question string = q.Question

// 	ctx := context.Background()

// 	client := mcp.NewClient(&mcp.Implementation{Name: "mcp-client", Version: "v1.0.0"}, nil)

// 	transport := mcp.NewCommandTransport(exec.Command(mcpServer.Host, mcpServer.Port))
// 	session, err := client.Connect(ctx, transport)
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	defer session.Close()

// 	params := &mcp.CallToolResultFor[string]{
// 		Content: []mcp.Content{
// 			&mcp.TextContent{
// 				Text: question,
// 			},
// 		},
// 	}

// 	answerItem := AnswerItem{Answer: params.Content[0].(*mcp.TextContent).Text}

// 	return answerItem
// }

// / OLLAMA
func queryAI(q QuestionItem) AnswerItem {
	var question string = q.Question

	client, err := talkative.New("http://127.0.0.1:11434")

	if err != nil {
		panic("Failed to create talkative client")
	}

	model := "mistral:7b"
	//model := "qwen2.5-coder:14b"

	responseAnswer := talkative.ChatResponse{}
	callback := func(cr *talkative.ChatResponse, err error) {
		if err != nil {
			llog.Err(err)
			return
		}

		responseAnswer = *cr
	}

	message := talkative.ChatMessage{
		Role:    talkative.USER,
		Content: question,
	}

	b := false
	done, err := client.Chat(model, callback, &talkative.ChatParams{
		Stream: &b,
	}, message)

	if err != nil {
		panic(err)
	}

	<-done

	answerItem := AnswerItem{Answer: responseAnswer.Message.Content}

	return answerItem
}
