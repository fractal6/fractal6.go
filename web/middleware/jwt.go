package middleware

import (
    "net/http"
    "github.com/go-chi/jwtauth/v5"
    "zerogov/fractal6.go/web/auth"
    "zerogov/fractal6.go/tools"
)


// Verifier http middleware handler will verify a JWT string from a http request.
//
// Verifier will search for a JWT token in a http request, in the order:
//   1. 'jwt' URI query parameter
//   2. 'Authorization: BEARER T' request header
//   3. Cookie 'jwt' value
//
// The first JWT string that is found as a query parameter, authorization header
// or cookie header is then decoded by the `jwt-go` library and a *jwt.Token
// object is set on the request context. In the case of a signature decoding error
// the Verifier will also set the error on the request context.
//
// The Verifier always calls the next http handler in sequence, which can either
// be the generic `jwtauth.Authenticator` middleware or your own custom handler
// which checks the request context jwt token and error to prepare a custom
// http response.
func JwtVerifier(ja *jwtauth.JWTAuth) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return jwtauth.Verify(ja, jwtauth.TokenFromHeader, TokenFromCookie)(next)
    }
}


// TokenFromCookie tries to retreive the token string from a cookie named "jwt".
// EDIT: Uncompress the cookie token.
func TokenFromCookie(r *http.Request) string {
	cookie, err := r.Cookie("jwt")
	if err != nil { return "" }
	return tools.Unpack64(cookie.Value)
}


func JwtDecode(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        ctx, err := auth.ContextWithUserCtx(r.Context())
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
