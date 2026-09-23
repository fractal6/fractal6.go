// f6-extract-clients converts a Fractale tensions xlsx export into a contacts workbook,
// one sheet per receiver, with fields extracted from the tension message.
package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/ncruces/zenity"
	"github.com/xuri/excelize/v2"
)

// field is an output column; keys are the line prefixes looked up in the tension message.
type field struct {
	header string
	width  float64
	keys   []string
}

// Keys are regexps where a space means zero or more spaces. Matched case insensitive.
var fields = []field{
	{"Nom de la société", 40, nil}, // tension title
	{"Nom du contact", 28, []string{"contacts?"}},
	{"Format", 20, []string{"formats?"}},
	{"Numéro de téléphone", 22, []string{"téléphone", "telephone", "tél", "tel", "phone"}},
	{"Adresse email", 32, []string{"adresse (?:e-?)?mails?", "(?:e-?)?mails?"}},
	{"Commercial", 22, []string{"commercials?"}},
	{"Date", 16, []string{"d[aâ]tes?"}},
	{"Prix", 16, []string{"prix", "cost", "pricing"}},
}

const (
	headerHeight = 20
	rowHeight    = 30
)

var fieldRegexps = func() []*regexp.Regexp {
	res := make([]*regexp.Regexp, len(fields))
	for i, f := range fields {
		if f.keys != nil {
			res[i] = keysRegexp(f.keys)
		}
	}
	return res
}()

// keysRegexp matches a line starting with one of the keys (after optional list marker / bold)
// and captures the rest of the line, separator excluded.
func keysRegexp(keys []string) *regexp.Regexp {
	alts := strings.ReplaceAll(strings.Join(keys, "|"), " ", `[ \t]*`)
	return regexp.MustCompile(`(?im)^[ \t]*(?:[-*+]|\d+[.)])?[ \t]*[*_]*(?:` + alts +
		`)(?:[*_ \t]*[:;][*_ \t:;]*|[*_ \t]+|$)(.*)$`)
}

// extract returns the value of the first line matching re, empty if none.
func extract(re *regexp.Regexp, message string) string {
	m := re.FindStringSubmatch(strings.ReplaceAll(message, "\r", ""))
	if m == nil {
		return ""
	}
	return strings.TrimSpace(m[1])
}

// sheetName makes a receiver name a valid xlsx sheet name (31 chars, no :\/?*[], no edge quote).
func sheetName(s string) string {
	s = regexp.MustCompile(`[:\\/?*\[\]]`).ReplaceAllString(s, "_")
	if r := []rune(s); len(r) > 31 {
		s = string(r[:31])
	}
	if s = strings.Trim(s, "' "); s == "" {
		return "Tensions"
	}
	return s
}

// convert reads the tensions export (first sheet) and builds the contacts workbook.
func convert(src *excelize.File) (*excelize.File, error) {
	rows, err := src.GetRows(src.GetSheetList()[0])
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, errors.New("empty spreadsheet")
	}
	col := map[string]int{}
	for i, h := range rows[0] {
		col[strings.ToLower(strings.TrimSpace(h))] = i
	}
	for _, h := range []string{"title", "receiver", "message"} {
		if _, ok := col[h]; !ok {
			return nil, fmt.Errorf("column %q not found: is this a Fractale tensions export?", h)
		}
	}
	cell := func(row []string, h string) string {
		if i := col[h]; i < len(row) {
			return row[i]
		}
		return ""
	}

	dst := excelize.NewFile()
	defaultSheet := dst.GetSheetName(0)
	next := map[string]int{} // sheet -> next row to write
	for _, row := range rows[1:] {
		sheet := sheetName(cell(row, "receiver"))
		if _, ok := next[sheet]; !ok {
			if _, err := dst.NewSheet(sheet); err != nil {
				return nil, err
			}
			if err := styleSheet(dst, sheet); err != nil {
				return nil, err
			}
			next[sheet] = 2
		}
		message := cell(row, "message")
		values := make([]any, len(fields))
		values[0] = cell(row, "title")
		for i, re := range fieldRegexps[1:] {
			values[i+1] = extract(re, message)
		}
		r := next[sheet]
		if err := dst.SetSheetRow(sheet, fmt.Sprintf("A%d", r), &values); err != nil {
			return nil, err
		}
		if err := dst.SetRowHeight(sheet, r, rowHeight); err != nil {
			return nil, err
		}
		next[sheet]++
	}

	if len(next) == 0 {
		return nil, errors.New("no tension found")
	}
	if _, ok := next[defaultSheet]; !ok {
		if err := dst.DeleteSheet(defaultSheet); err != nil {
			return nil, err
		}
	}
	return dst, nil
}

// styleSheet writes the navy header row (bold, frozen, filterable) and the column layout.
func styleSheet(f *excelize.File, sheet string) error {
	lastCol, err := excelize.ColumnNumberToName(len(fields))
	if err != nil {
		return err
	}
	header := make([]any, len(fields))
	for i, fd := range fields {
		header[i] = fd.header
		c, err := excelize.ColumnNumberToName(i + 1)
		if err != nil {
			return err
		}
		if err := f.SetColWidth(sheet, c, c, fd.width); err != nil {
			return err
		}
	}
	cellStyle, err := f.NewStyle(&excelize.Style{Alignment: &excelize.Alignment{WrapText: true, Vertical: "center"}})
	if err != nil {
		return err
	}
	if err := f.SetColStyle(sheet, "A:"+lastCol, cellStyle); err != nil {
		return err
	}
	if err := f.SetSheetRow(sheet, "A1", &header); err != nil {
		return err
	}
	// After SetColStyle: column styles overwrite existing cells, header style must win
	headerStyle, err := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"1F3864"}},
		Alignment: &excelize.Alignment{Vertical: "center"},
	})
	if err != nil {
		return err
	}
	if err := f.SetCellStyle(sheet, "A1", lastCol+"1", headerStyle); err != nil {
		return err
	}
	if err := f.SetRowHeight(sheet, 1, headerHeight); err != nil {
		return err
	}
	if err := f.SetPanes(sheet, &excelize.Panes{Freeze: true, YSplit: 1, TopLeftCell: "A2", ActivePane: "bottomLeft"}); err != nil {
		return err
	}
	return f.AutoFilter(sheet, "A1:"+lastCol+"1", nil)
}

// convertFile converts the export at path and returns the written file path.
func convertFile(in string) (string, error) {
	src, err := excelize.OpenFile(in)
	if err != nil {
		return "", err
	}
	defer src.Close()

	dst, err := convert(src)
	if err != nil {
		return "", err
	}
	defer dst.Close()

	out := strings.TrimSuffix(in, filepath.Ext(in)) + "_converted.xlsx"
	return out, dst.SaveAs(out)
}

func main() {
	// Files dropped on the binary come as arguments, otherwise ask for them
	files := os.Args[1:]
	if len(files) == 0 {
		var err error
		files, err = zenity.SelectFileMultiple(
			zenity.Title("Select the Fractale tensions exports to convert"),
			zenity.FileFilter{Name: "Excel", Patterns: []string{"*.xlsx"}},
		)
		if errors.Is(err, zenity.ErrCanceled) {
			return
		}
		if err != nil {
			zenity.Error(err.Error())
			os.Exit(1)
		}
	}

	// Native indeterminate progress bar, purely cosmetic: ignored if it cannot open
	loading, _ := zenity.Progress(zenity.Title("f6-extract-clients"), zenity.Pulsate(), zenity.NoCancel())
	var done, failed []string
	for i, in := range files {
		if loading != nil {
			loading.Text(fmt.Sprintf("Converting %s (%d/%d)...", filepath.Base(in), i+1, len(files)))
		}
		if out, err := convertFile(in); err != nil {
			failed = append(failed, filepath.Base(in)+": "+err.Error())
		} else {
			done = append(done, out)
		}
	}
	if loading != nil {
		loading.Close()
	}

	if len(failed) > 0 {
		msg := "Failed:\n" + strings.Join(failed, "\n")
		if len(done) > 0 {
			msg += "\n\nDone:\n" + strings.Join(done, "\n")
		}
		zenity.Error(msg)
		os.Exit(1)
	}
	zenity.Info("Done:\n" + strings.Join(done, "\n"))
}
