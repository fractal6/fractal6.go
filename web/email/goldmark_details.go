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

package email

import (
	"bytes"
	"regexp"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// ---------------------
// AST Nodes
// ---------------------

// KindDetails is the NodeKind for a Details block.
var KindDetails = ast.NewNodeKind("Details")

// Details represents a <details> HTML block whose body is parsed as markdown.
type Details struct {
	ast.BaseBlock
	Open bool // true if <details open>
}

func (n *Details) Kind() ast.NodeKind { return KindDetails }
func (n *Details) Dump(source []byte, level int) {
	ast.DumpHelper(n, source, level, nil, nil)
}

// KindSummary is the NodeKind for a Summary block.
var KindSummary = ast.NewNodeKind("Summary")

// Summary represents a <summary> element inside a Details block.
// Content holds the raw summary text (rendered as HTML-escaped text).
// Named "Content" to avoid shadowing ast.BaseBlock.Text().
type Summary struct {
	ast.BaseBlock
	Content []byte
}

func (n *Summary) Kind() ast.NodeKind { return KindSummary }
func (n *Summary) Dump(source []byte, level int) {
	ast.DumpHelper(n, source, level, nil, nil)
}

// ---------------------
// Block Parser
// ---------------------

var (
	detailsOpenRe  = regexp.MustCompile(`(?i)^\s*<details(\s+open)?\s*>\s*$`)
	summaryRe      = regexp.MustCompile(`(?i)^\s*<summary>(.*?)</summary>\s*$`)
	detailsCloseRe = regexp.MustCompile(`(?i)^\s*</details>\s*$`)
)

type detailsBlockParser struct{}

func (p *detailsBlockParser) Trigger() []byte {
	return []byte{'<'}
}

func (p *detailsBlockParser) Open(parent ast.Node, reader text.Reader, pc parser.Context) (ast.Node, parser.State) {
	line, _ := reader.PeekLine()
	matches := detailsOpenRe.FindSubmatch(line)
	if matches == nil {
		return nil, parser.NoChildren
	}

	node := &Details{
		Open: len(matches[1]) > 0,
	}

	// Consume the line content (leave newline for framework's AdvanceLine).
	reader.Advance(contentLen(line))
	return node, parser.HasChildren
}

func (p *detailsBlockParser) Continue(node ast.Node, reader text.Reader, pc parser.Context) parser.State {
	line, _ := reader.PeekLine()

	// Close on </details>
	if detailsCloseRe.Match(line) {
		reader.Advance(contentLen(line))
		return parser.Close
	}

	// Capture <summary>...</summary> as a child node.
	// Consume the line and return Continue without HasChildren so
	// goldmark does not try to parse this line as child blocks.
	if matches := summaryRe.FindSubmatch(line); matches != nil {
		summary := &Summary{
			Content: bytes.TrimSpace(matches[1]),
		}
		node.AppendChild(node, summary)
		reader.Advance(contentLen(line))
		return parser.Continue
	}

	// Body lines: don't advance, let goldmark parse the full line as children.
	return parser.Continue | parser.HasChildren
}

// contentLen returns the length of line content excluding the trailing newline.
func contentLen(line []byte) int {
	return len(bytes.TrimRight(line, "\n\r"))
}

func (p *detailsBlockParser) Close(node ast.Node, reader text.Reader, pc parser.Context) {
	// no-op
}

func (p *detailsBlockParser) CanInterruptParagraph() bool { return true }
func (p *detailsBlockParser) CanAcceptIndentedLine() bool { return false }

// ---------------------
// HTML Renderer
// ---------------------

type detailsHTMLRenderer struct{}

func (r *detailsHTMLRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(KindDetails, r.renderDetails)
	reg.Register(KindSummary, r.renderSummary)
}

func (r *detailsHTMLRenderer) renderDetails(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		n := node.(*Details)
		// Add bottom margin when this is the last block in the message,
		// to create spacing before the auto-generated email signature.
		style := ""
		if node.NextSibling() == nil {
			style = ` style="margin-bottom:1rem"`
		}
		if n.Open {
			_, _ = w.WriteString("<details open" + style + ">\n")
		} else {
			_, _ = w.WriteString("<details" + style + ">\n")
		}
	} else {
		// Close the body wrapper if a <summary> was present
		if node.FirstChild() != nil && node.FirstChild().Kind() == KindSummary {
			_, _ = w.WriteString("</div>\n")
		}
		_, _ = w.WriteString("</details>\n")
	}
	return ast.WalkContinue, nil
}

const detailsBodyStyle = `border-left:2px solid #d0d7de;margin-left:1rem;padding-left:1rem`

func (r *detailsHTMLRenderer) renderSummary(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		n := node.(*Summary)
		_, _ = w.WriteString("<summary>")
		_, _ = w.Write(util.EscapeHTML(n.Content))
		_, _ = w.WriteString("</summary>\n")
	} else {
		// Open a styled wrapper div for all body content that follows.
		// Closed by the parent Details' exiting renderer.
		_, _ = w.WriteString("<div style=\"" + detailsBodyStyle + "\">\n")
	}
	return ast.WalkSkipChildren, nil
}

// ---------------------
// Extension (Extender)
// ---------------------

// detailsExtension is a goldmark extension that adds support for
// collapsible <details>/<summary> blocks with markdown body content.
type detailsExtension struct{}

func (e *detailsExtension) Extend(m goldmark.Markdown) {
	m.Parser().AddOptions(
		parser.WithBlockParsers(
			// Priority 650: above HTMLBlock (600) so we match <details> first.
			util.Prioritized(&detailsBlockParser{}, 650),
		),
	)
	m.Renderer().AddOptions(
		renderer.WithNodeRenderers(
			util.Prioritized(&detailsHTMLRenderer{}, 500),
		),
	)
}
