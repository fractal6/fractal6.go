package graph

import (
	"strings"
	"testing"

	"fractale/fractal6.go/graph/model"
)

func governanceTestTension(fragment *model.NodeFragment, governed *model.Node) *model.Tension {
	return &model.Tension{
		ID:       "0x1",
		Receiver: &model.Node{Nameid: "org"},
		Blobs: []*model.Blob{{
			ID:   "0x2",
			Node: fragment,
		}},
		GovernedNode: governed,
	}
}

func TestResolveGovernanceSubject(t *testing.T) {
	role := model.NodeTypeRole
	circle := model.NodeTypeCircle
	peer := model.RoleTypePeer
	nameid := "designer"
	name := "Designer"
	fragment := func() *model.NodeFragment {
		return &model.NodeFragment{Nameid: &nameid, Name: &name, Type: &role, RoleType: &peer}
	}
	governed := func(archived bool) *model.Node {
		return &model.Node{ID: "0x3", Nameid: "org##designer", Type: role, IsArchived: archived}
	}

	badName := "X"
	guest := model.RoleTypeGuest
	rootFragment := &model.NodeFragment{Name: &name, Type: &circle}
	rootGoverned := &model.Node{ID: "0x5", Nameid: "org", Type: circle}
	rootArchived := func(archived bool) *model.Node {
		return &model.Node{ID: "0x5", Nameid: "org", Type: circle, IsArchived: true, IsRootArchived: &archived}
	}

	tests := []struct {
		name       string
		tension    *model.Tension
		operation  governanceOperation
		wantCreate bool
		wantNameid string
		wantError  string
	}{
		{name: "new role", tension: governanceTestTension(fragment(), nil), operation: governancePublish, wantCreate: true, wantNameid: "org##designer"},
		{name: "update role", tension: governanceTestTension(fragment(), governed(false)), operation: governancePublish, wantNameid: "org##designer"},
		{name: "root update", tension: governanceTestTension(rootFragment, rootGoverned), operation: governanceUpdate, wantNameid: "org"},
		{name: "invalid name", tension: governanceTestTension(&model.NodeFragment{Nameid: &nameid, Name: &badName, Type: &role, RoleType: &peer}, nil), operation: governancePublish, wantError: "too short"},
		{name: "membership role type rejected", tension: governanceTestTension(&model.NodeFragment{Nameid: &nameid, Name: &name, Type: &role, RoleType: &guest}, nil), operation: governancePublish, wantError: "Membership roles are protected"},
		{name: "missing blob", tension: &model.Tension{Receiver: &model.Node{Nameid: "org"}}, operation: governancePublish, wantError: "requires a blob"},
		{name: "empty blob slice", tension: &model.Tension{Receiver: &model.Node{Nameid: "org"}, Blobs: []*model.Blob{}}, operation: governancePublish, wantError: "requires a blob"},
		{name: "missing fragment", tension: governanceTestTension(nil, nil), operation: governancePublish, wantError: "must contain a node fragment"},
		{name: "missing create nameid", tension: governanceTestTension(&model.NodeFragment{Name: &name, Type: &role, RoleType: &peer}, nil), operation: governancePublish, wantError: "requires a nameid"},
		{name: "missing name", tension: governanceTestTension(&model.NodeFragment{Nameid: &nameid, Type: &role, RoleType: &peer}, nil), operation: governancePublish, wantError: "requires a name"},
		{name: "missing create kind", tension: governanceTestTension(&model.NodeFragment{Nameid: &nameid, Name: &name, RoleType: &peer}, nil), operation: governancePublish, wantError: "requires a valid type"},
		{name: "missing create role type", tension: governanceTestTension(&model.NodeFragment{Nameid: &nameid, Name: &name, Type: &role}, nil), operation: governancePublish, wantError: "requires a role_type"},
		{name: "existing relation required", tension: governanceTestTension(fragment(), nil), operation: governanceUpdate, wantError: "requires a governed node"},
		{name: "kind mismatch", tension: governanceTestTension(fragment(), &model.Node{ID: "0x3", Nameid: "org#circle", Type: circle}), operation: governancePublish, wantError: "does not match"},
		{name: "publish archived", tension: governanceTestTension(fragment(), governed(true)), operation: governancePublish, wantError: "cannot publish an archived node"},
		{name: "duplicate archive", tension: governanceTestTension(fragment(), governed(true)), operation: governanceArchive, wantError: "already archived"},
		{name: "duplicate unarchive", tension: governanceTestTension(fragment(), governed(false)), operation: governanceUnarchive, wantError: "is not archived"},
		{name: "archive transition", tension: governanceTestTension(fragment(), governed(false)), operation: governanceArchive, wantNameid: "org##designer"},
		{name: "unarchive transition", tension: governanceTestTension(fragment(), governed(true)), operation: governanceUnarchive, wantNameid: "org##designer"},
		// Root lifecycle reads isRootArchived only, isArchived is ignored.
		{name: "root archive transition", tension: governanceTestTension(rootFragment, rootGoverned), operation: governanceArchive, wantNameid: "org"},
		{name: "root duplicate archive", tension: governanceTestTension(rootFragment, rootArchived(true)), operation: governanceArchive, wantError: "already archived"},
		{name: "root unarchive transition", tension: governanceTestTension(rootFragment, rootArchived(true)), operation: governanceUnarchive, wantNameid: "org"},
		{name: "root duplicate unarchive", tension: governanceTestTension(rootFragment, rootArchived(false)), operation: governanceUnarchive, wantError: "is not archived"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			subject, err := resolveGovernanceSubject(test.tension, test.operation)
			if test.wantError != "" {
				if err == nil || !strings.Contains(err.Error(), test.wantError) {
					t.Fatalf("error = %v, want containing %q", err, test.wantError)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if subject.create != test.wantCreate {
				t.Fatalf("create = %v, want %v", subject.create, test.wantCreate)
			}
			if subject.nameid != test.wantNameid {
				t.Fatalf("nameid = %q, want %q", subject.nameid, test.wantNameid)
			}
		})
	}
}

func TestGovernanceActionsRejectInvalidStateBeforePersistence(t *testing.T) {
	role := model.NodeTypeRole
	peer := model.RoleTypePeer
	nameid := "designer"
	name := "Designer"
	fragment := &model.NodeFragment{Nameid: &nameid, Name: &name, Type: &role, RoleType: &peer}
	tension := governanceTestTension(fragment, &model.Node{ID: "0x3", Nameid: "org##designer", Type: role, IsArchived: true})

	if err := PushBlob(nil, tension, nil); err == nil || !strings.Contains(err.Error(), "archived") {
		t.Fatalf("PushBlob() = %v, want archived-state error", err)
	}
	eventType := model.TensionEventBlobArchived
	if err := ChangeArchiveBlob(nil, tension, &model.EventRef{EventType: &eventType}); err == nil || !strings.Contains(err.Error(), "already archived") {
		t.Fatalf("ChangeArchiveBlob() = %v, want duplicate-transition error", err)
	}

	username := "member"
	eventType = model.TensionEventMemberUnlinked
	if err := ChangeFirstLink(nil, tension, &model.EventRef{EventType: &eventType, Old: &username}); err == nil || !strings.Contains(err.Error(), "role type") {
		t.Fatalf("ChangeFirstLink() = %v, want missing-role-type error", err)
	}

	circle := model.NodeTypeCircle
	circleTension := governanceTestTension(
		&model.NodeFragment{Nameid: &nameid, Name: &name, Type: &circle},
		&model.Node{ID: "0x4", Nameid: "org#circle", Type: circle},
	)
	eventType = model.TensionEventMemberLinked
	if err := ChangeFirstLink(nil, circleTension, &model.EventRef{EventType: &eventType, New: &username}); err == nil || !strings.Contains(err.Error(), "previous circle member") {
		t.Fatalf("ChangeFirstLink() = %v, want missing-previous-member error", err)
	}
}

// ProcessEvent must pass ok=true through when the check is skipped (auth decided upstream).
func TestProcessEventSkippedCheckPassesOkThrough(t *testing.T) {
	eventType := model.TensionEventCreated
	tension := &model.Tension{ID: "0x1", Receiver: &model.Node{Nameid: "org"}}
	ok, _, err := ProcessEvent(nil, tension, &model.EventRef{EventType: &eventType}, nil, false, false)
	if err != nil || !ok {
		t.Fatalf("ProcessEvent(doCheck=false, doProcess=false) = (%v, %v), want (true, nil)", ok, err)
	}
}
