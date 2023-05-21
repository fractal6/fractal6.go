/*
 * Fractale - Self-organisation for humans.
 * Copyright (C) 2023 Fractale Co
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
	"errors"
	"fmt"
	re "regexp"
	"strings"
)

func FormatError(err error, loc string) error {
	return fmt.Errorf(`{
        "errors":[{
            "message": "%s",
            "location": "%s"
        }]
    }`, err.Error(), loc)
}

// Library errors
var (
	ErrBadUsernameFormat = errors.New(`{
        "errors":[{
            "message":"Please enter a valid username. Special characters (@:!,?%. etc) are not allowed.",
            "location": "username"
        }]
    }`)
	ErrUsernameTooLong = errors.New(`{
        "errors":[{
            "message":"Username too long.",
            "location": "username"
        }]
    }`)
	ErrUsernameTooShort = errors.New(`{
        "errors":[{
            "message":"Username too short.",
            "location": "username"
        }]
    }`)
	ErrBadNameidFormat = errors.New(`{
        "errors":[{
            "message":"Please enter a valid name.",
            "location": "nameid"
        }]
    }`)
	ErrBadNameFormat = errors.New(`{
        "errors":[{
            "message":"Please enter a valid name.",
            "location": "name"
        }]
    }`)
	ErrNameTooLong = errors.New(`{
        "errors":[{
            "message":"Name too long.",
            "location": "name"
        }]
    }`)
	ErrNameTooShort = errors.New(`{
        "errors":[{
            "message":"Name too short.",
            "location": "name"
        }]
    }`)
	ErrBadEmailFormat = errors.New(`{
        "errors":[{
            "message":"Please enter a valid email.",
            "location": "email"
        }]
    }`)
	ErrEmailTooLong = errors.New(`{
        "errors":[{
            "message":"Email too long.",
            "location": "name"
        }]
    }`)
	ErrWrongPassword = errors.New(`{
        "errors":[{
            "message":"Wrong Password.",
            "location": "password"
        }]
    }`)
	ErrPasswordTooShort = errors.New(`{
        "errors":[{
            "message":"Password too short.",
            "location": "password"
        }]
    }`)
	ErrPasswordTooLong = errors.New(`{
        "errors":[{
            "message":"Password too long.",
            "location": "password"
        }]
    }`)
	ErrPasswordRequirements = errors.New(`{
        "errors":[{
            "message":"Password need to contains at least one number and one letter.",
            "location": "password"
        }]
    }`)
	ErrReserverdNamed = errors.New(`{
        "errors":[{
            "message":"This name already exists, please use another one.",
            "location": "name"
        }]
    }`)
	// Upsert error
	ErrUsernameExist = errors.New(`{
        "errors":[{
            "message":"Username already exists.",
            "location": "username"
        }]
    }`)
	ErrEmailExist = errors.New(`{
        "errors":[{
            "message":"Email already exists.",
            "location": "email"
        }]
    }`)
	// User Rights
	ErrCantLogin = errors.New(`{
        "errors":[{
            "message": "You are not authorized to login.",
            "location": ""
        }]
    }`)
)

var stripReg *re.Regexp
var specialReg *re.Regexp
var specialSoftReg *re.Regexp
var reservedURIReg *re.Regexp
var numReg *re.Regexp
var letterReg *re.Regexp
var safeWordReg *re.Regexp
var emailReg *re.Regexp

func init() {
	reservedURI := `\(\)\?\|\&\=\+\/\[\[` + `\s`
	special := `\@\#\<\>\{\}\%\"\'\!\\` + "`" + `\*\^\%\;\~¨:,$£§`
	specialSoft := `\{\}\%\"\^\;¨\\` + "`"
	// --
	reservedURIReg = re.MustCompile(`[` + reservedURI + `]`) // nameid
	specialReg = re.MustCompile(`[` + special + `]`)         // nameid
	specialSoftReg = re.MustCompile(`[` + specialSoft + `]`) // name
	// --
	stripReg = re.MustCompile(`^\s|\s$`)
	numReg = re.MustCompile(`[0-9]`)
	letterReg = re.MustCompile(`[a-zA-Z]`)
	// --
	safeWordReg = re.MustCompile(`^[\w\.\-]+$`)                   // username
	emailReg = re.MustCompile(`^[\w\.\-\+]+@[\w\.\-]+\.[\w\-]+$`) // email
}

//
// Format Validation
//

func ValidateName(n string) error {
	// Size control
	if len(n) > 100 {
		return ErrNameTooLong
	}
	if len(n) < 2 {
		return ErrNameTooShort
	}

	// Character control
	if matchReg(n, stripReg) {
		return ErrBadNameFormat
	}
	if matchReg(n, specialSoftReg) {
		return ErrBadNameFormat
	}
	return nil
}

func ValidateUsername(u string) error {
	// Size control
	if len(u) > 42 {
		return ErrUsernameTooLong
	}
	if len(u) < 3 {
		return ErrUsernameTooShort
	}

	// Character control
	if matchReg(u, stripReg) {
		return ErrBadUsernameFormat
	}
	if !matchReg(u, safeWordReg) {
		return ErrBadUsernameFormat
	}

	for _, l := range []byte{u[0], u[len(u)-1]} {
		if l == '.' || l == '-' || l == '_' {
			return ErrBadUsernameFormat
		}
	}

	return nil
}

func ValidateNameid(nameid string, rootnameid string) error {
	ns := strings.Split(nameid, "#")
	if len(ns) == 0 {
		return ErrBadNameidFormat
	}

	for i, n := range ns {
		if i == 0 && n != rootnameid {
			return ErrBadNameidFormat
		}
		if i == 1 && len(ns) == 3 && n == "" {
			// assume role under root node
			continue
		}

		// Size control
		if len(n) > 42 {
			return ErrNameTooLong
		}
		if len(n) < 2 {
			return ErrNameTooShort
		}

		// character control
		if matchReg(n, stripReg) {
			return ErrBadNameidFormat
		}
		if matchReg(n, specialReg) {
			return ErrBadNameidFormat
		}
		if matchReg(n, reservedURIReg) {
			return ErrReserverdNamed
		}
	}

	return nil
}

func ValidateEmail(s string) error {
	// Size control
	if len(s) > 100 {
		return ErrUsernameTooLong
	}
	if len(s) < 3 {
		return ErrUsernameTooShort
	}

	// Character control
	if matchReg(s, stripReg) {
		return ErrBadEmailFormat
	}
	if !matchReg(s, emailReg) {
		return ErrBadEmailFormat
	}

	return nil
}

func ValidatePassword(p string) error {
	// Size control
	if len(p) < 8 {
		return ErrPasswordTooShort
	}
	if len(p) > 100 {
		return ErrPasswordTooLong
	}
	if !(numReg.MatchString(p) && letterReg.MatchString(p)) {
		return ErrPasswordRequirements
	}
	return nil
}

// This secondary validation method is here ensure compatibility between
// old user that may not satisfy the primary validation method.
// @hint: this could be used to sent email alert if password to weak.
func ValidateSimplePassword(p string) error {
	// Size control
	if len(p) < 8 {
		return ErrPasswordTooShort
	}
	if len(p) > 100 {
		return ErrPasswordTooLong
	}
	return nil
}

//
// String utils
//

func matchReg(s string, regexp *re.Regexp) bool {
	return regexp.MatchString(s)
}
