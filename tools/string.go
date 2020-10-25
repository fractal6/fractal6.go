package tools

import (
    "strings"
    "regexp"
    "strconv"
    "unicode"
    "bytes"
    "compress/gzip"
    "encoding/base64"
    "io/ioutil"
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

func QuoteString(data string) string {
    var d string = data
    // @DEBUG: better way to encode a json string ?
    d = strconv.Quote(d)
    // remove surrounding quote !
    d = d[1:len(d)-1]
    return d
}

func ToGoNameFormat(name string) string {
    var l []string
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

//
// Compression / Decompression
//
func Pack64(s string) string {
    var b bytes.Buffer
	gz := gzip.NewWriter(&b)
	if _, err := gz.Write([]byte(s)); err != nil {
		panic(err)
	}
	if err := gz.Flush(); err != nil {
		panic(err)
	}
	if err := gz.Close(); err != nil {
		panic(err)
	}
	c := base64.StdEncoding.EncodeToString(b.Bytes())
    return c
}

func Unpack64(c string) string {
    data, _ := base64.StdEncoding.DecodeString(c)
    rdata := bytes.NewReader(data)
    r,_ := gzip.NewReader(rdata)
    s, _ := ioutil.ReadAll(r)
    return string(s)
}
