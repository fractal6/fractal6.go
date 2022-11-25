/*
 * Fractale - Self-organisation for humans.
 * Copyright (C) 2022 Fractale Co
 *
 * This file is part of Fractale.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as
 * published by the Free Software Foundation, either version 3 of the
 * License, or (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with Fractale.  If not, see <http://www.gnu.org/licenses/>.
 */

package handlers


import (
	//"fmt"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/steambap/captcha"

	"fractale/fractal6.go/db"
	"fractale/fractal6.go/graph"
	"fractale/fractal6.go/graph/model"
	. "fractale/fractal6.go/tools"
	"fractale/fractal6.go/web/auth"
	"fractale/fractal6.go/web/email"
	"fractale/fractal6.go/web/sessions"
)

var cache *sessions.Session

func init() {
    cache = sessions.GetCache()
}

// Signup register a new user and gives it a token.
func Signup(w http.ResponseWriter, r *http.Request) {
	var creds model.UserCreds

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

    // Try to get PendingUser
    pending_, err := db.DB.GetFieldByEq("PendingUser.email", creds.Email, "uid PendingUser.updatedAt")
    pending, _ := pending_.(model.JsonAtom)

    // Delay to prevent attack and user creation hijacking
    now := Now()
    if pending["updatedAt"] != nil &&
    TimeDelta(now, pending["updatedAt"].(string)) < time.Minute * 5 {
        http.Error(w, auth.ErrUsernameExist.Error(), 401)
        return
    }

    email_token := sessions.GenerateToken()
    passwd := HashPassword(creds.Password)
    if pending["id"] != nil {
        // Update pending user
        err = db.DB.Update(db.DB.GetRootUctx(), "pendingUser", model.UpdatePendingUserInput{
            Filter: &model.PendingUserFilter{Email: &model.StringHashFilter{Eq: &creds.Email}},
            Set: &model.PendingUserPatch{
                Password: &passwd,
                EmailToken: &email_token,
                UpdatedAt: &now,
                Subscribe: creds.Subscribe.ToBoolPtr(),
            },
        })
        if err != nil {
            http.Error(w, err.Error(), 500)
            return
        }
        // @id field cant't be update with graphql (@debug dgraph)
        err = db.DB.SetFieldByEq("PendingUser.email", creds.Email, "PendingUser.username", creds.Username)
    } else {
        // Create pending user
        _, err = db.DB.Add(db.DB.GetRootUctx(), "pendingUser", model.AddPendingUserInput{
            Email: &creds.Email,
            Username: &creds.Username,
            Password: &passwd,
            EmailToken: &email_token,
            UpdatedAt: &now,
            Subscribe: creds.Subscribe.ToBoolPtr(),
        })
    }
    if err != nil {
        http.Error(w, err.Error(), 500)
        return
    }

    // Send verification email
    err = email.SendVerificationEmail(creds.Email, email_token)
    if err != nil { panic(err) }

    w.Write([]byte("true"))
}

// Signup confirmation
func SignupValidate(w http.ResponseWriter, r *http.Request) {
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

    // Update Creds depending on PendingUser
    pending := struct{
        Username string
        Email string
        Password string
        UpdatedAt *string
        Subscribe  bool
    }{}
    if creds.EmailToken != nil {
        // User signup parcour
        // --
        // User has already been validated and saved in UserPending
        if err = db.DB.Meta1("getPendingUser", map[string]string{"k":"email_token", "v":*creds.EmailToken}, &pending); err != nil {
            http.Error(w, err.Error(), 500)
            return
        } else if pending.UpdatedAt != nil &&
        TimeDelta(Now(), *pending.UpdatedAt) > time.Hour * 48 {
            http.Error(w, "The session has expired.", 500)
            return
        }
        // Overwrite creds to prevent CSRF
        StructMap(pending, &creds)
    } else if creds.Puid != nil {
        // User invitation parcour
        // --
        // User has not been registered in UserPending
        if err = db.DB.Meta1("getPendingUser", map[string]string{"k":"token", "v":*creds.Puid}, &pending); err != nil {
            http.Error(w, err.Error(), 500)
            return
        } else if pending.Email == "" {
            // If user exists, set uctx
            if ex, err := db.DB.Exists("User.username", creds.Username, nil); err != nil {
                http.Error(w, err.Error(), 500)
                return
            } else if ex {
                uctx, err = auth.GetAuthUserCtx(creds)
                if err != nil {
                    http.Error(w, err.Error(), 500)
                    return
                }
            } else {
                http.Error(w, "Token not found.", 400)
                return
            }
        } else {
            creds.Email = pending.Email
            if err = auth.ValidateNewUser(creds); err != nil {
                http.Error(w, err.Error(), 401)
                return
            }
            creds.Password = HashPassword(creds.Password)
        }
    } else {
        http.Error(w, "token validation error", 500)
        return
    }

    // Upsert new user and sync pending
    if uctx == nil {
        if creds.Username == "" || creds.Email == "" {
            http.Error(w, "User already exists.", 500)
            return
        }

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

        // Sync and remove pending user
        err = graph.SyncPendingUser(creds.Username, creds.Email)
        if err != nil {
            http.Error(w, err.Error(), 500)
            return
        }

        // Add welcome user notification
        anchorTid, err := db.GetDB().GetSubSubFieldByEq("Node.nameid", "f6", "Node.source", "Blob.tension", "uid" )
        if err != nil {
            http.Error(w, err.Error(), 500); return
        } else if anchorTid != nil {
            tid := anchorTid.(string)
            link := "/verification"
            graph.PushNotifNotifications(model.NotifNotif{
                Uctx: uctx,
                Tid: &tid,
                Cid: nil,
                Link: &link,
                Msg: "Welcome to Fractale",
                To: []string{uctx.Username},
                IsRead: true,
            }, true)
        }
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

    // @debug: use a thread to set the last ack Literal, no need to wait here.
    err = db.GetDB().SetFieldByEq("User.username", uctx.Username, "User.lastAck", Now())
    if err != nil {
        http.Error(w, err.Error(), 500)
		return
    }

    w.Write(data)
}

// Logout reset the jwt cookie.
func Logout(w http.ResponseWriter, r *http.Request) {
	// Finally, we set the client cookie for "token" as the JWT we just generated
	// we also set an expiry time which is the same as the token itself
	http.SetCookie(w, auth.ClearUserCookie())
    w.Write(nil)
}

// TokenAck update the user token.
func TokenAck(w http.ResponseWriter, r *http.Request) {
    oldUctx, err := auth.GetUserContextLight(r.Context())
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

func ResetPasswordChallenge(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    var token string

    // Get the visitor unique token or create a new one.
    c, err := r.Cookie("challenge_token")
    if err == http.ErrNoCookie {
        // generate a token
        token = sessions.GenerateToken()
    } else if err != nil {
        http.Error(w, err.Error(), 500)
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
    err = cache.SetEX(ctx, token, data.Text, time.Second * 300).Err()
    if err != nil {
        http.Error(w, err.Error(), 500)
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
    ctx := r.Context()
    var data  struct {
        Email string
        Challenge string
    }

	// Get the JSON body and decode it
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
        http.Error(w, "Unauthorized, please try again in a few seconds.", 400)
		return
    }
    token := c.Value

    // Get the challenge from cache
    //expected, err := redis.String(cache.Do("GET", token))
    expected, err := cache.Get(ctx, token).Result()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
    if expected != data.Challenge {
        w.Write([]byte("false"))
        return
    }

    // Return true after here in any case, to prevent
    // the email database to be probe.
    ex, _ := db.GetDB().Exists("User.email", data.Email, nil)
    if ex {
        // Actual send the reset email
        //
        // Set the cache with a token to identify the user
        token_url_redirect := sessions.GenerateToken()
        err = cache.SetEX(ctx, token_url_redirect, data.Email, time.Hour*1).Err()
        if err != nil {
			http.Error(w, err.Error(), 500)
            return
        }
        err = email.SendResetEmail(data.Email, token_url_redirect)
        if err != nil { panic(err) }
    }

    // Invalidate the challenge token if passed
    err = cache.Del(ctx, token).Err()
	if err != nil {
        http.Error(w, err.Error(), 500)
		return
	}

    w.Write([]byte("true"))
}

func ResetPassword2(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    var data  struct {
        Password string
        Password2 string
        Token string
    }

	// Get the JSON body and decode it
	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		// Body structure error
        http.Error(w, err.Error(), 400)
		return
	}

    // Check password
    if data.Password != data.Password2 {
        http.Error(w, "The passwords does not match.", 400)
		return
    }

	if err = auth.ValidatePassword(data.Password); err != nil {
        http.Error(w, err.Error(), 400)
		return
    }

    // Check that the cache contains the token
    mail, err := cache.Get(ctx, data.Token).Result()
	if err != nil {
        http.Error(w, err.Error(), 500)
		return
	}
    if mail == "" {
        http.Error(w, "Session expired, please try again.", 500)
        return
    }

	// Set the new password for the given user
    err = db.GetDB().SetFieldByEq("User.email", mail, "User.password", HashPassword(data.Password))
    if err != nil {
        http.Error(w, err.Error(), 500)
		return
    }

    // Invalidate the reset token if passed
    err = cache.Del(ctx, data.Token).Err()
	if err != nil { panic(err) }

    // Get the userctx to return
    uctx, err := auth.GetAuthUserCtx(model.UserCreds{Username:mail, Password:data.Password})
    if err != nil {
        http.Error(w, err.Error(), 500)
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

    data_out, err := json.Marshal(uctx)
    if err != nil {
        http.Error(w, err.Error(), 500)
		return
    }

    w.Write(data_out)
}

// Check that the cache contains the given token
func UuidCheck(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    var data struct {
        Token string
    }

	// Get the JSON body and decode it
	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		// Body structure error
        http.Error(w, err.Error(), 400)
		return
	}

    // Check that the cache contains the token
    x, err := cache.Get(ctx, data.Token).Result()
	if err != nil {
        http.Error(w, err.Error(), 500)
		return
	}
    if x == "" {
        w.Write([]byte("false"))
        return
    }

    w.Write([]byte("true"))
}



