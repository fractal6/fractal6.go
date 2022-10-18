package tools

import (
    "net/http"
    "io/ioutil"
    "encoding/base64"
    "encoding/pem"
    "crypto"
    "crypto/rsa"
    "crypto/x509"
    "crypto/sha1"
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
    key_priv, _ := x509.ParsePKCS1PrivateKey(block.Bytes)
    return key_priv
}

func ParseRsaPublic(key string) *rsa.PublicKey {
    block, _ := pem.Decode([]byte(key))
    if block == nil { panic("failed to parse PEM block for public key") }
    key_pub, _ := x509.ParsePKCS1PublicKey(block.Bytes)
    return key_pub
}

//
// Signature Check
// --
// For Postal, see https://github.com/postalserver/postal/issues/432
//

func ValidatePostalSignature(r *http.Request) error {
    postalWebhookPK := "MIGfMA0GCSqGSIb3DQEBAQUAA4GNADCBiQKBgQClPnCZ+8y6rO/zvNwsKF4vYKnEU4urIAzeOfnn8DoADtUP91UhsGkWvlludEBIa9xWtI+tXptadf7MYVqwXxKfTdj4H2tQXGdHnvSxLAK6jDEYTNVmed2y6XXwCMql87JLoDOiYXl9BiIyNbmWZfGmzAO5pt1qQFO5+1m77VC4rwIDAQAB" // Just the p=... part of the TXT record (without the semicolon at the end)

	// convert postal public key to PEM (X.509) format
    publicKeyPem :=  "-----BEGIN PUBLIC KEY-----\r\n" +
	ChunkSplit(postalWebhookPK, 64, "\r\n") +
	"-----END PUBLIC KEY-----"
    publicKey := ParseRsaPublic(publicKeyPem)

    rawSignature := r.Header.Get("X-Postal-Signature")
    signature, err := base64.StdEncoding.DecodeString(rawSignature)
    if err != nil { return err }

    body, err := ioutil.ReadAll(r.Body)
    if err != nil { return err }
    hash := sha1.Sum(body)

    err = rsa.VerifyPKCS1v15(publicKey, crypto.SHA1, hash[:], signature);
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
