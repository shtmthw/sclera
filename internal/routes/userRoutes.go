package routes

import (
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mattthew/sclera/internal/httpcallers"
	"github.com/mattthew/sclera/internal/middleware"
	"github.com/redis/go-redis/v9"
	"github.com/resend/resend-go/v3"
)

func RegisterUserRoutes(mux *http.ServeMux, pool *pgxpool.Pool, redisClient *redis.Client, resendClient *resend.Client) {
	mux.HandleFunc("/getUserData", middleware.CheckJwtToken(httpcallers.CallGetUser(pool)))
	mux.HandleFunc("/deleteUserData", middleware.CheckJwtToken(httpcallers.CallDeleteUser(pool)))
	mux.HandleFunc("/logoutUser", middleware.CheckJwtToken(httpcallers.CallLogoutUser()))
	mux.HandleFunc("/createUser", httpcallers.CallCreateUser(pool))
	mux.HandleFunc("/newUser", httpcallers.CallNewUser())
	mux.HandleFunc("/loginUser", httpcallers.CallLoginUser())
	mux.HandleFunc("/verifyLogin", httpcallers.CallVerifyUser(pool))
	mux.HandleFunc("/updateAccout", middleware.CheckJwtToken(httpcallers.CallUpdateUserClientSide()))
	mux.HandleFunc("/updateUserAccount", middleware.CheckJwtToken(httpcallers.CallUpdateUserServerSide(pool)))
	mux.HandleFunc("/sendVerificationMail", middleware.CheckJwtToken(httpcallers.CallSendVerificationMail(resendClient, redisClient)))
	mux.HandleFunc("/inputOTP", middleware.CheckJwtToken(httpcallers.CallVerifyOTPclientSide()))
	mux.HandleFunc("/verifyOTP", middleware.CheckJwtToken(httpcallers.CallVerifyOTPserverSide(redisClient)))

}
