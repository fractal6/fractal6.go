package handlers

import (
    "fmt"
    "strings"
    "time"
    "net/http"
    "encoding/json"
    "github.com/steambap/captcha"


    "zerogov/fractal6.go/db"
    "zerogov/fractal6.go/web/auth"
    "zerogov/fractal6.go/web/sessions"
    "zerogov/fractal6.go/web/email"
    "zerogov/fractal6.go/graph/model"
    . "zerogov/fractal6.go/tools"
)

var cache sessions.Session

func init() {
    cache = sessions.GetCache()
}


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
    oldUctx, err := auth.GetUserContext(ctx)
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


func ResetPasswordChallenge(w http.ResponseWriter, r *http.Request) {
    var token string

    // Get the visitor unique token or create a new one.
    c, err := r.Cookie("challenge_token")
    if err == http.ErrNoCookie {
        // generate a token
        token = sessions.GenerateToken()
    } else if err != nil {
        w.WriteHeader(http.StatusBadRequest)
        return
    } else {
        token = c.Value
    }

    // create a captcha of 150x50px
    data, _ := captcha.New(150, 50, func(options *captcha.Options) {
		options.CharPreset = "abcdefghkmnpqrstuvwxyz0123456789"
	})
    //data, _ := captcha.NewMathExpr(150, 50)

    // Save the token and challenge result in cache
    // with timeout to clear it.
    _, err = cache.Do("SETEX", token, "300", data.Text)
    if err != nil {
        w.WriteHeader(http.StatusInternalServerError)
        return
    }

	// Set the new token as the users `session_token` cookie
	http.SetCookie(w, &http.Cookie{
		Name:    "challenge_token",
		Value:   token,
        HttpOnly: true,
        Secure: true,
		Expires: time.Now().Add(300 * time.Second),
	})

    data.WriteImage(w)
}

func ResetPassword(w http.ResponseWriter, r *http.Request) {
    var data  struct {
        Email string
        Challenge string
    }

	// Get the JSON body and decode into UserCreds
	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		// Body structure error
        http.Error(w, err.Error(), 400)
		return
	}

    // Email is required
    if data.Email == "" {
        http.Error(w, "An email is required", 400)
		return
    }

    // Check email format
    err = auth.ValidateEmail(data.Email)
    if err != nil {
        http.Error(w, err.Error(), 400)
		return
    }

    // Try to Extract session token
    c, err := r.Cookie("challenge_token")
    if err != nil || c.Value == "" {
        http.Error(w, "Unauthorized, please try again later", 400)
		return
    }
    token := c.Value

    // Get the challenge from cache
    //expected, err := redis.String(cache.Do("GET", token))
    expected, err := cache.Do("GET", token)
    if fmt.Sprintf("%s", expected) != data.Challenge {
        w.Write([]byte("false"))
        return
    }

    // Return true after here in any case, to prevent
    // the email database to be probe.
    ex, _ := db.GetDB().Exists("User.email", data.Email, nil, nil)
    if ex {
        // Actual send the reset email
        //
        // Set the cache with a token to identify the user
        token_url_redirect := sessions.GenerateToken()
        _, err = cache.Do("SETEX", token_url_redirect, "3800", data.Email)
        if err != nil {
            w.WriteHeader(http.StatusInternalServerError)
            return
        }
        err = email.SendResetEmail(data.Email, token_url_redirect)
        if err != nil { panic(err) }
    }

    // Invalidate the challenge token if passed
    _, err = cache.Do("DEL", token)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

    w.Write([]byte("true"))
}

