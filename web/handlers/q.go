package handlers

import (
    //"fmt"
    "net/http"
    "encoding/json"

    "zerogov/fractal6.go/db"
    webauth "zerogov/fractal6.go/web/auth"
)

//
// Query data
// @Todo: token and check private status
//

func SubNodes(w http.ResponseWriter, r *http.Request) {
	var q string

	// Get the JSON body and decode into UserCreds
	err := json.NewDecoder(r.Body).Decode(&q)
	if err != nil {
		// Body structure error
        http.Error(w, err.Error(), 400)
		return
	}

    // Get sub children
    data, err := db.GetDB().GetSubNodes("nameid", q)
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
    data, err := db.GetDB().GetSubMembers("nameid", q)
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

func TopLabels(w http.ResponseWriter, r *http.Request) {
	var q string

	// Get the JSON body and decode into UserCreds
	err := json.NewDecoder(r.Body).Decode(&q)
	if err != nil {
		// Body structure error
        http.Error(w, err.Error(), 400)
		return
	}

    // Get top labels
    data, err := db.GetDB().GetTopLabels("nameid", q)
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
    data, err := db.GetDB().GetSubLabels("nameid", q)
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

func TopRoles(w http.ResponseWriter, r *http.Request) {
	var q string

	// Get the JSON body and decode into UserCreds
	err := json.NewDecoder(r.Body).Decode(&q)
	if err != nil {
		// Body structure error
        http.Error(w, err.Error(), 400)
		return
	}

    // Get top labels
    data, err := db.GetDB().GetTopRoles("nameid", q)
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

func SubRoles(w http.ResponseWriter, r *http.Request) {
	var q string

	// Get the JSON body and decode into UserCreds
	err := json.NewDecoder(r.Body).Decode(&q)
	if err != nil {
		// Body structure error
        http.Error(w, err.Error(), 400)
		return
	}

    // Get sub labels
    data, err := db.GetDB().GetSubRoles("nameid", q)
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

    // Filter the nameids according to the @auth directives
    uctx := webauth.GetUserContextOrEmpty(r.Context())
    newNameids, err := db.GetDB().QueryAuthFilter(uctx, "node", "nameid", q.Nameids)
    if err != nil {
        http.Error(w, err.Error(), 500)
		return
    }
    q.Nameids = newNameids

    // Get Int Tensions
    data, err := db.GetDB().GetTensions(q, "int")
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

    // Get Ext Tensions
    data, err := db.GetDB().GetTensions(q, "ext")
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

    // Filter the nameids according to the @auth directives
    uctx := webauth.GetUserContextOrEmpty(r.Context())
    newNameids, err := db.GetDB().QueryAuthFilter(uctx, "node", "nameid", q.Nameids)
    if err != nil {
        http.Error(w, err.Error(), 500)
		return
    }
    q.Nameids = newNameids

    // Get all tensions
    data, err := db.GetDB().GetTensions(q, "all")
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

    // Get tension counts
    data, err := db.GetDB().GetTensionsCount(q)
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
