package routes

import (
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mattthew/sclera/internal/httpcallers"
	"github.com/mattthew/sclera/internal/middleware"
)

func RegisterUserRoutes(mux *http.ServeMux, pool *pgxpool.Pool) {
	mux.HandleFunc("/getUserData", middleware.CheckJwtToken(httpcallers.CallGetUser(pool)))
}
