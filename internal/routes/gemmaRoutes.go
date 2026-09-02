package routes

import (
	"net/http"

	"github.com/mattthew/sclera/internal/httpcallers"
	"github.com/mattthew/sclera/internal/middleware"
)

func RegisterGemmaRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/chatWithGemma", middleware.CheckJwtToken(httpcallers.CallMessageGemmaClientSide()))
	mux.HandleFunc("/processUserSentMessage", middleware.CheckJwtToken(httpcallers.CallMessageGemmaServerSide()))
}
