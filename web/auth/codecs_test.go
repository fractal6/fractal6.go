/*
 * Fractale - Self-organisation for humans.
 * Copyright (C) 2022 Fractale Co
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

package auth

import (
	"reflect"
	"testing"
	//"os"
)

func init() {
	// Necessary to work with with Viper relative path and go test
	//os.Chdir("../")
	// ...doesn"t work, please run
	//    go test ./web/auth/codecs* -v
}

func TestValidateNameid(t *testing.T) {
	testcases := []struct {
		input string
		want  bool
	}{
		{"test", true},
		{"test#test", true},
		{"test#ok", true},
		{"test#ok#ok", true},
		{"test##ok", true},
		{"test##ok-ok", true},
		{"test#ok#ok#ok", true}, // watchout
		{"", false},
		{"ok", false},
		{"test@", false},
		{"ko#test", false},
		{"test#", false},
		{"test##", false},
		{"#test", false},
		{"test#o", false},
		{"test##ok@", false},
	}

	for _, test := range testcases {
		var got bool = true
		err := ValidateNameid(test.input, "test")
		if err != nil {
			//t.Errorf(err.Error())
			got = false
		}

		if !reflect.DeepEqual(got, test.want) {
			t.Errorf("For p = %s, want %v. Got %v",
				test.input, test.want, got)
		}
	}
}

func TestValidateUsername(t *testing.T) {
	testcases := []struct {
		input string
		want  bool
	}{
		{"test", true},
		{"tEst0123", true},
		{"test.ok", true},
		{"test-ok", true},
		{"test_ok", true},
		{"te", false},
		{"test-", false},
		{"test.", false},
		{"test_", false},
	}

	for _, test := range testcases {
		var got bool = true
		err := ValidateUsername(test.input)
		if err != nil {
			//t.Errorf(err.Error())
			got = false
		}

		if !reflect.DeepEqual(got, test.want) {
			t.Errorf("For p = %s, want %v. Got %v",
				test.input, test.want, got)
		}
	}
}

func TestValidateEmail(t *testing.T) {
	testcases := []struct {
		input string
		want  bool
	}{
		{"m@m.m", true},
		{"0@0.0", true},
		{"me@me.me", true},
		{"me0@m0.m0", true},
		{"me.me@me.me.m0", true},
		{"me_me@me_me.m0", true},
		{"me_m-e.@me_me.m0", true},
		{"m@.m", false},
		{"@.m", false},
		{"@me", false},
		{"me@", false},
		{"meme", false},
	}

	for _, test := range testcases {
		var got bool = true
		err := ValidateEmail(test.input)
		if err != nil {
			//t.Errorf(err.Error())
			got = false
		}

		if !reflect.DeepEqual(got, test.want) {
			t.Errorf("For p = %s, want %v. Got %v",
				test.input, test.want, got)
		}
	}
}
