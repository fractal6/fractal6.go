package email

import (
	"bytes"
	"strings"
	"testing"
)

func renderMD(t *testing.T, input string) string {
	t.Helper()
	var buf bytes.Buffer
	if err := md.Convert([]byte(input), &buf); err != nil {
		t.Fatalf("Convert failed: %v", err)
	}
	return buf.String()
}

// Bare angle-bracket words must survive: the original bug.
// Before: <tension> was parsed as raw HTML and dropped.
// After: it stays as literal text, escaped to &lt;tension&gt;.
func TestMarkdown_AngleBracketWordsArePreserved(t *testing.T) {
	input := "les messages circule <tension> <--> <canal de messagerie> (anonymiser numéro)"
	got := renderMD(t, input)

	for _, want := range []string{"&lt;tension&gt;", "&lt;canal de messagerie&gt;", "anonymiser numéro"} {
		if !strings.Contains(got, want) {
			t.Errorf("expected %q in output\ngot: %s", want, got)
		}
	}
}

// Inline HTML tags become literal text (the trade-off we accept).
func TestMarkdown_InlineHTMLBecomesLiteral(t *testing.T) {
	got := renderMD(t, "press <kbd>Ctrl+C</kbd> to copy")
	if !strings.Contains(got, "&lt;kbd&gt;") || !strings.Contains(got, "&lt;/kbd&gt;") {
		t.Errorf("expected <kbd> tags to be escaped as literal text\ngot: %s", got)
	}
}

// Bracketed autolinks must still work — handled by AutoLinkParser, not RawHTMLParser.
func TestMarkdown_BracketedAutolinkURL(t *testing.T) {
	got := renderMD(t, "see <https://example.com> for details")
	if !strings.Contains(got, `<a href="https://example.com">https://example.com</a>`) {
		t.Errorf("expected bracketed URL autolink\ngot: %s", got)
	}
}

func TestMarkdown_BracketedAutolinkEmail(t *testing.T) {
	got := renderMD(t, "contact <admin@example.com>")
	if !strings.Contains(got, `<a href="mailto:admin@example.com">admin@example.com</a>`) {
		t.Errorf("expected bracketed email autolink\ngot: %s", got)
	}
}

// GFM bare-URL autolink must still work.
func TestMarkdown_BareURLAutolink(t *testing.T) {
	got := renderMD(t, "visit https://example.com today")
	if !strings.Contains(got, `<a href="https://example.com">https://example.com</a>`) {
		t.Errorf("expected bare URL autolink (GFM)\ngot: %s", got)
	}
}

// XSS-style raw HTML must not pass through — independent of the sanitizer.
func TestMarkdown_ScriptTagDoesNotPassThrough(t *testing.T) {
	got := renderMD(t, "hello <script>alert(1)</script> world")
	if strings.Contains(got, "<script>") {
		t.Errorf("script tag must not pass through goldmark\ngot: %s", got)
	}
}

// Code spans still treat angle brackets literally — already worked, regression guard.
func TestMarkdown_CodeSpanWithAngleBrackets(t *testing.T) {
	got := renderMD(t, "use `<tension>` here")
	if !strings.Contains(got, "<code>&lt;tension&gt;</code>") {
		t.Errorf("expected code span with escaped angle brackets\ngot: %s", got)
	}
}

// Emphasis and links still parse normally — regression guard.
func TestMarkdown_EmphasisAndLinkStillWork(t *testing.T) {
	got := renderMD(t, "*bold-ish* and [link](https://example.com)")
	if !strings.Contains(got, "<em>bold-ish</em>") {
		t.Errorf("emphasis broken\ngot: %s", got)
	}
	if !strings.Contains(got, `<a href="https://example.com">link</a>`) {
		t.Errorf("link broken\ngot: %s", got)
	}
}

// Block-level <details> (custom extension) still renders.
func TestMarkdown_DetailsBlockStillWorks(t *testing.T) {
	input := "<details>\n<summary>Title</summary>\n\nBody\n</details>\n"
	got := renderMD(t, input)
	if !strings.Contains(got, "<details") || !strings.Contains(got, "<summary>Title</summary>") {
		t.Errorf("details block broken\ngot: %s", got)
	}
}
