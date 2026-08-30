package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/redis/go-redis/v9"

	"github.com/joho/godotenv"
	"github.com/mattthew/sclera/internal/db"
	"github.com/mattthew/sclera/internal/server"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, relying on real environment variables")
	}

	ctx := context.Background()

	var redisClient = redis.NewClient(&redis.Options{
		Addr: os.Getenv("REDIS_URL"),
	})

	defer redisClient.Close()

	fmt.Println("redis server connection prepared")

	pool, err := db.ConnectDatabase(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	server.RunServer(pool, redisClient)

	fmt.Println("Sclera server starting...")
}

//add "/" and improvise a postgre schema to hold the users table and data.
