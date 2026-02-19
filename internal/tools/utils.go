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
	"regexp"
	"strings"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"

	"fractale/fractal6.go/graph/model"
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

// StructMap convert/copy a interface to another
func StructMap(in any, out any) {
	raw, _ := json.Marshal(in)
	json.Unmarshal(raw, &out)
}

// StructToMap convert a struct to a map[string]any based on your JSON tag in your structs.
// Use mapstructure instead ?
// @deprectad: show omitempty tag, and values seems to be nil ?
func StructToMap(item any) map[string]any {
	res := map[string]any{}
	if item == nil {
		return res
	}
	v := reflect.TypeOf(item)
	reflectValue := reflect.ValueOf(item)
	reflectValue = reflect.Indirect(reflectValue)

	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	for i := 0; i < v.NumField(); i++ {
		tag := v.Field(i).Tag.Get("json")
		field := reflectValue.Field(i).Interface()
		if tag != "" && tag != "-" {
			if v.Field(i).Type.Kind() == reflect.Struct {
				res[tag] = StructToMap(field)
			} else {
				res[tag] = field
			}
		}
	}
	return res
}

// Use Marshaling and Unmarshaling to convert an object to a map of string.
// @DEBUG: mapstructure seems to not decode enum (RoleType) !
// @see also: https://stackoverflow.com/questions/23589564/function-for-converting-a-struct-to-map-in-golang#25117810
func Struct2Map(item any) map[string]any {
	var amap map[string]any
	itemRaw, _ := json.Marshal(item)
	json.Unmarshal(itemRaw, &amap)
	return amap
}

func Map2Struct(item map[string]any, res any) error {
	// @performance: use marshal/unmarshal instead ?
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Result:  res,
		TagName: "json",
	})
	err = decoder.Decode(item)
	return err
}

// MarshalWithoutNil marshal an struct but removed all empty (null) edges.
func MarshalWithoutNil(item any) ([]byte, error) {
	m := CleanNilMap(Struct2Map(item))
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
// see stackoverflow.com/a/12754757/4223749
// and https://ahmet.im/blog/golang-take-slices-of-any-type-as-input-parameter/
func InterfaceSlice(arg any) (out []any, ok bool) {
	// Keep the distinction between nil and empty slice input
	if arg == nil {
		return out, true
	}
	slice, success := takeArg(arg, reflect.Slice)
	// if slice.IsNil() { return out, true }
	if !success { // Non slice type
		return
	}
	c := slice.Len()
	out = make([]any, c)
	for i := 0; i < c; i++ {
		out[i] = slice.Index(i).Interface()
	}
	return out, true
}

func takeArg(arg any, kind reflect.Kind) (val reflect.Value, ok bool) {
	val = reflect.ValueOf(arg)
	if val.Kind() == kind {
		ok = true
	}
	return
}

// InterfaceToSlice safely converts an any (expected to be []any)
// into a typed []T slice. Elements that don't match type T are skipped.
// Returns nil if in is nil or not a slice.
func InterfaceToSlice[T any](in any) []T {
	items, ok := in.([]any)
	if !ok {
		return nil
	}
	out := make([]T, 0, len(items))
	for _, v := range items {
		if typed, ok := v.(T); ok {
			out = append(out, typed)
		}
	}
	return out
}

// CleanAliasedMap copy the input map by renaming all the keys
// recursively by removing trailing integers.
// @DEBUG: how to better handle aliasing (check gqlgen)
func CleanAliasedMap(m map[string]any) map[string]any {
	out := make(map[string]any, len(m))
	for k, v := range m {
		var nk string
		if IsDigit(k[len(k)-1]) {
			endDigits := regexp.MustCompile(`[0-9]+$`)
			nk = endDigits.ReplaceAllString(k, "")
		} else {
			nk = k
		}

		var nv any
		switch t := v.(type) {
		case map[string]any:
			nv = CleanAliasedMap(t)
		case []any:
			for i, x := range t {
				if m, ok := x.(model.JsonAtom); ok {
					t[i] = CleanAliasedMap(m)
				}
			}
			nv = t
		default:
			nv = t
		}
		out[nk] = nv
	}

	return out
}

// Keep the last key string when separated by dot (eg [a.key.name: 10] -> [name: 10])
// + replace uid field with ID (dgraph to gqlgen compatibility
func CleanCompositeName(m map[string]any, deep bool) map[string]any {
	out := make(map[string]any, len(m))
	for k, v := range m {
		ks := strings.Split(k, ".")
		nk := ks[len(ks)-1]

		if nk == "uid" {
			nk = "id"
		}

		var nv any
		switch t := v.(type) {
		case map[string]any:
			if deep {
				nv = CleanCompositeName(CleanAliasedMap(t), true)
			} else {
				nv = CleanAliasedMap(t)
			}
		case []any:
			for i, x := range t {
				if m, ok := x.(model.JsonAtom); ok {
					t[i] = CleanCompositeName(CleanAliasedMap(m), true)
				}
			}
			nv = t
		default:
			nv = t
		}
		out[nk] = nv
	}

	return out
}

func CleanAliasedMapHook() mapstructure.DecodeHookFunc {
	return func(from, to reflect.Kind, data any) (any, error) {
		if to == reflect.Struct {
			return CleanAliasedMap(data.(map[string]any)), nil
		}
		return data, nil
	}
}

func ToUnionHookFunc() mapstructure.DecodeHookFunc {
	// @DEBUG union type decoding
	// see  https://github.com/mitchellh/mapstructure/issues/159
	// and also https://github.com/99designs/gqlgen/issues/1055
	return func(f, t reflect.Type, data any) (any, error) {
		var u1 model.EventKind // because EventKind is an interface...else TypeOf(MyStrct) works.
		if t == reflect.TypeOf(&u1).Elem() {
			switch f.Kind() {
			case reflect.Map:
				var d any
				b, _ := json.Marshal(data)
				switch data.(model.JsonAtom)["__typename"] {
				case "Event":
					var partial model.Event
					if err := json.Unmarshal(b, &partial); err != nil {
						return data, err
					}
					d = &partial
				case "Contract":
					var partial model.Contract
					if err := json.Unmarshal(b, &partial); err != nil {
						return data, err
					}
					d = &partial
				case "Notif":
					var partial model.Notif
					if err := json.Unmarshal(b, &partial); err != nil {
						return data, err
					}
					d = &partial
				}

				return d, nil
			default:
				return data, nil
			}
		}

		var u2 model.CardKind
		if t == reflect.TypeOf(&u2).Elem() {
			switch f.Kind() {
			case reflect.Map:
				var d any
				b, _ := json.Marshal(data)
				switch data.(model.JsonAtom)["__typename"] {
				case "Tension":
					var partial model.Tension
					if err := json.Unmarshal(b, &partial); err != nil {
						return data, err
					}
					d = &partial
				case "ProjectDraft":
					var partial model.ProjectDraft
					if err := json.Unmarshal(b, &partial); err != nil {
						return data, err
					}
					d = &partial
				}

				return d, nil
			default:
				return data, nil
			}
		}

		return data, nil
	}
}
