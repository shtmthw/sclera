package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/mattthew/sclera/internal/authentication"
	"github.com/mattthew/sclera/internal/redisInternal"
	"github.com/redis/go-redis/v9" // official Redis client for Go
)

type contextKey string

//type of the contextKey sent to handler

const UserIDkey contextKey = "userID"

//the key being assigned to that type so that future collision between keys stored into context doesnt occurr

func WriteJSONError(w http.ResponseWriter, status int, logMessage string, clientMessage string, logResponseError string) {
	log.Println(logMessage)

	w.WriteHeader(status)

	jsonErr := json.NewEncoder(w).Encode(map[string]string{
		"error": clientMessage,
	})
	if jsonErr != nil {
		log.Println(logResponseError, jsonErr)
	}
}

func clearAuthorizationCookie(w http.ResponseWriter) {
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

	http.SetCookie(w, c)
}

// rateLimitKey picks the Redis identity used for the Allow() bucket:
//   - the caller's client IP when they are NOT authenticated (no
//     Authorization cookie, a bad "Bearer " prefix, or an invalid/expired
//     JWT) — pre-auth traffic is bucketed per-address;
//   - the caller's verified userID once their JWT is legit, so the bucket
//     follows the account instead of the network the user happens to be on.
//
// The token itself is only verified once here; CheckJwtToken relies on the
// result (plus rateKey) instead of re-parsing it.
func rateLimitKey(r *http.Request, clientIP string) (key string, userID int, tokenValid bool) {
	cookie, err := r.Cookie("Authorization")
	if err != nil {
		return clientIP, 0, false
	}

	tokenString := cookie.Value
	if !strings.HasPrefix(tokenString, "Bearer ") {
		return clientIP, 0, false
	}
	tokenString = strings.TrimPrefix(tokenString, "Bearer ")

	uid, err := authentication.VerifyToken(tokenString)
	if err != nil {
		return clientIP, 0, false
	}

	return strconv.Itoa(uid), uid, true
}

func CheckJwtToken(next http.HandlerFunc, trustedProxyNet *net.IPNet, redisClient *redis.Client, cost float64) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		ctx := r.Context()
		clientIP, ipErr := authentication.GetClientIP(r, trustedProxyNet)

		if ipErr != nil {
			log.Println("Error while executing GetClientIP()")

			WriteJSONError(
				w,
				http.StatusForbidden,
				"GetClientIP rejected: untrusted proxy or invalid IP", // detailed, for your logs only
				"forbidden", // vague, safe, sent to client
				"error writing forbidden response:",
			)
			return
		}

		//pick the rate limit identity: client IP while unauthenticated,
		//verified userID once the JWT checks out.
		rateKey, userID, tokenValid := rateLimitKey(r, clientIP)

		//count the request against that identity's token bucket
		allowed, remainingToken, err := redisInternal.Allow(ctx, redisClient, rateKey, cost)

		if err != nil {
			if errors.Is(err, redisInternal.ErrUnexpectedScriptError) {

				log.Println(redisInternal.ErrUnexpectedScriptError)
				WriteJSONError(
					w,
					http.StatusInternalServerError,
					redisInternal.ErrUnexpectedScriptError.Error(),
					"internal server error",
					"error writing internal server error response:",
				)
				return
			}
			WriteJSONError(
				w,
				http.StatusInternalServerError,
				"error allowing request: "+err.Error(),
				"internal server error",
				"error writing internal server error response:",
			)
			return
		}

		if !allowed {
			WriteJSONError(
				w,
				http.StatusTooManyRequests,
				"too many requests",
				"too many requests",
				"error writing too many requests response:",
			)
			return
		}

		log.Println("remainingTokens for userID: ", rateKey, ": ", remainingToken)

		if !tokenValid {
			clearAuthorizationCookie(w)
			log.Println("error in jwt auth, invalid or expired token")

			WriteJSONError(
				w,
				http.StatusUnauthorized,
				"invalid token, please log/sign in again",
				"invalid token, please log/sign in again",
				"error writing unauthorized response:",
			)
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
