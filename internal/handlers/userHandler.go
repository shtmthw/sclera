package handlers

import (
	"context"
	"errors"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mattthew/sclera/internal/models"
)

var NoUserFoundErr = errors.New("no user with this email found in database")

func GetUserData(ctx context.Context, pool *pgxpool.Pool, email string) (bool, models.User, error) {
	var userExists int
	var user models.User

	err := pool.QueryRow(ctx,
		`WITH fetched AS (
			SELECT * FROM users WHERE email = $1
		)
		SELECT 
			(SELECT count(*) FROM fetched) as userExists,	
			(SELECT name FROM fetched) as userName,
			(SELECT email FROM fetched) as userEmail,
			(SELECT age FROM fetched) as userAge,
			(SELECT favouriteTopics FROM fetched) as favouriteTopics`, email).Scan(&userExists, &user.Name, &user.Email, &user.Age, &user.FavouriteTopics)

	if err != nil {
		log.Println("Error occured in GetUserData while querying in users table")
		return false, models.User{}, err
	}

	if userExists == 0 {
		log.Println("No user exists with provided email")
		return false, models.User{}, NoUserFoundErr
	}

	log.Printf("Succesfully fetched user %s", user.Name)
	return true, user, nil

}
