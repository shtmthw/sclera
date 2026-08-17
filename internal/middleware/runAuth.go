package middleware

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/mattthew/sclera/internal/authentication"
)

func CheckJwtToken(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		tokenString := r.Header.Get("Authorization")
		if tokenString == "" {

			w.WriteHeader(http.StatusUnauthorized)
			jsonErr := json.NewEncoder(w).Encode(map[string]string{"error": "missing authorization token (congrats your cd now works)"})

			//this is fucking ridicolous
			if jsonErr != nil {
				log.Println("error writing unauthorized response:", jsonErr)
			}
			return
		}
		tokenString = tokenString[len("Bearer "):]

		err := authentication.VerifyToken(tokenString)
		if err != nil {
			writeErr := json.NewEncoder(w).Encode(map[string]string{
				"error": "invalid token",
			})

			//this is fucking ridicolous too..
			if writeErr != nil {
				log.Println("error writing unauthorized response:", writeErr)
			}

			log.Println("error in jwt auth, err: ", err)
			return
		}

		next(w, r)
	}

}
