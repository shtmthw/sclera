package main

import (
	"context"
	"fmt"
	"log"

	"github.com/joho/godotenv"
	"github.com/mattthew/sclera/internal/db"
	"github.com/mattthew/sclera/internal/server"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, relying on real environment variables")
	}

	ctx := context.Background()

	pool, err := db.ConnectDatabase(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	server.RunServer(pool)

	fmt.Println("Sclera server starting...")
}

//add "/" and improvise a postgre schema to hold the users table and data.
