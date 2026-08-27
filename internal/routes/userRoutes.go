package routes

import (
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mattthew/sclera/internal/httpcallers"
	"github.com/mattthew/sclera/internal/middleware"
)

func RegisterUserRoutes(mux *http.ServeMux, pool *pgxpool.Pool) {
	mux.HandleFunc("/getUserData", middleware.CheckJwtToken(httpcallers.CallGetUser(pool)))
	mux.HandleFunc("/deleteUserData", middleware.CheckJwtToken(httpcallers.DeleteUser(pool)))
	mux.HandleFunc("/logoutUser", middleware.CheckJwtToken(httpcallers.LogoutUser()))
	mux.HandleFunc("/createUser", httpcallers.CallCreateUser(pool))
	mux.HandleFunc("/newUser", httpcallers.CallNewUser())
	mux.HandleFunc("/loginUser", httpcallers.CallLoginUser())
	mux.HandleFunc("/verifyLogin", httpcallers.CallVerifyUser(pool))

}
