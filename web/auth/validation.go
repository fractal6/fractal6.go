package auth

import (
    "fmt"
    "time"
    "errors"
    "strings"
    "context"
    "encoding/json"
	"github.com/spf13/viper"

    "fractale/fractal6.go/db"
    "fractale/fractal6.go/tools"
    "fractale/fractal6.go/graph/model"
)

// Library errors
var (
    ErrUserUnknown = errors.New(`{
        "errors":[{
            "message":"User unknown.",
            "location": "nameid"
        }]
    }`)
    ErrBadNameidFormat = errors.New(`{
        "errors":[{
            "message":"Please enter a valid name.",
            "location": "nameid"
        }]
    }`)
    ErrBadUsernameFormat = errors.New(`{
        "errors":[{
            "message":"Please enter a valid username. Special characters (@:!,?%. etc) are not allowed.",
            "location": "username"
        }]
    }`)
    ErrUsernameTooLong = errors.New(`{
        "errors":[{
            "message":"Username too long.",
            "location": "username"
        }]
    }`)
    ErrUsernameTooShort = errors.New(`{
        "errors":[{
            "message":"Username too short.",
            "location": "username"
        }]
    }`)
    ErrBadNameFormat = errors.New(`{
        "errors":[{
            "message":"Please enter a valid name.",
            "location": "name"
        }]
    }`)
    ErrNameTooLong = errors.New(`{
        "errors":[{
            "message":"Name too long.",
            "location": "name"
        }]
    }`)
    ErrNameTooShort = errors.New(`{
        "errors":[{
            "message":"Name too short.",
            "location": "name"
        }]
    }`)
    ErrBadEmailFormat = errors.New(`{
        "errors":[{
            "message":"Please enter a valid email.",
            "location": "email"
        }]
    }`)
    ErrEmailTooLong = errors.New(`{
        "errors":[{
            "message":"Email too long.",
            "location": "name"
        }]
    }`)
    ErrWrongPassword = errors.New(`{
        "errors":[{
            "message":"Wrong Password.",
            "location": "password"
        }]
    }`)
    ErrPasswordTooShort = errors.New(`{
        "errors":[{
            "message":"Password too short.",
            "location": "password"
        }]
    }`)
    ErrPasswordTooLong = errors.New(`{
        "errors":[{
            "message":"Password too long.",
            "location": "password"
        }]
    }`)
    ErrPasswordRequirements = errors.New(`{
        "errors":[{
            "message":"Password need to contains at least one number and one letter.",
            "location": "password"
        }]
    }`)
    // Upsert error
    ErrUsernameExist = errors.New(`{
        "errors":[{
            "message":"Username already exists.",
            "location": "username"
        }]
    }`)
    ErrEmailExist = errors.New(`{
        "errors":[{
            "message":"Email already exists.",
            "location": "email"
        }]
    }`)
    // User Rights
    ErrCantLogin = errors.New(`{
        "errors":[{
            "message": "You are not authorized to login.",
            "location": ""
        }]
    }`)
)

var clientVersion string
var reservedUsername map[string]bool

func init() {
    clientVersion = viper.GetString("server.client_version")
    reservedUsername = map[string]bool{
        // Reserved email endpoint
        "admin": true,
        "sysadmin": true,
        "alert": true,
        "contact": true,
        "notifications": true,
        "noreply": true,
        "dmarc-reports": true,
        // Reserved URI
        // --
        // back
        "ping": true,
        "playground": true,
        "metrics": true,
        "mailing": true,
        "postal_webhook": true,
        "api": true,
        "auth": true,
        "data": true,
        "static": true,
        "index": true,
        "index.html": true,
        // front
        "new": true, // tension, orga, networks
        "explore": true, // orgas, networks, users
        "login": true,
        "logout": true,
        "signup": true,
        "verification": true,
        "password-reset": true,
        "user": true,
        "users": true,
        "tension": true,
        "tensions": true,
        "org": true,
        "network": true,
    }
}

func regularizeUctx(uctx *model.UserCtx) {
    // Hide the password !
    uctx.Password = ""
    // Set the client version
    uctx.ClientVersion = clientVersion
    // Set the date of expiration (based on the jwt token validity)
    uctx.ExpiresAt = time.Now().Add(tokenValidityTime).UTC().Format(time.RFC3339)
}

//
// Public methods
//

// GetUser returns the user ctx from a db.grpc request,
// **if they are authencitated** against their hashed password.
func GetAuthUserCtx(creds model.UserCreds) (*model.UserCtx, error) {
    // 1. get username/email or throw error
    // 3. if pass compare pasword or throw error
    // 4. if pass, returns UsertCtx from db request or throw error
    var fieldId string
    var userId string

    username := creds.Username
    password := creds.Password

    // Validate signin form
    err := ValidateSimplePassword(password)
    if err != nil {
        return nil, err
    } else if len(username) > 1 {
        if strings.Contains(username, "@") {
            fieldId = "email"
        } else {
            fieldId = "username"
        }
        userId = username
    } else {
        return nil, ErrBadUsernameFormat
    }

    // Try getting usetCtx
    userCtx, err := db.GetDB().GetUctx(fieldId, userId)
    if err != nil {
        return nil, err
    }

    if userCtx.Username == "" {
        return nil, ErrUserUnknown
    }

    // Compare hashed password.
    ok := tools.VerifyPassword(userCtx.Password, password)
    if !ok {
        return nil, ErrWrongPassword
    }

    regularizeUctx(userCtx)
    return userCtx, nil
}

// GetAuthUserFromCtx returns the user ctx from a db.grpc request,
// from the given user context.
func GetAuthUserFromCtx(uctx model.UserCtx) (*model.UserCtx, error) {
    // Try getting userCtx
    userCtx, err := db.GetDB().GetUctx("username", uctx.Username)
    if err != nil {
        return nil, err
    }

    // Update the user roles cache.
    ctx := context.Background()
    var key string = userCtx.Username + "roles"
    d, _ := json.Marshal(userCtx.Roles)
    err = cache.SetEX(ctx, key, d, time.Second * 12).Err()
    if err != nil { return nil, err }

    regularizeUctx(userCtx)
    return userCtx, nil
}

// ValidateNewuser check that an user doesn't exist,
// from a db.grpc request.
func ValidateNewUser(creds model.UserCreds) error {
    username := creds.Username
    email := creds.Email
    name := creds.Name
    password := creds.Password

    // Username validation
    err := ValidateUsername(username)
    if err != nil {
        return err
    } else if reservedUsername[username] {
        return ErrUsernameExist
    }
    // Email validation
    err = ValidateEmail(email)
    if err != nil {
        return err
    }
    // Name validation
    if name != nil {
        err = ValidateName(*name)
        if err != nil {
            return err
        }
    }
    // Password validation
    err = ValidatePassword(password)
    if err != nil {
        return err
    }
    // TODO: password complexity check

    // Check username existence
    ex1, err1 := db.DB.Exists("User.username", username, nil)
    if err1 != nil {
        return err1
    }
    if ex1 {
        return ErrUsernameExist
    }
    // Check email existence
    ex2, err2 := db.DB.Exists("User.email", email, nil)
    if err2 != nil {
        return err2
    }
    if ex2 {
        return ErrEmailExist
    }

    // New user can be created !
    return nil
}

// CreateNewUser Upsert an user,
// using db.graphql request.
func CreateNewUser(creds model.UserCreds) (*model.UserCtx, error) {
    now := tools.Now()
    // Rights
    canLogin := true
    canCreateRoot := false
    maxPublicOrga := 5
    userType := model.UserTypeRegular
    hasEmailNotifications := true

    userInput := model.AddUserInput{
        CreatedAt: now,
        LastAck: now,
        NotifyByEmail: true,
        Lang: model.LangEn,
        Username: creds.Username,
        Email: creds.Email,
        Name: creds.Name,
        Password: creds.Password,
        Rights: &model.UserRightsRef{
            CanLogin: &canLogin,
            CanCreateRoot: &canCreateRoot,
            MaxPublicOrga: &maxPublicOrga,
            Type: &userType,
            HasEmailNotifications: &hasEmailNotifications,
        },
    }

    _, err := db.GetDB().Add(db.GetDB().GetRootUctx(), "user", userInput)
    if err != nil {
        return nil, err
    }

    // Try getting usetCtx
    userCtx, err := db.GetDB().GetUctx("username", creds.Username)
    if err != nil {
        return nil, err
    }

    regularizeUctx(userCtx)
    return userCtx, nil
}

//
// Verify New orga right
//

func CanNewOrga(uctx model.UserCtx, form model.OrgaForm) (bool, error) {
    var ok bool
    var err error

    regex := fmt.Sprintf("@%s$", uctx.Username)
    nodes, err := db.GetDB().GetNodes(regex, true)
    if err != nil {return ok, err}

    switch uctx.Rights.Type {
    case model.UserTypeRegular:
        if len(nodes) >= uctx.Rights.MaxPublicOrga {
            return ok, fmt.Errorf("Number of personnal organisation are limited to %d, please contact us to create more.", uctx.Rights.MaxPublicOrga)
        }

    case model.UserTypePro:
        if len(nodes) >= 100 {
            return ok, fmt.Errorf("You own too many organisation, please contact us to create more.")
        }

    case model.UserTypeRoot:
        // pass

    }

    ok = true
    return ok, err
}

