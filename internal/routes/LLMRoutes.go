package routes

import (
	"net"
	"net/http"

	"github.com/mattthew/sclera/internal/httpcallers"
	"github.com/mattthew/sclera/internal/middleware"
	"github.com/mattthew/sclera/internal/redisInternal"
	"github.com/redis/go-redis/v9"
)

// gdgdsgd
func RegisterGemmaRoutes(mux *http.ServeMux, redisClient *redis.Client, trustedProxyNet *net.IPNet) {
	mux.HandleFunc("/ScleraChat", middleware.CheckJwtToken(httpcallers.CallMessageLLMClientSide(), trustedProxyNet, redisClient, redisInternal.JWTTokenCost))
	mux.HandleFunc("/gemmaProcessUserSentMessage", middleware.CheckJwtToken(httpcallers.CallMessageGemmaServerSide(), trustedProxyNet, redisClient, redisInternal.LLMTokenCost))
	mux.HandleFunc("/OSSProcessUserSentMessage", middleware.CheckJwtToken(httpcallers.CallMessageOSSServerSide(), trustedProxyNet, redisClient, redisInternal.LLMTokenCost))

}
