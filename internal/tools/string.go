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
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"io"
	"strings"
	"unicode"
	"unicode/utf8"

	re "regexp"
)

func IsDigit(s byte) bool {
	return unicode.IsDigit(rune(s))
}

func CleanString(data string, quote bool) string {
	var d string = data
	d = strings.ReplaceAll(d, `\n`, "")
	d = strings.ReplaceAll(d, "\n", "")
	space := re.MustCompile(`\s+`)
	d = space.ReplaceAllString(d, " ")
	if quote {
		d = QuoteString(d)
	}
	return d
}

// QuoteString escapes a string for safe embedding inside a JSON string value.
func QuoteString(data string) string {
	b, _ := json.Marshal(data)
	// Strip surrounding quotes added by json.Marshal
	return string(b[1 : len(b)-1])
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
	// r := re.MustCompile(`(^|\s|[^\w\[\` + "`" + `])@([\w\-\.]+)\b`)
	r := re.MustCompile(`(^|\s|[^\w\[])@([\w\-\.]+)\b`)
	all := r.FindAllStringSubmatch(msg, -1)
	match := []string{}
	for _, m := range all {
		match = append(match, m[2])
	}
	return match
}

// inlineImageRe matches a markdown image token `![alt](url)` and captures the
// URL portion. Mirrors markdownImageRe in web/handlers/files.go; kept in
// sync intentionally — both must agree on what counts as an inline paste
// reference, otherwise the upload-gate Register count will diverge from
// embedIfReferenced's Signal count and emails will stall waiting for an
// upload that will never arrive.
var inlineImageRe = re.MustCompile(`!\[[^\]]*\]\(([^)\s]+)\)`)

// CountInlineImageCandidates returns the number of `![alt](url)` references
// in `msg` whose URL is a plausible inline-paste filename — no scheme, no
// path separator, and not data:/cid:. Code regions are stripped via
// RemoveCodeBlocks so filenames mentioned in fenced/inline code do not
// count.
//
// Used by the tension resolver hooks (graph/tension_resolver.go) to
// pre-Register the per-tension upload gate BEFORE PublishTensionEvent fires,
// so the notifier daemon can wait for the matching /file/upload calls to
// arrive before reading Comment.files.
func CountInlineImageCandidates(msg string) int {
	if msg == "" {
		return 0
	}
	stripped := RemoveCodeBlocks(msg)
	matches := inlineImageRe.FindAllStringSubmatch(stripped, -1)
	n := 0
	for _, m := range matches {
		if len(m) < 2 {
			continue
		}
		u := m[1]
		if u == "" {
			continue
		}
		// Reject any URL that carries a path separator or a scheme delimiter:
		// only bare paste filenames qualify, matching rewriteMessageForFile.
		if strings.ContainsAny(u, "/:") {
			continue
		}
		n++
	}
	return n
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

// reEmailQuoteHeader matches quote headers from common clients across
// languages: EN (Gmail/Apple Mail), FR, DE, ES. Group 1 captures anything
// before the header on the same line — usually blank or ">"-quoted, but can
// be real reply text when the client didn't break the line. The line must
// end with ":" (typical header terminator), with optional content between
// the keyword and the colon (e.g. DE "schrieb Alice <a@b>:").
var reEmailQuoteHeader = re.MustCompile(`(?im)^(.*?)(?:` +
	`On\s[^\n]{1,300}?wrote` + // EN: "On Mon, 27 Mar 2026, Alice wrote:"
	`|Le\s[^\n]{1,300}?a\s+[eé]crit` + // FR: "Le lun. ... a écrit :"
	`|Am\s[^\n]{1,300}?schrieb` + // DE: "Am 27.03.2026 schrieb Alice:"
	`|El\s[^\n]{1,300}?escribi[oó]` + // ES: "El lun., 27 mar. ... escribió:"
	`)[^\n]*:\s*$`)

// reEmailSignature matches the standard "-- " signature delimiter.
var reEmailSignature = re.MustCompile(`^--\s*$`)

// reFractaleFooter matches our own notification footer: stable anchors we
// control, reliable even when the surrounding quote structure is mangled.
// Either of the two footer lines is accepted — clients sometimes strip one.
var reFractaleFooter = re.MustCompile(`(?im)^\s*>?\s*(?:—\s*)?(?:You are receiving this because\b|\[?View it on Fractale\b)`)

// StripEmailQuote removes the quoted reply and trailing signature from an
// email body. Signature can appear before or after the quoted block.
func StripEmailQuote(msg string) string {
	result := stripFractaleFooter(msg)
	result = stripTrailingSignature(result)
	result = stripEmailQuote(result)
	result = stripTrailingSignature(result)
	return result
}

// headerPrefixIsQuoteOnly: the text before a quote header is pure quoting/whitespace.
func headerPrefixIsQuoteOnly(prefix string) bool {
	return strings.TrimLeft(prefix, "> \t") == ""
}

func isQuoteOrBlank(line string) bool {
	t := strings.TrimSpace(line)
	return t == "" || strings.HasPrefix(t, ">")
}

// truncateAtQuoteHeader drops the quote-header line at index i and everything
// below. When the header is inline (real reply text before it on the same
// line), that reply text is preserved as the last kept line.
func truncateAtQuoteHeader(lines []string, i int, prefix string) []string {
	if headerPrefixIsQuoteOnly(prefix) {
		return lines[:i]
	}
	kept := append([]string{}, lines[:i]...)
	if reply := strings.TrimRight(prefix, " \t"); reply != "" {
		kept = append(kept, reply)
	}
	return kept
}

// keepOrFallback returns a trimmed candidate, or fallback if empty (refuses
// to strip everything).
func keepOrFallback(candidate, fallback string) string {
	if r := strings.TrimSpace(candidate); r != "" {
		return r
	}
	return fallback
}

// stripFractaleFooter cuts from the nearest preceding quote header when the
// Fractale footer marker is found. Handles inline headers that stripEmailQuote
// can't see (e.g. Gmail inlines "Le ... a écrit :" with the user's reply).
func stripFractaleFooter(msg string) string {
	loc := reFractaleFooter.FindStringIndex(msg)
	if loc == nil {
		return msg
	}
	footerLineStart := 0
	if nl := strings.LastIndexByte(msg[:loc[0]], '\n'); nl >= 0 {
		footerLineStart = nl + 1
	}
	prefix := strings.TrimRight(msg[:footerLineStart], "\n")
	preLines := []string{}
	if prefix != "" {
		preLines = strings.Split(prefix, "\n")
	}

	for i := len(preLines) - 1; i >= 0; i-- {
		m := reEmailQuoteHeader.FindStringSubmatch(preLines[i])
		if m == nil {
			continue
		}
		return keepOrFallback(strings.Join(truncateAtQuoteHeader(preLines, i, m[1]), "\n"), msg)
	}

	// No quote header: drop the footer block (and a preceding "—" line if any).
	if n := len(preLines); n > 0 {
		last := strings.TrimSpace(strings.TrimLeft(preLines[n-1], "> \t"))
		if last == "—" {
			preLines = preLines[:n-1]
		}
	}
	return keepOrFallback(strings.Join(preLines, "\n"), msg)
}

// stripEmailQuote strips the quoted block when it sits at the very start or
// very end of the message. Also handles an inline header on the last line.
func stripEmailQuote(msg string) string {
	lines := strings.Split(msg, "\n")

	type hit struct {
		line   int
		prefix string
	}
	var hits []hit
	for i, line := range lines {
		m := reEmailQuoteHeader.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		hits = append(hits, hit{line: i, prefix: m[1]})
	}
	if len(hits) == 0 {
		return msg
	}

	// Case 1: quote at the start — header on first non-blank line, followed by
	// ">" / blank lines, then the reply.
	first := hits[0]
	atStart := headerPrefixIsQuoteOnly(first.prefix)
	for i := 0; atStart && i < first.line; i++ {
		if strings.TrimSpace(lines[i]) != "" {
			atStart = false
		}
	}
	if atStart {
		end := first.line + 1
		for end < len(lines) && isQuoteOrBlank(lines[end]) {
			end++
		}
		if result := strings.TrimSpace(strings.Join(lines[end:], "\n")); result != "" {
			return result
		}
	}

	// Case 2: quote at the end — everything after the last header is blank/">".
	last := hits[len(hits)-1]
	for i := last.line + 1; i < len(lines); i++ {
		if !isQuoteOrBlank(lines[i]) {
			return msg
		}
	}
	kept := truncateAtQuoteHeader(lines, last.line, last.prefix)
	return keepOrFallback(strings.Join(kept, "\n"), msg)
}

// stripTrailingSignature strips the last "-- " delimited signature block at
// EOF (preceded by a blank line, followed by non-empty body then only blanks).
func stripTrailingSignature(msg string) string {
	lines := strings.Split(msg, "\n")
	// Scan backwards for the last signature delimiter.
	for i := len(lines) - 1; i >= 0; i-- {
		if !reEmailSignature.MatchString(lines[i]) {
			continue
		}
		// Must be preceded by at least one empty line.
		if i == 0 || strings.TrimSpace(lines[i-1]) != "" {
			continue
		}
		// Signature body: contiguous non-empty lines, then only blanks to EOF.
		j := i + 1
		for j < len(lines) && strings.TrimSpace(lines[j]) != "" {
			j++
		}
		if j == i+1 {
			// No content right after "--": not a signature.
			break
		}
		// Everything from j onward must be blank.
		trailingOk := true
		for _, l := range lines[j:] {
			if strings.TrimSpace(l) != "" {
				trailingOk = false
				break
			}
		}
		if !trailingOk {
			break
		}
		if result := strings.TrimSpace(strings.Join(lines[:i], "\n")); result != "" {
			return result
		}
		break
	}
	return msg
}

func ToGoNameFormat(name string) string {
	if name == "id" {
		return "ID"
	}
	var l []string
	for _, s := range strings.Split(name, "_") {
		l = append(l, strings.ToUpper(s[:1])+s[1:])
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
	s, _ := io.ReadAll(r)
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

// NameidEncoder sanitizes a string for use as a nameid component.
// Mirrors the frontend (Elm) nameidEncoder: replaces separator-like chars with '-',
// removes bracket-like chars with '_', and collapses consecutive '-' or '_'.
func NameidEncoder(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ToLower(s)

	var buf strings.Builder
	buf.Grow(len(s))
	for _, c := range s {
		switch c {
		case ' ', '/', '=', '?', '#', '&', '|', '%', '\\':
			buf.WriteByte('-')
		case '@', '(', ')', '<', '>', '[', ']', '{', '}', '"', '`', '\'':
			buf.WriteByte('_')
		default:
			buf.WriteRune(c)
		}
	}
	result := buf.String()
	result = cleanDup(result, "-")
	result = cleanDup(result, "_")
	return result
}

// cleanDup collapses consecutive occurrences of sep into a single one
// and trims leading/trailing sep.
func cleanDup(s, sep string) string {
	double := sep + sep
	for strings.Contains(s, double) {
		s = strings.ReplaceAll(s, double, sep)
	}
	s = strings.Trim(s, sep)
	return s
}
