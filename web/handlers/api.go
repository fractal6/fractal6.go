package handlers

import (
    //"fmt"
    "net/http"
    "encoding/json"

    "zerogov/fractal6.go/db"
)


// Signup register a new user and gives it a token.
func SubChildren(w http.ResponseWriter, r *http.Request) {
	var q string

	// Get the JSON body and decode into UserCreds
	err := json.NewDecoder(r.Body).Decode(&q)
	if err != nil {
		// Body structure error
        http.Error(w, err.Error(), 400)
		return
	}

    // Get sub children
    DB := db.GetDB()
    data, err := DB.GetAllChildren("nameid", q)
    if err != nil {
        http.Error(w, err.Error(), 500)
		return
    }


    // Return the user context
    jsonData, err := json.Marshal(data)
    if err != nil {
        http.Error(w, err.Error(), 500)
		return
    }
    w.Write(jsonData)
}

// Signup register a new user and gives it a token.
func SubMembers(w http.ResponseWriter, r *http.Request) {
	var q string

	// Get the JSON body and decode into UserCreds
	err := json.NewDecoder(r.Body).Decode(&q)
	if err != nil {
		// Body structure error
        http.Error(w, err.Error(), 400)
		return
	}

    // Get sub children
    DB := db.GetDB()
    data, err := DB.GetAllMembers("nameid", q)
    if err != nil {
        http.Error(w, err.Error(), 500)
		return
    }


    // Return the user context
    jsonData, err := json.Marshal(data)
    if err != nil {
        http.Error(w, err.Error(), 500)
		return
    }
    w.Write(jsonData)
}
