package redisInternal

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9" // official Redis client for Go
)

// Token costs charged per request. A fresh bucket holds 100 tokens and
// refills at ~1 token/min, so these numbers are the budget for each
// category of endpoint:
//   - PageTokenCost: pure HTML page renders (cheap)
//   - FormTokenCost: form processing / account mutations (cheap-ish)
//   - JWTTokenCost:  any JWT-protected request going through the middleware
//   - LLMTokenCost:  expensive LLM inference requests
const (
	PageTokenCost = 5
	FormTokenCost = 10
	JWTTokenCost  = 20
	LLMTokenCost  = 25
)

// Limiter wraps a Redis client and holds the default bucket config.
// You can override capacity/refill per-call if different endpoints
// need different budgets (e.g. LLM call vs cheap landing-page hit).

// tokenBucketScript is a Lua script executed ATOMICALLY inside Redis.
// This atomicity is the whole point: token bucket needs a
// read-modify-write (check tokens, maybe subtract, write back) and if
// that happened as separate GET/SET calls from Go, two concurrent
// requests from the same user could both read "5 tokens left" before
// either writes back the decrement — a classic race condition that
// would let more requests through than the bucket allows.
// Running it as a single Lua script means Redis executes the whole
// thing as one indivisible step, no race possible.
var tokenBucketScript = redis.NewScript(`
-- KEYS[1] = the Redis key for this identity (e.g. "ratelimit:user:123")
-- ARGV[1] = bucket capacity (max tokens the bucket can ever hold)
-- ARGV[2] = refill rate, in tokens per second
-- ARGV[3] = current unix timestamp (float, seconds) — passed in from Go
--           rather than using Redis TIME, so behavior is deterministic
--           and testable, and so all your app servers agree on "now"
--           even if their local clocks drift slightly (Redis is the
--           single source of truth for time here)
-- ARGV[4] = cost of a request in tokens (usually 1, but lets you
--           charge more for expensive operations like an LLM call)

local key = KEYS[1]
local capacity = tonumber(ARGV[1])
local refill_rate = tonumber(ARGV[2])
local now = tonumber(ARGV[3])
local cost = tonumber(ARGV[4])

-- Fetch the bucket's current state. HMGET returns nil for fields that
-- don't exist yet (i.e. this is the very first request from this key).
local bucket = redis.call("HMGET", key, "maxTokens", "last_refill")
local maxTokens = tonumber(bucket[1])
local last_refill = tonumber(bucket[2])

-- First-ever request for this key: initialize a full bucket.
-- This means a brand new user starts with full burst allowance,
-- which is the normal/expected token-bucket behavior.
if maxTokens == nil then
    maxTokens = capacity
    last_refill = now
end

-- Compute how much time has passed since we last touched this bucket,
-- and how many tokens should have regenerated in that time.
-- This "lazy refill" approach means we don't need a background job
-- ticking every bucket every second — we just calculate the correct
-- refill amount on-demand, whenever a request actually arrives.

local elapsed = math.max(0, now - last_refill)
local refill_amount = elapsed * refill_rate


-- Add the refilled tokens, but never exceed capacity (a bucket can't
-- overflow — excess refill is simply wasted, same as a real bucket).
maxTokens = math.min(capacity, maxTokens + refill_amount)


local allowed = 0
if maxTokens >= cost then
    -- Enough tokens available: consume them and allow the request.
    maxTokens = maxTokens - cost
    allowed = 1
end

-- Persist the updated state back to Redis, whether or not the
-- request was allowed — we still record refill progress either way,
-- so a denied request doesn't cause us to "lose" refill time.
redis.call("HMSET", key, "maxTokens", tostring(maxTokens), "last_refill", tostring(now))

-- Set an expiry on the key so buckets for inactive users don't sit in
-- Redis forever. Generous TTL (e.g. time to fully refill from empty,
-- doubled) so an active user's bucket never expires mid-use, but an
-- abandoned one eventually gets cleaned up automatically.
local ttl = math.ceil((capacity / refill_rate) * 2)
redis.call("EXPIRE", key, ttl)

-- Return: [allowed (1/0), tokens remaining after this request]
-- so the Go side can decide what to do and optionally report
-- remaining budget back to the client (e.g. in a header).
return {allowed, tostring(maxTokens)}
`)

// Allow checks whether a request identified by `key` is permitted
// under a token bucket with the given capacity (max burst size) and
// refillRate (tokens regenerated per second). cost lets you charge
// more than 1 token for expensive operations.
//
// Example: Allow(ctx, "user:123", 10, 0.5, 1)
//
//	-> bucket holds up to 10 tokens, refills at 0.5 tokens/sec
//	   (i.e. fully refills from empty in 20 seconds), this request
//	   costs 1 token.
var ErrUnexpectedScriptError = errors.New("unexpected rate limiter script, check server log")

func Allow(ctx context.Context, redisClient *redis.Client, key string, cost float64) (allowed bool, remaining float64, err error) {
	// Prefix the key so rate-limit keys can never collide with other
	// data you might store in the same Redis instance, and so you can
	// easily scan/inspect/flush just rate-limit keys later.
	fullKey := fmt.Sprintf("ratelimit:user:%s", key)

	var refillRate float32 = 0.01667
	capacity := 100
	// now is computed in Go and passed in as an argument (see ARGV[3]
	// above) rather than letting the Lua script call a Redis time
	// function, so the whole system agrees on one clock source.
	now := float64(time.Now().UnixNano()) / 1e9
	// Run the script. redis.Script.Run automatically handles the
	// EVALSHA/EVAL fallback dance (tries the cached script hash first,
	// falls back to sending the full script if Redis doesn't have it
	// cached yet) — you don't need to manage that yourself.
	res, err := tokenBucketScript.Run(ctx, redisClient, []string{fullKey},
		capacity, refillRate, now, cost,
	).Result()
	if err != nil {
		// Fail-closed vs fail-open is a real decision, not a default:
		// returning an error here and letting the CALLER decide
		// (reject the request, or let it through) means you can make
		// that call per-endpoint. For expensive backend calls you
		// likely want fail-closed (reject on Redis error, protect
		// your compute); for cheap/non-critical endpoints you might
		// fail-open (let it through, don't let a Redis blip take down
		// your whole app for real users).
		return false, 0, fmt.Errorf("rate limiter redis error: %w", err)
	}

	// The script returns a 2-element array: [allowed, remainingTokens]
	resSlice, ok := res.([]interface{})
	if !ok || len(resSlice) != 2 {
		log.Printf("unexpected rate limiter script result: %v", res)
		return false, 0, ErrUnexpectedScriptError
	}

	allowedInt, _ := resSlice[0].(int64)
	var remainingTokens float64
	if _, err := fmt.Sscanf(resSlice[1].(string), "%f", &remainingTokens); err != nil {
		return false, 0, fmt.Errorf("rate limiter script returned an unparsable balance: %w", err)
	}

	return allowedInt == 1, remainingTokens, nil
}

// --- Example usage sketches (not wired up, just illustrating call sites) ---

// Middleware use (handlerFunc3-style, userID guaranteed present):
//
// func RateLimitMiddleware(l *Limiter) func(http.Handler) http.Handler {
// 	return func(next http.Handler) http.Handler {
// 		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 			userID := r.Context().Value(userIDKey).(string)
// 			allowed, remaining, err := l.Allow(r.Context(), "user:"+userID, 20, 1.0, 1)
// 			if err != nil {
// 				http.Error(w, "internal error", http.StatusInternalServerError)
// 				return
// 			}
// 			if !allowed {
// 				w.Header().Set("Retry-After", "5")
// 				http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
// 				return
// 			}
// 			_ = remaining // could set as a response header for client visibility
// 			next.ServeHTTP(w, r)
// 		})
// 	}
// }

// Standalone handler use (handlerFunc1/2-style, optional auth):
//
// func HandlerFunc1(l *Limiter) http.HandlerFunc {
// 	return func(w http.ResponseWriter, r *http.Request) {
// 		key := deriveKey(r) // "user:<id>" if authenticated, else "ip:<addr>"
// 		allowed, _, err := l.Allow(r.Context(), key, 10, 0.5, 1)
// 		if err != nil || !allowed {
// 			http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
// 			return
// 		}
// 		// ... actual handler logic
// 	}
// }
