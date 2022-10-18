package tools

import (
    "testing"
    "encoding/base64"
    "crypto"
    "crypto/rsa"
    "crypto/sha1"
)

func TestValidatePostalSignature(t *testing.T) {
    postalWebhookPK := "MIGfMA0GCSqGSIb3DQEBAQUAA4GNADCBiQKBgQDu5GsIYzim3VVk7AY5cX7LkNoLK6LS7BYfjzmOHXuiNlOgkjKJEQfz/WkKcBGAYJfxIr5153ahbHk66eyJ+cQZZ5g1DJrxM/+VgEgGty8g8yS5wGv003wMgmCjz210Ur+03+9TbK1/FsrGc2feI18mAJ+9RwDzlWaKBpM+GZWDbwIDAQAB"

    // convert postal public key to PEM (X.509) format
    publicKeyPem :=  "-----BEGIN PUBLIC KEY-----\r\n" +
    ChunkSplit(postalWebhookPK, 64, "\r\n") +
    "-----END PUBLIC KEY-----"
    publicKey := ParseRsaPublic(publicKeyPem)

    rawSignature := "Y7gFPHXd5m+76qvgpke9xYmmkX/fWxheNOIyGJjBYNwZNdz8V+FCbAZyZPJi7hxqapcXHcC8cmF7h/SSGxXX/pQiNYMGvjf9mKPHidOKJeHn2sn+y8cZchqgzAtCcaj4UkvlHrwp6Xr9VT7rzhR+n7OtReFHaCbdLBa032/02UQ="
    signature, err := base64.StdEncoding.DecodeString(rawSignature)
    if err != nil { t.Errorf(err.Error()) }

    body := []byte(`{"id":935,"rcpt_to":"lectures@fractale.co","mail_from":"ypfwdy@psrp.fractale.co","token":"u3YnniaCoITx","subject":"Test","message_id":"0ee4b034-b7c6-4a0c-8e28-51b7cf138c4e@rp.postal.fractale.co","timestamp":1666059101.23531,"size":"1035","spam_status":"NotSpam","bounce":false,"received_with_ssl":true,"to":"lectures@fractale.co","cc":null,"from":"test@fractale.co","date":"Tue, 18 Oct 2022 02:11:41 +0000","in_reply_to":null,"references":null,"html_body":null,"attachment_quantity":0,"auto_submitted":null,"reply_to":null,"plain_body":"test\n","attachments":[]}`)
    if err != nil { t.Errorf(err.Error()) }
    hash := sha1.Sum(body)

    err = rsa.VerifyPKCS1v15(publicKey, crypto.SHA1, hash[:], signature)

    if err != nil {
        t.Errorf(err.Error())
    }

}
