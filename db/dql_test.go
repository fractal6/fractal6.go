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

package db_test

import (
	"bytes"
	"fmt"
	"testing"
	"text/template"

	. "fractale/fractal6.go/db"
	. "fractale/fractal6.go/internal/tools"
)

// dummyVars provides a value for every template variable used across
// DqlQueries and DqlMutations.
var dummyVars = map[string]string{
	"id":                    "0x1",
	"fieldName":             "Node.name",
	"fieldid":               "Node.nameid",
	"fieldNameSource":       "Node.parent",
	"fieldNameTarget":       "Node.name",
	"subFieldNameTarget":    "Node.nameid",
	"value":                 "test",
	"filter":                "",
	"f1":                    "Node.nameid",
	"v1":                    "a",
	"f2":                    "Node.isRoot",
	"v2":                    "true",
	"nameid":                "test-org#",
	"nameids":               `"test-org#"`,
	"nameidsProtected":      `"test-org#"`,
	"rootnameid":            "test-org#",
	"rootnameidProtected":   "test-org#",
	"nameid_old":            "test-org#old",
	"nameid_new":            "test-org#new",
	"regex":                 "test-org.*",
	"payload":               "{ uid }",
	"user_payload":          "{ uid }",
	"userid":                "user1",
	"username":              "user1",
	"email":                 "a@b.co",
	"token":                 "tok",
	"from":                  "test-org#a",
	"to":                    "test-org#b",
	"parent":                "test-org#",
	"child":                 "test-org#child",
	"query":                 "search",
	"tid":                   "0x2",
	"cid":                   "0x3",
	"colid":                 "0x4",
	"old_colid":             "0x5",
	"cardid":                "0x6",
	"ghostid":               "0x7",
	"pos":                   "1",
	"old_pos":               "0",
	"first":                 "10",
	"offset":                "0",
	"order":                 "orderdesc",
	"orderBy":               "Post.createdAt",
	"now":                   "2026-01-01T00:00:00Z",
	"old_name":              "old",
	"new_name":              "new",
	"k":                     "username",
	"v":                     "user1",
	"tensionFilter":         `@filter(eq(Tension.status, "Open"))`,
	"labelsFilter":          "",
	"authorsFilter":         "",
	"excludeSelf":           "",
	"extra_pre_vars":        "",
}

// assertTemplateRenders parses raw as a Go template and verifies it renders
// without error using dummyVars. label is used for error messages.
func assertTemplateRenders(t *testing.T, label, raw string) {
	t.Helper()
	tmpl := CleanString(raw, false)

	parsed, err := template.New(label).Parse(tmpl)
	if err != nil {
		t.Errorf("%s: failed to parse: %v", label, err)
		return
	}

	var buf bytes.Buffer
	if err := parsed.Execute(&buf, dummyVars); err != nil {
		t.Errorf("%s: failed to render: %v", label, err)
		return
	}
	if buf.Len() == 0 {
		t.Errorf("%s: rendered to empty string", label)
	}
}

// TestDqlQueriesRender verifies that every DQL query template parses and
// renders without error.
func TestDqlQueriesRender(t *testing.T) {
	for name, raw := range DqlQueries {
		assertTemplateRenders(t, name, raw)
	}
}

// TestDqlMutationsRender verifies that every DQL mutation template (query,
// set, delete, and condition parts) parses and renders without error.
func TestDqlMutationsRender(t *testing.T) {
	for name, qm := range DqlMutations {
		assertTemplateRenders(t, name+".Q", qm.Q)
		for i, x := range qm.M {
			if x.S != "" {
				assertTemplateRenders(t, fmt.Sprintf("%s.M[%d].S", name, i), x.S)
			}
			if x.D != "" {
				assertTemplateRenders(t, fmt.Sprintf("%s.M[%d].D", name, i), x.D)
			}
			if x.C != "" {
				assertTemplateRenders(t, fmt.Sprintf("%s.M[%d].C", name, i), x.C)
			}
		}
	}
}
