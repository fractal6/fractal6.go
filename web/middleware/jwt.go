package middleware

import (
    "net/http"

    "zerogov/fractal6.go/web/auth"
)

func JwtDecode(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        ctx := r.Context()
        ctx = auth.GetAuthContext(ctx)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
