// ai/ai.go
package ai

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	B "github.com/janevala/home_be/build"
	"github.com/janevala/home_be/config"
	_ "github.com/lib/pq"
	"github.com/rifaideen/talkative"
)

type QuestionItem struct {
	Question string `json:"question,omitempty"`
}

type AnswerItem struct {
	Answer string `json:"answer,omitempty"`
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
			var bodyBytes []byte

			if req.Body != nil {
				var err error
				bodyBytes, err = io.ReadAll(req.Body)
				if err != nil {
					return
				}
				defer req.Body.Close()
			}

			var questionObject QuestionItem
			var jsonString bytes.Buffer

			if len(bodyBytes) > 0 {
				if err := json.Indent(&jsonString, bodyBytes, "", "\t"); err != nil {
					B.LogErr(err)
					return
				}
				err := json.Unmarshal(bodyBytes, &questionObject)
				if err != nil {
					B.LogErr(err)
					return
				}
			} else {
				B.LogOut("ERROR: Empty body in request")
				return
			}

			var question string = questionObject.Question
			questionItem := QuestionItem{Question: question}

			answerItem := queryAI(questionItem, ollama)

			responseJson, _ := json.Marshal(answerItem)
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.WriteHeader(http.StatusOK)
			w.Write(responseJson)
		}
	}
}

func queryAI(q QuestionItem, ollama config.Ollama) AnswerItem {
	client, err := talkative.New("http://" + ollama.Host + ":" + ollama.Port)

	if err != nil {
		panic("Failed to create talkative client")
	}

	response := talkative.ChatResponse{}
	callback := func(cr *talkative.ChatResponse, err error) {
		if err != nil {
			B.LogErr(err)
			return
		}

		response = *cr
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
		B.LogErr(err)
	}

	<-done

	answerItem := AnswerItem{Answer: response.Message.Content}

	return answerItem
}
