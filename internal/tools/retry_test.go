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

package tools

import (
	"errors"
	"testing"
	"time"
)

func TestRetry(t *testing.T) {
	errRetry, errFatal := errors.New("retry"), errors.New("fatal")
	p := RetryPolicy{
		Name:     "test",
		Attempts: 3,
		Delay:    func(int) time.Duration { return 0 },
		RetryIf:  func(err error) bool { return errors.Is(err, errRetry) },
	}
	cases := []struct {
		name      string
		errs      []error // returned by successive calls; nil once exhausted
		wantCalls int
		wantErr   error
	}{
		{"success", nil, 1, nil},
		{"retry-then-success", []error{errRetry, errRetry}, 3, nil},
		{"fatal-stops", []error{errFatal}, 1, errFatal},
		{"exhausted", []error{errRetry, errRetry, errRetry, errRetry}, 3, errRetry},
	}
	for _, c := range cases {
		calls := 0
		_, err := Retry(p, func() (int, error) {
			calls++
			if calls <= len(c.errs) {
				return 0, c.errs[calls-1]
			}
			return 1, nil
		})
		if calls != c.wantCalls || !errors.Is(err, c.wantErr) || (c.wantErr == nil && err != nil) {
			t.Errorf("%s: calls=%d err=%v, want calls=%d err=%v", c.name, calls, err, c.wantCalls, c.wantErr)
		}
	}
}
