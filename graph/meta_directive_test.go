/*
 * Fractale - Self-organisation for humans.
 * Copyright (C) 2026 Fractale Co
 *
 * This file is part of Fractale.
 */

package graph

import (
	"encoding/json"
	"testing"

	"fractale/fractal6.go/graph/model"
)

// decodeSliceJSON mirrors metaSlice's inner JSON->[]*T step so the test
// doesn't need a live Dgraph -- the wrapper that runs QueryDql is trivial
// and exercised by the integration suite.
func decodeSliceJSON[T any](data []byte) ([]*T, error) {
	if len(data) == 0 {
		return []*T{}, nil
	}
	var w struct {
		All []*T `json:"all"`
	}
	if err := json.Unmarshal(data, &w); err != nil {
		return nil, err
	}
	if w.All == nil {
		return []*T{}, nil
	}
	return w.All, nil
}

func TestMetaRegistry_HasExpectedFunctions(t *testing.T) {
	want := []string{
		"getNodeHistory",
		"getNodeActivity",
		"getUserActivity",
		"getEventCount",
	}
	for _, f := range want {
		if _, ok := metaRegistry[f]; !ok {
			t.Errorf("metaRegistry missing entry %q", f)
		}
	}
}

func TestMetaSlice_DecodesAliasedActivity(t *testing.T) {
	// Mirrors the shape emitted by getUserActivity / getNodeActivity:
	// keys are pre-aliased in the DQL template to match model.Activity tags.
	json := []byte(`{"all":[
		{"activityid":"o#test-org#2026-05-01","count":3,"date":"2026-05-01"},
		{"activityid":"o#test-org#2026-05-02","count":7,"date":"2026-05-02"}
	]}`)

	got, err := decodeSliceJSON[model.Activity](json)
	if err != nil {
		t.Fatalf("decodeSliceJSON: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if got[0].Activityid != "o#test-org#2026-05-01" || got[0].Count != 3 || got[0].Date != "2026-05-01" {
		t.Errorf("row[0] = %+v", got[0])
	}
	if got[1].Count != 7 {
		t.Errorf("row[1].Count = %d, want 7", got[1].Count)
	}
}

func TestMetaSlice_EmptyResponse(t *testing.T) {
	got, err := decodeSliceJSON[model.Activity]([]byte(`{"all":[]}`))
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if len(got) != 0 {
		t.Errorf("len = %d, want 0", len(got))
	}
}

func TestMetaSlice_NullResponse(t *testing.T) {
	got, err := decodeSliceJSON[model.Activity](nil)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if got == nil {
		t.Errorf("got nil slice, want empty non-nil")
	}
}
