package httpcallers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
	"github.com/redis/go-redis/v9"
	"github.com/resend/resend-go/v3"
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

func verifyHTTPMethod(w http.ResponseWriter, r *http.Request, allowedMethod string) bool {
	if r.Method != allowedMethod {
		// Log and throw the error dynamically
		errMsg := fmt.Sprintf("The method %s is UNAUTHORIZED. Expected %s.", r.Method, allowedMethod)
		throwHTTPErrAndLog("unauthorized method!", nil, errMsg, w, http.StatusMethodNotAllowed)
		return false // Validation failed
	}
	return true // Validation passed
}

func CallGetUser(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		stat := verifyHTTPMethod(w, r, http.MethodGet)
		if !stat {
			return
		}
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
var parseUpdateAccountTemp = template.Must(template.ParseFiles("userHandling/updateAccount.html"))
var parseOTPverificationTemp = template.Must(template.ParseFiles("userHandling/OTPverification.html"))
var parseUpdatePasswordTemp = template.Must(template.ParseFiles("userHandling/updatePassword.html"))

func handleTokenEdgeCase(w http.ResponseWriter, r *http.Request) int {
	cookie, err := r.Cookie("Authorization")

	//edge case handling and token verification
	if err == nil {
		//if the token doesnt pass the verification, meaning user modified their token themseleves
		//and a token provided by the server will 100% of the time include the "Bearer " infront of it

		tokenString := strings.TrimPrefix(cookie.Value, "Bearer ")

		_, err := authentication.VerifyToken(tokenString)
		if err != nil {

			// Write the Set-Cookie header to the response
			// will not cause a superflous error as this is a declaration not a proccesion
			instantCookieDeletion(w, "Authorization", "/")

			// this is the close http contact or a preccesion call
			http.Redirect(w, r, "/loginUser", http.StatusSeeOther)
			return 0 // meaning invalid token
		}

		//if the token do pass the verification
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusForbidden)

		_, writeErr := w.Write([]byte("You have arleady logged in."))

		if writeErr != nil {
			throwHTTPErrAndLog("error while trying to write response", writeErr, "An error occured while trying to write the response", w, http.StatusInternalServerError)
			return 0 // internal error
		}

		return 1 // valid token
	}

	return 3 // token doesnt exist
}

// this is only used for /sendVerificationMail api
func handleTokenEdgeCase2(w http.ResponseWriter, r *http.Request) int {
	cookie, err := r.Cookie("Authorization")

	//edge case handling and token verification
	if err == nil {
		//if the token doesnt pass the verification, meaning user modified their token themseleves
		//and a token provided by the server will 100% of the time include the "Bearer " infront of it

		tokenString := strings.TrimPrefix(cookie.Value, "Bearer ")

		_, err := authentication.VerifyToken(tokenString)
		if err != nil {

			// Write the Set-Cookie header to the response
			// will not cause a superflous error as this is a declaration not a proccesion
			instantCookieDeletion(w, "Authorization", "/")

			// this is the close http contact or a preccesion call
			http.Redirect(w, r, "/loginUser", http.StatusSeeOther)
			return 0 // meaning invalid token
		}
		return 1 // valid token
	}

	return 3 // token doesnt exist
}

func loadTemplateAndHandleTokenEdgeCase(w http.ResponseWriter, r *http.Request, template *template.Template) {

	//check if user has already made an account and if he did show a diff html result
	//make use one html file is used an dynamically set
	//handles the token edge cases also

	stat := handleTokenEdgeCase(w, r)

	if stat == 0 || stat == 1 {
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
		stat := verifyHTTPMethod(w, r, http.MethodPost)
		if !stat {
			return
		}
		ctx := r.Context()

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
		stat := verifyHTTPMethod(w, r, http.MethodGet)
		if !stat {
			return
		}
		loadTemplateAndHandleTokenEdgeCase(w, r, parseLoginAccountTemp)

	}
}

// process the html form data and return accordingly
func CallVerifyUser(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		stat := verifyHTTPMethod(w, r, http.MethodPost)
		if !stat {
			return
		}
		ctx := r.Context()

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
func CallLogoutUser() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		stat := verifyHTTPMethod(w, r, http.MethodPost)
		if !stat {
			return
		}

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
func CallDeleteUser(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		stat := verifyHTTPMethod(w, r, http.MethodPost)
		if !stat {
			return
		}
		ctx := r.Context()

		userID, ok := ctx.Value(middleware.UserIDkey).(int) //this .(int) is NOT typecasting the value of type any, it is there to TYPECHECK what the actual type of the type any value is
		if !ok {
			throwHTTPErrAndLog("No user user id found in the context value", nil, "No user ID has been found linked to your account", w, http.StatusBadRequest)
			return
		}
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

// goes thru to middleware to make sure user is authenticated
// loads the http template to the user
type updateAccountPageData struct {
	Name            string
	FavouriteTopics map[string]bool
}

func CallUpdateUserClientSide(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		stat := verifyHTTPMethod(w, r, http.MethodGet)
		if !stat {
			return
		}
		ctx := r.Context()

		//apply error handing later
		userID, _ := ctx.Value(middleware.UserIDkey).(int)

		_, user, err := handlers.HandleGetUserData(ctx, pool, userID)

		if err != nil {
			if errors.Is(err, handlers.ErrNoUserFound) {
				throwHTTPErrAndLog("No user with this id exists in the database, err:", handlers.ErrNoUserFound, "The id has not been used to create an user.", w, http.StatusBadRequest)
				return
			}
			throwHTTPErrAndLog("an error occured while getting user data, err: ", err, "error finding user", w, http.StatusInternalServerError)
			return
		}

		topicsMap := make(map[string]bool, len(user.FavouriteTopics))

		for _, t := range user.FavouriteTopics {
			topicsMap[t] = true
		}

		data := updateAccountPageData{

			Name:            user.Name,
			FavouriteTopics: topicsMap,
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")

		tempParseErrr := parseUpdateAccountTemp.Execute(w, data)

		if tempParseErrr != nil {
			throwHTTPErrAndLog("failed to render template", tempParseErrr, "Internal server error", w, http.StatusInternalServerError)
			return
		}
	}
}

// should make sure the given pass is right
func CallUpdateUserServerSide(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		stat := verifyHTTPMethod(w, r, http.MethodPost)
		if !stat {
			return
		}

		ctx := r.Context()
		userID, ok := ctx.Value(middleware.UserIDkey).(int) //this .(int) is NOT typecasting the value of type any, it is there to TYPECHECK what the actual type of the type any value is
		if !ok {
			throwHTTPErrAndLog("No user user id found in the context value", nil, "No user ID has been found linked to your account", w, http.StatusBadRequest)
			return
		}

		parseErr := r.ParseForm()

		if parseErr != nil {
			throwHTTPErrAndLog("failed parsing the html form", parseErr, "An error occured while trying to parse the html form", w, http.StatusInternalServerError)
			return
		}

		var userData models.UpdatedUser

		//get the hashed pass
		_, hashedPassword, err := handlers.HandleHashedPasswordFetch(ctx, pool, userID)

		if err != nil {
			if errors.Is(err, handlers.ErrNoUserFound) {
				throwHTTPErrAndLog("user with this id does not exists", handlers.ErrNoUserFound, "User ID is INVALID", w, http.StatusBadRequest)
				return
			}
			throwHTTPErrAndLog("An error occcured while fetching users hashed password", err, "An error occured whilist fetching your password", w, http.StatusInternalServerError)
			return
		}

		//get the user give info from the html form
		name := strings.TrimSpace(r.FormValue("name"))

		if name == "" {
			throwHTTPErrAndLog("Name is empty", nil, "Name cannot be empty", w, http.StatusBadRequest)
			return
		}

		userData.Name = name

		unhashedPassword := strings.TrimSpace(r.FormValue("password"))

		success := hashing.VerifyPassword(unhashedPassword, hashedPassword) // comparing the user given pass with the hashed pass intially created upon accoutn creation

		if !success {
			throwHTTPErrAndLog("The password provided is incorrect", nil, "The password is INCORRECT", w, http.StatusBadRequest)
			return
		}

		//check if the values in that slice is under the max limit of 5 or no
		if len(r.Form["items"]) > 5 || len(r.Form["items"]) == 0 {
			throwHTTPErrAndLog("too many or no topics selected", nil, "Select no more than 5 maxium topics and no less than 1.", w, http.StatusBadRequest)
			return
		}
		// add the slice in the the user struct
		userData.FavouriteTopics = r.Form["items"]
		userData.Id = userID

		//send the userData sturct to get updated

		_, userDataUpdatingErr := handlers.HandleUserDataUpdating(ctx, pool, userData)

		if userDataUpdatingErr != nil {
			if errors.Is(userDataUpdatingErr, handlers.ErrNoUserFound) {
				throwHTTPErrAndLog("user with this id does not exists", handlers.ErrNoUserFound, "User ID is INVALID", w, http.StatusBadRequest)
				return
			}
			throwHTTPErrAndLog("An error occcured while updating users account", userDataUpdatingErr, "An error occured whilist updating your account", w, http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

// step 1 of password resetting
// send a verification email to user thru mailing then redirect to step 2 it being /verifyOTP

// making the html email boiler plate
func buildOTPEmailHTML(otp string) string {
	return fmt.Sprintf(`
<div style="font-family: -apple-system, Segoe UI, Roboto, Arial, sans-serif; max-width: 480px; margin: 0 auto; padding: 32px 24px; background-color: #ffffff;">
    
    <!-- Header -->
    <h2 style="color: #1a1a1a; font-size: 20px; margin-bottom: 8px;">Verify your account</h2>
    <p style="color: #555555; font-size: 14px; line-height: 1.5; margin-bottom: 24px;">
        Use the code below to finish verifying your account. This code expires in 5 minutes.
    </p>

    <!-- The OTP itself — this is the whole point of the email, make it unmissable -->
    <div style="background-color: #f4f4f7; border-radius: 8px; padding: 20px; text-align: center; margin-bottom: 24px;">
        <span style="font-size: 32px; font-weight: 700; letter-spacing: 8px; color: #1a1a1a; font-family: monospace;">
            %s
        </span>
    </div>

    <!-- Reassurance / anti-phishing line -->
    <p style="color: #888888; font-size: 13px; line-height: 1.5;">
        Didn't request this code? You can safely ignore this email — no changes will be made to your account.
    </p>

    <hr style="border: none; border-top: 1px solid #eeeeee; margin: 24px 0;">

    <p style="color: #aaaaaa; font-size: 12px;">
        This is an automated message, please don't reply directly to this email.
    </p>
</div>`, otp)
}

// makes sures if the ID the user is currently logged in is also the ID the user provided Email
func checkEmailAndIDRelation(userGivenEmail string, ctx context.Context, pool *pgxpool.Pool) error {

	email, err := handlers.HandleEmailCheck(ctx, pool, userGivenEmail)
	if err != nil {
		if errors.Is(err, handlers.ErrNoUserFound) {
			return handlers.ErrNoUserFound
		}
		return err
	}

	if email == userGivenEmail {
		return nil
	}
	return errors.New("wrong email provided by the user")
}

// this is /sendVerificationMail
func CallSendVerificationMail(resendClient *resend.Client, pool *pgxpool.Pool, redisClient *redis.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		stat := verifyHTTPMethod(w, r, http.MethodPost)
		if !stat {
			return
		}

		ctx := r.Context()
		parseErr := r.ParseForm()
		var userEmail string

		if parseErr != nil {
			throwHTTPErrAndLog("failed parsing the html form", parseErr, "An error occured while trying to parse the html form", w, http.StatusInternalServerError)
			return
		}

		tStat := handleTokenEdgeCase2(w, r)

		if tStat == 0 {
			return
		}

		storedCookieEmail, cookieErr := r.Cookie("Email")

		if cookieErr != nil {
			if errors.Is(cookieErr, http.ErrNoCookie) {
				userEmail = strings.ToLower(strings.TrimSpace(r.FormValue("email")))

				if userEmail == "" {
					throwHTTPErrAndLog("provided email is empty", nil, "Please fill up the email field.", w, http.StatusBadRequest)
					return
				}
				emailCheckErr := checkEmailAndIDRelation(userEmail, ctx, pool)

				if emailCheckErr != nil {
					throwHTTPErrAndLog("Email is not connected to users ID", emailCheckErr, "Please provide the correct email.", w, http.StatusBadRequest)
					return
				}

				log.Println("Email is connected to users ID.")

			} else {
				log.Println("", cookieErr)
				throwHTTPErrAndLog("An error occured whilist fetching the cookie form users request, err: ", cookieErr, "Error occured while fetching your Auth cookie.", w, http.StatusInternalServerError)
				return
			}

		} else {
			userEmail = storedCookieEmail.Value
		}

		//	OTP CREATION
		OTP, otpCreationError := authentication.CreateOTP(userEmail, ctx, redisClient)

		if otpCreationError != nil {
			if errors.Is(otpCreationError, authentication.ErrOTPcoolDown) {
				throwHTTPErrAndLog("resending OTP too fast, wait till the cooldown ends.", authentication.ErrOTPcoolDown, "Please wait before requesting a new OTP", w, http.StatusBadRequest)
				return
			}
			throwHTTPErrAndLog("An error occcured while creating users OTP", otpCreationError, "An error occured whilist creating your OTP", w, http.StatusInternalServerError)
			return
		}

		log.Println("Successfully made the OTP: ", OTP)

		// EMAIL HANDLING
		emailConfig := &resend.SendEmailRequest{
			From:    "Acme <onboarding@resend.dev>",
			To:      []string{userEmail},
			Subject: "Your OTP code from Sclera",
			Html:    buildOTPEmailHTML(OTP),
		}

		// Send the email
		sent, err := resendClient.Emails.Send(emailConfig)
		if err != nil {
			throwHTTPErrAndLog("Error sending email: %v\n", err, "An error occured while trying to send you the email", w, http.StatusInternalServerError)
			return
		}

		emailCookie := &http.Cookie{
			Name:     "Email",   // Legal name string
			Value:    userEmail, // Value becomes: "mathiwasbaroi@gmail.com"
			Path:     "/",
			MaxAge:   5 * 60,
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
			Secure:   false, // Set to true in production over HTTPS
		}

		http.SetCookie(w, emailCookie)

		log.Printf("Email sent successfully! ID: %s\n", sent.Id)

		http.Redirect(w, r, "/inputOTP", http.StatusTemporaryRedirect)

	}
}

// step 2 of passoword resseting
// the OTP verification handling using redis
// this is /inputOTP
func CallVerifyOTPclientSide() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		stat := verifyHTTPMethod(w, r, http.MethodPost)
		if !stat {
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")

		tempParseErrr := parseOTPverificationTemp.Execute(w, nil)

		if tempParseErrr != nil {
			throwHTTPErrAndLog("failed to render template", tempParseErrr, "Internal server error", w, http.StatusInternalServerError)
			return
		}

	}
}

func instantCookieDeletion(w http.ResponseWriter, name string, path string) {
	instantCookieDeletion := &http.Cookie{
		Name:     name,
		Value:    "", // Clear the value
		Path:     path,
		MaxAge:   -1,              // Signals immediate deletion
		Expires:  time.Unix(0, 0), // Backward compatibility for older browsers
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   false,
	}

	http.SetCookie(w, instantCookieDeletion)

}

// this is /veriyOTP
func CallVerifyOTPserverSide(redisClient *redis.Client, pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		stat := verifyHTTPMethod(w, r, http.MethodPost)
		if !stat {
			return
		}
		parseErr := r.ParseForm()
		ctx := r.Context()
		userEmail, err := r.Cookie("Email")
		if err != nil {
			log.Println("", err)
			throwHTTPErrAndLog("An error occured whilist fetching the cookie form users request, err: ", err, "Error occured while fetching your Auth cookie.", w, http.StatusInternalServerError)
			return
		}

		if parseErr != nil {
			throwHTTPErrAndLog("failed parsing the html form", parseErr, "An error occured while trying to parse the html form", w, http.StatusInternalServerError)
			return
		}
		userInputOTP := r.FormValue("otp")

		if userInputOTP == "" {
			throwHTTPErrAndLog("otp field missing from request", errors.New("empty otp"), "Please enter your verification code", w, http.StatusBadRequest)
			return
		}

		verificationErr := authentication.VerifyOTP(userEmail.Value, ctx, userInputOTP, redisClient)

		if verificationErr != nil {
			if errors.Is(verificationErr, authentication.ErrMaxAttempts) {

				instantCookieDeletion(w, "Email", "/")
				log.Println("too many attempts, please request a new OTP")
				http.Redirect(w, r, "/updateAccout", http.StatusSeeOther)
				return
			}
			if errors.Is(verificationErr, authentication.ErrInvalidOTP) {
				throwHTTPErrAndLog("invalid OTP provided by the user", authentication.ErrInvalidOTP, "invalid OTP", w, http.StatusBadRequest)
				return
			}
			throwHTTPErrAndLog("An error occcured while verifying users OTP", verificationErr, "An error occured whilist verifying your OTP", w, http.StatusInternalServerError)
			return
		}

		log.Println("users OTP sucessfully verified")

		//deltes the email cookie upon sucessfully verifying the OTP
		instantCookieDeletion(w, "Email", "/")

		//add a new authorization cookie, as OTP verification also marks the user as authentic
		_, userID, idErr := handlers.HandleFetchIdUsingEmail(ctx, pool, userEmail.Value)

		if idErr != nil {
			if errors.Is(idErr, handlers.ErrNoUserFound) {
				throwHTTPErrAndLog("No user with this email exists in database", handlers.ErrNoUserFound, "The email provided is INVALID", w, http.StatusBadRequest)
				return
			}
			throwHTTPErrAndLog("An error occured while trying to fetch userID from db", idErr, "An error occured while trying to fetch your ID form the database", w, http.StatusInternalServerError)
			return
		}

		tokenString, err := authentication.CreateToken(userID)

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

		http.Redirect(w, r, "/updateUsersPassword", http.StatusSeeOther)
	}
}

// step 3 of password resetting
// this is /runPasswordUpdation
// check if the passwords are same or not, if same dont allow user to change it BUT KEEP THE FUNC RUNNING, if the pass is diff then let user change it
func CallUpdateUserPasswordServerSide(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		stat := verifyHTTPMethod(w, r, http.MethodPost)
		if !stat {
			return
		}

		ctx := r.Context()
		userID, ok := ctx.Value(middleware.UserIDkey).(int) //this .(int) is NOT typecasting the value of type any, it is there to TYPECHECK what the actual type of the type any value is
		if !ok {
			throwHTTPErrAndLog("No user user id found in the context value", nil, "No user ID has been found linked to your account", w, http.StatusBadRequest)
			return
		}
		parseErr := r.ParseForm()

		if parseErr != nil {
			throwHTTPErrAndLog("failed parsing the html form", parseErr, "An error occured while trying to parse the html form", w, http.StatusInternalServerError)
			return
		}

		plaintTextPassword := r.FormValue("password")

		//hash the users password
		hashedPassword, hashingErr := hashing.HashPassword(plaintTextPassword)

		if hashingErr != nil {
			throwHTTPErrAndLog("failed hashing the password", hashingErr, "Your password was not successfully hashed by the server", w, http.StatusInternalServerError)
			return
		}

		//replace users password with the newly hashed one
		_, passUpdateErr := handlers.HandleUserPasswordUpdation(ctx, pool, hashedPassword, userID)

		if passUpdateErr != nil {
			if errors.Is(passUpdateErr, handlers.ErrNoUserFound) {
				throwHTTPErrAndLog("user with this id does not exists", handlers.ErrNoUserFound, "User ID is INVALID", w, http.StatusBadRequest)
				return
			}
			throwHTTPErrAndLog("An error occcured while updating users password", passUpdateErr, "An error occured whilist updating your password", w, http.StatusInternalServerError)
			return
		}

		log.Println("successfully updated users password")
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

// this is /updateUsersPassword
func CallUpdateUserPasswordClientSide() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		stat := verifyHTTPMethod(w, r, http.MethodGet)
		if !stat {
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")

		tempParseErrr := parseUpdatePasswordTemp.Execute(w, nil)

		if tempParseErrr != nil {
			throwHTTPErrAndLog("failed to render template", tempParseErrr, "Internal server error", w, http.StatusInternalServerError)
			return
		}
	}
}
