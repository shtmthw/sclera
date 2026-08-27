package middleware

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/mattthew/sclera/internal/authentication"
)

type contextKey string

//type of the contextKey sent to handler

const UserIDkey contextKey = "userID"

//the key being assigned to that type so that future collision between keys stored into context doesnt occurr

func CheckJwtToken(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		cookie, err := r.Cookie("Authorization")
		if err != nil {

			w.WriteHeader(http.StatusUnauthorized)
			jsonErr := json.NewEncoder(w).Encode(map[string]string{"error": "missing authorization token"})

			//this is fucking ridicolous
			if jsonErr != nil {
				log.Println("error writing unauthorized response:", jsonErr)
			}

			return
		}

		tokenString := cookie.Value

		if !strings.HasPrefix(tokenString, "Bearer ") {
			w.WriteHeader(http.StatusBadRequest)

			prefixErr := json.NewEncoder(w).Encode(map[string]string{
				"error": "invalid prefix",
			})
			log.Println("wrong prefix provided:", prefixErr)
			return

		}
		tokenString = strings.TrimPrefix(tokenString, "Bearer ")

		userID, err := authentication.VerifyToken(tokenString)

		if err != nil {
			c := &http.Cookie{
				Name:     "Authorization",
				Value:    "",              // Clear the value
				Path:     "/",             // Must match the original path
				MaxAge:   -1,              // Signals immediate deletion
				Expires:  time.Unix(0, 0), // Backward compatibility for older browsers
				HttpOnly: true,
				SameSite: http.SameSiteLaxMode,
				Secure:   false,
			}
			log.Println("error in jwt auth, err: ", err)

			http.SetCookie(w, c)
			w.WriteHeader(http.StatusUnauthorized)

			writeErr := json.NewEncoder(w).Encode(map[string]string{
				"error": "invalid token, please log/sign in again",
			})

			//this is fucking ridicolous too..
			if writeErr != nil {
				log.Println("error writing unauthorized response:", writeErr)
			}

			return
		}

		UserIDContext := context.WithValue(r.Context(), UserIDkey, userID) //takes the current http.Requests context
		//and adsd the userIDkey as the key using the userID as the value and genarates a new context holding the old contexts data
		//and a new key value
		newReq := r.WithContext(UserIDContext)
		next(w, newReq) //then passes it to the handlers new http.Request (that been made becuase of r.WithContext)
		//with the new contex and the handler then also gets the access of the added value, also remember the request is new but derived
		//meaning it still has the properties and metadata of the original request
	}

}
