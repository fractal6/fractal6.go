package tools

import (
    "golang.org/x/crypto/bcrypt"
    "crypto/rsa"
    "crypto/x509"
    "encoding/pem"
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

//
// Crypto/Cypher utils
//


func ParseRsaPrivate(key string) *rsa.PrivateKey {
    block, _ := pem.Decode([]byte(key))
    if block == nil { panic("failed to parse PEM block for private key.") }
    key_priv, _ := x509.ParsePKCS1PrivateKey(block.Bytes)
    return key_priv
}

func ParseRsaPublic(key string) *rsa.PublicKey {
    block, _ := pem.Decode([]byte(key))
    if block == nil { panic("failed to parse PEM block for public key") }
    key_pub, _ := x509.ParsePKCS1PublicKey(block.Bytes)
    return key_pub
}

