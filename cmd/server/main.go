package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/mattthew/sclera/internal/db"
	"github.com/mattthew/sclera/internal/server"
	"github.com/redis/go-redis/v9"
	"github.com/resend/resend-go/v3"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, relying on real environment variables")
	}

	ctx := context.Background()

	var redisClient = redis.NewClient(&redis.Options{
		Addr: os.Getenv("REDIS_URL"),
	})

	defer func() {
		deferErr := redisClient.Close()
		if deferErr != nil {
			log.Println("an error occured whilist trying to close the redis server")
		}

	}()

	fmt.Println("redis server connection prepared")

	resendApiKey := os.Getenv("RESEND_API_KEY")

	resendClient := resend.NewClient(resendApiKey)

	pool, err := db.ConnectDatabase(ctx)
	if err != nil {
		log.Fatal(err)
	}

	defer pool.Close()

	server.RunServer(pool, redisClient, resendClient)

	fmt.Println("Sclera server starting...")
}

//add "/" and improvise a postgre schema to hold the users table and data.
