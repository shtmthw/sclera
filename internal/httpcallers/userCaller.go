package httpcallers

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mattthew/sclera/internal/handlers"
)

func CallGetUser(pool *pgxpool.Pool) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		email := r.URL.Query().Get("email")
		stat, user, err := handlers.HandleGetUserData(ctx, pool, email)

		w.Header().Set("Content-Type", "application/json")

		if err != nil {
			log.Println("an error occured while getting user data, err: ", err)
			http.Error(w, "error finding user", http.StatusInternalServerError)
			return
		}
		if errors.Is(err, handlers.ErrNoUserFound) {

			log.Println("No user with this exists in the database, stat: ", stat)
			http.Error(w, "The email id has not been used to create an user.", http.StatusBadRequest)
			return
		}
		var response strings.Builder
		fmt.Fprintf(&response, "Username: %s, Email: %s\n", user.Name, user.Email)
		for _, topic := range user.FavouriteTopics {
			fmt.Fprintf(&response, "- %s\n", topic)
		}

		log.Println("user data has been fetched, stat: ", stat)

		jsonErr := json.NewEncoder(w).Encode(user)

		if jsonErr != nil {
			log.Println("error occured while trying to send json to header, error: ", jsonErr)
			return
		}

		_, writeErr := w.Write([]byte(response.String()))

		if writeErr != nil {
			log.Println("error occured while trying to write user's data to response, error: ", writeErr)
			return
		}

	}
}
