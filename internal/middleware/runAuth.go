package middleware

import (
	"fmt"
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
			fmt.Fprint(w, "Missing authorization header")
			return
		}
		tokenString = tokenString[len("Bearer "):]

		err := authentication.VerifyToken(tokenString)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			fmt.Fprint(w, "Invalid token")
			log.Println("error in jwt auth, err: ", err)
			return
		}

		next(w, r)
	}

}
