package middleware

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/mattthew/sclera/internal/authentication"
)

type contextKey string

//type of the contextKey sent to handler

const UserIDkey contextKey = "userID"

//the key being assigned to that type so that future collision between keys stored into context doesnt occurr

func CheckJwtToken(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		tokenString := r.Header.Get("Authorization")
		if tokenString == "" {

			w.WriteHeader(http.StatusUnauthorized)
			jsonErr := json.NewEncoder(w).Encode(map[string]string{"error": "missing authorization token"})

			//this is fucking ridicolous
			if jsonErr != nil {
				log.Println("error writing unauthorized response:", jsonErr)
			}

			return
		}
		if !strings.HasPrefix(tokenString, "Bearer ") {
			prefixErr := json.NewEncoder(w).Encode(map[string]string{
				"error": "invalid prefix",
			})
			log.Println("wrong prefix provided:", prefixErr)
			w.WriteHeader(http.StatusBadRequest)
			return

		}
		tokenString = tokenString[len("Bearer "):]

		userID, err := authentication.VerifyToken(tokenString)

		if err != nil {
			writeErr := json.NewEncoder(w).Encode(map[string]string{
				"error": "invalid token",
			})

			//this is fucking ridicolous too..
			if writeErr != nil {
				log.Println("error writing unauthorized response:", writeErr)
			}

			log.Println("error in jwt auth, err: ", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		UserIDContext := context.WithValue(r.Context(), UserIDkey, userID) //takes the current http.Requests context
		//and adsd the userIDkey as the key using the userID as the value and genarates a new context holding the old contexts data
		//and a new key value

		next(w, r.WithContext(UserIDContext)) //then passes it to the handlers new http.Request (that been made becuase of r.WithContext)
		//with the new contex and the handler then also gets the access of the added value, also remember the request is new but derived
		//meaning it still has the properties and metadata of the original request
	}

}
