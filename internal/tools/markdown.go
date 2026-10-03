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
	"fmt"
	"regexp"
	"strings"

	"github.com/microcosm-cc/bluemonday"
	"golang.org/x/net/html"
)

// mdSanitizer strips dangerous HTML (script, event handlers, javascript: URLs)
// while preserving structural tags used by HTMLToMarkdown.
var mdSanitizer = func() *bluemonday.Policy {
	p := bluemonday.NewPolicy()
	p.AllowElements(
		"p", "br", "strong", "b", "em", "i", "u",
		"ul", "ol", "li",
		"h1", "h2", "h3", "h4", "h5", "h6",
		"div", "span",
		"blockquote", "pre", "code",
		"del", "s",
		"hr",
		"table", "thead", "tbody", "tr", "th", "td",
		"details", "summary",
	)
	p.AllowAttrs("href").OnElements("a")
	p.AllowAttrs("data-mention").OnElements("a")
	p.AllowAttrs("src", "alt").OnElements("img")
	p.AllowAttrs("open").OnElements("details")
	return p
}()

// reHTMLSpace matches ASCII whitespace runs (not &nbsp;), collapsed to one space like a browser does.
var reHTMLSpace = regexp.MustCompile(`\s+`)

// reBlankLines matches 2+ line breaks with whitespace-only lines in between (e.g. from <br><br>).
var reBlankLines = regexp.MustCompile(`[ \t]*\n(?:[ \t]*\n)+`)

// HTMLToMarkdown converts HTML content to markdown.
// Handles: p, br, strong/b, em/i, a, ul/ol/li, h1-h6, blockquote,
// pre/code, del/s, hr, img, table, details/summary, and nested lists.
func HTMLToMarkdown(s string) (string, error) {
	if s == "" {
		return "", nil
	}
	s = mdSanitizer.Sanitize(s)
	doc, err := html.Parse(strings.NewReader("<div>" + s + "</div>"))
	if err != nil {
		return "", LogErr("HTMLToMarkdown", err)
	}
	hoistEmphasisSpaces(doc)
	var buf strings.Builder
	renderNode(&buf, doc, "")
	result := reBlankLines.ReplaceAllString(buf.String(), "\n\n")
	return strings.TrimSpace(result), nil
}

// hoistEmphasisSpaces moves edge whitespace out of emphasis elements: "** x **" is not emphasis in markdown.
func hoistEmphasisSpaces(n *html.Node) {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		hoistEmphasisSpaces(c)
	}
	if n.Type != html.ElementNode {
		return
	}
	switch n.Data {
	case "strong", "b", "em", "i", "del", "s":
	default:
		return
	}
	// A whitespace-only edge text is hoisted only if other content remains inside.
	if f := n.FirstChild; f != nil && f.Type == html.TextNode {
		if t := strings.TrimLeft(f.Data, " \t\r\n\f"); len(t) < len(f.Data) && (t != "" || f.NextSibling != nil) {
			f.Data = t
			n.Parent.InsertBefore(&html.Node{Type: html.TextNode, Data: " "}, n)
		}
	}
	if l := n.LastChild; l != nil && l.Type == html.TextNode {
		if t := strings.TrimRight(l.Data, " \t\r\n\f"); len(t) < len(l.Data) && (t != "" || l.PrevSibling != nil) {
			l.Data = t
			n.Parent.InsertBefore(&html.Node{Type: html.TextNode, Data: " "}, n.NextSibling)
		}
	}
}

// collectText renders a node's children into a temporary buffer and returns the result.
func collectText(n *html.Node, listPrefix string) string {
	var buf strings.Builder
	renderChildren(&buf, n, listPrefix)
	return buf.String()
}

// isInsidePre checks whether the node is inside a <pre> element.
func isInsidePre(n *html.Node) bool {
	for p := n.Parent; p != nil; p = p.Parent {
		if p.Type == html.ElementNode && p.Data == "pre" {
			return true
		}
	}
	return false
}

func renderNode(buf *strings.Builder, n *html.Node, listPrefix string) {
	switch n.Type {
	case html.TextNode:
		text := n.Data
		if !isInsidePre(n) {
			text = reHTMLSpace.ReplaceAllString(text, " ")
			// Collapse across nodes too; leading spaces at line start would make an indented code block.
			if s := buf.String(); s == "" || strings.HasSuffix(s, "\n") || strings.HasSuffix(s, " ") {
				text = strings.TrimLeft(text, " ")
			}
		}
		buf.WriteString(text)
		return
	case html.ElementNode:
		// handled below
	default:
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
	case "del", "s":
		buf.WriteString("~~")
		renderChildren(buf, n, listPrefix)
		buf.WriteString("~~")
	case "code":
		if n.Parent != nil && n.Parent.Type == html.ElementNode && n.Parent.Data == "pre" {
			// Handled by the "pre" case
			renderChildren(buf, n, listPrefix)
		} else {
			buf.WriteString("`")
			renderChildren(buf, n, listPrefix)
			buf.WriteString("`")
		}
	case "pre":
		buf.WriteString("\n```\n")
		renderChildren(buf, n, listPrefix)
		buf.WriteString("\n```\n")
	case "blockquote":
		inner := collectText(n, listPrefix)
		inner = strings.TrimSpace(inner)
		for _, line := range strings.Split(inner, "\n") {
			buf.WriteString("> ")
			buf.WriteString(line)
			buf.WriteString("\n")
		}
	case "hr":
		buf.WriteString("\n---\n")
	case "img":
		alt := getAttr(n, "alt")
		src := getAttr(n, "src")
		buf.WriteString("![")
		buf.WriteString(alt)
		buf.WriteString("](")
		buf.WriteString(src)
		buf.WriteString(")")
	case "a":
		href := getAttr(n, "href")
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
				buf.WriteString(listPrefix)
				buf.WriteString("- ")
				renderListItem(buf, c, listPrefix+"  ")
			}
		}
	case "ol":
		idx := 0
		buf.WriteString("\n")
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if c.Type == html.ElementNode && c.Data == "li" {
				idx++
				buf.WriteString(listPrefix)
				fmt.Fprintf(buf, "%d. ", idx)
				renderListItem(buf, c, listPrefix+"   ")
			}
		}
	case "li":
		renderChildren(buf, n, listPrefix)
	case "table":
		renderTable(buf, n)
	case "thead", "tbody", "tr", "th", "td":
		// Handled by renderTable; if encountered standalone, just render children.
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
	case "h5":
		buf.WriteString("\n##### ")
		renderChildren(buf, n, listPrefix)
		buf.WriteString("\n")
	case "h6":
		buf.WriteString("\n###### ")
		renderChildren(buf, n, listPrefix)
		buf.WriteString("\n")
	case "details":
		open := getAttr(n, "open")
		if open != "" {
			buf.WriteString("\n<details open>\n")
		} else {
			buf.WriteString("\n<details>\n")
		}
		renderChildren(buf, n, listPrefix)
		buf.WriteString("\n</details>\n")
	case "summary":
		buf.WriteString("<summary>")
		renderChildren(buf, n, listPrefix)
		buf.WriteString("</summary>\n")
	case "div", "span", "html", "head", "body":
		renderChildren(buf, n, listPrefix)
	default:
		renderChildren(buf, n, listPrefix)
	}
}

// renderTable collects rows from a <table> and emits a GFM pipe table.
func renderTable(buf *strings.Builder, table *html.Node) {
	var rows [][]string
	var isHeader []bool
	collectRows(table, &rows, &isHeader)
	if len(rows) == 0 {
		return
	}

	buf.WriteString("\n")
	for i, row := range rows {
		buf.WriteString("| ")
		buf.WriteString(strings.Join(row, " | "))
		buf.WriteString(" |\n")
		// Emit separator after header row
		if i == 0 && len(isHeader) > 0 && isHeader[0] {
			buf.WriteString("|")
			for range row {
				buf.WriteString(" --- |")
			}
			buf.WriteString("\n")
		}
	}
}

// collectRows walks the table tree to find <tr> elements and extract cell text.
func collectRows(n *html.Node, rows *[][]string, isHeader *[]bool) {
	if n.Type == html.ElementNode && n.Data == "tr" {
		var cells []string
		header := false
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if c.Type == html.ElementNode && (c.Data == "td" || c.Data == "th") {
				cells = append(cells, strings.TrimSpace(collectText(c, "")))
				if c.Data == "th" {
					header = true
				}
			}
		}
		*rows = append(*rows, cells)
		*isHeader = append(*isHeader, header)
		return
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		collectRows(c, rows, isHeader)
	}
}

// renderListItem renders a <li> element, ensuring exactly one trailing newline
// even when the item contains a nested list (which already ends with \n).
func renderListItem(buf *strings.Builder, li *html.Node, childPrefix string) {
	var tmp strings.Builder
	renderChildren(&tmp, li, childPrefix)
	text := strings.TrimRight(tmp.String(), "\n")
	buf.WriteString(text)
	buf.WriteString("\n")
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
