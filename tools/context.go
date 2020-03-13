package tools

import (
    "context"
    "net/http"
    "io/ioutil"
    "bytes"
    //"github.com/labstack/echo/v4"
    //"github.com/go-chi/chi"
)


func RequestContextMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        ctx := r.Context()
        if r.Method == "POST" {
            // for content-type != application/json
            //r.ParseForm()
            //fmt.Println(r.Proto)
            //fmt.Println(r.URL)
            //fmt.Println(r.Header)
            //fmt.Println(r.Body)
            //fmt.Println(r.Form)
            //fmt.Println(r.Form.Encode())

            body, _ := ioutil.ReadAll(r.Body)
            // Restore the io.ReadCloser to its original state
             r.Body = ioutil.NopCloser(bytes.NewBuffer(body))
            // Forward body in context
            ctx = context.WithValue(ctx, "request_body", body)
        }
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}


//type GlobalContext struct {
//    echo.Context
//    ctx    context.Context
//}
//
//func EchoContextToContextMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
//	return func(c echo.Context) error {
//		ctx := context.WithValue(c.Request().Context(), "RouterContextKey", c)
//		c.SetRequest(c.Request().WithContext(ctx))
//		return next(&GlobalContext{c, ctx})
//	}
//}
//
//func EchoContextFromContext(ctx context.Context) (echo.Context, error) {
//	routerContext := ctx.Value("RouterContextKey")
//	if routerContext == nil {
//		err := fmt.Errorf("could not retrieve router Context")
//		return nil, err
//	}
//
//	ec, ok := routerContext.(echo.Context)
//	if !ok {
//		err := fmt.Errorf("router Context has wrong type")
//		return nil, err
//	}
//	return ec, nil
//}

