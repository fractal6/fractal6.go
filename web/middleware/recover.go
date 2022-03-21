package middleware

import (
    "fmt"
    "context"
    "runtime/debug"
    "net/http"
    "github.com/go-chi/chi/v5/middleware"
	"github.com/99designs/gqlgen/graphql"
    "github.com/vektah/gqlparser/v2/gqlerror"
    "fractale/fractal6.go/web/email"
)


// Notifier recoverer
func NotifRecover(info string) {
    if r := recover(); r != nil {
        // Email notification
        fmt.Println("------------------notif recov ------------")
        email.SendMaintainerEmail(
            fmt.Sprintf("[f6-notifier][error][%s] %v", info, r),
            string(debug.Stack()),
        )

        // Log error
        fmt.Printf("error: Recovering from panic (%s): %v\n", info, r)
    }
}

// Graphql api recoverer
func GqlRecover(ctx context.Context, err interface{}) error {
        qn := graphql.GetResolverContext(ctx).Field.Name

        // Email Notification
        fmt.Println("------------------gql recov ------------")
        email.SendMaintainerEmail(
            fmt.Sprintf("[f6-graphql][error] %v", err),
            string(debug.Stack()),
        )

        // Log error
        //fmt.Printf("panic on `%s`:\n%s\n", qn, string(debug.Stack()))
        middleware.PrintPrettyStack(err)

        return gqlerror.Errorf("Internal error on '%s': %v", qn, err)
	}

// Standard server recover middleware
// Extension of https://github.com/go-chi/chi/blob/master/middleware/recoverer.go
// to send notification email.
func Recoverer(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rvr := recover(); rvr != nil {
				if rvr == http.ErrAbortHandler {
					// we don't recover http.ErrAbortHandler so the response
					// to the client is aborted, this should not be logged
					panic(rvr)
				}

                // Email notification
                fmt.Println("------------------rest recov ------------")
                email.SendMaintainerEmail(
                    fmt.Sprintf("[f6-rest][error] %v", rvr),
                    string(debug.Stack()),
                )
                // -----------------

				logEntry := middleware.GetLogEntry(r)
				if logEntry != nil {
					logEntry.Panic(rvr, debug.Stack())
				} else {
					middleware.PrintPrettyStack(rvr)
				}

				w.WriteHeader(http.StatusInternalServerError)
			}
		}()

		next.ServeHTTP(w, r)
	}

	return http.HandlerFunc(fn)
}
