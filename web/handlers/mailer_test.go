/*
 * Fractale - Self-organisation for humans.
 * Copyright (C) 2026 Fractale Co
 */

package handlers

import "testing"

// TestParseEmailReferences covers the inbound-reply routing header. The
// References value is sender-controlled, so the parser must reject anything
// that isn't a well-formed uid (it feeds DQL uid() roots downstream) and must
// not panic on malformed input.
func TestParseEmailReferences(t *testing.T) {
	cases := []struct {
		name, refs string
		wantTid    string
		wantCid    string
		wantErr    bool
	}{
		{name: "tension ref", refs: "<tension/0x123@fractale.co>", wantTid: "0x123"},
		{name: "contract ref", refs: "<contract/0xabc@fractale.co>", wantCid: "0xabc"},
		{name: "picks first of many", refs: "<other/x@d> <tension/0x1@d> <contract/0x2@d>", wantTid: "0x1"},
		{name: "no reference", refs: ""},
		{name: "unrelated reference", refs: "<CAF=abc123@mail.gmail.com>"},

		// Missing "@" used to panic on l[8:strings.Index(l,"@")] (index -1).
		{name: "no at sign", refs: "<tension/0x123>"},
		{name: "no at sign contract", refs: "<contract/0x123>"},

		// uid() accepts comma lists, so a forged header must not widen the query.
		{name: "comma list tid", refs: "<tension/0x1,0x2@d>", wantErr: true},
		{name: "comma list cid", refs: "<contract/0x1,0x2@d>", wantErr: true},
		{name: "non-hex tid", refs: "<tension/abc@d>", wantErr: true},
		{name: "empty tid", refs: "<tension/@d>"},
		{name: "quote breakout", refs: `<tension/0x1"){q(func:uid(0x2@d>`, wantErr: true},
		// contractid form ({tid}#{event}#old#new) reaches eq(Contract.contractid)
		// unquoted-safe, but is not a uid and never appears in our outbound mails.
		{name: "contractid form", refs: "<contract/0x1#Created##@d>", wantErr: true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			tid, cid, err := parseEmailReferences(c.refs)
			if (err != nil) != c.wantErr {
				t.Fatalf("err = %v, wantErr = %v", err, c.wantErr)
			}
			if tid != c.wantTid {
				t.Errorf("tid = %q, want %q", tid, c.wantTid)
			}
			if cid != c.wantCid {
				t.Errorf("cid = %q, want %q", cid, c.wantCid)
			}
		})
	}
}
