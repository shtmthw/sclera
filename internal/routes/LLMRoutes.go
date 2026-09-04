package routes

import (
	"net/http"

	"github.com/mattthew/sclera/internal/httpcallers"
	"github.com/mattthew/sclera/internal/middleware"
)

func RegisterGemmaRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/chatWithGemma", middleware.CheckJwtToken(httpcallers.CallMessageLLMClientSide()))
	mux.HandleFunc("/gemmaProcessUserSentMessage", middleware.CheckJwtToken(httpcallers.CallMessageGemmaServerSide()))
	mux.HandleFunc("/OSSProcessUserSentMessage", middleware.CheckJwtToken(httpcallers.CallMessageOSSServerSide()))

}
