package handlers

import (
    //"fmt"
    "net/http"
    "encoding/json"

    //"zerogov/fractal6.go/graph/model"
    "zerogov/fractal6.go/db"
)

//
// Query data
// @Todo: token and check private status
//

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

func SubMembers(w http.ResponseWriter, r *http.Request) {
	var q string

	// Get the JSON body and decode into UserCreds
	err := json.NewDecoder(r.Body).Decode(&q)
	if err != nil {
		// Body structure error
        http.Error(w, err.Error(), 400)
		return
	}

    // Get sub members
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

func SubLabels(w http.ResponseWriter, r *http.Request) {
	var q string

	// Get the JSON body and decode into UserCreds
	err := json.NewDecoder(r.Body).Decode(&q)
	if err != nil {
		// Body structure error
        http.Error(w, err.Error(), 400)
		return
	}

    // Get sub labels
    DB := db.GetDB()
    data, err := DB.GetAllLabels("nameid", q)
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

//
// Query Tensions
//


func TensionsInt(w http.ResponseWriter, r *http.Request) {
	var q db.TensionQuery

	// Get the JSON body and decode into UserCreds
	err := json.NewDecoder(r.Body).Decode(&q)
	if err != nil {
		// Body structure error
        http.Error(w, err.Error(), 400)
		return
	}

    // Get sub labels
    DB := db.GetDB()
    data, err := DB.GetTensions(q, "int")
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

func TensionsExt(w http.ResponseWriter, r *http.Request) {
	var q db.TensionQuery

	// Get the JSON body and decode into UserCreds
	err := json.NewDecoder(r.Body).Decode(&q)
	if err != nil {
		// Body structure error
        http.Error(w, err.Error(), 400)
		return
	}

    // Get sub labels
    DB := db.GetDB()
    data, err := DB.GetTensions(q, "ext")
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

func TensionsAll(w http.ResponseWriter, r *http.Request) {
	var q db.TensionQuery

	// Get the JSON body and decode into UserCreds
	err := json.NewDecoder(r.Body).Decode(&q)
	if err != nil {
		// Body structure error
        http.Error(w, err.Error(), 400)
		return
	}

    // Get sub labels
    DB := db.GetDB()
    data, err := DB.GetTensions(q, "all")
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

func TensionsCount(w http.ResponseWriter, r *http.Request) {
	var q db.TensionQuery

	// Get the JSON body and decode into UserCreds
	err := json.NewDecoder(r.Body).Decode(&q)
	if err != nil {
		// Body structure error
        http.Error(w, err.Error(), 400)
		return
	}

    // Get sub labels
    DB := db.GetDB()
    data, err := DB.GetTensionsCount(q)
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
