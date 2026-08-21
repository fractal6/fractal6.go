//go:build integration

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

// Regression: moving an ordinary tension (no blob, no governed node) must not go
// through the governance subject resolution. Own org (movorg) to avoid leaking
// roles into the shared test orgs.

package handlers_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"fractale/fractal6.go/db"
	"fractale/fractal6.go/internal/testutil"
	"fractale/fractal6.go/web/sessions"
)

const (
	movSrc = "movorg#movsrc"
	movDst = "movorg#movdst"
)

func TestMoveOrdinaryTension(t *testing.T) {
	nqMovCircle := func(blank, nameid, parent string) string {
		name := nameid[strings.LastIndex(nameid, "#")+1:]
		return fmt.Sprintf(`
			%[1]s <dgraph.type> "Node" .
			%[1]s <Node.nameid> %[2]q .
			%[1]s <Node.rootnameid> "movorg" .
			%[1]s <Node.name> %[3]q .
			%[1]s <Node.type_> "Circle" .
			%[1]s <Node.mode> "Coordinated" .
			%[1]s <Node.visibility> "Public" .
			%[1]s <Node.isArchived> "false" .
			%[1]s <Node.isRoot> "false" .
			%[1]s <Node.parent> %[4]s .
			%[4]s <Node.children> %[1]s .
		`, blank, nameid, name, parent)
	}
	set := `
		_:org <dgraph.type> "Node" .
		_:org <Node.nameid> "movorg" .
		_:org <Node.rootnameid> "movorg" .
		_:org <Node.name> "movorg" .
		_:org <Node.type_> "Circle" .
		_:org <Node.mode> "Coordinated" .
		_:org <Node.visibility> "Public" .
		_:org <Node.isArchived> "false" .
		_:org <Node.isRoot> "true" .
		_:coordo <dgraph.type> "Node" .
		_:coordo <Node.nameid> "movorg##@movcoordo" .
		_:coordo <Node.rootnameid> "movorg" .
		_:coordo <Node.name> "movcoordo" .
		_:coordo <Node.type_> "Role" .
		_:coordo <Node.role_type> "Coordinator" .
		_:coordo <Node.mode> "Coordinated" .
		_:coordo <Node.visibility> "Public" .
		_:coordo <Node.isArchived> "false" .
		_:coordo <Node.isRoot> "false" .
		_:coordo <Node.parent> _:org .
		_:org <Node.children> _:coordo .
		_:coordo <Node.first_link> uid(u1) .
		uid(u1) <User.roles> _:coordo .
		_:t <dgraph.type> "Tension" .
		_:t <Tension.title> "movorg-plain-tension" .
		_:t <Tension.status> "Open" .
		_:t <Tension.type_> "Operational" .
		_:t <Tension.emitter> _:src .
		_:t <Tension.emitterid> ` + fmt.Sprintf("%q", movSrc) + ` .
		_:t <Tension.receiver> _:src .
		_:t <Tension.receiverid> ` + fmt.Sprintf("%q", movSrc) + ` .
		_:t <Post.createdBy> uid(u1) .
		_:t <Post.createdAt> "2027-01-01T00:00:00Z" .
	` + nqMovCircle("_:src", movSrc, "_:org") + nqMovCircle("_:dst", movDst, "_:org")

	qm := db.QueryMut{
		Q: `query {
			u1 as var(func: eq(User.username, "` + testutil.TestUser + `"))
		}`,
		M: []db.X{{S: set}},
	}
	res, err := db.GetDB().UpsertDql(qm, map[string]string{})
	if err != nil {
		t.Fatalf("seed move tree: %v", err)
	}
	uids := res.Uids
	t.Cleanup(func() {
		var q, del strings.Builder
		q.WriteString(`query {
			u1 as var(func: eq(User.username, "` + testutil.TestUser + `"))`)
		i := 0
		for _, uid := range uids {
			fmt.Fprintf(&q, "\n\t\t\tx%d as var(func: uid(%s))", i, uid)
			fmt.Fprintf(&del, "\n\t\t\tuid(x%d) * * .", i)
			fmt.Fprintf(&del, "\n\t\t\tuid(u1) <User.roles> uid(x%d) .", i)
			i++
		}
		q.WriteString("\n\t\t}")
		if _, err := db.GetDB().Gamma(db.QueryMut{Q: q.String(), M: []db.X{{D: del.String()}}}, nil); err != nil {
			t.Logf("cleanup move tree: %v", err)
		}
		sessions.GetCache().Del(context.Background(), testutil.TestUser+"roles")
	})
	sessions.GetCache().Del(context.Background(), testutil.TestUser+"roles")

	cookie := loginAs(testutil.TestUser, testutil.TestPassword)
	requireGraphQLSuccess(t, runTensionEvent(t, cookie, uids["t"], "Moved", map[string]any{
		"old": movSrc,
		"new": movDst,
	}))
	if got, _ := db.GetDB().GetByUid(uids["t"], "Tension.receiverid"); got != movDst {
		t.Fatalf("Tension.receiverid = %v, want %s", got, movDst)
	}
}
