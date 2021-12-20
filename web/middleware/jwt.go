package middleware

import (
    "net/http"
    "zerogov/fractal6.go/web/auth"
    "zerogov/fractal6.go/web/middleware/jwtauth"
)

func JwtDecode(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        ctx := r.Context()
        ctx, err := auth.ContextWithUserCtx(ctx)
        switch err {
        case jwtauth.ErrExpired:
            // pass for now...
            //http.Error(w, err.Error(), 400)
            //return
        default:
            // pass
        }
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
