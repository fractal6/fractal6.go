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
	"reflect"
	"testing"

	. "fractale/fractal6.go/internal/tools"
)

func TestFindUsername(t *testing.T) {
	testcases := []struct {
		input string
		want  []string
	}{
		{"me", []string{}},
		{"@me", []string{"me"}},
		{"@me.", []string{"me"}},
		{"me @me me", []string{"me"}},
		{"@me @me_me", []string{"me", "me_me"}},
		{"(@me)", []string{"me"}},
		{"[@me]", []string{}},
	}

	for _, test := range testcases {
		got := FindUsernames(test.input)
		if !reflect.DeepEqual(got, test.want) {
			t.Errorf("For p = %s, want %s. Got %s (len %d).",
				test.input, test.want, got, len(got))
		}
	}
}

func TestNameidEncoder(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Simple Name", "simple-name"},
		{"Cercle d'Ancrage UdN (CA)", "cercle-d_ancrage-udn-_ca"},
		{"Com&Médias", "com-médias"},
		{"Offres inter-orga", "offres-inter-orga"},
		{"RH - Richesses Humaines", "rh-richesses-humaines"},
		{"  spaces  around  ", "spaces-around"},
		{"a///b", "a-b"},
		{"hello@world", "hello_world"},
		{"a(b)c", "a_b_c"},
		{"test#hash&amp", "test-hash-amp"},
		{"already-clean", "already-clean"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := NameidEncoder(tt.input)
			if got != tt.want {
				t.Errorf("NameidEncoder(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestStripEmailQuote(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "no quote",
			input:    "Hello, this is my reply.",
			expected: "Hello, this is my reply.",
		},
		{
			name:     "english quote at end",
			input:    "Here is my reply.\n\nOn Mon, 27 Mar 2026 at 10:00, Alice <alice@example.com> wrote:\n> Original message\n> second line",
			expected: "Here is my reply.",
		},
		{
			name:     "french quote at end",
			input:    "Voici ma réponse.\n\nLe lun. 27 mars 2026 à 10:00, Alice <alice@example.com> a écrit :\n> Message original\n> deuxième ligne",
			expected: "Voici ma réponse.",
		},
		{
			name:     "french ecrit without accent",
			input:    "Ma réponse.\n\nLe 27 mars 2026, Bob a ecrit:\n> texte",
			expected: "Ma réponse.",
		},
		{
			name:     "quote at start with > lines",
			input:    "On Mon, 27 Mar 2026, Alice wrote:\n> quoted line\n> another quoted\n\nMy actual reply here.",
			expected: "My actual reply here.",
		},
		{
			name:     "quote in the middle not stripped",
			input:    "Before.\n\nOn Mon, 27 Mar 2026, Alice wrote:\n> quoted\n\nAfter this line.",
			expected: "Before.\n\nOn Mon, 27 Mar 2026, Alice wrote:\n> quoted\n\nAfter this line.",
		},
		{
			name:     "blank lines between quoted lines at end",
			input:    "Reply.\n\nOn Mon, 27 Mar 2026, Alice wrote:\n> line 1\n\n> line 2",
			expected: "Reply.",
		},
		{
			name:     "only quote returns original",
			input:    "On Mon, 27 Mar 2026, Alice wrote:\n> everything is quoted",
			expected: "On Mon, 27 Mar 2026, Alice wrote:\n> everything is quoted",
		},
		{
			name:     "signature at end",
			input:    "Hello, this is my reply.\n\n--\nJohn Doe\ntel: 02020",
			expected: "Hello, this is my reply.",
		},
		{
			name:     "signature at end with trailing blank lines",
			input:    "Hello, this is my reply.\n\n--\nJohn Doe\ntel: 02020\n\n",
			expected: "Hello, this is my reply.",
		},
		{
			name:     "signature without preceding blank line not stripped",
			input:    "Hello, this is my reply.\n--\nJohn Doe",
			expected: "Hello, this is my reply.\n--\nJohn Doe",
		},
		{
			name:     "bare -- with no content after not stripped",
			input:    "Hello, this is my reply.\n\n--\n\n",
			expected: "Hello, this is my reply.\n\n--\n\n",
		},
		{
			name:     "signature after quote stripped",
			input:    "My reply.\n\n--\nJohn Doe\n\nOn Mon, 27 Mar 2026, Alice wrote:\n> quoted",
			expected: "My reply.",
		},
		{
			name:     "only signature returns original",
			input:    "\n--\nJohn Doe",
			expected: "\n--\nJohn Doe",
		},
		{
			name:     "text after signature block not stripped",
			input:    "Hello.\n\n--\nJohn Doe\n\nSome extra text here.",
			expected: "Hello.\n\n--\nJohn Doe\n\nSome extra text here.",
		},
		{
			name:     "text between signature and quote not stripped",
			input:    "Reply.\n\n--\nJohn Doe\n\nPS: forgot to mention this.\n\nOn Mon, 27 Mar 2026, Alice wrote:\n> quoted",
			expected: "Reply.\n\n--\nJohn Doe\n\nPS: forgot to mention this.",
		},
		{
			name:     "signature after quote both stripped",
			input:    "My reply.\n\nOn Mon, 27 Mar 2026, Alice wrote:\n> quoted\n\n--\nJohn Doe\ntel: 02020",
			expected: "My reply.",
		},
		{
			name:     "signature before quote both stripped",
			input:    "My reply.\n\n--\nJohn Doe\ntel: 02020\n\nOn Mon, 27 Mar 2026, Alice wrote:\n> quoted",
			expected: "My reply.",
		},
		{
			name:     "text between signature and EOF not stripped",
			input:    "Reply.\n\n--\nJohn Doe\n\nActually one more thing.",
			expected: "Reply.\n\n--\nJohn Doe\n\nActually one more thing.",
		},
		{
			name:     "signature delimiter with trailing space (RFC 3676)",
			input:    "Hello.\n\n-- \nJohn Doe\ntel: 02020",
			expected: "Hello.",
		},
		{
			name:     "quote at start and signature at end",
			input:    "On Mon, 27 Mar 2026, Alice wrote:\n> quoted\n\nMy reply.\n\n--\nJohn Doe",
			expected: "My reply.",
		},
		{
			name:     "two -- blocks only last one is signature",
			input:    "First part.\n\n--\nSeparator text\n\nSecond part.\n\n--\nJohn Doe",
			expected: "First part.\n\n--\nSeparator text\n\nSecond part.",
		},
		{
			name:     "inline french header on same line as reply",
			input:    "Yes, it's fine. Le jeu. 23 avr. 2026 à 05:49, Adrien D. <notifications@fractale.co> a écrit :\n> Hi @maud.srd, message content.",
			expected: "Yes, it's fine.",
		},
		{
			name:     "inline english header on same line as reply",
			input:    "Yes, it's fine. On Mon, 27 Mar 2026 at 10:00, Alice <alice@example.com> wrote:\n> Hi, message content.",
			expected: "Yes, it's fine.",
		},
		{
			name:     "fractale footer anchors inline header strip",
			input:    "Yes, it's fine. Le jeu. 23 avr. 2026 à 05:49, Adrien D. <notifications@fractale.co> a écrit :\n> Hi @maud.srd, message content.\n> —\n> You are receiving this because you have been mentionned.\n> View it on Fractale or reply to this email directly.",
			expected: "Yes, it's fine.",
		},
		{
			name:     "fractale footer strips quoted block without > prefix",
			input:    "My reply.\nLe jeu. 23 avr. 2026 à 05:49, Adrien D. <notifications@fractale.co> a écrit :\n\nHi @maud.srd, message content.\n—\nYou are receiving this because you have been mentionned.\nView it on Fractale or reply to this email directly.",
			expected: "My reply.",
		},
		{
			name:     "fractale footer alone without quote header",
			input:    "My reply.\n—\nYou are receiving this because you have been mentionned.\nView it on Fractale or reply to this email directly.",
			expected: "My reply.",
		},
		{
			name:     "german quote at end",
			input:    "Meine Antwort.\n\nAm 27.03.2026 um 10:00 schrieb Alice <alice@example.com>:\n> Original Nachricht\n> zweite Zeile",
			expected: "Meine Antwort.",
		},
		{
			name:     "spanish quote at end",
			input:    "Mi respuesta.\n\nEl lun., 27 mar. 2026 a las 10:00, Alice <alice@example.com> escribió:\n> Mensaje original\n> segunda línea",
			expected: "Mi respuesta.",
		},
		{
			name:     "quote header directly after reply without blank line",
			input:    "My reply.\nOn Mon, 27 Mar 2026, Alice <alice@example.com> wrote:\n> quoted line\n> second line",
			expected: "My reply.",
		},
		{
			name:     "unquoted quote block after header stripped",
			input:    "My reply.\nOn Mon, 27 Mar 2026, Alice <alice@example.com> wrote:\nquoted line\nsecond quoted",
			expected: "My reply.",
		},
		{
			name:     "unquoted quote block after header and blank line stripped",
			input:    "My reply.\n\nOn Mon, 27 Mar 2026, Alice <alice@example.com> wrote:\n\nquoted line",
			expected: "My reply.",
		},
		{
			name:     "unmarked text below quoted lines kept (bottom-post)",
			input:    "Before.\n\nOn Mon, 27 Mar 2026, Alice wrote:\n> quoted\n\nAfter this line.",
			expected: "Before.\n\nOn Mon, 27 Mar 2026, Alice wrote:\n> quoted\n\nAfter this line.",
		},
		{
			name:     "outlook original message divider",
			input:    "My reply.\n\n-----Original Message-----\nFrom: Alice <alice@example.com>\nSent: Monday, March 27, 2026\nSubject: Re: something\n\nOriginal text here.",
			expected: "My reply.",
		},
		{
			name:     "outlook french message d'origine divider",
			input:    "Ma réponse.\n\n-----Message d'origine-----\nDe : Alice\nEnvoyé : lundi 27 mars 2026\n\nTexte original.",
			expected: "Ma réponse.",
		},
		{
			name:     "only View it on Fractale anchor (first line stripped)",
			input:    "My reply.\n> [View it on Fractale](https://fractale.co/tension/x/y), reply to this email directly.",
			expected: "My reply.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StripEmailQuote(tt.input)
			if got != tt.expected {
				t.Errorf("StripEmailQuote():\n  got:  %q\n  want: %q", got, tt.expected)
			}
		})
	}
}

// TestStripEmailQuote_GmailHTML exercises the real notifier path:
// raw Gmail reply HTML -> HTMLToMarkdown -> StripEmailQuote.
func TestStripEmailQuote_GmailHTML(t *testing.T) {
	tests := []struct {
		name     string
		html     string
		expected string
	}{
		{
			// No <br> separator: <div> emits no newline, so the reply and
			// the gmail_attr quote-header collapse onto one line.
			name: "gmail reply without br between reply and quote",
			html: `<div dir="ltr">Yes, it's fine.</div>` +
				`<div class="gmail_quote gmail_quote_container">` +
				`<div dir="ltr" class="gmail_attr">Le jeu. 23 avr. 2026 à 05:49, Adrien D. &lt;<a href="mailto:notifications@fractale.co">notifications@fractale.co</a>&gt; a écrit :<br></div>` +
				`<blockquote class="gmail_quote">Hi @maud.srd, is it ok for you?<br>— <br>You are receiving this because you have been mentionned.<br><a href="https://fractale.co/tension/x/y">View it on Fractale</a> or reply to this email directly.</blockquote>` +
				`</div>`,
			expected: "Yes, it's fine.",
		},
		{
			name: "gmail reply with br between reply and quote",
			html: `<div dir="ltr">Yes, it's fine.</div><br>` +
				`<div class="gmail_quote">` +
				`<div dir="ltr" class="gmail_attr">Le jeu. 23 avr. 2026 à 05:49, Adrien D. &lt;notifications@fractale.co&gt; a écrit :<br></div>` +
				`<blockquote class="gmail_quote">Hi @maud.srd, short message.<br>— <br>You are receiving this because you have been mentionned.</blockquote>` +
				`</div>`,
			expected: "Yes, it's fine.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			md, err := HTMLToMarkdown(tt.html)
			if err != nil {
				t.Fatalf("HTMLToMarkdown error: %v", err)
			}
			got := StripEmailQuote(md)
			if got != tt.expected {
				t.Errorf("StripEmailQuote(HTMLToMarkdown(html)):\n  got:  %q\n  want: %q\n  intermediate md: %q", got, tt.expected, md)
			}
		})
	}
}

func TestFindTension(t *testing.T) {
	testcases := []struct {
		input string
		want  []string
	}{
		{"123", []string{}},
		{"0x0123f", []string{"0x0123f"}},
		{"0x0123f.", []string{"0x0123f"}},
		{"0x0123fg", []string{}},
		{"me 0x123 me", []string{"0x123"}},
		{"0x123 0xabc", []string{"0x123", "0xabc"}},
		{"(0x123)", []string{"0x123"}},
		{"[0x123]", []string{}},
	}

	for _, test := range testcases {
		got := FindTensions(test.input)
		if !reflect.DeepEqual(got, test.want) {
			t.Errorf("For p = %s, want %s. Got %s (len %d).",
				test.input, test.want, got, len(got))
		}
	}
}
