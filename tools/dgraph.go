package tools

import (
    "net/http"
    "bytes"
    "encoding/json"
    //"io/ioutil"
)

// Dgraph graphql client from scratch
type Dgraph struct {
    Url string
}

func (d Dgraph) Request(data []byte, res interface{}) error {
    req, err := http.NewRequest("POST", d.Url, bytes.NewBuffer(data))
    req.Header.Set("Content-Type", "application/json")

    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    // Get the string/byte response
    //res, _ := ioutil.ReadAll(resp.Body)
    return json.NewDecoder(resp.Body).Decode(res)
}



