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
    ErrBadUsername = errors.New(`{
        "user_ctx":{
            "field": "username",
            "msg":"Bad username""
        }
    }`)
    ErrBadEmail = errors.New(`{
        "user_ctx":{
            "field": "email",
            "msg":"Bad email""
        }
    }`)
    ErrBadName = errors.New(`{
        "user_ctx":{
            "field": "name",
            "msg":"Bad name""
        }
    }`)
    ErrBadPassword = errors.New(`{
        "user_ctx":{
            "field": "password",
            "msg":"Bad Password""
        }
    }`)
    ErrUsernameExist = errors.New(`{
        "user_ctx":{
            "field": "username",
            "msg":"Username already exists""
        }
    }`)
    ErrEmailExist = errors.New(`{
        "user_ctx":{
            "field": "email",
            "msg":"Email already exists""
        }
    }`)
    ErrPasswordTooShort = errors.New(`{
        "user_ctx":{
            "field": "password",
            "msg":"Password too short""
        }
    }`)
    ErrPasswordTooLong = errors.New(`{
        "user_ctx":{
            "field": "password",
            "msg":"Password too long""
        }
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
    var userCtx model.UserCtx

    username := creds.Username
    password := creds.Password

    // Validate signin form
    if password == "" {
        return nil, ErrBadPassword
    } else if username != "" {
        if strings.Contains(username, "@") {
            fieldId = "email"
        } else {
            fieldId = "username"
        }
        userId = username
    } else {
        return nil, ErrBadUsername
    }

    // Try getting usetCtx
    DB := db.GetDB()
    err := DB.GetUser(fieldId, userId, &userCtx)
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
    return &userCtx, nil
}


// ValidateNewuser check that an user doesn't exist,
// from a db.grpc request.
func ValidateNewUser(creds model.UserCreds) error {
    username := creds.Username
    email := creds.Email
    name := creds.Name
    password := creds.Password

    // Structure check
    if len(username) < 2 || len(username) > 42 {
        return ErrBadUsername
    }
    if len(email) < 3 || len(email) > 42 {
        return ErrBadEmail
    }
    if !strings.Contains(email, ".") || !strings.Contains(email, "@") {
        return ErrBadEmail
    }
    if name != nil && len(*name) > 100 {
        return ErrBadName
    }
    if len(password) < 8 {
        return ErrPasswordTooShort
    }
    if len(password) > 100 {
        return ErrPasswordTooLong
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
    var userCtx model.UserCtx

    user := model.AddUserInput{                                 
        CreatedAt:      time.Now().Format(time.RFC3339),
        Username:       creds.Username,
        Email:          creds.Email,
        //EmailHash:      *string,
        EmailValidated: false,
        Name:           creds.Name,
        Password:       tools.HashPassword(creds.Password),
        //Roles:          []*NodeRef 
        //BackedRoles:    []*NodeRef 
        //Bio            *string    
        //Utc            *string    
    }

    // @DEBUG: ensure that dgraph graphql add requests are atomic (i.e honor @id field)
    DB := db.GetDB()
    err := DB.AddUser(user, &userCtx)
    if err != nil {
        return nil, err 
    }

    // Hide the password !
    userCtx.Passwd = ""
    return &userCtx, nil
}



