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

package tools

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"io/ioutil"
	"regexp"
	re "regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
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
		d = QuoteString(d)
	}
	return d
}

// @DEBUG: better way to encode a json string ?
func QuoteString(data string) string {
	// Quote
	d := strconv.Quote(data)
	// remove surrounding quote !
	d = d[1 : len(d)-1]
	return d
}

func PrettyString(str string) (string, error) {
	var prettyJSON bytes.Buffer
	if err := json.Indent(&prettyJSON, []byte(str), "", "    "); err != nil {
		return "", err
	}
	return prettyJSON.String(), nil
}

func RemoveCodeBlocks(msg string) string {
	var split []string

	split = strings.Split(msg, "```")
	if len(split)%2 == 1 {
		var subsplit []string
		for i := 0; i < len(split); i += 2 {
			subsplit = append(subsplit, split[i])
		}
		msg = strings.Join(subsplit, " ")
	}

	split = strings.Split(msg, "`")
	if len(split)%2 == 1 {
		var subsplit []string
		for i := 0; i < len(split); i += 2 {
			subsplit = append(subsplit, split[i])
		}
		msg = strings.Join(subsplit, " ")
	}

	return msg
}

func FindUsernames(msg string) []string {
	//r := re.MustCompile(`(^|\s|[^\w\[\` + "`" + `])@([\w\-\.]+)\b`)
	r := re.MustCompile(`(^|\s|[^\w\[])@([\w\-\.]+)\b`)
	all := r.FindAllStringSubmatch(msg, -1)
	match := []string{}
	for _, m := range all {
		match = append(match, m[2])
	}
	return match
}

func FindTensions(msg string) []string {
	r := re.MustCompile(`(^|\s|[^\w\[])(0x[0-9a-f]+)\b`)
	all := r.FindAllStringSubmatch(msg, -1)
	match := []string{}
	for _, m := range all {
		match = append(match, m[2])
	}
	return match
}

func ToGoNameFormat(name string) string {
	if name == "id" {
		return "ID"
	}
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

// Compression / Decompression
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
	r, _ := gzip.NewReader(rdata)
	s, _ := ioutil.ReadAll(r)
	return string(s)
}

//
// String splitting
//

// SplitCamelCase splits the camelcase word and returns a list of words. It also
// supports digits. Both lower camel case and upper camel case are supported.
// For more info please check: http://en.wikipedia.org/wiki/CamelCase
//
// Examples
//
//	"" =>                     [""]
//	"lowercase" =>            ["lowercase"]
//	"Class" =>                ["Class"]
//	"MyClass" =>              ["My", "Class"]
//	"MyC" =>                  ["My", "C"]
//	"HTML" =>                 ["HTML"]
//	"PDFLoader" =>            ["PDF", "Loader"]
//	"AString" =>              ["A", "String"]
//	"SimpleXMLParser" =>      ["Simple", "XML", "Parser"]
//	"vimRPCPlugin" =>         ["vim", "RPC", "Plugin"]
//	"GL11Version" =>          ["GL", "11", "Version"]
//	"99Bottles" =>            ["99", "Bottles"]
//	"May5" =>                 ["May", "5"]
//	"BFG9000" =>              ["BFG", "9000"]
//	"BöseÜberraschung" =>     ["Böse", "Überraschung"]
//	"Two  spaces" =>          ["Two", "  ", "spaces"]
//	"BadUTF8\xe2\xe2\xa1" =>  ["BadUTF8\xe2\xe2\xa1"]
//
// Splitting rules
//
//  1. If string is not valid UTF-8, return it without splitting as
//     single item array.
//  2. Assign all unicode characters into one of 4 sets: lower case
//     letters, upper case letters, numbers, and all other characters.
//  3. Iterate through characters of string, introducing splits
//     between adjacent characters that belong to different sets.
//  4. Iterate through array of split strings, and if a given string
//     is upper case:
//     if subsequent string is lower case:
//     move last character of upper case string to beginning of
//     lower case string
func SplitCamelCase(src string) (entries []string) {
	// don't split invalid utf8
	if !utf8.ValidString(src) {
		return []string{src}
	}
	entries = []string{}
	var runes [][]rune
	lastClass := 0
	class := 0
	// split into fields based on class of unicode character
	for _, r := range src {
		switch true {
		case unicode.IsLower(r):
			class = 1
		case unicode.IsUpper(r):
			class = 2
		case unicode.IsDigit(r):
			class = 3
		default:
			class = 4
		}
		if class == lastClass {
			runes[len(runes)-1] = append(runes[len(runes)-1], r)
		} else {
			runes = append(runes, []rune{r})
		}
		lastClass = class
	}
	// handle upper case -> lower case sequences, e.g.
	// "PDFL", "oader" -> "PDF", "Loader"
	for i := 0; i < len(runes)-1; i++ {
		if unicode.IsUpper(runes[i][0]) && unicode.IsLower(runes[i+1][0]) {
			runes[i+1] = append([]rune{runes[i][len(runes[i])-1]}, runes[i+1]...)
			runes[i] = runes[i][:len(runes[i])-1]
		}
	}
	// construct []string from results
	for _, s := range runes {
		if len(s) > 0 {
			entries = append(entries, string(s))
		}
	}
	return
}

func Humanize(src string) (t string) {
	return strings.Join(SplitCamelCase(src), " ")
}
