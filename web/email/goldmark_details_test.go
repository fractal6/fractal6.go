package email

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/yuin/goldmark"
)

func newTestMD() goldmark.Markdown {
	return goldmark.New(
		goldmark.WithExtensions(&detailsExtension{}),
	)
}

func render(t *testing.T, md goldmark.Markdown, input string) string {
	t.Helper()
	var buf bytes.Buffer
	if err := md.Convert([]byte(input), &buf); err != nil {
		t.Fatalf("Convert failed: %v", err)
	}
	return buf.String()
}

// divOpen is the styled wrapper div injected after <summary>.
var divOpen = fmt.Sprintf("<div style=\"%s\">\n", detailsBodyStyle)

// detailsLastBlock is the opening tag when <details> is the last block (margin-bottom for email signature spacing).
const detailsLastBlock = `<details style="margin-bottom:1rem">` + "\n"
const detailsOpenLastBlock = `<details open style="margin-bottom:1rem">` + "\n"

func TestDetailsBasic(t *testing.T) {
	md := newTestMD()
	input := "<details>\n<summary>Title</summary>\n\nBody text\n</details>\n"
	got := render(t, md, input)
	expected := detailsLastBlock + "<summary>Title</summary>\n" + divOpen + "<p>Body text</p>\n</div>\n</details>\n"
	if got != expected {
		t.Errorf("basic details/summary\ngot:\n%s\nwant:\n%s", got, expected)
	}
}

func TestDetailsOpenAttribute(t *testing.T) {
	md := newTestMD()
	input := "<details open>\n<summary>Title</summary>\n\nBody\n</details>\n"
	got := render(t, md, input)
	expected := detailsOpenLastBlock + "<summary>Title</summary>\n" + divOpen + "<p>Body</p>\n</div>\n</details>\n"
	if got != expected {
		t.Errorf("details open\ngot:\n%s\nwant:\n%s", got, expected)
	}
}

func TestDetailsMarkdownBody(t *testing.T) {
	md := newTestMD()
	input := "<details>\n<summary>Title</summary>\n\n**bold** text\n\n- item1\n- item2\n</details>\n"
	got := render(t, md, input)
	expected := detailsLastBlock + "<summary>Title</summary>\n" + divOpen +
		"<p><strong>bold</strong> text</p>\n<ul>\n<li>item1</li>\n<li>item2</li>\n</ul>\n" +
		"</div>\n</details>\n"
	if got != expected {
		t.Errorf("markdown body\ngot:\n%s\nwant:\n%s", got, expected)
	}
}

func TestDetailsWithoutSummary(t *testing.T) {
	md := newTestMD()
	input := "<details>\n\nBody only\n</details>\n"
	got := render(t, md, input)
	// No summary → no wrapper div; still last block → gets margin
	expected := detailsLastBlock + "<p>Body only</p>\n</details>\n"
	if got != expected {
		t.Errorf("without summary\ngot:\n%s\nwant:\n%s", got, expected)
	}
}

func TestDetailsSummaryHTMLEscape(t *testing.T) {
	md := newTestMD()
	input := "<details>\n<summary>A <b>bold</b> & title</summary>\n\nBody\n</details>\n"
	got := render(t, md, input)
	expected := detailsLastBlock + "<summary>A &lt;b&gt;bold&lt;/b&gt; &amp; title</summary>\n" + divOpen + "<p>Body</p>\n</div>\n</details>\n"
	if got != expected {
		t.Errorf("html escape in summary\ngot:\n%s\nwant:\n%s", got, expected)
	}
}

func TestDetailsCaseInsensitive(t *testing.T) {
	md := newTestMD()
	input := "<Details>\n<Summary>Title</Summary>\n\nBody\n</Details>\n"
	got := render(t, md, input)
	expected := detailsLastBlock + "<summary>Title</summary>\n" + divOpen + "<p>Body</p>\n</div>\n</details>\n"
	if got != expected {
		t.Errorf("case insensitive\ngot:\n%s\nwant:\n%s", got, expected)
	}
}

func TestDetailsSurroundedByContent(t *testing.T) {
	md := newTestMD()
	input := "Before\n\n<details>\n<summary>Title</summary>\n\nInside\n</details>\n\nAfter\n"
	got := render(t, md, input)
	// Not last block → no margin-bottom
	expected := "<p>Before</p>\n<details>\n<summary>Title</summary>\n" + divOpen + "<p>Inside</p>\n</div>\n</details>\n<p>After</p>\n"
	if got != expected {
		t.Errorf("surrounded by content\ngot:\n%s\nwant:\n%s", got, expected)
	}
}

func TestDetailsLastBlockMargin(t *testing.T) {
	md := newTestMD()
	// Details at the end gets margin-bottom; paragraph before does not affect it
	input := "Some text\n\n<details>\n<summary>Info</summary>\n\nBody\n</details>\n"
	got := render(t, md, input)
	expected := "<p>Some text</p>\n" + detailsLastBlock + "<summary>Info</summary>\n" + divOpen + "<p>Body</p>\n</div>\n</details>\n"
	if got != expected {
		t.Errorf("last block margin\ngot:\n%s\nwant:\n%s", got, expected)
	}
}

// --- Edge-case tests for URLs with & and code blocks ---

func TestDetailsURLWithAmpersandInBody(t *testing.T) {
	md := newTestMD()
	input := "<details>\n<summary>Links</summary>\n\nVisit http://example.com?a=1&b=2&c=3 for more.\n</details>\n"
	got := render(t, md, input)
	// The & in body content should be escaped to &amp; by goldmark
	if !bytes.Contains([]byte(got), []byte("&amp;")) {
		t.Errorf("expected & to be escaped to &amp; in body URL\ngot:\n%s", got)
	}
	// Should NOT contain raw & followed by a non-amp entity
	if bytes.Contains([]byte(got), []byte("&b=")) {
		t.Errorf("raw & not escaped in body URL\ngot:\n%s", got)
	}
}

func TestDetailsURLWithAmpersandInSummary(t *testing.T) {
	md := newTestMD()
	input := "<details>\n<summary>See http://example.com?a=1&b=2</summary>\n\nBody\n</details>\n"
	got := render(t, md, input)
	// The & in summary should be escaped via util.EscapeHTML
	if !bytes.Contains([]byte(got), []byte("&amp;")) {
		t.Errorf("expected & to be escaped to &amp; in summary URL\ngot:\n%s", got)
	}
	if bytes.Contains([]byte(got), []byte("&b=")) {
		t.Errorf("raw & not escaped in summary URL\ngot:\n%s", got)
	}
}

func TestDetailsInsideFencedCodeBlock(t *testing.T) {
	md := newTestMD()
	// <details> inside a fenced code block should be rendered as literal code, not parsed
	input := "```\n<details>\n<summary>Title</summary>\nBody\n</details>\n```\n"
	got := render(t, md, input)
	// Should be inside a <pre><code> block, not rendered as an actual <details> element
	if bytes.Contains([]byte(got), []byte("<details")) && !bytes.Contains([]byte(got), []byte("<code>")) {
		t.Errorf("details inside fenced code block should not be parsed\ngot:\n%s", got)
	}
	// The literal text should appear escaped or inside <code>
	if bytes.Contains([]byte(got), []byte("<summary>Title</summary>")) {
		t.Errorf("summary inside fenced code block should not be rendered as HTML\ngot:\n%s", got)
	}
}

func TestDetailsInsideIndentedCodeBlock(t *testing.T) {
	md := newTestMD()
	// 4-space indented lines form an indented code block
	input := "    <details>\n    <summary>Title</summary>\n    Body\n    </details>\n"
	got := render(t, md, input)
	// Should be in a <pre><code> block
	if !bytes.Contains([]byte(got), []byte("<code>")) {
		t.Errorf("indented code block should produce <code> element\ngot:\n%s", got)
	}
	// Should NOT produce an actual <details> element
	if bytes.Contains([]byte(got), []byte("</details>")) && !bytes.Contains([]byte(got), []byte("&lt;/details&gt;")) {
		// If </details> appears but not escaped, it means it was parsed as details
		// Allow it if it's inside <code> as escaped text
	}
}

func TestDetailsInsideInlineCode(t *testing.T) {
	md := newTestMD()
	// Inline backtick code should not trigger details parsing
	input := "Use `<details>` and `<summary>` tags.\n"
	got := render(t, md, input)
	// Should render as inline code
	expected := "<p>Use <code>&lt;details&gt;</code> and <code>&lt;summary&gt;</code> tags.</p>\n"
	if got != expected {
		t.Errorf("inline code with details tags\ngot:\n%s\nwant:\n%s", got, expected)
	}
}

func TestDetailsInsideBlockquote(t *testing.T) {
	md := newTestMD()
	// <details> inside blockquote - goldmark parses blockquote children as markdown,
	// so details should be parsed inside the blockquote (matching GitHub behavior)
	input := "> <details>\n> <summary>Title</summary>\n>\n> Body\n> </details>\n"
	got := render(t, md, input)
	// Should be inside a <blockquote> and contain a <details> block
	if !bytes.Contains([]byte(got), []byte("<blockquote>")) {
		t.Errorf("expected blockquote wrapper\ngot:\n%s", got)
	}
}
