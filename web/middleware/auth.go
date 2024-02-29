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
	//"fmt"
	"bytes"
	"encoding/json"
	"github.com/spf13/viper"
	"io/ioutil"
	"net/http"
	//"fractale/fractal6.go/db"
	//"fractale/fractal6.go/web/auth"
)

var CREDENTIAL_PROM string

func init() {
	CREDENTIAL_PROM = viper.GetString("server.prometheus_credentials")
}

func CheckBearer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		val := r.Header.Get("Authorization")
		if val == CREDENTIAL_PROM || val == "Bearer "+CREDENTIAL_PROM {
			next.ServeHTTP(w, r.WithContext(r.Context()))
		}
	})
}

// CheckRecursiveQueryRights check if the query can be executed where a
// a the body is expected to be string/nameid.
func CheckRecursiveQueryRights(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		//ctx := r.Context()
		//uctx := auth.GetUserContextOrEmpty(r.Context())
		//var q string

		//// Keep this to reset the body reader later
		//body, _ := ioutil.ReadAll(r.Body)

		//// reset the body reader
		//r.Body = ioutil.NopCloser(bytes.NewReader(body))
		//// Get the JSON body and decode it
		//err := json.NewDecoder(r.Body).Decode(&q)
		//if err != nil {
		//    // Body structure error
		//    http.Error(w, err.Error(), 400)
		//    return
		//}
		//// reset the body reader agin
		//r.Body = ioutil.NopCloser(bytes.NewReader(body))

		//// This test is not enough, as private node will be return below.
		//input := map[string]string{"key":"nameid", "value": q}
		//res, err := db.GetDB().Get(uctx, "node", input)
		//if err != nil { http.Error(w, err.Error(), 400); return }

		//// Failed silently, or with discretion....
		//if res == "" {
		//    w.Write([]byte("[]"))
		//    return
		//}

		//next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// CheckTensionQueryRights remove all unauthorize nameids where a
// the body represent a {db.TensionQuery}.
func CheckTensionQueryRights(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		//uctx := auth.GetUserContextOrEmpty(r.Context())
		var q struct{ Nameids []string }

		// Keep this to reset the body reader later
		body, _ := ioutil.ReadAll(r.Body)
		// reset the body reader
		r.Body = ioutil.NopCloser(bytes.NewReader(body))
		// Get the JSON body and decode it
		err := json.NewDecoder(r.Body).Decode(&q)
		if err != nil {
			// Body structure error
			http.Error(w, err.Error(), 400)
			return
		}
		// Restore the io.ReadCloser to its original state
		r.Body = ioutil.NopCloser(bytes.NewBuffer(body))

		// Authentification tasks...
		// to be completed, rewrite body !?

		//// Failed silently, or with discretion....
		//if len(newNameids) == 0 {
		//    w.Write([]byte("[]"))
		//    return
		//}

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
