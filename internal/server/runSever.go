package server

import (
	"log"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mattthew/sclera/internal/routes"
)

func newServer(mux http.Handler) *http.Server {

	return &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}
}

func RunServer(pool *pgxpool.Pool) {
	mux := http.NewServeMux()

	routes.RegisterRoutes(mux, pool)

	server := newServer(mux)

	log.Fatal(server.ListenAndServe())
}
