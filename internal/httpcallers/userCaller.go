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

func CallGetUser(pool *pgxpool.Pool) http.HandlerFunc {
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
var parseLoginAccountTemp = template.Must(template.ParseFiles("userHandling/loginAccount.html"))

func loadTemplateAndHandleTokenEdgeCase(w http.ResponseWriter, r *http.Request, template *template.Template) {

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

			// Write the Set-Cookie header to the response
			// will not cause a superflous error as this is a declaration not a proccesion
			http.SetCookie(w, c)

			// this is the close http contact or a preccesion call
			http.Redirect(w, r, "/loginUser", http.StatusSeeOther)
			return
		}

		//if the token do pass the verification
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusForbidden)

		_, writeErr := w.Write([]byte("You have arleady logged in."))

		if writeErr != nil {
			throwHTTPErrAndLog("error while trying to write response", writeErr, "An error occured while trying to write the response", w, http.StatusInternalServerError)
			return
		}

		return
	}

	tempParseErrr := template.Execute(w, nil)

	if tempParseErrr != nil {
		throwHTTPErrAndLog("failed to render signup template", tempParseErrr, "Internal server error", w, http.StatusInternalServerError)
		return
	}

}

func CallNewUser() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		loadTemplateAndHandleTokenEdgeCase(w, r, parseNewAccountTemp)
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
		hash, hashErr := hashing.HashPassword(strings.TrimSpace(r.FormValue("password")))

		if hashErr != nil {
			throwHTTPErrAndLog("failed hashing the password", hashErr, "Your password was not successfully hashed by the server", w, http.StatusInternalServerError)
			return
		}

		//submitting the hash into the database
		userData.Password = hash

		//check if the values in that slice is under the max limit of 5 or no
		if len(r.Form["items"]) > 5 || len(r.Form["items"]) == 0 {
			throwHTTPErrAndLog("too many or no topics selected", nil, "Select no more than 5 maxium topics and no less than 1.", w, http.StatusBadRequest)
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
			MaxAge:   7 * 24 * 60 * 60,
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
			Secure:   false, // Set to true in production over HTTPS
		}

		http.SetCookie(w, authCookie)

		//one sucessfully OR unsucessfully checked and ran, make sure to redirect the user to either /newAccount (if fails) or /home (if passes) after letting him know the result
		http.Redirect(w, r, "/", http.StatusSeeOther)

	}

}

// shows the login html after verifying that user is not logged in
// same logic as CallNewUser
func CallLoginUser() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		loadTemplateAndHandleTokenEdgeCase(w, r, parseLoginAccountTemp)

	}
}

// process the html form data and return accordingly
func CallVerifyUser(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		if r.Method != http.MethodPost {
			throwHTTPErrAndLog("unauthorized method!", nil, "The method you are trying to execute is UNAUTHORIZED", w, http.StatusMethodNotAllowed)
			return
		}

		parseErr := r.ParseForm()

		if parseErr != nil {
			throwHTTPErrAndLog("failed parsing the html form", parseErr, "An error occured while trying to parse the html form", w, http.StatusInternalServerError)
			return
		}

		email := strings.ToLower(strings.TrimSpace(r.FormValue("email")))
		password := strings.TrimSpace(r.FormValue("password"))

		_, userData, err := handlers.HandleVerifyUserData(ctx, pool, email)

		if err != nil {
			if errors.Is(err, handlers.ErrNoUserFound) {
				throwHTTPErrAndLog("No user with this email exists in database", handlers.ErrNoUserFound, "The email provided is INVALID", w, http.StatusBadRequest)
				return
			}
			throwHTTPErrAndLog("An error occured while trying to fetch data from db", err, "An error occured while trying to fetch your data form the database", w, http.StatusInternalServerError)
			return
		}

		success := hashing.VerifyPassword(password, userData.Password) // comparing the user given pass with the hashed pass intially created upon accoutn creation

		if !success {
			throwHTTPErrAndLog("The password provided is incorrect", nil, "The password is INCORRECT", w, http.StatusBadRequest)
			return
		}

		tokenString, err := authentication.CreateToken(userData.Id)

		if err != nil {
			throwHTTPErrAndLog("Error occured while making an JWT token using the created users ID", err, "Error occured while creating an token for you.", w, http.StatusInternalServerError)
			return
		}

		fullCookieValue := "Bearer " + tokenString

		authCookie := &http.Cookie{
			Name:     "Authorization", // Legal name string
			Value:    fullCookieValue, // Value becomes: "Bearer 752059293"
			Path:     "/",             // Available on all endpoints
			MaxAge:   7 * 24 * 60 * 60,
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
			Secure:   false, // Set to true in production over HTTPS
		}

		http.SetCookie(w, authCookie)
		http.Redirect(w, r, "/getUserData", http.StatusSeeOther)

	}
}

// put this thru the middleware as user cant logout if hes not logged in at the first place, same for the /deleteAccount api
func LogoutUser() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, err := r.Cookie("Authorization")
		if err != nil {
			log.Println("", err)
			throwHTTPErrAndLog("An error occured whilist fetching the cookie form users request, err: ", err, "Error occured while fetching your Auth cookie.", w, http.StatusInternalServerError)
			//always throws an http.Error
			return
			//why do i need a return here
			//ans: return statements are preciesly there to prematurely end a function
		}

		instantCookieDeletion := &http.Cookie{
			Name:     "Authorization",
			Value:    "",              // Clear the value
			Path:     "/",             // Must match the original path
			MaxAge:   -1,              // Signals immediate deletion
			Expires:  time.Unix(0, 0), // Backward compatibility for older browsers
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
			Secure:   false,
		}

		http.SetCookie(w, instantCookieDeletion)
		http.Redirect(w, r, "/", http.StatusSeeOther)
		//why do i not need a return here
		//ans: because this is the natural end of this function
	}
}

// call a database look up the uses existance and perform the deletion of both the user data and cookie
func DeleteUser(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		userID := ctx.Value(middleware.UserIDkey).(int)
		_, err := handlers.HandleUserDataDeletion(ctx, pool, userID)
		// maybe someday ill need the stat bool in use :((

		if err != nil {
			if errors.Is(err, handlers.ErrNoUserFound) {
				throwHTTPErrAndLog("No user with this ID exists in database", handlers.ErrNoUserFound, "The ID provided is INVALID", w, http.StatusBadRequest)
				return
			}
			throwHTTPErrAndLog("An error occured while deleting user", handlers.ErrNoUserFound, "An error occured trying to delete your account", w, http.StatusInternalServerError)
			return
		}

		_, cookieError := r.Cookie("Authorization")
		if cookieError != nil {
			log.Println("", err)
			throwHTTPErrAndLog("An error occured whilist fetching the cookie form users request, err: ", err, "Error occured while fetching your Auth cookie.", w, http.StatusInternalServerError)
			return
		}
		instantCookieDeletion := &http.Cookie{
			Name:     "Authorization",
			Value:    "",              // Clear the value
			Path:     "/",             // Must match the original path
			MaxAge:   -1,              // Signals immediate deletion
			Expires:  time.Unix(0, 0), // Backward compatibility for older browsers
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
			Secure:   false,
		}

		http.SetCookie(w, instantCookieDeletion)
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

//add these upper funcs to the mux http router
