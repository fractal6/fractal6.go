//go:build integration

/*
 * Fractale - Self-organisation for humans.
 * Copyright (C) 2026 Fractale Co
 *
 * This file is part of Fractale.
 */

package graph_test

import (
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"fractale/fractal6.go/db"
	"fractale/fractal6.go/graph"
	"fractale/fractal6.go/graph/model"
	"fractale/fractal6.go/internal/testutil"
	"fractale/fractal6.go/internal/tools"
)

func TestContractEventDecisionAndPersistence(t *testing.T) {
	tests := []struct {
		name   string
		vote   int
		status model.ContractStatus
	}{
		{"pending", 0, model.ContractStatusOpen},
		{"canceled", -1, model.ContractStatusCanceled},
		{"accepted", 1, model.ContractStatusClosed},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			org := "contract-event-" + tt.name
			vars := map[string]string{
				"org": org, "owner": testutil.TestUser, "candidate": testutil.TestUser2,
				"vote": strconv.Itoa(tt.vote),
			}
			// Each invitation owns its org and membership nodes; shared users stay intact.
			seed := db.QueryMut{
				Q: `query {
					owner as var(func: eq(User.username, "{{.owner}}"))
					candidate as var(func: eq(User.username, "{{.candidate}}"))
				}`,
				M: []db.X{{S: `
					_:org <dgraph.type> "Node" .
					_:org <Node.nameid> "{{.org}}" .
					_:org <Node.rootnameid> "{{.org}}" .
					_:org <Node.name> "Contract event test" .
					_:org <Node.type_> "Circle" .
					_:org <Node.mode> "Coordinated" .
					_:org <Node.visibility> "Public" .
					_:org <Node.isRoot> "true" .
					_:org <Node.isArchived> "false" .
					_:org <Node.updatedAt> "2026-01-01T00:00:00Z" .
					_:org <Node.children> _:ownerRole .
					_:org <Node.children> _:member .
					_:ownerRole <dgraph.type> "Node" .
					_:ownerRole <Node.nameid> "{{.org}}##@{{.owner}}" .
					_:ownerRole <Node.role_type> "Owner" .
					_:ownerRole <Node.isArchived> "false" .
					_:ownerRole <Node.parent> _:org .
					_:ownerRole <Node.first_link> uid(owner) .
					uid(owner) <User.roles> _:ownerRole .
					_:member <dgraph.type> "Node" .
					_:member <Node.nameid> "{{.org}}##@{{.candidate}}" .
					_:member <Node.role_type> "Pending" .
					_:member <Node.parent> _:org .
					_:member <Node.first_link> uid(candidate) .
					uid(candidate) <User.roles> _:member .
					_:tension <dgraph.type> "Tension" .
					_:tension <dgraph.type> "Post" .
					_:tension <Post.createdBy> uid(owner) .
					_:tension <Tension.emitter> _:ownerRole .
					_:tension <Tension.receiver> _:org .
					_:tension <Tension.receiverid> "{{.org}}" .
					_:tension <Tension.contracts> _:contract .
					_:contract <dgraph.type> "Contract" .
					_:contract <dgraph.type> "Post" .
					_:contract <Contract.contractid> "{{.org}}#invitation" .
					_:contract <Contract.status> "Open" .
					_:contract <Contract.contract_type> "AnyCandidates" .
					_:contract <Contract.tension> _:tension .
					_:contract <Contract.event> _:event .
					_:contract <Contract.candidates> uid(candidate) .
					_:contract <Contract.participants> _:ownerVote .
					_:event <dgraph.type> "EventFragment" .
					_:event <EventFragment.event_type> "UserJoined" .
					_:event <EventFragment.new> "{{.candidate}}" .
					_:ownerVote <dgraph.type> "Vote" .
					_:ownerVote <Vote.voteid> "{{.org}}#owner-vote" .
					_:ownerVote <Vote.node> _:ownerRole .
					_:ownerVote <Vote.data> "1" .
				`}},
			}
			if tt.vote != 0 {
				seed.M[0].S += `
					_:contract <Contract.participants> _:candidateVote .
					_:candidateVote <dgraph.type> "Vote" .
					_:candidateVote <Vote.voteid> "{{.org}}#candidate-vote" .
					_:candidateVote <Vote.node> _:member .
					_:candidateVote <Vote.data> "{{.vote}}" .
				`
			}
			res, err := db.GetDB().UpsertDql(seed, vars)
			if err != nil {
				t.Fatalf("seed invitation: %v", err)
			}
			t.Cleanup(func() {
				ids := make([]string, 0, len(res.Uids))
				for _, id := range res.Uids {
					ids = append(ids, id)
				}
				cleanup := db.QueryMut{
					Q: `query {
						u as var(func: eq(User.username, ["{{.owner}}", "{{.candidate}}"] ))
						a as var(func: eq(Activity.ownerid, "o#{{.org}}"))
						n as var(func: uid(` + strings.Join(ids, ",") + `))
					}`,
					M: []db.X{{D: `uid(u) <User.roles> uid(n) .
						uid(u) <User.watching> uid(n) .
						uid(n) * * .
						uid(a) * * .`}},
				}
				if _, err := db.GetDB().Gamma(cleanup, vars); err != nil {
					t.Errorf("cleanup invitation: %v", err)
				}
			})

			uctx, err := db.GetDB().GetUctx("username", testutil.TestUser2)
			if err != nil {
				t.Fatal(err)
			}
			tension, err := db.GetDB().GetTensionHook(res.Uids["tension"], true, nil)
			if err != nil || tension == nil {
				t.Fatalf("GetTensionHook = %v, %v", tension, err)
			}
			loadContract := func() *model.Contract {
				t.Helper()
				contract, err := db.GetDB().GetContractHook(res.Uids["contract"])
				if err != nil || contract == nil || contract.Event == nil {
					t.Fatalf("GetContractHook = %v, %v", contract, err)
				}
				return contract
			}
			readState := func() model.Contract {
				t.Helper()
				q := db.QueryMut{Q: `query {
					all(func: uid(` + res.Uids["contract"] + `)) {
						Contract.status
						Contract.contractid
						Contract.participants { uid Vote.voteid }
						Contract.tension { Tension.receiver {
							Node.updatedAt
							Node.children @filter(uid(` + res.Uids["member"] + `)) { Node.role_type }
							Node.watchers @filter(eq(User.username, "{{.candidate}}")) { User.username }
						} }
					}
				}`}
				state, err := tools.First(db.Gamma[model.Contract](q, vars))
				if err != nil || state.Tension == nil || state.Tension.Receiver == nil {
					t.Fatalf("read invitation state = %+v, %v", state, err)
				}
				return state
			}
			before := readState()
			contract := loadContract()
			event := tools.StructMap[model.EventRef](contract.Event)
			wantOK := tt.status != model.ContractStatusOpen

			ok, got, err := graph.AuthorizeEvent(uctx, tension, &event, contract)
			if err != nil || ok != wantOK || got != contract || got.Status != tt.status {
				t.Fatalf("AuthorizeEvent = (%v, %+v, %v), want (%v, %s)", ok, got, err, wantOK, tt.status)
			}
			if after := readState(); !reflect.DeepEqual(before, after) {
				t.Fatalf("AuthorizeEvent changed persisted state: before %+v, after %+v", before, after)
			}

			// ProcessEvent must make its own decision from the still-open persisted contract.
			contract = loadContract()
			ok, got, err = graph.ProcessEvent(uctx, tension, &event, contract)
			if err != nil || ok != wantOK || got != contract || got.Status != tt.status {
				t.Fatalf("ProcessEvent = (%v, %+v, %v), want (%v, %s)", ok, got, err, wantOK, tt.status)
			}
			if tt.status == model.ContractStatusClosed {
				// Wait for the final trace write before cleaning up its isolated org.
				key := "o#" + org + "#" + time.Now().UTC().Format("2006-01-02")
				deadline := time.Now().Add(5 * time.Second)
				for activityCount(t, key) == 0 {
					if time.Now().After(deadline) {
						t.Fatal("accepted invitation did not leave an activity trace")
					}
					time.Sleep(10 * time.Millisecond)
				}
			}
			after := readState()
			if !wantOK {
				if !reflect.DeepEqual(before, after) {
					t.Fatal("pending invitation changed persisted state")
				}
				return
			}
			if after.Status != tt.status || after.Contractid != "" || len(after.Participants) != len(before.Participants) {
				t.Fatalf("contract not finalized: %+v", after)
			}
			for _, vote := range after.Participants {
				if vote.Voteid != "" {
					t.Errorf("vote deduplication key not cleared: %q", vote.Voteid)
				}
			}
			if tt.status == model.ContractStatusCanceled {
				if !reflect.DeepEqual(before.Tension, after.Tension) {
					t.Fatal("canceled invitation ran the join action or changed receiver state")
				}
				return
			}
			receiver := after.Tension.Receiver
			if len(receiver.Children) != 1 || receiver.Children[0].RoleType == nil || *receiver.Children[0].RoleType != model.RoleTypeGuest {
				t.Fatalf("accepted invitation did not upgrade Pending to Guest: %+v", receiver.Children)
			}
			if len(receiver.Watchers) != 1 || receiver.Watchers[0].Username != testutil.TestUser2 {
				t.Fatalf("accepted invitation did not subscribe candidate to org: %+v", receiver.Watchers)
			}
		})
	}
}
