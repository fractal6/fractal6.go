package tools

import (
    "golang.org/x/crypto/bcrypt" 
)

//
// Crypt utils
//

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
