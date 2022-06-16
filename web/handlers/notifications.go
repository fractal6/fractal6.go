package handlers

import (
    "fmt"
    "net/http"
    "encoding/json"

    //"zerogov/fractal6.go/db"
    //"zerogov/fractal6.go/web/auth"
    //"zerogov/fractal6.go/graph"
    //"zerogov/fractal6.go/graph/model"
    //. "zerogov/fractal6.go/tools"
)


// Signup register a new user and gives it a token.
func Notifications(w http.ResponseWriter, r *http.Request) {

    // Get request form
    var form map[string]interface{}
    err := json.NewDecoder(r.Body).Decode(&form)
	if err != nil { http.Error(w, err.Error(), 400); return }

    // return result on success
    data, _ := json.Marshal(form)
    fmt.Println(form)
    w.Write(data)
}
