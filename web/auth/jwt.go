package auth

import (
    "log"
    "time"
    jwt "github.com/dgrijalva/jwt-go"
    "github.com/go-chi/jwtauth"
)

var tokenMaster *Jwt

func init () {
    tokenMaster = Jwt{}.New()
}

type Jwt struct {
    // @FIX: How to initialize mapClaims with another map
    // in order to node decode evething at each request
    tokenClaim string 
	tokenAuth  *jwtauth.JWTAuth
}

// New create a token auth master
func (Jwt) New() *Jwt {
    secret := "frctl6"
	tk := &Jwt{
        tokenClaim: "user_ctx",
		tokenAuth: jwtauth.New("HS256", []byte(secret), nil),
	}
    token, _ := tk.issue(UserCtx{Username:"debugger"}, time.Hour*1)
	log.Println("DEBUG JWT:", token)
	return tk
}

func (tk Jwt) GetAuth() *jwtauth.JWTAuth {
    return tk.tokenAuth
}

// Issue generate and encode a new token
func (tk *Jwt) issue(d UserCtx, t time.Duration) (string, error){
    claims := jwt.MapClaims{ tk.tokenClaim: d }
    jwtauth.SetExpiry(claims, time.Now().Add(t))
	_, tokenString, err := tk.tokenAuth.Encode(claims)
	return tokenString, err
}

//
// Global functions
//

func GetTokenMaster() *Jwt {
    return tokenMaster
}

// NewUserToken create a new user token from master key
func NewUserToken(userCtx UserCtx) (string, error) {
    token, err := tokenMaster.issue(userCtx, time.Hour*1)
    return token, err
}

