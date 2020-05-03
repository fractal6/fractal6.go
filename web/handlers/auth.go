package handlers

import (
    "net/http"
    "encoding/json"

    "zerogov/fractal6.go/web/auth"
)


// Create the Login handler
func Login(w http.ResponseWriter, r *http.Request) {
	// Get the JSON body and decode into UserCreds
	var creds auth.UserCreds
	err := json.NewDecoder(r.Body).Decode(&creds)
	if err != nil {
		// If the structure of the body is wrong, return an HTTP error
		w.WriteHeader(http.StatusBadRequest)
		return
	}

    // This is protected
    userCtx, err := auth.GetUserCtx(creds)
    if err != nil {
        //return nil, &echo.HTTPError{
        //    Code: http.StatusBadRequest,
        //    Message: "invalid email or password"
        //}
		// Returns Json Structured error
		w.WriteHeader(http.StatusUnauthorized)
		return
    }

	// Create a new token string
	tokenString, err := auth.NewUserToken(userCtx)
	if err != nil {
		// If there is an error in creating the JWT return an internal server error
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Finally, we set the client cookie for "token" as the JWT we just generated
	// we also set an expiry time which is the same as the token itself
	http.SetCookie(w, &http.Cookie{
		Name:    "token",
		Value:   tokenString,
		//Expires: expirationTime,
	})
}
