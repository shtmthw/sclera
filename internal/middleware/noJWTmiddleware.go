package middleware

import (
	"errors"
	"log"
	"net"
	"net/http"

	"github.com/mattthew/sclera/internal/authentication"
	"github.com/mattthew/sclera/internal/redisInternal"
	helpers "github.com/mattthew/sclera/internal/usefulHelpers"
	"github.com/redis/go-redis/v9"
)

// WithIPRateLimit wraps a NON-middleware (pre-auth) handler so its requests
// are bucketed against the caller's client IP via redisInternal.Allow().
// Authenticated endpoints go through middleware.CheckJwtToken instead, which
// buckets by verified userID. cost comes from the redisInternal token constants.
func CheckUserIPandVerify(next http.HandlerFunc, redisClient *redis.Client, trustedProxyNet *net.IPNet, cost float64) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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
		allowed, remainingTokens, err := redisInternal.Allow(r.Context(), redisClient, clientIP, cost)

		if err != nil {
			if errors.Is(err, redisInternal.ErrUnexpectedScriptError) {
				helpers.ThrowHTTPErrAndLog("rate limiter script error", redisInternal.ErrUnexpectedScriptError, "internal server error", w, http.StatusInternalServerError)
				return
			}
			helpers.ThrowHTTPErrAndLog("error allowing request: ", err, "internal server error", w, http.StatusInternalServerError)
			return
		}

		if !allowed {
			helpers.ThrowHTTPErrAndLog("too many requests from this IP", nil, "too many requests", w, http.StatusTooManyRequests)
			return
		}

		log.Println("remainingTokens for IP ", clientIP, ": ", remainingTokens)

		next(w, r)
	}
}
