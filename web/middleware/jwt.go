package middleware

import (
	"fmt"
    "context"
    "errors"
    "net/http"
    "github.com/go-chi/jwtauth"

    //"zerogov/fractal6.go/web/auth"
)

func JwtDecode(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        ctx := r.Context()
		token, claims, err := jwtauth.FromContext(ctx)

		if err != nil {
            //errMsg := fmt.Errorf("%v", err)
            switch err {
            case jwtauth.ErrUnauthorized:
            case jwtauth.ErrExpired:
            case jwtauth.ErrNBFInvalid:
            case jwtauth.ErrIATInvalid:
            case jwtauth.ErrNoTokenFound:
            case jwtauth.ErrAlgoInvalid:
            }
            //http.Error(w, http.StatusText(401), 401)
			//return
		} else if token == nil || !token.Valid {
            err = errors.New("jwtauth: token is invalid")
			//http.Error(w, http.StatusText(401), 401)
			//return
        }

        if err == nil {
            fmt.Println("Got jwt claims:", claims)
            userCtx := claims
            ctx = context.WithValue(ctx, "user_context", userCtx)
        } else {
            fmt.Println("Got jwt error:", err)
        }

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
