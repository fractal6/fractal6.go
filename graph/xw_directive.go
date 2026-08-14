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

package graph

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/99designs/gqlgen/graphql"

	"fractale/fractal6.go/db"
	"fractale/fractal6.go/graph/codec"
	"fractale/fractal6.go/graph/model"
	"fractale/fractal6.go/internal/tools"
	. "fractale/fractal6.go/internal/tools"
	"fractale/fractal6.go/web/auth"
)

//
// Input directives — @x_* (authorization) and @w_* (transformation)
//
// Both families share the same shape: the directive entrypoint dispatches
// to a named rule/action via a string key (`r:` for @x_*, `a:` for @w_*).
// Adding a new rule/transform is a one-line addition to the registries.
//
// Wiring lives in graph/resolver.go::Init().
//

// xRule is the signature of every @x_* rule. It receives the directive's
// optional `f` (field selector), `e` (event filter) and `n` (numeric param).
type xRule func(context.Context, any, graphql.Resolver, *string, []model.TensionEvent, *int) (any, error)

// wTransform is the signature of every @w_* action. The action runs the
// resolver and post-processes the produced value.
type wTransform func(context.Context, graphql.Resolver) (any, error)

// FieldAuthorizationFunc is the registry of @x_* rules, indexed by the `r:`
// argument in the SDL. Each entry is a pure check — auth context, ownership,
// shape, length, etc.
var FieldAuthorizationFunc map[string]xRule

// FieldTransformFunc is the registry of @w_* actions, indexed by the `a:`
// argument in the SDL.
var FieldTransformFunc map[string]wTransform

func init() {
	FieldAuthorizationFunc = map[string]xRule{
		"isOwner":          isOwner,
		"unique":           unique,
		"oneByOne":         oneByOne,
		"hasEvent":         hasEvent,
		"tensionTypeCheck": tensionTypeCheck,
		"ref":              ref,
		"minLen":           minLength,
		"maxLen":           maxLength,
		"json":             validJSON,
	}
	FieldTransformFunc = map[string]wTransform{
		"lower": lower,
		"now":   now,
	}
}

//
// Directive entrypoints
//

// FieldAuthorization is the @x_* directive entrypoint. The directive may
// appear without a rule (no-op pass-through) or carry a rule name to dispatch.
func FieldAuthorization(ctx context.Context, obj any, next graphql.Resolver, r *string, f *string, e []model.TensionEvent, n *int) (any, error) {
	// If the directives exists withtout a rule, it pass through.
	if r == nil {
		return next(ctx)
	}

	// @TODO: Seperate function for Set and Remove + test if the input comply with the directives

	if fun := FieldAuthorizationFunc[*r]; fun != nil {
		return fun(ctx, obj, next, f, e, n)
	}
	return nil, LogErr("directive error", fmt.Errorf("unknown rule '%s'", *r))
}

// FieldTransform is the @w_* directive entrypoint.
func FieldTransform(ctx context.Context, obj any, next graphql.Resolver, a string) (any, error) {
	if fun := FieldTransformFunc[a]; fun != nil {
		return fun(ctx, next)
	}
	return nil, LogErr("directive error", fmt.Errorf("unknown function '%s'", a))
}

//
// Hook input directives — populate context for downstream @x_*/@w_* rules.
//

// setContextWithID hoists the standard identifier fields from the mutation
// input into the request context so subsequent directives (e.g. @isOwner,
// @unique) can read them without re-walking the args tree.
func setContextWithID(ctx context.Context, obj any, next graphql.Resolver) (any, error) {
	var err error
	for _, n := range []string{"id", "nameid", "rootnameid", "username"} {
		ctx, _, err = setContextWith(ctx, obj, n)
		if err != nil {
			return nil, err
		}
	}
	return next(ctx)
}

// setUpdateContextInfo flags whether the update payload uses set/remove and
// stashes the target id, both required by @hasEvent and @isOwner.
func setUpdateContextInfo(ctx context.Context, obj any, next graphql.Resolver) (any, error) {
	hasSet := obj.(model.JsonAtom)["set"] != nil
	hasRemove := obj.(model.JsonAtom)["remove"] != nil
	ctx = context.WithValue(ctx, "hasSet", hasSet)
	ctx = context.WithValue(ctx, "hasRemove", hasRemove)
	ctx, _, err := setContextWith(ctx, obj, "id")
	if err != nil {
		return nil, err
	}
	return next(ctx)
}

// meta_patch is the @w_meta_patch input-field directive: it stages function
// name + (optional) parameter key/value into Redis so the matching update
// hook can pick them up. See User.markAllAsRead for the canonical use.
func meta_patch(ctx context.Context, obj any, next graphql.Resolver, f string, k *string) (any, error) {
	uctx := auth.GetUserContextOrEmpty(ctx)
	// @FIX this hack ! Redis push ?
	var ok bool
	var v string
	// Set function
	key := uctx.Username + "meta_patch_f"
	err := cache.SetEX(ctx, key, f, time.Second*5).Err()
	if err != nil {
		return nil, err
	}
	if k != nil {
		// Set attribute name
		if v, ok = ctx.Value(*k).(string); !ok {
			o := reflect.ValueOf(obj).Elem().FieldByName(ToGoNameFormat(*k))
			if !o.IsValid() {
				rc := graphql.GetResolverContext(ctx)
				fieldName := rc.Field.Name
				return nil, fmt.Errorf("'%s' field on '%s' seems not valid or unknown", *k, fieldName)
			}
			v = o.String()
		}
		if v == "" {
			rc := graphql.GetResolverContext(ctx)
			fieldName := rc.Field.Name
			err := fmt.Errorf("'%s' field is needed to query '%s'", *k, fieldName)
			return nil, err
		}

		key = uctx.Username + "meta_patch_k"
		err := cache.SetEX(ctx, key, *k, time.Second*5).Err()
		if err != nil {
			return nil, err
		}

		// Set attribute value
		key = uctx.Username + "meta_patch_v"
		err = cache.SetEX(ctx, key, v, time.Second*5).Err()
		if err != nil {
			return nil, err
		}
	}
	return next(ctx)
}

//
// @x_* rule implementations
//

// isOwner Check that object is own by the user.
// If user(u) field is empty, assume a user object, else field should match the user(u) credential.
func isOwner(ctx context.Context, obj any, next graphql.Resolver, f *string, e []model.TensionEvent, n *int) (any, error) {
	// Retrieve userCtx from token
	ctx, uctx, err := auth.GetUserContext(ctx)
	if err != nil {
		return nil, LogErr("Access denied", err)
	}

	// Get attributes and check everything is ok
	userObj := make(model.JsonAtom)
	var userField string
	if f == nil {
		userField = "user"
		userObj[userField] = obj
	} else {
		userField = *f
		userObj = obj.(model.JsonAtom)
	}

	ok, err := CheckUserOwnership(ctx, uctx, userField, userObj)
	if err != nil {
		return nil, LogErr("Access denied", err)
	}
	if ok {
		return next(ctx)
	}

	return nil, LogErr("Access Denied", fmt.Errorf("bad ownership."))
}

// unique Check uniqueness (@DEBUG follow @unique dgraph field iplementation)
// Ensure the field value is unique. If a field is given, it check the uniqueness on a subset of the parent type.
func unique(ctx context.Context, obj any, next graphql.Resolver, f *string, e []model.TensionEvent, n *int) (any, error) {
	data, err := next(ctx)
	if err != nil {
		return nil, err
	}

	var v string
	switch d := data.(type) {
	case *string:
		v = *d
	case string:
		v = d
	}

	field := *graphql.GetPathContext(ctx).Field
	if f != nil {
		// Extract the fieldname and type of the object queried
		_, typeName, _, err := queryTypeFromGraphqlContext(ctx)
		if err != nil {
			return nil, LogErr("unique", err)
		}
		fieldName := typeName + "." + field
		filterName := typeName + "." + *f
		s := obj.(model.JsonAtom)[*f]
		if s != nil {
			// *f is present in the inut
			// pass
		} else if ctx.Value("id") != nil {
			s, err = db.GetDB().GetByUid(ctx.Value("id").(string), filterName)
			if err != nil || s == nil {
				return nil, LogErr("Internal error", err)
			}
		} else {
			return nil, LogErr("Value Error", fmt.Errorf("'%s' or id is required.", *f))
		}
		filterValue := s.(string)

		// Check existence
		filter := fmt.Sprintf(`eq(%s, "%s")`, filterName, filterValue)
		ex, err := db.GetDB().Exists(fieldName, v, &filter)
		if err != nil {
			return nil, LogErr("Internal error", err)
		}
		if !ex {
			return data, err
		}
	} else {
		return nil, fmt.Errorf("@unique alone not implemented.")
	}

	return data, LogErr("Duplicate error", fmt.Errorf("%s '%s' is already taken", field, v))
}

// oneByOne ensure that the mutation on the given field should contains at least one element.
func oneByOne(ctx context.Context, obj any, next graphql.Resolver, f *string, e []model.TensionEvent, n *int) (any, error) {
	data, err := next(ctx)
	slice, ok := InterfaceSlice(data)
	if !ok {
		field := *graphql.GetPathContext(ctx).Field
		return nil, fmt.Errorf("Data must be an array '%s'", field)
	}
	if len(slice) > 1 {
		field := *graphql.GetPathContext(ctx).Field
		return nil, LogErr("@oneByOne error", fmt.Errorf("Only one object allowed in slice '%s'", field))
	}
	return data, err
}

// hasEvent ensure the given events are present in the `history` property.
func hasEvent(ctx context.Context, obj any, next graphql.Resolver, f *string, e []model.TensionEvent, n *int) (any, error) {
	var events []any
	events_ := obj.(model.JsonAtom)["history"]
	if events_ != nil {
		events = events_.([]any)
	}

	for _, event := range e {
		for _, eventPresent := range events {
			eventType := eventPresent.(model.JsonAtom)["event_type"].(string)
			if model.TensionEvent(eventType) == event {
				// ok
				return next(ctx)
			}
		}
	}

	// Exception if we got Blob with just an ID. Use to identify the blob user are working on.
	blobs_ := obj.(model.JsonAtom)["blobs"]
	blobs, ok := InterfaceSlice(blobs_)
	if !ok {
		field := *graphql.GetPathContext(ctx).Field
		return nil, fmt.Errorf("Blobs must be an array '%s'", field)
	}
	if len(blobs) == 1 {
		b := blobs[0].(model.JsonAtom)
		if len(b) == 1 && b["id"] != nil {
			// Allows just reference to a blob (@DEBUG: add a 'cut_blob' to remove it as we do for history to prevent blob hack.
			// ok
			return next(ctx)
		}
	}

	field := *graphql.GetPathContext(ctx).Field
	if ctx.Value("hasSet") != nil && ctx.Value("hasRemove") != nil && (field == "labels" || field == "assignees") {
		// @DEBUG: detect events when remove is used in in updates
		// @DEBUG: detect that the number of event is equal to the current field len
		// when removing labels or assigness...
		return next(ctx)
	}
	return nil, LogErr("Event error", fmt.Errorf("missing event for field '%s'", field))
}

// tensionTypeCheck check is the user can use a tension type.
func tensionTypeCheck(ctx context.Context, obj any, next graphql.Resolver, f *string, e []model.TensionEvent, n *int) (any, error) {
	data, err := next(ctx)
	if err != nil {
		return nil, err
	}

	type_, ok := data.(model.TensionType)
	if !ok { // Happend if tension type is optional => when referenced...
		return data, err
	}

	// Handle Special Tension
	for _, x := range []model.TensionType{model.TensionTypeAlert, model.TensionTypeAnnouncement} {
		if type_ != x {
			continue
		}

		ctx, uctx, err := auth.GetUserContext(ctx)
		if err != nil {
			return nil, err
		}
		// Get receiverid
		var receiverid string
		if v := obj.(model.JsonAtom)["receiverid"]; v != nil {
			receiverid = v.(string)
		} else if ctx.Value("id") != nil {
			x, err := db.GetDB().GetByUid(ctx.Value("id").(string), "Tension.receiverid")
			if err != nil || x == nil {
				return nil, LogErr("Internal error", err)
			}
			receiverid = x.(string)
		} else {
			return nil, LogErr("Value Error", fmt.Errorf("'%s' or id is required.", *f))
		}

		// Check auth
		switch x {
		case model.TensionTypeAlert:
			// User need circle authority
			ok, err := auth.HasCoordoAuth(uctx, receiverid, nil)
			if err != nil {
				return nil, err
			}
			if ok {
				return data, err
			}
		case model.TensionTypeAnnouncement:
			// User need circle authority + root only
			if rid, err := codec.Nid2rootid(receiverid); err != nil {
				return nil, err
			} else if rid != receiverid {
				return nil, fmt.Errorf("Announcement can only be created a the root circle")
			}

			ok, err := auth.HasCoordoAuth(uctx, receiverid, nil)
			if err != nil {
				return nil, err
			}
			if ok {
				return data, err
			}
		}

		return nil, fmt.Errorf("You need to be a coordinator of this circle to create an %s tensions.", string(x))
	}

	return data, err
}

// ref ensure the given objects are just linked to an existing one, no more. (@weak: by testing that its size if not equal to one.)
func ref(ctx context.Context, obj any, next graphql.Resolver, f *string, e []model.TensionEvent, n *int) (any, error) {
	data, err := next(ctx)
	if err != nil {
		return nil, err
	}
	test := func(x any) bool {
		return len(CleanNilMap(StructMap[map[string]any](x))) == 1
	}
	var pass bool
	data_list, ok := InterfaceSlice(data)
	if ok {
		for _, d := range data_list {
			if test(d) {
				pass = true
			} else {
				pass = false
				break
			}
		}
	} else {
		pass = test(data)
	}

	if pass {
		return data, err
	}
	field := *graphql.GetPathContext(ctx).Field
	return nil, fmt.Errorf("ref: only referecence allowed for: %s", field)
}

// minLength rejects values whose length is strictly less than n.
func minLength(ctx context.Context, obj any, next graphql.Resolver, f *string, e []model.TensionEvent, n *int) (any, error) {
	var l int
	data, err := next(ctx)
	if err != nil {
		return nil, err
	}

	switch d := data.(type) {
	case *string:
		l = len(*d)
	case string:
		l = len(d)
	default:
		field := *graphql.GetPathContext(ctx).Field
		return nil, fmt.Errorf("Type unknwown for field '%s'", field)
	}
	if l < *n {
		field := *graphql.GetPathContext(ctx).Field
		return nil, fmt.Errorf("`%s' to short. Minimum length is '%d'", field, *n)
	}
	return data, err
}

// validJSON ensures the field value parses as JSON. The string-typed field is
// otherwise opaque to the schema (e.g. ProjectTemplate.columns_json), so this
// rule rejects malformed payloads at write time instead of letting them reach
// downstream consumers.
func validJSON(ctx context.Context, obj any, next graphql.Resolver, f *string, e []model.TensionEvent, n *int) (any, error) {
	data, err := next(ctx)
	if err != nil {
		return nil, err
	}

	var s string
	switch d := data.(type) {
	case nil:
		return data, nil
	case *string:
		if d == nil {
			return data, nil
		}
		s = *d
	case string:
		s = d
	default:
		field := *graphql.GetPathContext(ctx).Field
		return nil, fmt.Errorf("Type unknown for field '%s'", field)
	}

	var v any
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		field := *graphql.GetPathContext(ctx).Field
		return nil, fmt.Errorf("`%s' is not valid JSON: %v", field, err)
	}
	return data, nil
}

// maxLength rejects values whose length is strictly greater than n.
func maxLength(ctx context.Context, obj any, next graphql.Resolver, f *string, e []model.TensionEvent, n *int) (any, error) {
	var l int
	data, err := next(ctx)
	if err != nil {
		return nil, err
	}

	switch d := data.(type) {
	case *string:
		l = len(*d)
	case string:
		l = len(d)
	default:
		field := *graphql.GetPathContext(ctx).Field
		return nil, fmt.Errorf("Type unknwown for field %s", field)
	}
	if l > *n {
		field := *graphql.GetPathContext(ctx).Field
		return nil, fmt.Errorf("`%s' to short. Maximum length is %d", field, *n)
	}
	return data, err
}

//
// @w_* transform implementations
//

// need https://github.com/golang/go/issues/51977
//type StringEqFilter interface {
//    model.StringExactFilter |
//    model.StringHashFilter |
//    model.StringHashFilterStringRegExpFilter |
//    model.StringHashFilterStringTermFilter
//    //SetEq(s string)
//}

func lower(ctx context.Context, next graphql.Resolver) (any, error) {
	data, err := next(ctx)
	switch d := data.(type) {
	case *string:
		v := strings.ToLower(*d)
		return &v, err
	case string:
		v := strings.ToLower(d)
		return v, err
	case *model.StringExactFilter:
		v := *d
		if v.Eq != nil {
			s := strings.ToLower(*v.Eq)
			v.Eq = &s
		}
		return &v, err
	case *model.StringHashFilter:
		v := *d
		if v.Eq != nil {
			s := strings.ToLower(*v.Eq)
			v.Eq = &s
		}
		return &v, err
	case *model.StringHashFilterStringRegExpFilter:
		v := *d
		if v.Eq != nil {
			s := strings.ToLower(*v.Eq)
			v.Eq = &s
		}
		return &v, err
	case *model.StringHashFilterStringTermFilter:
		v := *d
		if v.Eq != nil {
			s := strings.ToLower(*v.Eq)
			v.Eq = &s
		}
		return &v, err
	}
	field := *graphql.GetPathContext(ctx).Field
	return nil, fmt.Errorf("Type unknwown for field %s", field)
}

func now(ctx context.Context, next graphql.Resolver) (any, error) {
	data, err := next(ctx)
	now := tools.Now()
	switch data.(type) {
	case *string:
		return &now, err
	case string:
		return now, err
	}
	field := *graphql.GetPathContext(ctx).Field
	return nil, fmt.Errorf("Type unknwown for field %s", field)
}

//
// Auth utility functions
// * (could be done in Dgraph Lambda ?)
//

// CheckUserOwnership returns true when uctx owns the parent object via the
// named user field. Falls back to a Dgraph lookup when the input only carries
// the parent uid (id was previously stashed by setContextWithID).
func CheckUserOwnership(ctx context.Context, uctx *model.UserCtx, userField string, userObj any) (bool, error) {
	// Get user ID
	var username string
	var err error
	user := userObj.(model.JsonAtom)[userField]
	if user == nil || user.(model.JsonAtom)["username"] == nil {
		// Non user type here (userField/createdBy must be present)
		id := ctx.Value("id")
		if id == nil || id.(string) == "" {
			return false, fmt.Errorf("object target unknown(id), see setContextWithID...")
		}
		// Request the database to get the field
		// @DEBUG: in the dgraph graphql schema, @createdBy is in the Post interface: ToTypeName(reflect.TypeOf(nodeObj).String())
		username_, err := db.GetDB().GetByUid(id.(string), "Post."+userField, "User.username")
		if err != nil {
			return false, err
		}
		username = username_.(string)
	} else {
		// User here
		username = user.(model.JsonAtom)["username"].(string)
	}

	// Check user ID match
	return uctx.Username == username, err
}
