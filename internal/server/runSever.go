package server

import (
	"log"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mattthew/sclera/internal/routes"
	"github.com/redis/go-redis/v9"
	"github.com/resend/resend-go/v3"
)

func newServer(mux http.Handler) *http.Server {

	//takes the handler and assigns the server with it.
	httpServer := http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	pointerHttpServer := &httpServer

	return pointerHttpServer
}

func RunServer(pool *pgxpool.Pool, redisClient *redis.Client, resendClient *resend.Client) {
	//creates the router
	mux := http.NewServeMux()

	//assigns the mux router/ServeMux ( has .HandleFunc within it ) to connect the endpoints to the handlerfuncs
	routes.RegisterUserRoutes(mux, pool, redisClient, resendClient)
	routes.RegisterGemmaRoutes(mux)

	//creates the server and hosts the mux handler to the provided port.
	server := newServer(mux)

	//logs any error when running the server.
	log.Fatal(server.ListenAndServe())
}
