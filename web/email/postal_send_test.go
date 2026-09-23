/*
 * Fractale - Self-organisation for humans.
 * Copyright (C) 2026 Fractale Co
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

// Postal answers HTTP 200 even for refused messages, carrying the failure in
// the JSON body's status field; these tests pin sendPostal's handling of both
// transport-level and body-level errors, and the retry on 5xx.

package email

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestSendPostalStatusHandling(t *testing.T) {
	type resp struct {
		code int
		body string
	}
	var resps []resp // served in order, the last one repeats
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r0 := resps[min(calls, len(resps)-1)]
		calls++
		w.WriteHeader(r0.code)
		w.Write([]byte(r0.body))
	}))
	defer srv.Close()

	oldUrl, oldSecret := emailUrl, emailSecret
	SetTestConfig(srv.URL, "test-secret")
	defer SetTestConfig(oldUrl, oldSecret)
	oldDelay := postalRetry.Delay
	postalRetry.Delay = func(int) time.Duration { return 0 }
	defer func() { postalRetry.Delay = oldDelay }()

	ok := resp{200, `{"status":"success"}`}
	cases := []struct {
		name      string
		resps     []resp
		wantErr   bool
		wantCalls int
	}{
		{"success", []resp{ok}, false, 1},
		{"no-status-in-body", []resp{{200, `{}`}}, false, 1},
		{"postal-error-behind-200", []resp{{200, `{"status":"error","data":{"code":"ValidationError","message":"bad rcpt"}}`}}, true, 1},
		{"5xx-then-success", []resp{{500, `boom`}, ok}, false, 2},
		{"5xx-exhausted", []resp{{500, `boom`}}, true, 3},
		{"4xx-no-retry", []resp{{401, `nope`}}, true, 1},
	}
	for _, c := range cases {
		resps, calls = c.resps, 0
		err := sendPostal([]byte(`{"to":["x@y.z"]}`))
		if (err != nil) != c.wantErr || calls != c.wantCalls {
			t.Errorf("%s: err=%v calls=%d, wantErr=%v wantCalls=%d", c.name, err, calls, c.wantErr, c.wantCalls)
		}
	}
}
