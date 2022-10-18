package tools

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"io/ioutil"
	"net/http"
	"strings"

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

//
// Crypto/Cypher utils
//


func ParseRsaPrivate(key string) *rsa.PrivateKey {
    block, _ := pem.Decode([]byte(key))
    if block == nil { panic("failed to parse PEM block for private key.") }
    key_priv, err := x509.ParsePKCS1PrivateKey(block.Bytes)
    if err != nil { panic(err.Error()) }
    return key_priv
}

func ParseRsaPublic(key string) (key_pub *rsa.PublicKey) {
    var err error
    block, _ := pem.Decode([]byte(key))
    if block == nil { panic("failed to parse PEM block for public key") }
    if strings.HasPrefix(key, "-----BEGIN PUBLIC KEY") {
        key_pub_, err := x509.ParsePKIXPublicKey(block.Bytes)
        if err != nil { panic(err.Error()) }
        key_pub = key_pub_.(*rsa.PublicKey)
    } else {
        key_pub, err = x509.ParsePKCS1PublicKey(block.Bytes)
        if err != nil { panic(err.Error()) }
    }
    return
}

//
// Signature Check
// --
// For Postal, see https://github.com/postalserver/postal/issues/432
//

func ValidatePostalSignature(r *http.Request, pk string) error {
	// convert postal public key to PEM (X.509) format
    publicKeyPem :=  "-----BEGIN PUBLIC KEY-----\r\n" +
	ChunkSplit(pk, 64, "\r\n") +
	"-----END PUBLIC KEY-----"
    publicKey := ParseRsaPublic(publicKeyPem)

    rawSignature := r.Header.Get("X-Postal-Signature")
    signature, err := base64.StdEncoding.DecodeString(rawSignature)
    if err != nil { return err }

    body, err := ioutil.ReadAll(r.Body)
    if err != nil { return err }
    hash := sha1.Sum(body)

    err = rsa.VerifyPKCS1v15(publicKey, crypto.SHA1, hash[:], signature)
    return err
}

func ChunkSplit(body string, limit int, end string) string {
    if limit < 1 { return body }

	var charSlice []rune

	// push characters to slice
	for _, char := range body {
		charSlice = append(charSlice, char)
	}

	var result string = ""

	for len(charSlice) >= 1 {
		// Convert slice/array back to string but insert end at specified limit
		result = result + string(charSlice[:limit]) + end

		// Discard the elements that were copied over to result
		charSlice = charSlice[limit:]

		// Change the limit to cater for the last few words in charSlice
		if len(charSlice) < limit {
			limit = len(charSlice)
		}
	}

	return result
}
