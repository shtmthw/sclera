package handlers

import (
	"context"
	"errors"
	"log"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mattthew/sclera/internal/models"
)

var ErrNoUserFound = errors.New("no user with this id found in database")
var ErrUserEmailAlreadyTaken = errors.New("the provided email is already attached to another users data")

func HandleGetUserData(ctx context.Context, pool *pgxpool.Pool, userID int) (bool, models.User, error) {
	var userExists int
	var user models.User

	err := pool.QueryRow(ctx,
		`WITH fetched AS (
			SELECT * FROM users WHERE id = $1
		)
		SELECT 
			(SELECT count(*) FROM fetched) as userExists,	
			(SELECT name FROM fetched) as userName,
			(SELECT email FROM fetched) as userEmail,
			(SELECT age FROM fetched) as userAge,
			(SELECT favouriteTopics FROM fetched) as favouriteTopics`, userID).Scan(&userExists, &user.Name, &user.Email, &user.Age, &user.FavouriteTopics)

	if err != nil {
		log.Println("Error occured in GetUserData while querying in users table")
		return false, models.User{}, err
	}

	if userExists == 0 {
		log.Println("No user exists with provided id")
		return false, models.User{}, ErrNoUserFound
	}

	log.Printf("Succesfully fetched user %s", user.Name)
	return true, user, nil

}

func HandlePushUserData(ctx context.Context, pool *pgxpool.Pool, userdata models.User) (bool, int, error) {
	var newID int
	err := pool.QueryRow(ctx,
		`INSERT INTO users (name , email, age, favouriteTopics) 
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (email) DO NOTHING
		RETURNING id`,
		userdata.Name, userdata.Email, userdata.Age, userdata.FavouriteTopics,
	).Scan(&newID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			log.Println("Email is linked to an existing users data")
			return false, 0, ErrUserEmailAlreadyTaken
		}
		log.Println("Error occured in GetUserData while inserting in users table")
		return false, 0, err
	}

	log.Println("Successfully created user account")
	return true, newID, nil

}
