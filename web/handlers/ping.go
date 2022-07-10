package handlers

import (
    "net/http"
)

// Ping is simple keep-alive/ping handler
func Ping(w http.ResponseWriter, r *http.Request) {
    //user := r.Context().Value("user").(string)
    w.Write([]byte("OK"))
}

