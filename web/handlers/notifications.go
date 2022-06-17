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


// Handle email responses.
func Notifications(w http.ResponseWriter, r *http.Request) {

    // Get request form
    var form map[string]interface{}
    err := json.NewDecoder(r.Body).Decode(&form)
	if err != nil { http.Error(w, err.Error(), 400); return }

    // return result on success
    fmt.Println(form)
}
