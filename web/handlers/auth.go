package handlers

import (
    "net/http"
    "encoding/json"

    "zerogov/fractal6.go/web/auth"
)


// Login create and pass a token to the authenticated user.
func Login(w http.ResponseWriter, r *http.Request) {
	// Get the JSON body and decode into UserCreds
	var creds auth.UserCreds
	err := json.NewDecoder(r.Body).Decode(&creds)
	if err != nil {
		// If the structure of the body is wrong, return an HTTP error
		//w.WriteHeader(http.StatusBadRequest)
        http.Error(w, err.Error(), 500)
		return
	}

    // This is protected
    userCtx, err := auth.GetUserCtx(creds)
    if err != nil {
		// Credentials validation error
		//w.WriteHeader(http.StatusUnauthorized)
        http.Error(w, err.Error(), 401)
		return
    }

	// Create a new token string
	tokenString, err := auth.NewUserToken(*userCtx)
	if err != nil {
		// If there is an error in creating the JWT return an internal server error
		//w.WriteHeader(http.StatusInternalServerError)
        http.Error(w, err.Error(), 500)
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

// Logout deletes the user token.
func Logout(w http.ResponseWriter, r *http.Request) {
}

// Signup register a new user and gives it a token.
func Signup(w http.ResponseWriter, r *http.Request) {
}
