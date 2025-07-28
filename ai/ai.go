package ai

import (
	"net/http"
	"strings"

	"bytes"
	"encoding/json"
	"io"

	"github.com/janevala/home_be/llog"
	_ "github.com/lib/pq"
	"github.com/rifaideen/talkative"
)

type Database struct {
	Postgres string `json:"postgres"`
}

type QuestionItem struct {
	Question string `json:"question,omitempty"`
}

type AnswerItem struct {
	Answer string `json:"answer,omitempty"`
}

func ExplainHandler() http.HandlerFunc {
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

			var bodyBytes []byte
			var err error

			if req.Body != nil {
				bodyBytes, err = io.ReadAll(req.Body)
				if err != nil {
					llog.Err(err)
					return
				}
				defer req.Body.Close()
			}

			var questionItem QuestionItem
			var jsonString bytes.Buffer

			if len(bodyBytes) > 0 {
				if err = json.Indent(&jsonString, bodyBytes, "", "\t"); err != nil {
					llog.Err(err)
					return
				}
				err := json.Unmarshal(bodyBytes, &questionItem)
				if err != nil {
					llog.Err(err)
					return
				}
			} else {
				llog.Out("Body: No Body Supplied\n")
			}

			answerItem := queryAI(questionItem)

			responseJson, _ := json.Marshal(answerItem)
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Write(responseJson)
			w.WriteHeader(http.StatusOK)
		}
	}
}

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
