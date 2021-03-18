package handlers

import (
    //"fmt"
    "strings"
    "net/http"
    "encoding/json"

    "zerogov/fractal6.go/db"
    "zerogov/fractal6.go/web/auth"
    "zerogov/fractal6.go/graph/model"
    . "zerogov/fractal6.go/tools"
)


// Signup register a new user and gives it a token.
func Signup(w http.ResponseWriter, r *http.Request) {
	var creds model.UserCreds
    var uctx *model.UserCtx

	// Get the JSON body and decode into UserCreds
	err := json.NewDecoder(r.Body).Decode(&creds)
	if err != nil {
		// Body structure error
        http.Error(w, err.Error(), 400)
		return
	}
    // Ignore username/email case
    creds.Username = strings.ToLower(creds.Username)

    // Validate user form and ensure user uniquenesss.
    err = auth.ValidateNewUser(creds)
    if err != nil {
        http.Error(w, err.Error(), 401)
        return
    }

    // Upsert new user
    uctx, err = auth.CreateNewUser(creds)
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

	// Create a new cookie with token
    httpCookie, err := auth.NewUserCookie(*uctx)
	if err != nil {
		// Token issuing error
        http.Error(w, err.Error(), 500)
		return
	}

	// Finally, we set the client cookie for "token" as the JWT we just generated
	// we also set an expiry time which is the same as the token itself
	http.SetCookie(w, httpCookie)

    // Return the user context
    data, err := json.Marshal(uctx)
    if err != nil {
        http.Error(w, err.Error(), 500)
		return
    }
    w.Write(data)
}

// Login create and pass a token to the authenticated user.
func Login(w http.ResponseWriter, r *http.Request) {
	var creds model.UserCreds
    var uctx *model.UserCtx

	// Get the JSON body and decode into UserCreds
	err := json.NewDecoder(r.Body).Decode(&creds)
	if err != nil {
		// Body structure error
        http.Error(w, err.Error(), 400)
		return
	}
    // Ignore username/email case
    creds.Username = strings.ToLower(creds.Username)

    // === This is protected ===
    // Returns the user ctx if authenticated.
    uctx, err = auth.GetAuthUserCtx(creds)
    if err != nil {
		// Credentials validation error
        http.Error(w, err.Error(), 401)
		return
    }

    // Check if the user has login right
    if !uctx.Rights.CanLogin  {
        http.Error(w, auth.ErrCantLogin.Error(), 401)
    }

	// Create a new cookie with token
    httpCookie, err := auth.NewUserCookie(*uctx)
	if err != nil {
		// Token issuing error
        http.Error(w, err.Error(), 500)
		return
	}

	// Finally, we set the client cookie for "token" as the JWT we just generated
	// we also set an expiry time which is the same as the token itself
	http.SetCookie(w, httpCookie)

    // Return the user context
    data, err := json.Marshal(uctx)
    if err != nil {
        http.Error(w, err.Error(), 500)
		return
    }

    // @debug: use a thread to set the last ack Literal, no need to wait here.
    err = db.GetDB().SetFieldByEq("User.username", uctx.Username, "User.lastAck", Now())
    if err != nil {
        http.Error(w, err.Error(), 500)
		return
    }

    w.Write(data)
}

// TokenAck update the user token.
func TokenAck(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    oldUctx, err := auth.UserCtxFromContext(ctx)
    if err != nil {
		// User authentication error
		//w.WriteHeader(http.StatusUnauthorized)
        http.Error(w, err.Error(), 401)
		return
    }

    // Refresh the user context
    uctx, err := auth.GetAuthUserFromCtx(*oldUctx)
    if err != nil {
		// Credentials validation error
		//w.WriteHeader(http.StatusUnauthorized)
        http.Error(w, err.Error(), 401)
		return
    }

	// Create a new cookie with token
    httpCookie, err := auth.NewUserCookie(*uctx)
	if err != nil {
		// Token issuing error
		//w.WriteHeader(http.StatusInternalServerError)
        http.Error(w, err.Error(), 500)
		return
	}

	// Finally, we set the client cookie for "token" as the JWT we just generated
	// we also set an expiry time which is the same as the token itself
	http.SetCookie(w, httpCookie)

    // Return the user context
    data, err := json.Marshal(uctx)
    if err != nil {
        http.Error(w, err.Error(), 500)
		return
    }

    // @debug: use a thread to set the last ack Literal, no need to wait here.
    err = db.GetDB().SetFieldByEq("User.username", uctx.Username, "User.lastAck", Now())
    if err != nil {
        http.Error(w, err.Error(), 500)
		return
    }

    w.Write(data)
}

// Logout deletes the user token.
//func Logout(w http.ResponseWriter, r *http.Request) {
//    // The client deletes the cookies or session.
//}

