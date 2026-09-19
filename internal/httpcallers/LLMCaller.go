package httpcallers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	llm "github.com/mattthew/sclera/internal/LLM"
	helpers "github.com/mattthew/sclera/internal/usefulHelpers"
	llmhandling "github.com/mattthew/sclera/tempFrontend/LLMHandling"
)

type ChatResponse struct {
	Reply string `json:"reply"`
}

func serverSideLLMboilerPlate(w http.ResponseWriter, r *http.Request, LLMcall func(string) (string, error)) {
	stat := helpers.VerifyHTTPMethod(w, r, http.MethodPost)
	if !stat {
		return
	}

	parseError := r.ParseForm()

	if parseError != nil {
		helpers.ThrowHTTPErrAndLog("failed parsing the html form", parseError, "Your data was not successfully handled by the server", w, http.StatusInternalServerError)
		return
	}

	userSentMessage := strings.TrimSpace(r.FormValue("userSentMessage"))

	if userSentMessage == "" {
		helpers.ThrowHTTPErrAndLog("the message cant be nil", nil, "Please input a message first.", w, http.StatusBadRequest)
		return
	}

	// send the user sent message to the LLM

	gemmaReply, gemmaErr := LLMcall(userSentMessage)

	if gemmaErr != nil {
		if errors.Is(gemmaErr, llm.ErrOverLimitToolUsage) {
			helpers.ThrowHTTPErrAndLog("AI request timed out, reason: LLM using too many Tool calls:", llm.ErrOverLimitToolUsage, "Error occured while AI replying.", w, http.StatusInternalServerError)
			return
		}

		helpers.ThrowHTTPErrAndLog("error occured whilist giving AI reply, err:", gemmaErr, "Error occured while AI replying.", w, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	jsonErr := json.NewEncoder(w).Encode(ChatResponse{Reply: gemmaReply})

	if jsonErr != nil {
		helpers.ThrowHTTPErrAndLog("error occured while trying to send json to header, error: ", jsonErr, "Your token was not successfully send to the header.", w, http.StatusInternalServerError)
		return
	}

}

// this is /ScleraChat
func CallMessageLLMClientSide() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		stat := helpers.VerifyHTTPMethod(w, r, http.MethodGet)
		if !stat {
			return
		}

		renderComponent(w, r, llmhandling.ChatWindow())
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
