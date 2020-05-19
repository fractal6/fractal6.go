package auth

import (
    //"fmt"
    "log"
    "time"
    "errors"
    "context"
    "encoding/json"
    "net/http"
    //"github.com/mitchellh/mapstructure"
    jwt "github.com/dgrijalva/jwt-go"
    "github.com/go-chi/jwtauth"

	"zerogov/fractal6.go/graph/model"
)

var tkMaster *Jwt

func init () {
    tkMaster = Jwt{}.New()
}

type Jwt struct {
    // @FIX: How to initialize mapClaims with another map
    // in order to node decode evething at each request
	tokenAuth  *jwtauth.JWTAuth
    tokenClaim string 
    tokenClaimErr string 
}

// New create a token auth master
func (Jwt) New() *Jwt {
    secret := "frctl6"
	tk := &Jwt{
        tokenClaim: "user_ctx",
        tokenClaimErr: "user_ctx_err",
		tokenAuth: jwtauth.New("HS256", []byte(secret), nil),
	}
    uctx := model.UserCtx{
        Username: "yoda",
        Rights: model.UserRights{CanLogin:false, CanCreateRoot:true},
        Roles: []model.Role{
            {Rootnameid:"SKU", Nameid:"SKU", RoleType:model.RoleTypeCoordinator},
        },
    }
    token, _ := tk.issue(uctx, time.Hour*24)
	log.Println("DEBUG JWT:", token)
	return tk
}

func (tk Jwt) GetAuth() *jwtauth.JWTAuth {
    return tk.tokenAuth
}

// Issue generate and encode a new token
func (tk *Jwt) issue(d model.UserCtx, t time.Duration) (string, error) {
    claims := jwt.MapClaims{ tk.tokenClaim: d }
    jwtauth.SetExpiry(claims, time.Now().Add(t))
	_, tokenString, err := tk.tokenAuth.Encode(claims)
	return tokenString, err
}

//
// Global functions
//

func GetTokenMaster() *Jwt {
    return tkMaster 
}

// NewUserToken create a new user token from master key
func NewUserToken(userCtx model.UserCtx) (string, error) {
    token, err := tkMaster.issue(userCtx, time.Hour*1)
    return token, err
}

// NexuserCookie create an http cookie that embed a token
func NewUserCookie(userCtx model.UserCtx) (*http.Cookie, error) {
    tokenString, err := NewUserToken(userCtx)
    if err != nil {
        return nil, err
    }

    httpCookie := http.Cookie{
            Name: "jwt",
            Value: tokenString,
            Path: "/", 
            //HttpOnly: true,
            //Secure: true, 
            //Expires: expirationTime,
            //MaxAge: 90000,
        }

    return &httpCookie, nil
}

func ContextWithUserCtx(ctx context.Context) context.Context {
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
    } else if claims[tkMaster.tokenClaim] == nil {
        err = errors.New("auth: user claim is invalid")
    }

    if err == nil {
        userCtx := model.UserCtx{}
        uRaw, err := json.Marshal(claims[tkMaster.tokenClaim])
        if err != nil {
            panic(err)
        }
        json.Unmarshal(uRaw, &userCtx)
        // @DEBUG: mapstructure seems to not decode enum (RoleType) !
        //mapstructure.Decode(claims[tkMaster.tokenClaim], &userCtx)
        ctx = context.WithValue(ctx, tkMaster.tokenClaim, userCtx)
    } else {
        ctx = context.WithValue(ctx, tkMaster.tokenClaimErr, err)
    }
    return ctx
}

func UserCtxFromContext(ctx context.Context) (model.UserCtx, error) {
    userCtxErr := ctx.Value(tkMaster.tokenClaimErr)
    if userCtxErr != nil {
        return model.UserCtx{}, userCtxErr.(error)
    } 
    return ctx.Value(tkMaster.tokenClaim).(model.UserCtx), nil

}
