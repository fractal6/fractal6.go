/*
 * Fractale - Self-organisation for humans.
 * Copyright (C) 2026 Fractale Co
 *
 * This file is part of Fractale.
 */

package graph

import (
	"context"
	"strings"
	"testing"

	"github.com/99designs/gqlgen/graphql"
)

// TestFieldAuthorizationFunc_HasExpectedRules pins the @x_* registry so that
// silently dropping a rule from the wiring trips a failing test before it
// reaches schema compilation.
func TestFieldAuthorizationFunc_HasExpectedRules(t *testing.T) {
	want := []string{
		"isOwner",
		"unique",
		"oneByOne",
		"hasEvent",
		"tensionTypeCheck",
		"ref",
		"minLen",
		"maxLen",
		"json",
	}
	for _, r := range want {
		if _, ok := FieldAuthorizationFunc[r]; !ok {
			t.Errorf("FieldAuthorizationFunc missing rule %q", r)
		}
	}
}

// TestFieldTransformFunc_HasExpectedActions pins the @w_* registry.
func TestFieldTransformFunc_HasExpectedActions(t *testing.T) {
	want := []string{"lower", "now"}
	for _, a := range want {
		if _, ok := FieldTransformFunc[a]; !ok {
			t.Errorf("FieldTransformFunc missing action %q", a)
		}
	}
}

// TestFieldAuthorization_NilRulePassesThrough verifies the "directive present
// but no rule" shape (e.g. `@x_alter` on its own) is a transparent passthrough.
func TestFieldAuthorization_NilRulePassesThrough(t *testing.T) {
	called := false
	next := graphql.Resolver(func(ctx context.Context) (interface{}, error) {
		called = true
		return "ok", nil
	})
	out, err := FieldAuthorization(context.Background(), nil, next, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Fatal("next() was not called")
	}
	if got, _ := out.(string); got != "ok" {
		t.Fatalf("out = %v, want \"ok\"", out)
	}
}

// TestFieldAuthorization_UnknownRuleErrors guards against typos in the SDL.
func TestFieldAuthorization_UnknownRuleErrors(t *testing.T) {
	rule := "no_such_rule"
	_, err := FieldAuthorization(context.Background(), nil, nil, &rule, nil, nil, nil)
	if err == nil {
		t.Fatal("expected error for unknown rule, got nil")
	}
	if !strings.Contains(err.Error(), "no_such_rule") {
		t.Errorf("error %q should mention the offending rule", err.Error())
	}
}

// TestFieldTransform_UnknownActionErrors mirrors the @x_* check for @w_*.
func TestFieldTransform_UnknownActionErrors(t *testing.T) {
	_, err := FieldTransform(context.Background(), nil, nil, "no_such_action")
	if err == nil {
		t.Fatal("expected error for unknown action, got nil")
	}
	if !strings.Contains(err.Error(), "no_such_action") {
		t.Errorf("error %q should mention the offending action", err.Error())
	}
}

// TestFieldTransform_LowerString covers the most-used branch of the `lower`
// transform without needing a graphql resolver context.
func TestFieldTransform_LowerString(t *testing.T) {
	next := graphql.Resolver(func(ctx context.Context) (interface{}, error) {
		return "HeLLo", nil
	})
	out, err := FieldTransform(context.Background(), nil, next, "lower")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "hello" {
		t.Errorf("out = %v, want \"hello\"", out)
	}
}

// TestFieldTransform_LowerStringPtr covers the *string branch.
func TestFieldTransform_LowerStringPtr(t *testing.T) {
	in := "WoRLd"
	next := graphql.Resolver(func(ctx context.Context) (interface{}, error) {
		return &in, nil
	})
	out, err := FieldTransform(context.Background(), nil, next, "lower")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got, ok := out.(*string)
	if !ok || got == nil {
		t.Fatalf("out = %T %v, want *string", out, out)
	}
	if *got != "world" {
		t.Errorf("*got = %q, want \"world\"", *got)
	}
}

// TestFieldTransform_NowReturnsRFCString sanity-checks the `now` transform:
// the produced value should be a non-empty string in RFC3339 shape.
func TestFieldTransform_NowReturnsRFCString(t *testing.T) {
	var seed string
	next := graphql.Resolver(func(ctx context.Context) (interface{}, error) {
		return seed, nil
	})
	out, err := FieldTransform(context.Background(), nil, next, "now")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	s, ok := out.(string)
	if !ok {
		t.Fatalf("out type = %T, want string", out)
	}
	if s == "" || !strings.Contains(s, "T") {
		t.Errorf("now() = %q, expected RFC3339-like timestamp", s)
	}
}
