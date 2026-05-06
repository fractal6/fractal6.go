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
	"github.com/vektah/gqlparser/v2/ast"
)

// withFieldName builds a minimal context that satisfies graphql.GetResolverContext
// + graphql.GetPathContext for directives that only read `Field.Name` /
// path's `Field` to format error messages. It avoids spinning up a real
// gqlgen request.
func withFieldName(parent context.Context, queryName, fieldName string) context.Context {
	fc := &graphql.FieldContext{
		Field: graphql.CollectedField{
			Field: &ast.Field{Name: queryName, Alias: queryName},
		},
	}
	ctx := graphql.WithFieldContext(parent, fc)
	field := fieldName
	pc := &graphql.PathContext{Field: &field}
	return graphql.WithPathContext(ctx, pc)
}

// TestHidden_AlwaysErrors ensures the @hidden directive never lets a value
// through, regardless of caller. It's the cheapest line of defence on
// password / token fields, so a regression here would silently expose
// secrets.
func TestHidden_AlwaysErrors(t *testing.T) {
	// hidden reads rc.Field.Name (the field being resolved); use that slot.
	ctx := withFieldName(context.Background(), "password", "password")
	called := false
	next := graphql.Resolver(func(ctx context.Context) (interface{}, error) {
		called = true
		return "secret", nil
	})

	out, err := hidden(ctx, nil, next)
	if err == nil {
		t.Fatal("expected error from @hidden, got nil")
	}
	if called {
		t.Error("@hidden must short-circuit and not invoke next()")
	}
	if out != nil {
		t.Errorf("out = %v, want nil", out)
	}
	if !strings.Contains(err.Error(), "password") {
		t.Errorf("error %q should mention the offending field", err.Error())
	}
}

// TestReadOnly_AlwaysErrors mirrors @hidden for write-side @x_ro fields.
func TestReadOnly_AlwaysErrors(t *testing.T) {
	ctx := withFieldName(context.Background(), "updateComment", "files")
	called := false
	next := graphql.Resolver(func(ctx context.Context) (interface{}, error) {
		called = true
		return nil, nil
	})

	_, err := readOnly(ctx, nil, next)
	if err == nil {
		t.Fatal("expected error from @x_ro, got nil")
	}
	if called {
		t.Error("@x_ro must short-circuit and not invoke next()")
	}
}
