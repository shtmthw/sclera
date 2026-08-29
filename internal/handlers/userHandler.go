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
	var user models.User

	err := pool.QueryRow(ctx,
		`SELECT name, email, age, password, favouriteTopics FROM sclera.users WHERE id = $1`, userID).Scan(&user.Name, &user.Email, &user.Age, &user.Password, &user.FavouriteTopics)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			log.Println("No user with this id exists in database")
			return false, models.User{}, ErrNoUserFound
		}

		log.Println("Error occured in GetUserData while querying in users table")
		return false, models.User{}, err
	}

	log.Printf("Succesfully fetched user %s", user.Name)
	return true, user, nil

}

func HandlePushUserData(ctx context.Context, pool *pgxpool.Pool, userdata models.User) (bool, int, error) {
	var newID int
	err := pool.QueryRow(ctx,
		`INSERT INTO sclera.users (name , email, age, password, favouriteTopics) 
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (email) DO NOTHING
		RETURNING id`,
		userdata.Name, userdata.Email, userdata.Age, userdata.Password, userdata.FavouriteTopics,
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

func HandleVerifyUserData(ctx context.Context, pool *pgxpool.Pool, email string) (bool, models.LoggedUser, error) {
	var userData models.LoggedUser

	err := pool.QueryRow(ctx,
		`SELECT id, password FROM sclera.users WHERE email = $1`, email).Scan(&userData.Id, &userData.Password) //the hashed password being saved into the userData struct

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			log.Println("No user with this email exists")
			return false, models.LoggedUser{}, ErrNoUserFound
		}
		log.Println("An error occured while trying to fetch user data")
		return false, models.LoggedUser{}, err
	}

	log.Println("Successfully fetched user data")
	return true, userData, nil

}

func HandleUserDataDeletion(ctx context.Context, pool *pgxpool.Pool, userID int) (bool, error) {
	commandTag, err := pool.Exec(ctx, `DELETE FROM sclera.users WHERE id = $1`, userID)

	if err != nil {
		log.Println("an error occured whilist looking for user in the database")
		return false, err
	}

	if commandTag.RowsAffected() == 0 {
		log.Println("No user with this ID exist")
		return false, ErrNoUserFound
	}

	log.Println("Successfully deleted user data")
	return true, nil
}

func HandleHashedPasswordFetch(ctx context.Context, pool *pgxpool.Pool, userID int) (bool, string, error) {
	var password string
	err := pool.QueryRow(ctx, `SELECT password FROM sclera.users WHERE id = $1`, userID).Scan(&password)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			log.Println("No user with this id exists")
			return false, "", ErrNoUserFound
		}
		log.Println("An error occured while trying to fetch users hashed password")
		return false, "", err
	}

	log.Println("Successfully fetched users hashed password")
	return true, password, nil

}

func HandleUserDataUpdating(ctx context.Context, pool *pgxpool.Pool, userData models.UpdatedUser) (bool, error) {
	commandTag, err := pool.Exec(ctx, `UPDATE sclera.users SET name = $2, favouriteTopics = $3 WHERE id = $1;`, userData.Id, userData.Name, userData.FavouriteTopics)

	if err != nil {
		log.Println("an error occured whilist looking for user in the database")
		return false, err
	}

	if commandTag.RowsAffected() == 0 {
		log.Println("No user with this ID exist")
		return false, ErrNoUserFound
	}

	return true, nil

}
