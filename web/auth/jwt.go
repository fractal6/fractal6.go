package auth

import (
    "fmt"
    "os"
    "time"
    "errors"
    "context"
    "encoding/json"
    "net/http"
    "github.com/go-chi/jwtauth/v5"
    "github.com/lestrrat-go/jwx/jwt"
	"github.com/spf13/viper"

    . "fractale/fractal6.go/tools"
	"fractale/fractal6.go/graph/model"
	"fractale/fractal6.go/db"
	"fractale/fractal6.go/web/sessions"

)

var buildMode string
var cache *sessions.Session
var tkMaster *Jwt
var jwtSecret string

func init () {
    // Get env mode
    if buildMode != "PROD" {
        buildMode = "DEV"
    }

    // Initializa cache
    cache = sessions.GetCache()

    // Get Jwt private key
    jwtSecret = viper.GetString("server.jwt_secret")
    if jwtSecret == "" {
        jwtSecret = os.Getenv("JWT_SECRET")
        if jwtSecret == "" {
            fmt.Println("JWT_SECRET not found. JWT token disabled.")
            return
        }
    }

    // Init token master
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
	tk := &Jwt{
        tokenClaim: "user_ctx",
        tokenClaimErr: "user_ctx_err",
		tokenAuth: jwtauth.New("HS256", []byte(jwtSecret), nil),
	}

    if buildMode == "DEV" {
        // Api debug token

        uctx := db.DB.GetRootUctx()
        //uctx := db.DB.GetRegularUctx()

        o := model.RoleTypeOwner
        uctx.Roles = []*model.Node{&model.Node{Nameid: "f6", RoleType: &o}}
        apiToken, _ := tk.issue(uctx, time.Hour*48)

        // Dgraph token
        dgraphToken := db.GetDB().BuildGqlToken(uctx, time.Hour*48)

        // Log
        fmt.Println("Api token:", Unpack64(apiToken))
        fmt.Println("Dgraph token:", dgraphToken)
    }

	return tk
}

func (tk Jwt) GetAuth() *jwtauth.JWTAuth {
    return tk.tokenAuth
}

// Issue generate and encode a new token
func (tk *Jwt) issue(d model.UserCtx, t time.Duration) (string, error) {
    claims := map[string]interface{}{ tk.tokenClaim: d }
    jwtauth.SetIssuedNow(claims)
    jwtauth.SetExpiry(claims, time.Now().UTC().Add(t))
	_, token, err := tk.tokenAuth.Encode(claims)
	return Pack64(token), err
}

//
// Global functions
//

func GetTokenMaster() *Jwt {
    return tkMaster
}

// NewUserToken create a new user token from master key
func NewUserToken(userCtx model.UserCtx) (string, error) {
    var token string
    var err error
    if buildMode == "PROD" {
        token, err = tkMaster.issue(userCtx, time.Hour*24*30)
    } else {
        token, err = tkMaster.issue(userCtx, time.Hour*12)
        //token, err = tkMaster.issue(userCtx, time.Second*60)
    }
    return token, err
}

// NexuserCookie create an http cookie that embed a token
func NewUserCookie(userCtx model.UserCtx) (*http.Cookie, error) {

    // Erase growing value
    userCtx.Roles = nil
    // Ignore internal Hit value
    userCtx.Hit = 0

    token, err := NewUserToken(userCtx)
    if err != nil {
        return nil, err
    }
    var httpCookie http.Cookie
    if buildMode == "PROD" {
        httpCookie = http.Cookie{
            Name: "jwt",
            Value: token,
            Path: "/",
            HttpOnly: true,
            Secure: true,
            SameSite: 2, // https://golang.org/src/net/http/cookie.go
            //Expires: expirationTime,
            //MaxAge: 90000,
        }
    } else {
        httpCookie = http.Cookie{
            Name: "jwt",
            Value: token,
            Path: "/",
            Secure: false,
            SameSite: 2,
        }
    }

    return &httpCookie, nil
}

func ContextWithUserCtx(ctx context.Context) (context.Context, error) {
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
    } else if token == nil || jwt.Validate(token) != nil {
        err = errors.New("jwtauth: token is invalid")
    } else if claims[tkMaster.tokenClaim] == nil {
        err = errors.New("auth: user claim is invalid")
    }

    if err != nil { // Set the user error token
        ctx = context.WithValue(ctx, tkMaster.tokenClaimErr, err)
    }

    // Set the user token
    userCtx := model.UserCtx{}
    uRaw, e := json.Marshal(claims[tkMaster.tokenClaim])
    if e != nil { panic(e) }
    json.Unmarshal(uRaw, &userCtx)
    ctx = context.WithValue(ctx, tkMaster.tokenClaim, &userCtx)

    // Set the Iat
    if claims["iat"] != nil {
        //iat := claims["iat"].(int64)
        //return context.WithValue(ctx, "iat", time.Unix(iat, 0).Format(time.RFC3339)), err
        iat := claims["iat"].(time.Time)
        return context.WithValue(ctx, "iat", iat.Format(time.RFC3339)), err
    }
    //LogErr("jwt error", fmt.Errorf("Can't set the iat jwt claims. This would breaks the user context synchronisation logics."))
    return ctx, err
}

func GetUserContext(ctx context.Context) (context.Context, *model.UserCtx, error) {
    uctx, err := GetUserContextLight(ctx)
    if err != nil { return ctx,  uctx, err }

    // Set the roles and report.
    if uctx.Hit == 0 {
        uctx, err := MaybeRefresh(uctx)
        if err != nil { return ctx, uctx, err }
        ctx = context.WithValue(ctx, tkMaster.tokenClaim, uctx)
    }

    return ctx, uctx, nil
}

func GetUserContextLight(ctx context.Context) (*model.UserCtx, error) {
    uctx := ctx.Value(tkMaster.tokenClaim).(*model.UserCtx)
    userCtxErr := ctx.Value(tkMaster.tokenClaimErr)
    if userCtxErr != nil { return uctx, userCtxErr.(error) }

    uctx.Iat = ctx.Value("iat").(string)
    return uctx, nil
}

func GetUserContextOrEmpty(ctx context.Context) model.UserCtx {
    _, uctx, err := GetUserContext(ctx)
    if err != nil { return model.UserCtx{} }
    return *uctx
}

func GetUserContextOrEmptyLight(ctx context.Context) model.UserCtx {
    uctx, err := GetUserContextLight(ctx)
    if err != nil { return model.UserCtx{} }
    return *uctx
}

// CheckUserCtxIat update the user token if the given
// node has been updated after that the token's iat.
// @deprecated: has been replaced by MaybeRefresh in resolvers
func CheckUserCtxIat(uctx *model.UserCtx, nid string) (*model.UserCtx, error) {
    var u *model.UserCtx
    var e error
    var updatedAt string

    // Pass for ROOT user
    if uctx.Rights.Type == model.UserTypeRoot {
        return uctx, e
    }

    // Check if User context need to be updated
    for _, v := range uctx.CheckedNameid {
        if v == nid {
            return uctx, e
        }
    }

    // Check last node update date
    DB := db.GetDB()
    updatedAt_, e := DB.GetFieldByEq("Node.nameid", nid, "Node.updatedAt")
    if e != nil { return uctx, e }
    if updatedAt_ == nil {
        updatedAt = uctx.Iat
    } else {
        updatedAt = updatedAt_.(string)
    }

    // Update User context if node is newer
    if IsOlder(uctx.Iat, updatedAt) {
        u, e = DB.GetUctx("username", uctx.Username)
        // @DEBUG: UserCtx is update from their fields for propagation
        uctx.Username = u.Username
        uctx.Name     = u.Name
        uctx.Password = u.Password
        uctx.Rights   = u.Rights
        uctx.Roles    = u.Roles
        uctx.CheckedNameid = u.CheckedNameid
        if e != nil { return uctx, e }
        //return nil, fmt.Errorf("refresh token")
    }
    uctx.Hit++
    uctx.CheckedNameid = append(uctx.CheckedNameid, nid)
    return uctx, e
}

func MaybeRefresh(uctx *model.UserCtx) (*model.UserCtx, error) {
    ctx := context.Background()
    var roles []*model.Node
    var err error
    var key string = uctx.Username + "roles"

    if uctx.Hit > 0 {
        //1. Is fresh data
        return uctx, nil
    } else if d, err := cache.Get(ctx, key).Bytes(); err == nil && len(d) != 0 && !uctx.NoCache {
        //2. Check the cache
        err = json.Unmarshal(d, &roles)
        if err != nil { return nil, err }
    } else {
        //3. Query the database
        roles, err = db.GetDB().GetUserRoles(uctx.Username)
        if err != nil { return nil, err }
        d, _ := json.Marshal(roles)
        err = cache.SetEX(ctx, key, d, time.Second * 10).Err()
        if err != nil { return nil, err }
    }

    // Return a updated version of *Uctx
    uctx.Roles = roles
    uctx.Hit++
    return uctx, err
}
