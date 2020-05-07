package handlers

import (
    "net/http"
    "encoding/json"

    "zerogov/fractal6.go/db"
    "zerogov/fractal6.go/web/auth"
    "zerogov/fractal6.go/graph/model"
)


// Login create and pass a token to the authenticated user.
func Login(w http.ResponseWriter, r *http.Request) {
	// Get the JSON body and decode into UserCreds
	var creds model.UserCreds
	err := json.NewDecoder(r.Body).Decode(&creds)
	if err != nil {
		// Body structure error
		//w.WriteHeader(http.StatusBadRequest)
        http.Error(w, err.Error(), 400)
		return
	}

    // === This is protected ===
    // Returns the user ctx if authenticated.
    userCtx, err := auth.GetAuthUserCtx(creds)
    if err != nil {
		// Credentials validation error
		//w.WriteHeader(http.StatusUnauthorized)
        http.Error(w, err.Error(), 401)
		return
    }

	// Create a new token string
	tokenString, err := auth.NewUserToken(*userCtx)
	if err != nil {
		// Token issuing error
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

    data := `["OK"]`
    w.Write([]byte(data))
}


// Signup register a new user and gives it a token.
func Signup(w http.ResponseWriter, r *http.Request) {
	// Get the JSON body and decode into UserCreds
	var creds model.UserCreds
	err := json.NewDecoder(r.Body).Decode(&creds)
	if err != nil {
		// Body structure error
        http.Error(w, err.Error(), 400)
		return
	}

    // Validate user form and ensure user uniquenesss.
    err = auth.ValidateNewUser(creds)
    if err != nil {
        http.Error(w, err.Error(), 401)
        return
    }

    // Upsert new user
    userCtx, err := auth.CreateNewUser(creds)
    if err != nil {
		// Credentials validation error
        switch err.(type) {
        case *db.GraphQLError:
            http.Error(w, err.Error(), 401)
        default:
            http.Error(w, err.Error(), 500)
        }
		return
    }

	// Create a new token string
	tokenString, err := auth.NewUserToken(*userCtx)
	if err != nil {
		// Token issuing error
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

    data := `["OK"]`
    w.Write([]byte(data))
}

// Logout deletes the user token.
//func Logout(w http.ResponseWriter, r *http.Request) {
//    // The client deletes the cookies or session.
//}
