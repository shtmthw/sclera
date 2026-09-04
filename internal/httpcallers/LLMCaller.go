package httpcallers

import (
	"encoding/json"
	"errors"
	"html/template"
	"net/http"
	"strings"

	llm "github.com/mattthew/sclera/internal/LLM"
)

type ChatResponse struct {
	Reply string `json:"reply"`
}

var parseGemmaChatWindowTemp = template.Must(template.ParseFiles("LLMHandling/LLMChatWindow.html"))

func serverSideLLMboilerPlate(w http.ResponseWriter, r *http.Request, LLMcall func(string) (string, error)) {
	stat := VerifyHTTPMethod(w, r, http.MethodPost)
	if !stat {
		return
	}

	parseError := r.ParseForm()

	if parseError != nil {
		ThrowHTTPErrAndLog("failed parsing the html form", parseError, "Your data was not successfully handled by the server", w, http.StatusInternalServerError)
		return
	}

	userSentMessage := strings.TrimSpace(r.FormValue("userSentMessage"))

	if userSentMessage == "" {
		ThrowHTTPErrAndLog("the message cant be nil", nil, "Please input a message first.", w, http.StatusBadRequest)
		return
	}

	// send the user sent message to the LLM

	gemmaReply, gemmaErr := LLMcall(userSentMessage)

	if gemmaErr != nil {
		if errors.Is(gemmaErr, llm.OverLimitToolUsage) {
			ThrowHTTPErrAndLog("AI request timed out, reason: LLM using too many Tool calls:", llm.OverLimitToolUsage, "Error occured while AI replying.", w, http.StatusInternalServerError)
			return
		}

		ThrowHTTPErrAndLog("error occured whilist giving AI reply, err:", gemmaErr, "Error occured while AI replying.", w, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	jsonErr := json.NewEncoder(w).Encode(ChatResponse{Reply: gemmaReply})

	if jsonErr != nil {
		ThrowHTTPErrAndLog("error occured while trying to send json to header, error: ", jsonErr, "Your token was not successfully send to the header.", w, http.StatusInternalServerError)
		return
	}

}

// this is /chatWithGemma
func CallMessageLLMClientSide() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		stat := VerifyHTTPMethod(w, r, http.MethodGet)
		if !stat {
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")

		tempParseErrr := parseGemmaChatWindowTemp.Execute(w, nil)

		if tempParseErrr != nil {
			ThrowHTTPErrAndLog("failed to render template", tempParseErrr, "Internal server error", w, http.StatusInternalServerError)
			return
		}

	}
}

// this is /gemmaProcessUserSentMessage
func CallMessageGemmaServerSide() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		serverSideLLMboilerPlate(w, r, llm.AskGemma)
	}
}

//this is /OSSProcessUserSentMessage

func CallMessageOSSServerSide() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		serverSideLLMboilerPlate(w, r, llm.AskOss)

	}
}
