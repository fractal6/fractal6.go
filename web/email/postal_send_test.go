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
// transport-level and body-level errors.

package email

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSendPostalStatusHandling(t *testing.T) {
	var respCode int
	var respBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(respCode)
		w.Write([]byte(respBody))
	}))
	defer srv.Close()

	oldUrl, oldSecret := emailUrl, emailSecret
	SetTestConfig(srv.URL, "test-secret")
	defer SetTestConfig(oldUrl, oldSecret)

	cases := []struct {
		name    string
		code    int
		body    string
		wantErr bool
	}{
		{"success", 200, `{"status":"success"}`, false},
		{"no-status-in-body", 200, `{}`, false},
		{"postal-error-behind-200", 200, `{"status":"error","data":{"code":"ValidationError","message":"bad rcpt"}}`, true},
		{"http-error", 500, `boom`, true},
	}
	for _, c := range cases {
		respCode, respBody = c.code, c.body
		err := sendPostal([]byte(`{"to":["x@y.z"]}`))
		if (err != nil) != c.wantErr {
			t.Errorf("%s: err=%v, wantErr=%v", c.name, err, c.wantErr)
		}
	}
}
