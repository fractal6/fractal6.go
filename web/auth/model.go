package auth

import (
    //"fmt"
    "time"
    "errors"
    "strings"

    "zerogov/fractal6.go/db"
    "zerogov/fractal6.go/tools"
    "zerogov/fractal6.go/graph/model"
)

// Library errors
var (
    ErrBadNameidFormat = errors.New(`{
        "errors":[{
            "message":"Please enter a valid name",
            "location": "nameid"
        }]
    }`)
    ErrBadUsernameFormat = errors.New(`{
        "errors":[{
            "message":"Please enter a valid username",
            "location": "username"
        }]
    }`)
    ErrUsernameTooLong = errors.New(`{
        "errors":[{
            "message":"Username too long",
            "location": "username"
        }]
    }`)
    ErrUsernameTooShort = errors.New(`{
        "errors":[{
            "message":"Username too short",
            "location": "username"
        }]
    }`)
    ErrBadNameFormat = errors.New(`{
        "errors":[{
            "message":"Please enter a valid name",
            "location": "name"
        }]
    }`)
    ErrNameTooLong = errors.New(`{
        "errors":[{
            "message":"Name too long",
            "location": "name"
        }]
    }`)
    ErrNameTooShort = errors.New(`{
        "errors":[{
            "message":"Name too short",
            "location": "name"
        }]
    }`)
    ErrBadEmailFormat = errors.New(`{
        "errors":[{
            "message":"Please enter a valid email",
            "location": "email"
        }]
    }`)
    ErrEmailTooLong = errors.New(`{
        "errors":[{
            "message":"Email too long",
            "location": "name"
        }]
    }`)
    ErrBadPassword = errors.New(`{
        "errors":[{
            "message":"Bad Password",
            "location": "password"
        }]
    }`)
    ErrPasswordTooShort = errors.New(`{
        "errors":[{
            "message":"Password too short",
            "location": "password"
        }]
    }`)
    ErrPasswordTooLong = errors.New(`{
        "errors":[{
            "message":"Password too long",
            "location": "password"
        }]
    }`)
    // Upsert error
    ErrUsernameExist = errors.New(`{
        "errors":[{
            "message":"Username already exists",
            "location": "username"
        }]
    }`)
    ErrEmailExist = errors.New(`{
        "errors":[{
            "message":"Email already exists",
            "location": "email"
        }]
    }`)
    // User Rights
    ErrCantLogin = errors.New(`{
        "errors":[{
            "message": "You are not authorized to login",
            "location": ""
        }]
    }`)
)

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
    err := ValidatePassword(password)
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
    DB := db.GetDB()
    userCtx, err := DB.GetUser(fieldId, userId)
    if err != nil {
        return nil, err
    }

    // Compare hashed password.
    ok := tools.VerifyPassword(userCtx.Passwd, password)
    if !ok {
        return nil, ErrBadPassword
    }
    // Hide the password !
    userCtx.Passwd = ""
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
    }
    // Email validation
    err = ValidateEmail(email)
    if err != nil {
        return err
    }
    // Password validation
    err = ValidatePassword(password)
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
    // TODO: password complexity check

    DB := db.GetDB()

    // Chech username existence
    ex1, err1 := DB.Exists("User", "username", username)
    if err1 != nil {
        return err1
    }
    if ex1 {
        return ErrUsernameExist
    }
    // Chech email existence
    ex2, err2 := DB.Exists("User", "email", email)
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
    // Rights
    canLogin := true
    canCreateRoot := false

    userInput := model.AddUserInput{
        CreatedAt:      time.Now().Format(time.RFC3339),
        Username:       creds.Username,
        Email:          creds.Email,
        //EmailHash:      *string,
        EmailValidated: false,
        Name:           creds.Name,
        Password:       tools.HashPassword(creds.Password),
        Rights: &model.UserRightsRef{
            CanLogin: &canLogin,
            CanCreateRoot: &canCreateRoot,
        },
        //Utc            *string
    }

    DB := db.GetDB()
    // @DEBUG: ensure that dgraph graphql add requests are atomic (i.e honor @id field)
    _, err := DB.Add("user", userInput)
    if err != nil {
        return nil, err
    }

    // Try getting usetCtx
    userCtx, err := DB.GetUser("username", creds.Username)
    if err != nil {
        return nil, err
    }

    // Hide the password !
    userCtx.Passwd = ""
    return userCtx, nil
}

// GetAuthUserFromCtx returns the user ctx from a db.grpc request,
// from the given user context.
func GetAuthUserFromCtx(uctx model.UserCtx) (*model.UserCtx, error) {
    fieldId := "username"
    userId := uctx.Username

    // Try getting userCtx
    DB := db.GetDB()
    userCtx, err := DB.GetUser(fieldId, userId)
    if err != nil {
        return nil, err
    }

    // Hide the password !
    userCtx.Passwd = ""
    return userCtx, nil
}
