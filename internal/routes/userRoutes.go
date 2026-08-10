package routes

import (
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mattthew/sclera/internal/httpcallers"
)

func RegisterRoutes(mux *http.ServeMux, pool *pgxpool.Pool) {
	mux.HandleFunc("/getUserData", httpcallers.CallGetUser(pool))
}
