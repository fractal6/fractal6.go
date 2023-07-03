/*
 * Fractale - Self-organisation for humans.
 * Copyright (C) 2023 Fractale Co
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

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.
package graph

//go:generate go run github.com/99designs/gqlgen generate

import (
	"context"
	"fmt"
	"github.com/99designs/gqlgen/graphql"
	"reflect"
	"time"

	"fractale/fractal6.go/db"
	gen "fractale/fractal6.go/graph/generated"
	"fractale/fractal6.go/graph/model"
	. "fractale/fractal6.go/tools"
	"fractale/fractal6.go/web/auth"
)

//
// Resolver initialisation
//

type Resolver struct {
	// Pointer on Dgraph client
	db *db.Dgraph
}

// Init initialize shema config and Directives...
func Init() gen.Config {

	c := gen.Config{
		Resolvers: &Resolver{db: db.GetDB()},
	}

	//
	// Query / Payload fields
	//

	// Fields directives
	c.Directives.Hidden = hidden
	c.Directives.Private = private
	c.Directives.Meta = meta
	c.Directives.IsContractValidator = isContractValidator

	//
	//  Input Fields directives
	//

	// Auth directive
	// - add fields are allowed by default
	c.Directives.X_add = FieldAuthorization
	c.Directives.X_set = FieldAuthorization
	c.Directives.X_remove = FieldAuthorization
	c.Directives.X_patch = FieldAuthorization
	c.Directives.X_alter = FieldAuthorization
	c.Directives.X_patch_ro = readOnly
	c.Directives.X_ro = readOnly

	// Transformation directives
	c.Directives.W_add = FieldTransform
	c.Directives.W_set = FieldTransform
	c.Directives.W_remove = FieldTransform
	c.Directives.W_patch = FieldTransform
	c.Directives.W_alter = FieldTransform
	c.Directives.W_meta_patch = meta_patch

	//
	// Hook
	//

	//User
	c.Directives.Hook_getUserInput = setContextWithID   // used by @private
	c.Directives.Hook_queryUserInput = setContextWithID // used by @private
	c.Directives.Hook_addUserInput = nothing
	c.Directives.Hook_updateUserInput = setContextWithID // used by @meta_patch
	c.Directives.Hook_deleteUserInput = nothing
	// --
	c.Directives.Hook_addUser = nothing
	c.Directives.Hook_updateUser = nothing
	c.Directives.Hook_deleteUser = nothing
	//RoleExt
	c.Directives.Hook_getRoleExtInput = nothing
	c.Directives.Hook_queryRoleExtInput = nothing
	c.Directives.Hook_addRoleExtInput = nothing
	c.Directives.Hook_updateRoleExtInput = setContextWithID // used by @unique
	c.Directives.Hook_deleteRoleExtInput = nothing
	// --
	c.Directives.Hook_addRoleExt = addNodeArtefactHook
	c.Directives.Hook_updateRoleExt = updateNodeArtefactHook
	c.Directives.Hook_deleteRoleExt = nothing
	//Label
	c.Directives.Hook_getLabelInput = nothing
	c.Directives.Hook_queryLabelInput = nothing
	c.Directives.Hook_addLabelInput = nothing
	c.Directives.Hook_updateLabelInput = setContextWithID // used by the @unique
	c.Directives.Hook_deleteLabelInput = nothing
	// --
	c.Directives.Hook_addLabel = addNodeArtefactHook
	c.Directives.Hook_updateLabel = updateNodeArtefactHook
	c.Directives.Hook_deleteLabel = nothing
	//Project
	c.Directives.Hook_getProjectInput = nothing
	c.Directives.Hook_queryProjectInput = nothing
	c.Directives.Hook_addProjectInput = nothing
	c.Directives.Hook_updateProjectInput = setContextWithID // used by the @unique
	c.Directives.Hook_deleteProjectInput = nothing
	// --
	c.Directives.Hook_addProject = addNodeArtefactHook
	c.Directives.Hook_updateProject = updateNodeArtefactHook
	c.Directives.Hook_deleteProject = nothing
	//ProjectColumn
	c.Directives.Hook_getProjectColumnInput = nothing
	c.Directives.Hook_queryProjectColumnInput = nothing
	c.Directives.Hook_addProjectColumnInput = nothing
	c.Directives.Hook_updateProjectColumnInput = nothing
	c.Directives.Hook_deleteProjectColumnInput = nothing
	// --
	c.Directives.Hook_addProjectColumn = addProjectColumnHook
	c.Directives.Hook_updateProjectColumn = updateProjectColumnHook
	c.Directives.Hook_deleteProjectColumn = deleteProjectColumnHook
	//ProjectCard
	c.Directives.Hook_getProjectCardInput = nothing
	c.Directives.Hook_queryProjectCardInput = nothing
	c.Directives.Hook_addProjectCardInput = nothing
	c.Directives.Hook_updateProjectCardInput = nothing
	c.Directives.Hook_deleteProjectCardInput = nothing
	// --
	c.Directives.Hook_addProjectCard = addProjectCardHook
	c.Directives.Hook_updateProjectCard = updateProjectCardHook
	c.Directives.Hook_deleteProjectCard = deleteProjectCardHook
	//Tension
	c.Directives.Hook_getTensionInput = nothing
	c.Directives.Hook_queryTensionInput = nothing
	//ProjectDraft
	c.Directives.Hook_getProjectDraftInput = nothing
	c.Directives.Hook_queryProjectDraftInput = nothing
	c.Directives.Hook_addProjectDraftInput = nothing
	c.Directives.Hook_updateProjectDraftInput = nothing
	c.Directives.Hook_deleteProjectDraftInput = nothing
	// --
	c.Directives.Hook_addProjectDraft = nothing
	c.Directives.Hook_updateProjectDraft = updateProjectDraftHook
	c.Directives.Hook_deleteProjectDraft = nothing
	// @DEBUG: input rawQuery isssue (input modification not propagated with rawQuery whil rawQuery loose field with argument) !!!
	// to be reviewed...
	//c.Directives.Hook_addTensionInput = tensionInputHook
	//c.Directives.Hook_updateTensionInput = tensionInputHook
	c.Directives.Hook_addTensionInput = nothing
	c.Directives.Hook_updateTensionInput = setUpdateContextInfo // for @hasEvent+@isOwner
	c.Directives.Hook_deleteTensionInput = nothing
	// --
	c.Directives.Hook_addTension = addTensionHook
	c.Directives.Hook_updateTension = updateTensionHook
	c.Directives.Hook_deleteTension = nothing
	//Comment
	c.Directives.Hook_getCommentInput = nothing
	c.Directives.Hook_queryCommentInput = nothing
	c.Directives.Hook_addCommentInput = nothing
	c.Directives.Hook_updateCommentInput = setContextWithID // used by @isOwner
	c.Directives.Hook_deleteCommentInput = nothing
	// --
	c.Directives.Hook_addComment = nothing
	c.Directives.Hook_updateComment = nothing
	c.Directives.Hook_deleteComment = nothing
	//Reaction
	c.Directives.Hook_getReactionInput = nothing
	c.Directives.Hook_queryReactionInput = nothing
	c.Directives.Hook_addReactionInput = addReactionInputHook
	c.Directives.Hook_updateReactionInput = nothing
	c.Directives.Hook_deleteReactionInput = nothing
	// --
	c.Directives.Hook_addReaction = nothing
	c.Directives.Hook_updateReaction = nothing
	c.Directives.Hook_deleteReaction = nothing
	//Contract
	c.Directives.Hook_getContractInput = nothing
	c.Directives.Hook_queryContractInput = nothing
	c.Directives.Hook_addContractInput = addContractInputHook
	c.Directives.Hook_updateContractInput = setContextWithID // used by @isOwner
	c.Directives.Hook_deleteContractInput = nothing
	// --
	c.Directives.Hook_addContract = addContractHook
	c.Directives.Hook_updateContract = updateContractHook
	c.Directives.Hook_deleteContract = deleteContractHook
	//Vote
	c.Directives.Hook_getVoteInput = nothing
	c.Directives.Hook_queryVoteInput = nothing
	c.Directives.Hook_addVoteInput = nothing
	c.Directives.Hook_updateVoteInput = nothing
	c.Directives.Hook_deleteVoteInput = nothing
	// --
	c.Directives.Hook_addVote = addVoteHook
	c.Directives.Hook_updateVote = nothing
	c.Directives.Hook_deleteVote = nothing

	return c
}

// https://stackoverflow.com/questions/58468134/how-to-compose-functions-in-go
// @generics
// @debug: do not workd with resolvers
func compose(manyv ...func(ctx context.Context, obj interface{}, next graphql.Resolver) (interface{}, error)) func(ctx context.Context, obj interface{}, next graphql.Resolver) (interface{}, error) {
	return func(ctx context.Context, obj interface{}, next graphql.Resolver) (interface{}, error) {
		var err error
		for _, v := range manyv {
			obj, err = v(ctx, obj, next)
			if err != nil {
				return obj, err
			}
		}
		return obj, err
	}
}

func ExtractInputs[T any](ctx context.Context, inputs *[]T) {
	a := graphql.GetResolverContext(ctx).Args["input"]
	ExtractSlice(a, inputs)
}

func ExtractInput[T any](ctx context.Context, input *T) {
	a := graphql.GetResolverContext(ctx).Args["input"]
	StructMap(a, input)
}

func ExtractFilter[T any](ctx context.Context, filter *T) {
	a := graphql.GetResolverContext(ctx).Args["filter"]
	StructMap(a, filter)
}

/*
*
* Field Directives Logics
*
 */

// Reminder: Api to access to input query:
//  rc := graphql.GetResolverContext(ctx)
//  rqc := graphql.GetRequestContext(ctx)
//  cfc := graphql.CollectFieldsCtx(ctx, nil)
//  fc := graphql.GetFieldContext(ctx)
//  pc := graphql.GetPathContext(ctx) // .*.Field to get the field name

func nothing(ctx context.Context, obj interface{}, next graphql.Resolver) (interface{}, error) {
	return next(ctx)
}

func hidden(ctx context.Context, obj interface{}, next graphql.Resolver) (interface{}, error) {
	rc := graphql.GetResolverContext(ctx)
	fieldName := rc.Field.Name
	return nil, fmt.Errorf("'%s' field is hidden", fieldName)
}

func private(ctx context.Context, obj interface{}, next graphql.Resolver) (interface{}, error) {
	ctx, uctx, err := auth.GetUserContext(ctx)
	if err != nil {
		return nil, LogErr("Access denied", err)
	}

	rc := graphql.GetResolverContext(ctx)
	fieldName := rc.Field.Name

	// @DEBUG: not workng; ContextWith do not propagage value here, why, gqlgen !
	//         Probably because directive for returned value are not in the same context as of before ?
	// @AFTER_DEBUG: if uid is given, or other @id...
	if username, ok := ctx.Value("username").(string); ok && username == uctx.Username {
		return next(ctx)
	}

	switch v := obj.(type) {
	case *model.User:
		// @debug: username field required in graph
		if v.Username == uctx.Username {
			return next(ctx)
		}
	default:
		return nil, fmt.Errorf("Private directive not implemented for this field: %s", fieldName)
	}

	return nil, fmt.Errorf("'%s' field is private", fieldName)
}

// Use DQL query to fetch the given field=k.
// If k is not given, "id" is automatically pass to the query template.
func meta(ctx context.Context, obj interface{}, next graphql.Resolver, f string, k *string) (interface{}, error) {
	data, err := next(ctx)
	if err != nil {
		return nil, err
	}

	var ok bool
	var v string
	maps := map[string]string{}
	// Get query field
	if k != nil {
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
		maps[*k] = v
	}

	// Query
	res, err := db.GetDB().Meta(f, maps)
	if err != nil {
		return nil, err
	}

	// Map result
	switch data.(type) {
	case *int:
		rc := graphql.GetResolverContext(ctx)
		fieldName := rc.Field.Name
		for _, s := range res {
			// assumes only one element is returned
			v, _ := s[fieldName].(float64)
			v2 := int(v)
			data = &v2
		}
	default:
		rt := reflect.TypeOf(data)
		switch rt.Kind() {
		case reflect.Slice:
			// Convert list of map to the desired list of interface
			t := reflect.MakeSlice(rt, 1, 1)
			newData := reflect.MakeSlice(rt, 0, len(res))
			for i := 0; i < len(res); i++ {
				v := reflect.ValueOf(t.Interface()).Index(0).Interface()
				if err := Map2Struct(res[i], &v); err != nil {
					return data, err
				}
				newData = reflect.Append(newData, reflect.ValueOf(v))
			}
			data = newData.Interface()
		default:
			// Assume interface
			// Merge results (needed for user defined returns (i.e. EventCouts))
			m := make(map[string]interface{}, 2)
			for _, s := range res {
				for k, v := range s {
					m[k] = v
				}
			}
			err = Map2Struct(m, &data)
		}
	}
	return data, err
}

func meta_patch(ctx context.Context, obj interface{}, next graphql.Resolver, f string, k *string) (interface{}, error) {
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
// Input directives
//

func setContextWithID(ctx context.Context, obj interface{}, next graphql.Resolver) (interface{}, error) {
	var err error
	for _, n := range []string{"id", "nameid", "rootnameid", "username"} {
		ctx, _, err = setContextWith(ctx, obj, n)
		if err != nil {
			return nil, err
		}
	}
	return next(ctx)
}

func setContextWithNameid(ctx context.Context, obj interface{}, next graphql.Resolver) (interface{}, error) {
	ctx, _, err := setContextWith(ctx, obj, "nameid")
	if err != nil {
		return nil, err
	}
	return next(ctx)
}

func setUpdateContextInfo(ctx context.Context, obj interface{}, next graphql.Resolver) (interface{}, error) {
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

//
// Input Field directives
//

func readOnly(ctx context.Context, obj interface{}, next graphql.Resolver) (interface{}, error) {
	rc := graphql.GetResolverContext(ctx)
	pc := graphql.GetPathContext(ctx)
	queryName := rc.Field.Name
	fieldName := *pc.Field
	return nil, LogErr("Forbiden", fmt.Errorf("Read only field on %s:%s", queryName, fieldName))
}

func FieldAuthorization(ctx context.Context, obj interface{}, next graphql.Resolver, r *string, f *string, e []model.TensionEvent, n *int) (interface{}, error) {
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

func FieldTransform(ctx context.Context, obj interface{}, next graphql.Resolver, a string) (interface{}, error) {
	if fun := FieldTransformFunc[a]; fun != nil {
		return fun(ctx, next)
	}
	return nil, LogErr("directive error", fmt.Errorf("unknown function '%s'", a))
}
