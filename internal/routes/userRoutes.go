package routes

import (
	"net"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mattthew/sclera/internal/httpcallers"
	"github.com/mattthew/sclera/internal/middleware"
	"github.com/mattthew/sclera/internal/redisInternal"
	"github.com/redis/go-redis/v9"
	"github.com/resend/resend-go/v3"
)

func RegisterUserRoutes(mux *http.ServeMux, pool *pgxpool.Pool, redisClient *redis.Client, resendClient *resend.Client, trustedProxyNet *net.IPNet) {
	mux.HandleFunc("/getUserData", middleware.CheckJwtToken(httpcallers.CallGetUser(pool), trustedProxyNet, redisClient, redisInternal.JWTTokenCost))
	mux.HandleFunc("/deleteUserData", middleware.CheckJwtToken(httpcallers.CallDeleteUser(pool), trustedProxyNet, redisClient, redisInternal.JWTTokenCost))
	mux.HandleFunc("/logoutUser", middleware.CheckJwtToken(httpcallers.CallLogoutUser(), trustedProxyNet, redisClient, redisInternal.JWTTokenCost))
	mux.HandleFunc("/createUser", httpcallers.WithIPRateLimit(httpcallers.CallCreateUser(pool), redisClient, trustedProxyNet, redisInternal.FormTokenCost))
	mux.HandleFunc("/newUser", httpcallers.WithIPRateLimit(httpcallers.CallNewUser(), redisClient, trustedProxyNet, redisInternal.PageTokenCost))
	mux.HandleFunc("/loginUser", httpcallers.WithIPRateLimit(httpcallers.CallLoginUser(), redisClient, trustedProxyNet, redisInternal.PageTokenCost))
	mux.HandleFunc("/verifyLogin", httpcallers.WithIPRateLimit(httpcallers.CallVerifyUser(pool), redisClient, trustedProxyNet, redisInternal.FormTokenCost))
	mux.HandleFunc("/updateAccout", middleware.CheckJwtToken(httpcallers.CallUpdateUserClientSide(pool), trustedProxyNet, redisClient, redisInternal.JWTTokenCost))
	mux.HandleFunc("/updateUserAccount", middleware.CheckJwtToken(httpcallers.CallUpdateUserServerSide(pool), trustedProxyNet, redisClient, redisInternal.JWTTokenCost))
	mux.HandleFunc("/sendVerificationMail", httpcallers.WithIPRateLimit(httpcallers.CallSendVerificationMail(resendClient, pool, redisClient), redisClient, trustedProxyNet, redisInternal.FormTokenCost))
	mux.HandleFunc("/inputOTP", httpcallers.WithIPRateLimit(httpcallers.CallVerifyOTPclientSide(), redisClient, trustedProxyNet, redisInternal.PageTokenCost))
	mux.HandleFunc("/verifyOTP", httpcallers.WithIPRateLimit(httpcallers.CallVerifyOTPserverSide(redisClient, pool), redisClient, trustedProxyNet, redisInternal.FormTokenCost))
	mux.HandleFunc("/updateUsersPassword", middleware.CheckJwtToken(httpcallers.CallUpdateUserPasswordClientSide(), trustedProxyNet, redisClient, redisInternal.JWTTokenCost))
	mux.HandleFunc("/runPasswordUpdation", middleware.CheckJwtToken(httpcallers.CallUpdateUserPasswordServerSide(pool), trustedProxyNet, redisClient, redisInternal.JWTTokenCost))

}