package routes

import (
	"net/http"

	"github.com/redis/go-redis/v9"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mattthew/sclera/internal/httpcallers"
	"github.com/mattthew/sclera/internal/middleware"
)

func RegisterUserRoutes(mux *http.ServeMux, pool *pgxpool.Pool, redisClient *redis.Client) {
	mux.HandleFunc("/getUserData", middleware.CheckJwtToken(httpcallers.CallGetUser(pool)))
	mux.HandleFunc("/deleteUserData", middleware.CheckJwtToken(httpcallers.CallDeleteUser(pool)))
	mux.HandleFunc("/logoutUser", middleware.CheckJwtToken(httpcallers.CallLogoutUser()))
	mux.HandleFunc("/createUser", httpcallers.CallCreateUser(pool))
	mux.HandleFunc("/newUser", httpcallers.CallNewUser())
	mux.HandleFunc("/loginUser", httpcallers.CallLoginUser())
	mux.HandleFunc("/verifyLogin", httpcallers.CallVerifyUser(pool))
	mux.HandleFunc("/updateAccout", middleware.CheckJwtToken(httpcallers.CallUpdateUserClientSide()))
	mux.HandleFunc("/updateUserAccount", middleware.CheckJwtToken(httpcallers.CallUpdateUserServerSide(pool)))

}
