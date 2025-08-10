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

func ExplainHandler(ollama config.Ollama) http.HandlerFunc {
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

						answerItem := queryAI(questionItem, ollama)

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

			responseJson, _ := json.Marshal(r)
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Write(responseJson)
			w.WriteHeader(http.StatusOK)
		}
	}
}

func queryAI(q QuestionItem, ollama config.Ollama) AnswerItem {
	client, err := talkative.New("http://" + ollama.Host + ":" + ollama.Port)

	if err != nil {
		panic("Failed to create talkative client")
	}

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
		Content: q.Question,
	}

	b := false
	done, err := client.Chat(ollama.Model, callback, &talkative.ChatParams{
		Stream: &b,
	}, message)

	if err != nil {
		llog.Err(err)
	}

	<-done

	answerItem := AnswerItem{Answer: responseAnswer.Message.Content}

	return answerItem
}
