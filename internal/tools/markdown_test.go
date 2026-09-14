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

package tools_test

import (
	"testing"

	. "fractale/fractal6.go/internal/tools"
)

func TestHtmlToMarkdown(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "plain text",
			input:    "hello world",
			expected: "hello world",
		},
		{
			name:     "paragraph",
			input:    "<p>hello world</p>",
			expected: "hello world",
		},
		{
			name:     "bold",
			input:    "<p><strong>bold text</strong></p>",
			expected: "**bold text**",
		},
		{
			name:     "italic",
			input:    "<p><em>italic text</em></p>",
			expected: "*italic text*",
		},
		{
			name:     "link",
			input:    `<a href="https://example.com">click here</a>`,
			expected: "[click here](https://example.com)",
		},
		{
			name:     "data-mention link stripped",
			input:    `<a data-mention="60993e048b3bea14f250e5f4|role">Outils numériques</a>`,
			expected: "Outils numériques",
		},
		{
			name:     "unordered list",
			input:    "<ul><li>item 1</li><li>item 2</li></ul>",
			expected: "- item 1\n- item 2",
		},
		{
			name:     "br becomes line break",
			input:    "line1<br />line2",
			expected: "line1  \nline2",
		},
		{
			name:     "nested bold in paragraph",
			input:    "<p>This is <strong>important</strong> text</p>",
			expected: "This is **important** text",
		},
		{
			name:     "inline code",
			input:    "<p>Use <code>fmt.Println</code> here</p>",
			expected: "Use `fmt.Println` here",
		},
		{
			name:     "fenced code block",
			input:    "<pre><code>func main() {\n  fmt.Println(\"hi\")\n}</code></pre>",
			expected: "```\nfunc main() {\n  fmt.Println(\"hi\")\n}\n```",
		},
		{
			name:     "strikethrough del",
			input:    "<p><del>removed</del></p>",
			expected: "~~removed~~",
		},
		{
			name:     "strikethrough s",
			input:    "<p><s>struck</s></p>",
			expected: "~~struck~~",
		},
		{
			name:     "blockquote",
			input:    "<blockquote><p>quoted text</p></blockquote>",
			expected: "> quoted text",
		},
		{
			name:     "horizontal rule",
			input:    "<p>above</p><hr><p>below</p>",
			expected: "above\n\n---\n\nbelow",
		},
		{
			name:     "image",
			input:    `<img src="https://example.com/img.png" alt="logo">`,
			expected: "![logo](https://example.com/img.png)",
		},
		{
			name:     "h5",
			input:    "<h5>Title Five</h5>",
			expected: "##### Title Five",
		},
		{
			name:     "h6",
			input:    "<h6>Title Six</h6>",
			expected: "###### Title Six",
		},
		{
			name:     "ordered list",
			input:    "<ol><li>first</li><li>second</li></ol>",
			expected: "1. first\n2. second",
		},
		{
			name:     "nested unordered list",
			input:    "<ul><li>a<ul><li>a1</li><li>a2</li></ul></li><li>b</li></ul>",
			expected: "- a\n  - a1\n  - a2\n- b",
		},
		{
			name:     "simple table",
			input:    "<table><thead><tr><th>Name</th><th>Age</th></tr></thead><tbody><tr><td>Alice</td><td>30</td></tr></tbody></table>",
			expected: "| Name | Age |\n| --- | --- |\n| Alice | 30 |",
		},
		{
			name:     "details with summary",
			input:    "<details><summary>More info</summary><p>Hidden content</p></details>",
			expected: "<details>\n<summary>More info</summary>\n\nHidden content\n\n</details>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := HTMLToMarkdown(tt.input)
			if err != nil {
				t.Fatalf("HTMLToMarkdown(%q): unexpected error: %v", tt.input, err)
			}
			if got != tt.expected {
				t.Errorf("HTMLToMarkdown(%q):\n  got:  %q\n  want: %q", tt.input, got, tt.expected)
			}
		})
	}
}
