package auth

import (
    "log"
    "time"
    "errors"
    "context"
    "github.com/mitchellh/mapstructure"
    jwt "github.com/dgrijalva/jwt-go"
    "github.com/go-chi/jwtauth"

	"zerogov/fractal6.go/graph/model"
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
    token, _ := tk.issue(model.UserCtx{Username:"debugger"}, time.Hour*1)
	log.Println("DEBUG JWT:", token)
	return tk
}

func (tk Jwt) GetAuth() *jwtauth.JWTAuth {
    return tk.tokenAuth
}

// Issue generate and encode a new token
func (tk *Jwt) issue(d model.UserCtx, t time.Duration) (string, error){
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
func NewUserToken(userCtx model.UserCtx) (string, error) {
    token, err := tokenMaster.issue(userCtx, time.Hour*1)
    return token, err
}

func GetAuthContext(ctx context.Context) context.Context {
    token, claims, err := jwtauth.FromContext(ctx)

    if err != nil {
        //errMsg := fmt.Errorf("%v", err)
        switch err {
        case jwtauth.ErrUnauthorized:
        case jwtauth.ErrExpired:
        case jwtauth.ErrNBFInvalid:
        case jwtauth.ErrIATInvalid:
        case jwtauth.ErrNoTokenFound:
        case jwtauth.ErrAlgoInvalid:
        }
        //http.Error(w, http.StatusText(401), 401)
        //return
    } else if token == nil || !token.Valid {
        err = errors.New("jwtauth: token is invalid")
    } else if claims[tokenMaster.tokenClaim] == nil {
        err = errors.New("auth: user claim is invalid")
    }

    if err == nil {
        userCtx := model.UserCtx{}
        mapstructure.Decode(claims[tokenMaster.tokenClaim], &userCtx)
        ctx = context.WithValue(ctx, "user_context", userCtx)
    } else {
        ctx = context.WithValue(ctx, "user_context_error", err)
    }
    return ctx
}
