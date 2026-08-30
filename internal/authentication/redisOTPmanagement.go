package authentication

// mock code

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	otpTTD         = 5 * time.Minute
	resendCooldown = 60 * time.Second
	maxAttempts    = 5
	otpLength      = 6
)

// ---------- Helpers ----------

// genarates a random number by making use of rand and pow18 func under this
func generateOTP(length int) (string, error) {
	max := big.NewInt(int64(pow10(length)))
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%0*d", length, n.Int64()), nil
}

func pow10(n int) int {
	result := 1
	for i := 0; i < n; i++ {
		result *= 10
	}

	return result
}

// hashes the newly made randomized plaintext OTP using the sha256 cryptography
func hashOTP(otp string) string {
	sum := sha256.Sum256([]byte(otp))

	//returns the string encoded from the bytes slice
	return hex.EncodeToString(sum[:])
}

// the predefined keys boilerplate
func keys(identifier string) (otpKey, attemptsKey, cooldownKey string) {
	return "otp:code:" + identifier,
		"otp:attempts:" + identifier,
		"otp:cooldown:" + identifier
}

// ---------- Core Logic ----------

// this takes the users id as identifier, gets triggered upon "get otp button" and "resend OTP button"
func CreateOTP(identifier string, ctx context.Context, rdb *redis.Client) (string, error) {
	otpKey, attemptsKey, cooldownKey := keys(identifier) // assiging the users id to the predifined keys

	// check resend cooldown
	exists, err := rdb.Exists(ctx, cooldownKey).Result()
	if err != nil {
		return "", err
	}

	//if the "resend OTP" button has been hit before the 60 sec expiration of coolDownkey passed
	//this is the guard clause preventing use from recreating the OTP multiple times in a short period of time
	if exists == 1 {
		return "", errors.New("please wait before requesting another OTP")
	}

	//create the new plaintext OTP if the guard clause passed
	otp, err := generateOTP(otpLength)
	if err != nil {
		return "", err
	}

	//hash the OTP using the sha256 cryptography method
	hashedOTP := hashOTP(otp)

	//declare the redis database pipeline
	//this will store the key:value pain into the redis in-memory server
	// both of the .Sets make independent key:value pairs

	pipe := rdb.Pipeline()
	pipe.Set(ctx, otpKey, hashedOTP, otpTTD)
	pipe.Set(ctx, cooldownKey, "1", resendCooldown)

	// deletes the previous attemp count upot resending of OTP
	pipe.Del(ctx, attemptsKey)
	if _, err := pipe.Exec(ctx); err != nil {
		return "", err
	}

	// sends the OTP, this also gets sent to user thru a auto mailing service
	return otp, nil
}

// this takes the users provided plaintext OTP and verifies it against the server saved OTP
func VerifyOTP(identifier string, ctx context.Context, inputOTP string, rdb *redis.Client) error {
	otpKey, attemptsKey, _ := keys(identifier) //fetches the keys using the users ID

	// this gets the users saved OTP form the redis in-mem server, if the OTP has already expired the user faces the error
	storedHash, err := rdb.Get(ctx, otpKey).Result()
	if errors.Is(err, redis.Nil) {
		return errors.New("OTP expired or not found, please request a new one")
	} else if err != nil {
		return err
	}

	// if the OTP does exist then the user gets assigned a new key:value incremental that also gets stored in the redis server
	// on each /VerifyOTP hit the the attepmts get incremented by 1
	attempts, err := rdb.Incr(ctx, attemptsKey).Result()
	if err != nil {
		return err
	}

	// as soon as the user hits /VerifyOTP the attempts gets incremented to 1 and gets attached an expiry date to it
	// ths expiry date is there so that after long time of not resending or verifying the attempts get deleted
	if attempts == 1 {
		rdb.Expire(ctx, attemptsKey, otpTTD)
	}

	//if the user attempts to verify gets over the maxAttempts amount dont let the user the OTP and tell to resend it
	if attempts > maxAttempts {
		rdb.Del(ctx, otpKey)
		return errors.New("too many attempts, please request a new OTP")
	}

	// takes the user send plaintext OTP and hasesh it so that it can be checked against the server saved OTP
	inputHash := hashOTP(inputOTP)

	//compares the OTP saved into redises in-memory storage with the newly made hashedOTP that userprovided
	if subtle.ConstantTimeCompare([]byte(storedHash), []byte(inputHash)) != 1 {
		return fmt.Errorf("invalid OTP, %d attempt(s) left", maxAttempts-attempts)
	}

	// success — one-time use
	rdb.Del(ctx, otpKey, attemptsKey)
	return nil
}
