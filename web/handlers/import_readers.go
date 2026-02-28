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

package handlers

import (
	"encoding/csv"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/microcosm-cc/bluemonday"
	"github.com/xuri/excelize/v2"
	"golang.org/x/net/html"
)

const maxImportRows = 10000

// importSanitizer strips dangerous HTML (script, event handlers, javascript: URLs)
// while preserving structural tags used by htmlToMarkdown.
var importSanitizer = func() *bluemonday.Policy {
	p := bluemonday.NewPolicy()
	// Allow structural tags that htmlToMarkdown understands
	p.AllowElements("p", "br", "strong", "b", "em", "i", "u",
		"ul", "ol", "li", "h1", "h2", "h3", "h4", "div", "span")
	p.AllowAttrs("href").OnElements("a")
	p.AllowAttrs("data-mention").OnElements("a")
	return p
}()

// readXLSX reads an xlsx file and returns a map of sheet name -> rows (each row is []string).
func readXLSX(file io.Reader) (map[string][][]string, error) {
	f, err := excelize.OpenReader(file)
	if err != nil {
		return nil, fmt.Errorf("failed to open xlsx: %w", err)
	}
	defer f.Close()

	sheets := make(map[string][][]string)
	for _, name := range f.GetSheetList() {
		rows, err := f.GetRows(name)
		if err != nil {
			return nil, fmt.Errorf("failed to read sheet %q: %w", name, err)
		}
		if len(rows) > maxImportRows {
			return nil, fmt.Errorf("sheet %q exceeds maximum of %d rows", name, maxImportRows)
		}
		sheets[name] = rows
	}
	return sheets, nil
}

// readCSV reads a CSV file and returns it as a single-sheet map.
// The sheet name is derived from the filename (without extension),
// so "Circles & Roles.csv" becomes sheet "Circles & Roles".
func readCSV(file io.Reader, filename string) (map[string][][]string, error) {
	reader := csv.NewReader(file)
	reader.LazyQuotes = true
	reader.TrimLeadingSpace = true

	rows, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV: %w", err)
	}
	if len(rows) > maxImportRows {
		return nil, fmt.Errorf("CSV exceeds maximum of %d rows", maxImportRows)
	}

	sheetName := strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename))
	return map[string][][]string{sheetName: rows}, nil
}

// readSpreadsheet dispatches to the appropriate reader based on file extension.
func readSpreadsheet(file io.Reader, filename string) (map[string][][]string, error) {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".xlsx":
		return readXLSX(file)
	case ".csv":
		return readCSV(file, filename)
	default:
		return nil, fmt.Errorf("unsupported file format %q: only .xlsx and .csv are supported", ext)
	}
}

// htmlToMarkdown converts simple HTML content to markdown.
// Handles: p, br, strong/b, em/i, a, ul/ol/li, h1-h6, and strips other tags.
func htmlToMarkdown(s string) string {
	if s == "" {
		return ""
	}
	// Sanitize HTML to strip dangerous content (scripts, event handlers, javascript: URLs)
	s = importSanitizer.Sanitize(s)
	// Wrap in a root element so the parser handles fragments
	doc, err := html.Parse(strings.NewReader("<div>" + s + "</div>"))
	if err != nil {
		return s
	}
	var buf strings.Builder
	renderNode(&buf, doc, "")
	result := buf.String()
	// Clean up excessive newlines
	for strings.Contains(result, "\n\n\n") {
		result = strings.ReplaceAll(result, "\n\n\n", "\n\n")
	}
	return strings.TrimSpace(result)
}

func renderNode(buf *strings.Builder, n *html.Node, listPrefix string) {
	switch n.Type {
	case html.TextNode:
		text := n.Data
		// Collapse whitespace in inline text
		text = strings.ReplaceAll(text, "\n", " ")
		buf.WriteString(text)
		return
	case html.ElementNode:
		// handled below
	default:
		// Recurse into children for document/fragment nodes
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			renderNode(buf, c, listPrefix)
		}
		return
	}

	switch n.Data {
	case "br":
		buf.WriteString("  \n")
	case "p":
		buf.WriteString("\n")
		renderChildren(buf, n, listPrefix)
		buf.WriteString("\n")
	case "strong", "b":
		buf.WriteString("**")
		renderChildren(buf, n, listPrefix)
		buf.WriteString("**")
	case "em", "i":
		buf.WriteString("*")
		renderChildren(buf, n, listPrefix)
		buf.WriteString("*")
	case "u":
		renderChildren(buf, n, listPrefix)
	case "a":
		href := getAttr(n, "href")
		// Skip data-mention links, just render the text
		if getAttr(n, "data-mention") != "" {
			renderChildren(buf, n, listPrefix)
			return
		}
		buf.WriteString("[")
		renderChildren(buf, n, listPrefix)
		buf.WriteString("](")
		buf.WriteString(href)
		buf.WriteString(")")
	case "ul":
		buf.WriteString("\n")
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if c.Type == html.ElementNode && c.Data == "li" {
				buf.WriteString("- ")
				renderChildren(buf, c, "  ")
				buf.WriteString("\n")
			}
		}
	case "ol":
		idx := 0
		buf.WriteString("\n")
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if c.Type == html.ElementNode && c.Data == "li" {
				idx++
				buf.WriteString(fmt.Sprintf("%d. ", idx))
				renderChildren(buf, c, "   ")
				buf.WriteString("\n")
			}
		}
	case "li":
		renderChildren(buf, n, listPrefix)
	case "h1":
		buf.WriteString("\n# ")
		renderChildren(buf, n, listPrefix)
		buf.WriteString("\n")
	case "h2":
		buf.WriteString("\n## ")
		renderChildren(buf, n, listPrefix)
		buf.WriteString("\n")
	case "h3":
		buf.WriteString("\n### ")
		renderChildren(buf, n, listPrefix)
		buf.WriteString("\n")
	case "h4":
		buf.WriteString("\n#### ")
		renderChildren(buf, n, listPrefix)
		buf.WriteString("\n")
	case "div", "span", "html", "head", "body":
		renderChildren(buf, n, listPrefix)
	default:
		renderChildren(buf, n, listPrefix)
	}
}

func renderChildren(buf *strings.Builder, n *html.Node, listPrefix string) {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		renderNode(buf, c, listPrefix)
	}
}

func getAttr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}
