package tools

import (
    "testing"
    "fmt"
    "encoding/base64"
    "crypto"
    "crypto/rsa"
    "crypto/sha1"
)

func TestValidatePostalSignature(t *testing.T) {
    postalWebhookPK := "MIGfMA0GCSqGSIb3DQEBAQUAA4GNADCBiQKBgQClPnCZ+8y6rO/zvNwsKF4vYKnEU4urIAzeOfnn8DoADtUP91UhsGkWvlludEBIa9xWtI+tXptadf7MYVqwXxKfTdj4H2tQXGdHnvSxLAK6jDEYTNVmed2y6XXwCMql87JLoDOiYXl9BiIyNbmWZfGmzAO5pt1qQFO5+1m77VC4rwIDAQAB" // Just the p=... part of the TXT record (without the semicolon at the end)

	// convert postal public key to PEM (X.509) format
    publicKeyPem :=  "-----BEGIN PUBLIC KEY-----\r\n" +
	ChunkSplit(postalWebhookPK, 64, "\r\n") +
	"-----END PUBLIC KEY-----"
    publicKey := ParseRsaPublic(publicKeyPem)

    rawSignature := "hB8VE3R/7HX9sUgOGU2eIKq3+7oz5YGh4BSTvHRoKr8JMQBg8nAz0zkftxa8y6R/6sE1uA/Ztjx47/0Jpxv68JoaP28OmEpYVoStoL820njcNWUSeQ1ZjTt7YO3dYfstt4Dm5W7QyAICHAMMzXJL8+zRi9DWsNYEFsgpi5lBVv4="
    signature, err := base64.StdEncoding.DecodeString(rawSignature)
    if err != nil { t.Errorf(err.Error()) }

    //body, err := ioutil.ReadAll(r.Body)
    body := []byte("test")
    if err != nil { t.Errorf(err.Error()) }
    hash := sha1.Sum(body)

    fmt.Println(publicKey)
    fmt.Println(publicKeyPem)
    err = rsa.VerifyPKCS1v15(publicKey, crypto.SHA1, hash[:], signature)

    if publicKey == nil {
        t.Errorf("bad value")
    }

 }
