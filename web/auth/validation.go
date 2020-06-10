package auth 

import (
    "strings"
    re "regexp"
)

var stripReg *re.Regexp
var specialReg *re.Regexp
var reservedURIReg *re.Regexp

func init() {
    //special := "@!#<>{}`'\"" + `%\\`
    //reservedURI := "&=+'/[]" + `\s`
    special := `\@\!\#\<\>\{\}\%\'\"\\` + "`"
    reservedURI := `\(\)\?\|\&\=\+\/\[\[` + `\s`
    stripReg = re.MustCompile(`^\s|\s$`)
    specialReg = re.MustCompile(`[`+special+`]`)
    reservedURIReg = re.MustCompile(`[`+reservedURI+`]`)
}

func ValidateName(n string) error {
    // Structure check
    if len(n) > 100 {
        return ErrNameTooLong
    }
    if len(n) < 3 {
        return ErrNameTooLong // too short
    }

    // Format/Security Check
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
    // Structure check
    if len(u) > 42 {
        return ErrUsernameTooLong
    }
    if len(u) < 2 {
        return ErrUsernameTooLong // too short
    }

    // Format/Security Check
    // * do not contains space at begining or end.
    // * unsafe character.
    // * avoid URI special character and spaces.
    if hasStrip(u) {
        return ErrBadUsernameFormat
    }
    if hasSpecial(u) {
        return ErrBadUsernameFormat
    }
    if hasReservedURI(u) {
        return ErrBadUsernameFormat
    }
    return nil
}

func ValidateNameid(nameid string, rootnameid string, name string) error {
    ns := strings.Split(nameid, "#")
    if len(ns) == 0 {
        return ErrBadNameidFormat
    } else {
        for i, n := range ns {
            if i == 0 && n != rootnameid {
                return ErrBadNameidFormat
            }
            if n == "" {
                // assume role under root node
                continue
            }
            //if i == len(ns)-1 && formatName(n) != n {
            //    return ErrBadNameidFormat
            //}
            err := ValidateUsername(n)
            if err != nil {
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
                return ErrBadNameidFormat
            }
            err := ValidateUsername(n)
            if err != nil {
                return ErrBadEmailFormat 
            }
        }
    } else {
        return ErrBadEmailFormat
    }
    return nil
}


func ValidatePassword(p string) error {
    // Structure check
    if len(p) < 8 {
        return ErrPasswordTooShort
    }
    if len(p) > 100 {
        return ErrPasswordTooLong
    }

    // Format/Security Check
    return nil
}

//
// String utils
//

func hasStrip(s string) bool {
    return stripReg.MatchString(s)
}

func hasSpecial(s string) bool {
    return specialReg.MatchString(s)
}

func hasReservedURI(s string) bool {
    return reservedURIReg.MatchString(s)
}