package auth

import (
    //"fmt"
    "errors"
    "golang.org/x/crypto/bcrypt" 

    "zerogov/fractal6.go/db"
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
    ErrBadPassword = errors.New(`{
        "user_ctx":{
            "field": "password",
            "msg":"Bad Password""
        }
    }`)
)


//
// User Data structure
//

// UserCtx are data encoded in the thoken (e.g Jwt claims)
type UserCtx struct {
    Username string `json:"User.username"`
    Name string     `json:"User.name"`
    Passwd string   `json:"User.password"`
	Roles []Role    `json:"User.roles"`
}
type Role struct {
    Nameid string    `json:"Node.nameid"` // the node identifier
    RoleType string  `json:"Node.role_type"` // the role type
}

//
// Public methods
//

// UserCreds are data sink for login request
type UserCreds struct {
    Username string `json:"username"`
    Email string `json:"email"`
    Password string `json:"password"`
}

// GetUser returns the user ctx **if they are authencitated** against their
// hashed password.
func GetUserCtx(creds UserCreds) (*UserCtx, error) {
    // 1. get username or email
    // 2. if not present, throw error 
    // 3. compare pasword
    // 4. if good, return UsertCtx from db request, else throw error
    
    var fieldId string
    var userId string
    var userCtx UserCtx

    username := creds.Username
    email := creds.Email
    password := creds.Password

    if password == "" {
        return nil, ErrBadPassword
    } else if username != "" {
        fieldId = "username"
        userId = username
    } else if email != "" {
        fieldId = "email"
        userId = email
    } else {
        return nil, ErrBadUsername
    }

    DB := db.GetDB()
    //
    // @TODO: check if user exists
    //        before getting it.
    //
    err := DB.GetUser(fieldId, userId, &userCtx)
    if err != nil {
        return nil, err 
    }

    // Compared hashed password:
    ok := VerifyPassword(userCtx.Passwd, password)
    if !ok {
        return nil, ErrBadPassword
    }

    userCtx.Passwd = ""
    return &userCtx, nil
}

// HashPassword generates a hash using the bcrypt.GenerateFromPassword
func HashPassword(password string) string {
    hash, err := bcrypt.GenerateFromPassword([]byte(password), 10)
    if err != nil {
        panic(err)
    }
    return string(hash)
}

// VerifyPassword compares the hash of a password with the password
func VerifyPassword(hash string, password string) bool {
    if len(password) == 0 || len(hash) == 0 {
        return false
    }

    err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
    return err == nil
}




