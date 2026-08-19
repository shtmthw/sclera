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
	"github.com/mattthew/sclera/internal/middleware"
)

// random comment to run ci pipeline

func throwHTTPErrAndLog(logText string, logErr error, errorText string, w http.ResponseWriter, httpStat int) {
	if logErr != nil {
		log.Println(logText, logErr)
	} else {
		log.Println(logText)
	}

	http.Error(w, errorText, httpStat)
}

func CallGetUser(pool *pgxpool.Pool) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		userID, ok := ctx.Value(middleware.UserIDkey).(int)
		if !ok {
			throwHTTPErrAndLog("No user user id found in the context values", nil, "No user ID has been found linked to your account", w, http.StatusBadRequest)
			return
		}
		stat, user, err := handlers.HandleGetUserData(ctx, pool, userID) // send the user context, database pool and id to check existance

		w.Header().Set("Content-Type", "application/json")

		if err != nil {
			if errors.Is(err, handlers.ErrNoUserFound) {
				throwHTTPErrAndLog("No user with this id exists in the database, err:", handlers.ErrNoUserFound, "The id has not been used to create an user.", w, http.StatusBadRequest)
				return
			}
			throwHTTPErrAndLog("an error occured while getting user data, err: ", err, "error finding user", w, http.StatusInternalServerError)
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
			throwHTTPErrAndLog("error occured while trying to send json to header, error: ", jsonErr, "Your token was not successfully send to the header.", w, http.StatusInternalServerError)
			return
		}

		_, writeErr := w.Write([]byte(response.String()))

		// must be changed when frontend is added to properly show the userdata struct to the user
		// this is temporary just to see the user in the html servers page

		if writeErr != nil {
			throwHTTPErrAndLog("error occured while trying to write user's data to response, error: ", writeErr, "Your profile was not successfully retrieved.", w, http.StatusInternalServerError)
			return
		}

	}
}

func CallCreateUser() {

	// to be added
}
