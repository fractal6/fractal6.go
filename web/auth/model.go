package auth

import (
    "golang.org/x/crypto/bcrypt" 

    "zerogov/fractal6.go/db"
)

var DB db.Dgraph

func init () {
    DB = db.InitDB()
}

// UserCtx are data encoded in the thoken (e.g Jwt claims)
type UserCtx struct {
    Username string `json:"username"`
    Name string     `json:"name"`
	Roles []Role    `json:"roles"`
}
type Role struct {
    Nameid string    `json:"nameid"` // the node identifier
    RoleType string  `json:"role_type"` // the role type
}

// UserCreds are data sink for login request
type UserCreds struct {
    Username string `json:"username"`
    Email string `json:"email"`
    Password string `json:"password"`
}

func GetUserCtx(creds UserCreds) (UserCtx, error) {
    // 1. get username or email
    // 2. if not present, throw error 
    // 3. compare pasword
    // 4. if good, return UsertCtx from db request, else throw error
    dtest := UserCtx{"dtr", "dtr", []Role{Role{"Coordo", "coordinator"}}}
    return dtest, nil
}

// HashPassword generates a hash using the bcrypt.GenerateFromPassword
func HashPassword(password string) string {
    hash, err := bcrypt.GenerateFromPassword([]byte(password), 10)
    if err != nil {
        panic(err)
    }
    return string(hash)
}

// CheckPassword compares the hash of a password with the password
func CheckPassword(hash string, password string) bool {

    if len(password) == 0 || len(hash) == 0 {
        return false
    }

    err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
    return err == nil
}




