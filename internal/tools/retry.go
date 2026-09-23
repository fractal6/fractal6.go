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
	"fmt"
	"time"
)

// RetryPolicy configures Retry: up to Attempts calls while RetryIf(err) holds,
// sleeping Delay(attempt) between them (attempt starts at 0).
type RetryPolicy struct {
	Name     string // log prefix, e.g. "dgraph", "postal"
	Attempts int
	Delay    func(attempt int) time.Duration
	RetryIf  func(error) bool
}

// Retry runs fn until it returns an error rejected by p.RetryIf (or nil), or
// p.Attempts is exhausted. Returns the last result. Logs once when retries fire.
func Retry[T any](p RetryPolicy, fn func() (T, error)) (T, error) {
	var (
		res T
		err error
	)
	for attempt := 0; attempt < p.Attempts; attempt++ {
		res, err = fn()
		if err == nil || !p.RetryIf(err) {
			if attempt > 0 {
				fmt.Printf("%s: succeeded after %d retries\n", p.Name, attempt)
			}
			return res, err
		}
		if attempt < p.Attempts-1 {
			time.Sleep(p.Delay(attempt))
		}
	}
	return res, err
}
