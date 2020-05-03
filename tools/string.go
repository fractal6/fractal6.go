package tools

import (
    "strings"
    "regexp"
    "strconv"
    "unicode"
)

func IsDigit(s byte) bool {
    return unicode.IsDigit(rune(s))
}

func CleanString(data string, quote bool) string {
    var d string = data
    d = strings.Replace(d, `\n`, "", -1)
    d = strings.Replace(d, "\n", "", -1)
    space := regexp.MustCompile(`\s+`)
    d = space.ReplaceAllString(d, " ")
    if quote {
        // @DEBUG: better way to encode a json string ?
        d = strconv.Quote(d)
        // remove surrounding quote !
        d = d[1:len(d)-1]
    }
    return d
}

func ToGoNameFormat(name string) string {
    var l  []string
    for _, s := range strings.Split(name, "_") {
        l = append(l, strings.Title(s))
    }
    goName := strings.Join(l, "")
    return goName
}

func ToTypeName(name string) string {
    l := strings.Split(name, ".")
    typeName := l[len(l)-1]
    return typeName 
}
