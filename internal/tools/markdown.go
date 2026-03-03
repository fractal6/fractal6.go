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
	"strings"

	"github.com/microcosm-cc/bluemonday"
	"golang.org/x/net/html"
)

// mdSanitizer strips dangerous HTML (script, event handlers, javascript: URLs)
// while preserving structural tags used by HTMLToMarkdown.
var mdSanitizer = func() *bluemonday.Policy {
	p := bluemonday.NewPolicy()
	p.AllowElements("p", "br", "strong", "b", "em", "i", "u",
		"ul", "ol", "li", "h1", "h2", "h3", "h4", "div", "span")
	p.AllowAttrs("href").OnElements("a")
	p.AllowAttrs("data-mention").OnElements("a")
	return p
}()

// HTMLToMarkdown converts simple HTML content to markdown.
// Handles: p, br, strong/b, em/i, a, ul/ol/li, h1-h4, and strips other tags.
func HTMLToMarkdown(s string) string {
	if s == "" {
		return ""
	}
	s = mdSanitizer.Sanitize(s)
	doc, err := html.Parse(strings.NewReader("<div>" + s + "</div>"))
	if err != nil {
		return s
	}
	var buf strings.Builder
	renderNode(&buf, doc, "")
	result := buf.String()
	for strings.Contains(result, "\n\n\n") {
		result = strings.ReplaceAll(result, "\n\n\n", "\n\n")
	}
	return strings.TrimSpace(result)
}

func renderNode(buf *strings.Builder, n *html.Node, listPrefix string) {
	switch n.Type {
	case html.TextNode:
		text := n.Data
		text = strings.ReplaceAll(text, "\n", " ")
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
				fmt.Fprintf(buf, "%d. ", idx)
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
