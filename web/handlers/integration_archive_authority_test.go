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

// End-to-end coverage of the recursive archive: subtree authority + the `new="true"`
// close flag. The tree lives in its own org (arcorg): testuser is Owner of test-org
// (owners pass everywhere) and seeding roles in sec-org leaks into web/auth tests.

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

// The seeded subtree (nameids are flat, the hierarchy lives in Node.parent/children):
//
//	arcorg#arcauth        (coordo: testuser)      <- governance tension receiver
//	  ├── arcorg#arcdeny        target, blocked
//	  │     └── arcorg#arcdenysub    own coordo: testuser2
//	  ├── arcorg#arcallow       target, fully owned
//	  │     └── arcorg#arcallowsub   no coordo -> inherits arcauth authority
//	  └── arcorg#arcnoflag      target, archived without the close flag
const (
	arcauthRoot = "arcorg#arcauth"
	arcDeny     = "arcorg#arcdeny"
	arcDenySub  = "arcorg#arcdenysub"
	arcAllow    = "arcorg#arcallow"
	arcAllowSub = "arcorg#arcallowsub"
	arcNoFlag   = "arcorg#arcnoflag"
)

func nqCircle(blank, nameid, parent string) string {
	name := nameid[strings.LastIndex(nameid, "#")+1:]
	return fmt.Sprintf(`
		%[1]s <dgraph.type> "Node" .
		%[1]s <Node.nameid> %[2]q .
		%[1]s <Node.rootnameid> "arcorg" .
		%[1]s <Node.name> %[3]q .
		%[1]s <Node.type_> "Circle" .
		%[1]s <Node.mode> "Coordinated" .
		%[1]s <Node.visibility> "Private" .
		%[1]s <Node.isArchived> "false" .
		%[1]s <Node.isRoot> "false" .
		%[1]s <Node.parent> %[4]s .
		%[4]s <Node.children> %[1]s .
	`, blank, nameid, name, parent)
}

func nqCoordo(blank, nameid, parent, userVar string) string {
	name := nameid[strings.LastIndex(nameid, "@")+1:]
	return fmt.Sprintf(`
		%[1]s <dgraph.type> "Node" .
		%[1]s <Node.nameid> %[2]q .
		%[1]s <Node.rootnameid> "arcorg" .
		%[1]s <Node.name> %[3]q .
		%[1]s <Node.type_> "Role" .
		%[1]s <Node.role_type> "Coordinator" .
		%[1]s <Node.mode> "Coordinated" .
		%[1]s <Node.visibility> "Private" .
		%[1]s <Node.isArchived> "false" .
		%[1]s <Node.isRoot> "false" .
		%[1]s <Node.parent> %[4]s .
		%[4]s <Node.children> %[1]s .
		%[1]s <Node.first_link> %[5]s .
		%[5]s <User.roles> %[1]s .
	`, blank, nameid, name, parent, userVar)
}

func nqTension(blank, title, receiver, receiverid string) string {
	return fmt.Sprintf(`
		%[1]s <dgraph.type> "Tension" .
		%[1]s <Tension.title> %[2]q .
		%[1]s <Tension.status> "Open" .
		%[1]s <Tension.type_> "Operational" .
		%[1]s <Tension.emitter> %[3]s .
		%[1]s <Tension.emitterid> %[4]q .
		%[1]s <Tension.receiver> %[3]s .
		%[1]s <Tension.receiverid> %[4]q .
		%[1]s <Post.createdBy> uid(u1) .
		%[1]s <Post.createdAt> "2027-01-01T00:00:00Z" .
	`, blank, title, receiver, receiverid)
}

// nqGovernanceTension builds the archive tension: received by arcauth (testuser is
// coordo there, so TargetCoordoHook passes) and governing the target.
func nqGovernanceTension(blank, target, targetName string) string {
	return fmt.Sprintf(`
		%[1]s <dgraph.type> "Tension" .
		%[1]s <Tension.title> "archive %[3]s" .
		%[1]s <Tension.status> "Open" .
		%[1]s <Tension.type_> "Governance" .
		%[1]s <Tension.emitter> _:arcp .
		%[1]s <Tension.emitterid> "arcorg#arcauth" .
		%[1]s <Tension.receiver> _:arcp .
		%[1]s <Tension.receiverid> "arcorg#arcauth" .
		%[1]s <Tension.governed_node> %[2]s .
		%[1]s <Post.createdBy> uid(u1) .
		%[1]s <Post.createdAt> "2027-01-01T00:00:00Z" .
		%[1]s <Tension.blobs> %[1]sb .
		%[1]sb <dgraph.type> "Blob" .
		%[1]sb <Blob.tension> %[1]s .
		%[1]sb <Blob.node> %[1]sf .
		%[1]sb <Post.createdBy> uid(u1) .
		%[1]sb <Post.createdAt> "2027-01-01T00:00:01Z" .
		%[1]sf <dgraph.type> "NodeFragment" .
		%[1]sf <NodeFragment.nameid> %[3]q .
		%[1]sf <NodeFragment.type_> "Circle" .
		%[1]sf <NodeFragment.name> %[3]q .
	`, blank, target, targetName)
}

// seedArchiveAuthTree seeds the tree and returns the blank-node -> uid map.
func seedArchiveAuthTree(t *testing.T) map[string]string {
	t.Helper()
	set := `
		_:org <dgraph.type> "Node" .
		_:org <Node.nameid> "arcorg" .
		_:org <Node.rootnameid> "arcorg" .
		_:org <Node.name> "arcorg" .
		_:org <Node.type_> "Circle" .
		_:org <Node.mode> "Coordinated" .
		_:org <Node.visibility> "Public" .
		_:org <Node.isArchived> "false" .
		_:org <Node.isRoot> "true" .
	` +
		nqCircle("_:arcp", arcauthRoot, "_:org") +
		nqCoordo("_:coordo1", arcauthRoot+"#@coordo1", "_:arcp", "uid(u1)") +
		nqCircle("_:deny", arcDeny, "_:arcp") +
		nqCircle("_:denysub", arcDenySub, "_:deny") +
		nqCoordo("_:coordo2", arcDenySub+"#@coordo2", "_:denysub", "uid(u2)") +
		nqCircle("_:allow", arcAllow, "_:arcp") +
		nqCircle("_:allowsub", arcAllowSub, "_:allow") +
		nqCircle("_:noflag", arcNoFlag, "_:arcp") +
		nqTension("_:tdeny", "arcauth-deny-open", "_:deny", arcDeny) +
		nqTension("_:tallow", "arcauth-allow-open", "_:allow", arcAllow) +
		nqTension("_:tallowsub", "arcauth-allowsub-open", "_:allowsub", arcAllowSub) +
		nqTension("_:tnoflag", "arcauth-noflag-open", "_:noflag", arcNoFlag) +
		nqGovernanceTension("_:gdeny", "_:deny", "arcdeny") +
		nqGovernanceTension("_:gallow", "_:allow", "arcallow") +
		nqGovernanceTension("_:gnoflag", "_:noflag", "arcnoflag")

	qm := db.QueryMut{
		Q: `query {
			u1 as var(func: eq(User.username, "` + testutil.TestUser + `"))
			u2 as var(func: eq(User.username, "` + testutil.TestUser2 + `"))
		}`,
		M: []db.X{{S: set}},
	}
	res, err := db.GetDB().UpsertDql(qm, map[string]string{})
	if err != nil {
		t.Fatalf("seed archive-authority tree: %v", err)
	}
	uids := res.Uids
	t.Cleanup(func() {
		var q, del strings.Builder
		q.WriteString(`query {
			u1 as var(func: eq(User.username, "` + testutil.TestUser + `"))
			u2 as var(func: eq(User.username, "` + testutil.TestUser2 + `"))`)
		i := 0
		for _, uid := range uids {
			fmt.Fprintf(&q, "\n\t\t\tx%d as var(func: uid(%s))", i, uid)
			fmt.Fprintf(&del, "\n\t\t\tuid(x%d) * * .", i)
			fmt.Fprintf(&del, "\n\t\t\tuid(u1) <User.roles> uid(x%d) .", i)
			fmt.Fprintf(&del, "\n\t\t\tuid(u2) <User.roles> uid(x%d) .", i)
			i++
		}
		q.WriteString("\n\t\t}")
		if _, err := db.GetDB().Gamma(db.QueryMut{Q: q.String(), M: []db.X{{D: del.String()}}}, nil); err != nil {
			t.Logf("cleanup archive-authority tree: %v", err)
		}
		// The role cache still holds the seeded roles.
		sessions.GetCache().Del(context.Background(), testutil.TestUser+"roles")
	})
	// Bust the 12s role cache so the fresh coordo roles are seen right away.
	sessions.GetCache().Del(context.Background(), testutil.TestUser+"roles")
	return uids
}

func requireArchived(t *testing.T, uid, nameid string, want bool) {
	t.Helper()
	got, err := db.GetDB().GetByUid(uid, "Node.isArchived")
	if err != nil {
		t.Fatalf("Node.isArchived(%s): %v", nameid, err)
	}
	if b, _ := got.(bool); b != want {
		t.Fatalf("%s isArchived = %v, want %v", nameid, got, want)
	}
}

func requireTensionStatus(t *testing.T, tid, title, want string) {
	t.Helper()
	got, err := db.GetDB().GetByUid(tid, "Tension.status")
	if err != nil {
		t.Fatalf("Tension.status(%s): %v", title, err)
	}
	if got != want {
		t.Fatalf("tension %s status = %v, want %s", title, got, want)
	}
}

func TestRecursiveArchiveSubtreeAuthority(t *testing.T) {
	uids := seedArchiveAuthTree(t)
	cookie := loginAs(testutil.TestUser, testutil.TestPassword)

	t.Run("denied without authority on every sub-circle", func(t *testing.T) {
		result := runTensionEvent(t, cookie, uids["gdeny"], "BlobArchived", map[string]any{"new": "true"})
		requireGraphQLError(t, result, arcDenySub)
		// Nothing written: the check runs before the archive upsert.
		requireArchived(t, uids["deny"], arcDeny, false)
		requireArchived(t, uids["denysub"], arcDenySub, false)
		requireTensionStatus(t, uids["tdeny"], "arcauth-deny-open", "Open")
	})

	t.Run("archives the subtree and closes its tensions", func(t *testing.T) {
		requireGraphQLSuccess(t, runTensionEvent(t, cookie, uids["gallow"], "BlobArchived", map[string]any{"new": "true"}))
		requireArchived(t, uids["allow"], arcAllow, true)
		requireArchived(t, uids["allowsub"], arcAllowSub, true)
		requireTensionStatus(t, uids["tallow"], "arcauth-allow-open", "Closed")
		requireTensionStatus(t, uids["tallowsub"], "arcauth-allowsub-open", "Closed")
	})

	t.Run("without the flag the tensions stay open", func(t *testing.T) {
		requireGraphQLSuccess(t, runTensionEvent(t, cookie, uids["gnoflag"], "BlobArchived", nil))
		requireArchived(t, uids["noflag"], arcNoFlag, true)
		requireTensionStatus(t, uids["tnoflag"], "arcauth-noflag-open", "Open")
	})
}
