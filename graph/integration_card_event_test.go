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

package graph_test

import (
	"testing"
	"time"

	"fractale/fractal6.go/db"
	. "fractale/fractal6.go/graph"
	"fractale/fractal6.go/graph/model"
	"fractale/fractal6.go/internal/testutil"
)

// seededCardEnv holds the uids for an isolated project + columns + tension + cards
// scaffold used to drive the project-card event helpers in tests.
type seededCardEnv struct {
	tensionID string
	projectID string
	colAID    string
	colBID    string
	cardID    string // ProjectCard wrapping the tension, in col A
	draftID   string
	draftCard string // ProjectCard wrapping a ProjectDraft, in col A
}

func seedCardEnv(t *testing.T) seededCardEnv {
	t.Helper()
	seed := db.QueryMut{
		Q: `query {
            org as var(func: eq(Node.nameid, "test-org"))
            u   as var(func: eq(User.username, "` + testutil.TestUser + `"))
        }`,
		M: []db.X{{
			S: `
            _:tension <dgraph.type> "Tension" .
            _:tension <Tension.title> "evt-test-tension" .
            _:tension <Tension.status> "Open" .
            _:tension <Tension.type_> "Operational" .
            _:tension <Tension.emitter> uid(org) .
            _:tension <Tension.emitterid> "test-org" .
            _:tension <Tension.receiver> uid(org) .
            _:tension <Tension.receiverid> "test-org" .
            _:tension <Post.createdBy> uid(u) .
            _:tension <Post.createdAt> "2026-01-01T00:00:00Z" .
            uid(org) <Node.tensions_out> _:tension .
            uid(org) <Node.tensions_in> _:tension .

            _:proj <dgraph.type> "Project" .
            _:proj <Project.nameid> "evt-test-project" .
            _:proj <Project.rootnameid> "test-org" .
            _:proj <Project.parentnameid> "test-org" .
            _:proj <Project.name> "Evt Test Project" .
            _:proj <Project.status> "Open" .
            _:proj <Project.peerCanEditProject> "false" .
            _:proj <Project.guestCanEditProject> "false" .
            _:proj <Project.createdBy> uid(u) .
            _:proj <Project.createdAt> "2026-01-01T00:00:00Z" .
            _:proj <Project.updatedAt> "2026-01-01T00:00:00Z" .
            _:proj <Project.nodes> uid(org) .
            uid(org) <Node.projects> _:proj .

            _:cola <dgraph.type> "ProjectColumn" .
            _:cola <ProjectColumn.name> "Evt Col A" .
            _:cola <ProjectColumn.color> "#aaa111" .
            _:cola <ProjectColumn.pos> "0" .
            _:cola <ProjectColumn.col_type> "NormalColumn" .
            _:cola <ProjectColumn.project> _:proj .
            _:proj <Project.columns> _:cola .

            _:colb <dgraph.type> "ProjectColumn" .
            _:colb <ProjectColumn.name> "Evt Col B" .
            _:colb <ProjectColumn.color> "#bbb222" .
            _:colb <ProjectColumn.pos> "1" .
            _:colb <ProjectColumn.col_type> "NormalColumn" .
            _:colb <ProjectColumn.project> _:proj .
            _:proj <Project.columns> _:colb .

            _:card <dgraph.type> "ProjectCard" .
            _:card <ProjectCard.pos> "0" .
            _:card <ProjectCard.card> _:tension .
            _:card <ProjectCard.pc> _:cola .
            _:cola <ProjectColumn.cards> _:card .

            _:draft <dgraph.type> "ProjectDraft" .
            _:draft <ProjectDraft.title> "evt-test-draft" .
            _:draft <Post.createdBy> uid(u) .
            _:draft <Post.createdAt> "2026-01-01T00:00:00Z" .

            _:dcard <dgraph.type> "ProjectCard" .
            _:dcard <ProjectCard.pos> "1" .
            _:dcard <ProjectCard.card> _:draft .
            _:dcard <ProjectCard.pc> _:cola .
            _:cola <ProjectColumn.cards> _:dcard .
            `,
		}},
	}
	res, err := db.GetDB().UpsertDql(seed, map[string]string{})
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
	pick := func(k string) string {
		v, ok := res.Uids[k]
		if !ok || v == "" {
			t.Fatalf("seed: missing uid for blank node %q", k)
		}
		return v
	}
	return seededCardEnv{
		tensionID: pick("tension"),
		projectID: pick("proj"),
		colAID:    pick("cola"),
		colBID:    pick("colb"),
		cardID:    pick("card"),
		draftID:   pick("draft"),
		draftCard: pick("dcard"),
	}
}

func cleanupCardEnv(t *testing.T, s seededCardEnv) {
	t.Helper()
	// Drop Event nodes attached to the tension's history first.
	evq := db.QueryMut{
		Q: `query {
            var(func: uid(` + s.tensionID + `)) {
                ev as Tension.history
            }
        }`,
		M: []db.X{{D: `uid(ev) * * .`}},
	}
	if _, err := db.GetDB().Gamma(evq, map[string]string{}); err != nil {
		t.Logf("cleanup events: %v", err)
	}
	// Drop all triples on our scaffold nodes.
	uids := []string{s.cardID, s.draftCard, s.draftID, s.colAID, s.colBID, s.projectID, s.tensionID}
	for _, u := range uids {
		dq := db.QueryMut{
			Q: `query { x as var(func: uid(` + u + `)) }`,
			M: []db.X{{D: `uid(x) * * .`}},
		}
		if _, err := db.GetDB().Gamma(dq, map[string]string{}); err != nil {
			t.Logf("cleanup %s: %v", u, err)
		}
	}
}

// historyEntry is a denormalized projection of Event nodes hanging off a tension
// for one particular event_type, used to assert PushProject* behavior.
type historyEntry struct {
	UID  string `json:"uid"`
	Type string `json:"event_type"`
	Old  string `json:"old"`
	New  string `json:"new"`
}

func fetchHistory(t *testing.T, tid string, et model.TensionEvent) []historyEntry {
	t.Helper()
	q := db.QueryMut{
		Q: `query {
            all(func: uid(` + tid + `)) @normalize {
                Tension.history @filter(eq(Event.event_type, "` + string(et) + `")) {
                    uid: uid
                    event_type: Event.event_type
                    old: Event.old
                    new: Event.new
                }
            }
        }`,
	}
	out, err := db.Gamma[historyEntry](q, map[string]string{})
	if err != nil {
		t.Fatalf("fetch history (%s): %v", et, err)
	}
	return out
}

// TestPushProjectAdded_TensionCard verifies that PushProjectAdded writes a
// ProjectAdded event with `new = "{projectid}§{projectname}§"` to the tension's
// history when the card wraps a Tension.
func TestPushProjectAdded_TensionCard(t *testing.T) {
	env := seedCardEnv(t)
	defer cleanupCardEnv(t, env)

	uctx := &model.UserCtx{Username: testutil.TestUser}
	if err := PushProjectAdded(uctx, env.cardID); err != nil {
		t.Fatalf("PushProjectAdded: %v", err)
	}

	got := fetchHistory(t, env.tensionID, model.TensionEventProjectAdded)
	if len(got) != 1 {
		t.Fatalf("expected 1 ProjectAdded event, got %d", len(got))
	}
	wantNew := env.projectID + "§Evt Test Project§"
	if got[0].New != wantNew {
		t.Errorf("ProjectAdded.new = %q, want %q", got[0].New, wantNew)
	}
	if got[0].Old != "" {
		t.Errorf("ProjectAdded.old = %q, want empty", got[0].Old)
	}
}

// TestPushProjectAdded_DraftIsSkipped verifies drafts produce no event.
func TestPushProjectAdded_DraftIsSkipped(t *testing.T) {
	env := seedCardEnv(t)
	defer cleanupCardEnv(t, env)

	uctx := &model.UserCtx{Username: testutil.TestUser}
	if err := PushProjectAdded(uctx, env.draftCard); err != nil {
		t.Fatalf("PushProjectAdded(draft): %v", err)
	}
	// No tension to attach an event to; just assert no event landed on the
	// real tension either (ensures we didn't accidentally cross-link).
	if got := fetchHistory(t, env.tensionID, model.TensionEventProjectAdded); len(got) != 0 {
		t.Fatalf("expected 0 ProjectAdded events on real tension, got %d", len(got))
	}
}

// TestPushProjectColumnMoved verifies that moving a tension card between two
// columns writes a ProjectColumnMoved event with both column descriptors.
func TestPushProjectColumnMoved(t *testing.T) {
	env := seedCardEnv(t)
	defer cleanupCardEnv(t, env)

	// Build the pre-move loc as the update hook would.
	oldLoc := ProjectCardLoc{
		ID:          env.cardID,
		Colid:       env.colAID,
		Colname:     "Evt Col A",
		Colcolor:    "#aaa111",
		Projectid:   env.projectID,
		Projectname: "Evt Test Project",
		Pos:         0,
		Contentid:   env.tensionID,
		Typenames:   []string{"Tension"},
	}

	newCol := ProjectColumnDesc{ID: env.colBID, Name: "Evt Col B", Color: "#bbb222"}
	uctx := &model.UserCtx{Username: testutil.TestUser}
	if err := PushProjectColumnMoved(uctx, oldLoc, newCol); err != nil {
		t.Fatalf("PushProjectColumnMoved: %v", err)
	}

	got := fetchHistory(t, env.tensionID, model.TensionEventProjectColumnMoved)
	if len(got) != 1 {
		t.Fatalf("expected 1 ProjectColumnMoved event, got %d", len(got))
	}
	wantOld := env.colAID + "§Evt Col A§#aaa111§" + env.projectID
	wantNew := env.colBID + "§Evt Col B§#bbb222§" + env.projectID
	if got[0].Old != wantOld {
		t.Errorf("ProjectColumnMoved.old = %q, want %q", got[0].Old, wantOld)
	}
	if got[0].New != wantNew {
		t.Errorf("ProjectColumnMoved.new = %q, want %q", got[0].New, wantNew)
	}
}

// TestPushProjectColumnMoved_SameColumnSkipped verifies that staying in the same
// column emits no event (in-column position shuffle).
func TestPushProjectColumnMoved_SameColumnSkipped(t *testing.T) {
	env := seedCardEnv(t)
	defer cleanupCardEnv(t, env)

	oldLoc := ProjectCardLoc{
		ID: env.cardID, Colid: env.colAID, Colname: "Evt Col A",
		Colcolor: "#aaa111", Projectid: env.projectID,
		Projectname: "Evt Test Project", Pos: 0,
		Contentid: env.tensionID, Typenames: []string{"Tension"},
	}
	sameCol := ProjectColumnDesc{ID: env.colAID, Name: "Evt Col A", Color: "#aaa111"}
	uctx := &model.UserCtx{Username: testutil.TestUser}
	if err := PushProjectColumnMoved(uctx, oldLoc, sameCol); err != nil {
		t.Fatalf("PushProjectColumnMoved (same col): %v", err)
	}
	if got := fetchHistory(t, env.tensionID, model.TensionEventProjectColumnMoved); len(got) != 0 {
		t.Fatalf("expected 0 ProjectColumnMoved events on same-col, got %d", len(got))
	}
}

// TestPushProjectRemoved verifies that PushProjectRemoved writes a
// ProjectRemoved event with `old = "{projectid}§{projectname}§"`.
func TestPushProjectRemoved(t *testing.T) {
	env := seedCardEnv(t)
	defer cleanupCardEnv(t, env)

	loc := ProjectCardLoc{
		ID: env.cardID, Colid: env.colAID, Colname: "Evt Col A",
		Colcolor: "#aaa111", Projectid: env.projectID,
		Projectname: "Evt Test Project", Pos: 0,
		Contentid: env.tensionID, Typenames: []string{"Tension"},
	}
	uctx := &model.UserCtx{Username: testutil.TestUser}
	if err := PushProjectRemoved(uctx, loc); err != nil {
		t.Fatalf("PushProjectRemoved: %v", err)
	}

	got := fetchHistory(t, env.tensionID, model.TensionEventProjectRemoved)
	if len(got) != 1 {
		t.Fatalf("expected 1 ProjectRemoved event, got %d", len(got))
	}
	wantOld := env.projectID + "§Evt Test Project§"
	if got[0].Old != wantOld {
		t.Errorf("ProjectRemoved.old = %q, want %q", got[0].Old, wantOld)
	}
	if got[0].New != "" {
		t.Errorf("ProjectRemoved.new = %q, want empty", got[0].New)
	}
}

// activityCount returns the current Activity.count for the given activityid,
// or 0 if no Activity node exists yet.
func activityCount(t *testing.T, activityid string) int {
	t.Helper()
	type row struct {
		Count int `json:"count"`
	}
	q := db.QueryMut{
		Q: `query {
            all(func: eq(Activity.activityid, "` + activityid + `")) @normalize {
                count: Activity.count
            }
        }`,
	}
	out, err := db.Gamma[row](q, map[string]string{})
	if err != nil {
		t.Fatalf("activityCount(%s): %v", activityid, err)
	}
	if len(out) == 0 {
		return 0
	}
	return out[0].Count
}

// TestPushProjectAdded_TracksActivity verifies that emitting a project event
// also bumps the daily Activity counters for the user and the root org.
func TestPushProjectAdded_TracksActivity(t *testing.T) {
	env := seedCardEnv(t)
	defer cleanupCardEnv(t, env)

	today := time.Now().UTC().Format("2006-01-02")
	userKey := "u#" + testutil.TestUser + "#" + today
	orgKey := "o#test-org#" + today

	beforeUser := activityCount(t, userKey)
	beforeOrg := activityCount(t, orgKey)

	uctx := &model.UserCtx{Username: testutil.TestUser}
	if err := PushProjectAdded(uctx, env.cardID); err != nil {
		t.Fatalf("PushProjectAdded: %v", err)
	}

	// These daily counters are shared with the concurrently-running web/handlers
	// test binary and other card-event tests (async trackActivity goroutines),
	// which may also increment them between the reads. They are increment-only,
	// so assert >= rather than an exact delta.
	if got := activityCount(t, userKey); got < beforeUser+1 {
		t.Errorf("user activity count = %d, want >= %d", got, beforeUser+1)
	}
	if got := activityCount(t, orgKey); got < beforeOrg+1 {
		t.Errorf("org activity count = %d, want >= %d", got, beforeOrg+1)
	}
}

// TestPushProjectRemoved_DraftIsSkipped verifies drafts are skipped.
func TestPushProjectRemoved_DraftIsSkipped(t *testing.T) {
	env := seedCardEnv(t)
	defer cleanupCardEnv(t, env)

	loc := ProjectCardLoc{
		ID: env.draftCard, Colid: env.colAID, Colname: "Evt Col A",
		Colcolor: "#aaa111", Projectid: env.projectID,
		Projectname: "Evt Test Project", Pos: 1,
		Contentid: env.draftID, Typenames: []string{"ProjectDraft"},
	}
	uctx := &model.UserCtx{Username: testutil.TestUser}
	if err := PushProjectRemoved(uctx, loc); err != nil {
		t.Fatalf("PushProjectRemoved(draft): %v", err)
	}
	if got := fetchHistory(t, env.tensionID, model.TensionEventProjectRemoved); len(got) != 0 {
		t.Fatalf("expected 0 ProjectRemoved events from draft, got %d", len(got))
	}
}
