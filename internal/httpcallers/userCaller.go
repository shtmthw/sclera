package httpcallers

import (
	"encoding/json"
	"errors"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mattthew/sclera/internal/authentication"
	"github.com/mattthew/sclera/internal/handlers"
	"github.com/mattthew/sclera/internal/hashing"
	"github.com/mattthew/sclera/internal/middleware"
	"github.com/mattthew/sclera/internal/models"
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

		userID, ok := ctx.Value(middleware.UserIDkey).(int) //this .(int) is NOT typecasting the value of type any, it is there to TYPECHECK what the actual type of the type any value is
		if !ok {
			throwHTTPErrAndLog("No user user id found in the context value", nil, "No user ID has been found linked to your account", w, http.StatusBadRequest)
			return
		}
		stat, user, err := handlers.HandleGetUserData(ctx, pool, userID) // send the user context, database pool and id to check existance

		if err != nil {
			if errors.Is(err, handlers.ErrNoUserFound) {
				throwHTTPErrAndLog("No user with this id exists in the database, err:", handlers.ErrNoUserFound, "The id has not been used to create an user.", w, http.StatusBadRequest)
				return
			}
			throwHTTPErrAndLog("an error occured while getting user data, err: ", err, "error finding user", w, http.StatusInternalServerError)
			return
		}

		log.Println("user data has been fetched, stat: ", stat)

		var logSB strings.Builder
		for _, topic := range user.FavouriteTopics {
			logSB.WriteString("[")
			logSB.WriteString(topic)
			logSB.WriteString("] ")
		}

		log.Printf("User retrieved: Name: %s, Topics: %s\n", user.Name, logSB.String())

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		jsonErr := json.NewEncoder(w).Encode(user)

		if jsonErr != nil {
			throwHTTPErrAndLog("error occured while trying to send json to header, error: ", jsonErr, "Your token was not successfully send to the header.", w, http.StatusInternalServerError)
			return
		}

	}
}

//this displays the html file to user when he enters /newUser and fills up the form then onclick sends the get request to /createUser happens

// features
// must check if user is logged in or no
// must make sure the form is provided with proper info or no
// the form info then gets written to the users request body then passed onto /createUser

// make sure to preload this SOMEHOW, but this is NOT GOOD FOR OPTIMIZATION
var parseNewAccountTemp = template.Must(template.ParseFiles("userHandling/newAccount.html"))

func CallNewUser() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		//check if user has already made an account and if he did show a diff html result
		//make use one html file is used an dynamically set
		cookie, err := r.Cookie("Authorization")

		//edge case handling and token verification
		if err == nil {
			//if the token doesnt pass the verification, meaning user modified their token themseleves
			//and a token provided by the server will 100% of the time include the "Bearer " infront of it

			tokenString := strings.TrimPrefix(cookie.Value, "Bearer ")

			_, err := authentication.VerifyToken(tokenString)
			if err != nil {
				throwHTTPErrAndLog("provided token is not authorized", err, "The token associated with your account is INVALID", w, http.StatusBadRequest)
				return
			}

			//if the token do pass the verification
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusForbidden)

			_, writeErr := w.Write([]byte("You have arleady logged in."))

			if writeErr != nil {
				throwHTTPErrAndLog("error while trying to write response", writeErr, "An error occured while trying to write the response", w, http.StatusInternalServerError)
				return
			}

			return
		}

		tempParseErrr := parseNewAccountTemp.Execute(w, nil)

		if tempParseErrr != nil {
			throwHTTPErrAndLog("failed to render signup template", tempParseErrr, "Internal server error", w, http.StatusInternalServerError)
			return
		}
	}

}

// this is the func that gets attached to the /createUser api endpoint
// get handler for /newUser
func CallCreateUser(pool *pgxpool.Pool) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		// verify the request
		if r.Method != http.MethodPost {
			throwHTTPErrAndLog("unauthorized method!", nil, "The method you are trying to execute is UNAUTHORIZED", w, http.StatusMethodNotAllowed)
			return
		}

		err := r.ParseForm()

		if err != nil {
			throwHTTPErrAndLog("failed parsing the html form", err, "Your data was not successfully handled by the server", w, http.StatusBadRequest)
			return
		}

		// assign the data to userData struct

		var userData models.User
		userData.Name = strings.TrimSpace(r.FormValue("name"))
		userData.Email = strings.ToLower(strings.TrimSpace(r.FormValue("email"))) // Normalize email casing
		ageStr := r.FormValue("age")

		//typecast the age str into int CAUSE THE FUCKING HTML SENDS type="number" AS STRINGS
		ageInt, err := strconv.Atoi(ageStr)
		if err != nil {
			// This triggers if the input wasn't a valid number string
			http.Error(w, "Invalid number format", http.StatusBadRequest)
			return
		}
		userData.Age = ageInt

		//this is the raw password given by user being turned into an hash
		hash, hashErr := hashing.HashPassword(r.FormValue("password"))

		if hashErr != nil {
			throwHTTPErrAndLog("failed hashing the password", hashErr, "Your password was not successfully hashed by the server", w, http.StatusInternalServerError)
			return
		}

		//submitting the hash into the database
		userData.Password = hash

		favouriteTopics := r.Form["items"]

		//check if the values in that slice is under the max limit of 5 or no
		if len(favouriteTopics) > 5 || len(favouriteTopics) == 0 {
			throwHTTPErrAndLog("too many or no topics selected", err, "Select no more than 5 maxium topics and no less than 1.", w, http.StatusBadRequest)
			return
		}
		// add the slice in the the user struct
		userData.FavouriteTopics = r.Form["items"]

		//run the database work
		_, newID, err := handlers.HandlePushUserData(ctx, pool, userData)

		if err != nil {
			if errors.Is(err, handlers.ErrUserEmailAlreadyTaken) {
				throwHTTPErrAndLog("provided email is already taken", err, "The email you are trying to use is already in use.", w, http.StatusBadRequest)
				return
			}
			throwHTTPErrAndLog("error occured whilist inserting userdata", err, "An error occured while trying to create your account.", w, http.StatusInternalServerError)
			return
		}

		//create an token after the database work is done and add the token to users "Authentication" header
		tokenString, err := authentication.CreateToken(newID)
		if err != nil {
			throwHTTPErrAndLog("Error occured while making an JWT token using the created users ID", err, "Error occured while creating an token for you.", w, http.StatusInternalServerError)
			return
		}
		//adds the token into the cookie
		fullCookieValue := "Bearer " + tokenString

		authCookie := &http.Cookie{
			Name:     "Authorization", // Legal name string
			Value:    fullCookieValue, // Value becomes: "Bearer 752059293"
			Path:     "/",             // Available on all endpoints
			Expires:  time.Now().Add(168 * time.Hour),
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
			Secure:   false, // Set to true in production over HTTPS
		}

		http.SetCookie(w, authCookie)

		//one sucessfully OR unsucessfully checked and ran, make sure to redirect the user to either /newAccount (if fails) or /home (if passes) after letting him know the result
		http.Redirect(w, r, "/", http.StatusSeeOther)

	}

}

//to do
// work on /userLogin
