package tools

import (
    "testing"
)

func TestValidatePostalSignature(t *testing.T) {
    postalWebhookPK := "MIGfMA0GCSqGSIb3DQEBAQUAA4GNADCBiQKBgQClPnCZ+8y6rO/zvNwsKF4vYKnEU4urIAzeOfnn8DoADtUP91UhsGkWvlludEBIa9xWtI+tXptadf7MYVqwXxKfTdj4H2tQXGdHnvSxLAK6jDEYTNVmed2y6XXwCMql87JLoDOiYXl9BiIyNbmWZfGmzAO5pt1qQFO5+1m77VC4rwIDAQAB" // Just the p=... part of the TXT record (without the semicolon at the end)

	// convert postal public key to PEM (X.509) format
    publicKeyPem :=  "-----BEGIN PUBLIC KEY-----\r\n" +
	ChunkSplit(postalWebhookPK, 64, "\r\n") +
	"-----END PUBLIC KEY-----"
    publicKey := ParseRsaPublic(publicKeyPem)

    if publicKey == nil {
        t.Errorf("bad value")
    }

 }
