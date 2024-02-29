/*
 * Fractale - Self-organisation for humans.
 * Copyright (C) 2024 Fractale Co
 *
 * This file is part of Fractale.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as
 * published by the Free Software Foundation, either version 3 of the
 * License, or (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with Fractale.  If not, see <http://www.gnu.org/licenses/>.
 */

package middleware

import (
	"context"
	"fmt"
	"fractale/fractal6.go/web/email"
	"github.com/99designs/gqlgen/graphql"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/vektah/gqlparser/v2/gqlerror"
	"net/http"
	"runtime/debug"
)

// Notifier recoverer
func NotifRecover(info string) {
	if r := recover(); r != nil {
		// Email notification
		fmt.Println("------------------notif recov ------------")
		e := email.SendMaintainerEmail(
			fmt.Sprintf("[f6-notifier][error][%s] %v", info, r),
			string(debug.Stack()),
		)
		if e != nil {
			fmt.Println(e)
		}

		// Log error
		fmt.Printf("error: Recovering from panic (%s): %v\n", info, r)
	}
}

// Graphql api recoverer
func GqlRecover(ctx context.Context, err interface{}) error {
	qn := graphql.GetResolverContext(ctx).Field.Name

	// Email Notification
	fmt.Println("------------------gql recov ------------")
	e := email.SendMaintainerEmail(
		fmt.Sprintf("[f6-graphql][error] %v", err),
		string(debug.Stack()),
	)
	if err != nil {
		fmt.Println(e)
	}

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
				e := email.SendMaintainerEmail(
					fmt.Sprintf("[f6-rest][error] %v", rvr),
					string(debug.Stack()),
				)
				if e != nil {
					fmt.Println(e)
				}
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
