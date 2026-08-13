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
			ID:       "0x2",
			BlobType: model.BlobTypeOnNode,
			Node:     fragment,
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

	tests := []struct {
		name       string
		tension    *model.Tension
		operation  governanceOperation
		wantCreate bool
		wantError  string
	}{
		{name: "new role", tension: governanceTestTension(fragment(), nil), operation: governancePublish, wantCreate: true},
		{name: "update role", tension: governanceTestTension(fragment(), governed(false)), operation: governancePublish},
		{name: "missing blob", tension: &model.Tension{Receiver: &model.Node{Nameid: "org"}}, operation: governancePublish, wantError: "requires a blob"},
		{name: "missing fragment", tension: governanceTestTension(nil, nil), operation: governancePublish, wantError: "must contain a node fragment"},
		{name: "missing create nameid", tension: governanceTestTension(&model.NodeFragment{Name: &name, Type: &role, RoleType: &peer}, nil), operation: governancePublish, wantError: "requires a nameid"},
		{name: "missing create kind", tension: governanceTestTension(&model.NodeFragment{Nameid: &nameid, Name: &name, RoleType: &peer}, nil), operation: governancePublish, wantError: "requires a valid type"},
		{name: "missing create role type", tension: governanceTestTension(&model.NodeFragment{Nameid: &nameid, Name: &name, Type: &role}, nil), operation: governancePublish, wantError: "requires a role_type"},
		{name: "existing relation required", tension: governanceTestTension(fragment(), nil), operation: governanceUpdate, wantError: "requires a governed node"},
		{name: "kind mismatch", tension: governanceTestTension(fragment(), &model.Node{ID: "0x3", Nameid: "org#circle", Type: circle}), operation: governancePublish, wantError: "does not match"},
		{name: "publish archived", tension: governanceTestTension(fragment(), governed(true)), operation: governancePublish, wantError: "cannot publish an archived node"},
		{name: "duplicate archive", tension: governanceTestTension(fragment(), governed(true)), operation: governanceArchive, wantError: "already archived"},
		{name: "duplicate unarchive", tension: governanceTestTension(fragment(), governed(false)), operation: governanceUnarchive, wantError: "is not archived"},
		{name: "archive transition", tension: governanceTestTension(fragment(), governed(false)), operation: governanceArchive},
		{name: "unarchive transition", tension: governanceTestTension(fragment(), governed(true)), operation: governanceUnarchive},
		{name: "Md rejected", tension: &model.Tension{Receiver: &model.Node{Nameid: "org"}, Blobs: []*model.Blob{{BlobType: model.BlobTypeOnDoc, Md: &name}}}, operation: governancePublish, wantError: "cannot govern a node"},
		{name: "unknown blob type rejected", tension: &model.Tension{Receiver: &model.Node{Nameid: "org"}, Blobs: []*model.Blob{{BlobType: model.BlobType("Future"), Node: fragment()}}}, operation: governancePublish, wantError: "cannot govern a node"},
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

	if ok, err := PushBlob(nil, tension, nil, nil); ok || err == nil || !strings.Contains(err.Error(), "archived") {
		t.Fatalf("PushBlob() = (%v, %v), want archived-state error", ok, err)
	}
	eventType := model.TensionEventBlobArchived
	if ok, err := ChangeArchiveBlob(nil, tension, &model.EventRef{EventType: &eventType}, nil); ok || err == nil || !strings.Contains(err.Error(), "already archived") {
		t.Fatalf("ChangeArchiveBlob() = (%v, %v), want duplicate-transition error", ok, err)
	}

	username := "member"
	eventType = model.TensionEventMemberUnlinked
	if ok, err := ChangeFirstLink(nil, tension, &model.EventRef{EventType: &eventType, Old: &username}, nil); ok || err == nil || !strings.Contains(err.Error(), "role type") {
		t.Fatalf("ChangeFirstLink() = (%v, %v), want missing-role-type error", ok, err)
	}

	circle := model.NodeTypeCircle
	circleTension := governanceTestTension(
		&model.NodeFragment{Nameid: &nameid, Name: &name, Type: &circle},
		&model.Node{ID: "0x4", Nameid: "org#circle", Type: circle},
	)
	eventType = model.TensionEventMemberLinked
	if ok, err := ChangeFirstLink(nil, circleTension, &model.EventRef{EventType: &eventType, New: &username}, nil); ok || err == nil || !strings.Contains(err.Error(), "previous circle member") {
		t.Fatalf("ChangeFirstLink() = (%v, %v), want missing-previous-member error", ok, err)
	}
}
