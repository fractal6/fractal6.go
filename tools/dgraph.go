package tools

import (
    "net/http"
    "bytes"
    "io/ioutil"
)

// Dgraph graphql client from scratch
type Dgraph struct {
    Url string
}

func (d Dgraph) Request(body []byte) []byte {
    req, err := http.NewRequest("POST", d.Url, bytes.NewBuffer(body))
    req.Header.Set("Content-Type", "application/json")

    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        panic(err)
    }
    defer resp.Body.Close()

    res, _ := ioutil.ReadAll(resp.Body)
    return res
}



