package db

import (
	"context"
	"errors"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

func ConnectDatabase(ctx context.Context) (*pgxpool.Pool, error) {
	connString := os.Getenv("DATABASE_URL")
	if connString == "" {
		return nil, errors.New("DATABASE_URL is not set")
	}
	pool, err := pgxpool.New(context.Background(), connString)

	if err != nil {
		log.Println("Failed connecting of pool")
		return nil, err
	}
	// a random change
	if err := pool.Ping(ctx); err != nil {

		log.Println("Database is inactive")
		return nil, err
	}

	log.Println("Database is active")
	return pool, nil
}
