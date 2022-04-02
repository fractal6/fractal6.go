package auth

import (
    //"fmt"
    "strings"
    re "regexp"
)

var stripReg *re.Regexp
var specialReg *re.Regexp
var specialSoftReg *re.Regexp
var reservedURIReg *re.Regexp
var numReg *re.Regexp
var letterReg *re.Regexp
var safeWordReg *re.Regexp

func init() {
    //special := "@!#<>{}`'\"" + `%\\`
    //reservedURI := "&=+'/[]" + `\s`
    special :=   `\@\!\#\<\>\{\}\%\'\"\\` + "`" + `\*\^\%\;\~¨:,$£§` // username
    specialSoft := `\!\#\<\>\{\}\%\'\"\\` + "`" + `\*\^\%\;¨`        // nameid
    reservedURI := `\(\)\?\|\&\=\+\/\[\[` + `\s`
    numReg = re.MustCompile(`[0-9]`)
    letterReg = re.MustCompile(`[a-zA-Z]`)
    stripReg = re.MustCompile(`^\s|\s$`)
    specialReg = re.MustCompile(`[`+special+`]`)
    specialSoftReg = re.MustCompile(`[`+specialSoft+`]`)
    reservedURIReg = re.MustCompile(`[`+reservedURI+`]`)
    safeWordReg = re.MustCompile(`[\w\.\-]+`)
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
    // * do not contains space at begining or end.
    // * unsafe character.
    if hasStrip(n) {
        return ErrBadNameFormat
    }
    if hasSpecial(n) {
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
    // * do not contains space at begining or end.
    // * unsafe character.
    // * avoid URI special character and spaces.
    if hasStrip(u)  {
        return ErrBadUsernameFormat
    }
    if !isSafeWord(u) {
        return ErrBadUsernameFormat
    }

    for _, l := range []byte{u[0], u[len(u)-1]} {
        if l == '.' || l == '-' || l == '_'{
            return ErrBadUsernameFormat
        }
    }

    return nil
}

func ValidateNameid(nameid string, rootnameid string) error {
    ns := strings.Split(nameid, "#")
    if len(ns) == 0 {
        return ErrBadNameidFormat
    } else {
        for i, n := range ns {
            if i == 0 && n != rootnameid {
                return ErrBadNameidFormat
            }
            if i==1 && len(ns) == 3 &&  n == "" {
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
            if hasStrip(n) {
                return ErrBadNameidFormat
            }
            if hasSpecialSoft(n) {
                return ErrBadNameidFormat
            }
            if hasReservedURI(n) {
                return ErrBadNameidFormat
            }
        }
    }
    return nil
}

func ValidateEmail(e string) error {
    ns := strings.Split(e, "@")
    if len(ns) == 2 {
        for i, n := range ns {
            if i == len(ns)-1 && !strings.Contains(n, ".")  {
                return ErrBadEmailFormat
            }
            err := ValidateName(n)
            if err != nil || hasReservedURI(n) {
                return ErrBadEmailFormat
            }
        }
    } else {
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
    if (numReg.MatchString(p) && letterReg.MatchString(p)) == false {
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

func isSafeWord(s string) bool {
    return safeWordReg.MatchString(s)
}

func hasStrip(s string) bool {
    return stripReg.MatchString(s)
}

func hasSpecial(s string) bool {
    return specialReg.MatchString(s)
}

func hasSpecialSoft(s string) bool {
    return specialSoftReg.MatchString(s)
}

func hasReservedURI(s string) bool {
    return reservedURIReg.MatchString(s)
}
