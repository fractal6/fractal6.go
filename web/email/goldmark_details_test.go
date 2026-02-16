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
