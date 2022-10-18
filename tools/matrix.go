package tools

import (
    "fmt"
    "time"
    "bytes"
    "strconv"
    "net/http"
    "io/ioutil"
)

type MatrixError struct {
    Errcode string `json:"errcode"`
    Error string   `json:"error"`
}

// Send a JSON formatted string to a matrix room
func MatrixJsonSend(body, roomid, access_token string) error {
    // Pretiffy JSON string
    data, err := PrettyString(string(body))
    if err != nil { return err }

    // Send message to matrix room
    // @debug: triple backquote doesnt work with matrix; How to encode backquote ???
    //data = ```json\n" + QuoteString(data) + "\n```",
    data = QuoteString(data)
    data = fmt.Sprintf(`{
        "msgtype":"m.text",
        "body":"%s",
        "format": "org.matrix.custom.html",
        "formatted_body": "<pre><code class=\"language-json\">%s</code></pre>"
    }`, data, data)
    txnId := strconv.FormatInt(time.Now().UnixNano() / 1000000, 10)
    matrix_url := fmt.Sprintf(
        "https://matrix.org/_matrix/client/r0/rooms/%s/send/m.room.message/%s?access_token=%s",
        roomid,
        txnId,
        access_token)
    req, err := http.NewRequest("PUT", matrix_url, bytes.NewBuffer([]byte(data)))
    if err != nil { return err }
    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil { return err }
    defer resp.Body.Close()
    if resp.StatusCode != 200 {
        b, _ := ioutil.ReadAll(resp.Body)
        fmt.Println(string(b))
        return fmt.Errorf("http matrix error, see body. (code %s)", resp.Status)
    }

    return nil
}
