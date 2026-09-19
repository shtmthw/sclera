package helpers

// random comment to run ci pipeline

import (
	"fmt"
	"log"
	"net/http"
)

func ThrowHTTPErrAndLog(logText string, logErr error, errorText string, w http.ResponseWriter, httpStat int) {
	if logErr != nil {
		log.Println(logText, logErr)
	} else {
		log.Println(logText)
	}

	http.Error(w, errorText, httpStat)
}

func VerifyHTTPMethod(w http.ResponseWriter, r *http.Request, allowedMethod string) bool {
	if r.Method != allowedMethod {
		// Log and throw the error dynamically
		errMsg := fmt.Sprintf("The method %s is UNAUTHORIZED. Expected %s.", r.Method, allowedMethod)
		ThrowHTTPErrAndLog("unauthorized method!", nil, errMsg, w, http.StatusMethodNotAllowed)
		return false // Validation failed
	}
	return true // Validation passed
}
