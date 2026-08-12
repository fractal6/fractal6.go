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
	"encoding/json"
	"reflect"
	"time"

	"github.com/spf13/viper"
)

// Now returns the current time formated with RFC3339
func Now() string {
	return time.Now().UTC().Format(time.RFC3339)
}

func ZeroTime() string {
	return time.Time{}.UTC().Format(time.RFC3339)
}

// IsOlder returns true if d1 is older than d2.
func IsOlder(d1, d2 string) bool {
	date1, _ := time.Parse(time.RFC3339, d1)
	date2, _ := time.Parse(time.RFC3339, d2)
	return date1.Before(date2)
}

// TimeDelta returns the time difference (Duration) d1 - d2
func TimeDelta(d1, d2 string) time.Duration {
	date1, _ := time.Parse(time.RFC3339, d1)
	date2, _ := time.Parse(time.RFC3339, d2)
	return date1.Sub(date2)
}

// ViperPositiveInt reads `key` from viper and returns it when > 0; falls back
// to `fallback` for missing/zero/negative values. Centralises the
// "config-with-sane-default" pattern used across the codebase (storage TTLs,
// upload size caps, attachment caps, gate timings…).
func ViperPositiveInt(key string, fallback int) int {
	v := viper.GetInt(key)
	if v <= 0 {
		return fallback
	}
	return v
}

// InitViper Read the config file
func InitViper() {
	viper.AddConfigPath("./")
	viper.AddConfigPath("../")    // `go test` change directory !
	viper.AddConfigPath("../../") // `go test` change directory !
	viper.SetConfigName("config") // name of config file (without extension)
	// viper.AutomaticEnv() // read in environment variables that match
	if err := viper.ReadInConfig(); err != nil {
		// Panic on config reading error
		panic(err)
	}
}

// MarshalWithoutNil marshal an struct but removed all empty (null) edges.
func MarshalWithoutNil(item any) ([]byte, error) {
	m := CleanNilMap(StructMap[map[string]any](item))
	return json.Marshal(m)
}

// CleanNilMap remove all empty (nil) edges recursively.
func CleanNilMap(m map[string]any) map[string]any {
	out := make(map[string]any, len(m))
	for k, v := range m {
		if v == nil {
			continue
		}

		var nv any
		switch t := v.(type) {
		case map[string]any:
			nv = CleanNilMap(t)
		default:
			nv = t
		}
		out[k] = nv
	}

	return out
}

// InterfaceSlice tries to convert an interface to a Slice of interface.
func InterfaceSlice(arg any) (out []any, ok bool) {
	if arg == nil {
		return out, true
	}
	val := reflect.ValueOf(arg)
	if val.Kind() != reflect.Slice {
		return
	}
	c := val.Len()
	out = make([]any, c)
	for i := 0; i < c; i++ {
		out[i] = val.Index(i).Interface()
	}
	return out, true
}
