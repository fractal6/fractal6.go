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
	"bytes"
	"context"
	"io/ioutil"
	"net/http"
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
