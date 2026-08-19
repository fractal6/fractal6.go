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

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.
package graph

//go:generate go run github.com/99designs/gqlgen generate

import (
	"context"

	"github.com/99designs/gqlgen/graphql"

	"fractale/fractal6.go/db"
	gen "fractale/fractal6.go/graph/generated"
	. "fractale/fractal6.go/internal/tools"
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

	// User
	c.Directives.Hook_getUserInput = setContextWithID   // used by @private
	c.Directives.Hook_queryUserInput = setContextWithID // used by @private
	c.Directives.Hook_addUserInput = nothing
	c.Directives.Hook_updateUserInput = setContextWithID // used by @meta_patch
	c.Directives.Hook_deleteUserInput = nothing
	// --
	c.Directives.Hook_addUser = nothing
	c.Directives.Hook_updateUser = nothing
	c.Directives.Hook_deleteUser = nothing
	// RoleExt
	c.Directives.Hook_getRoleExtInput = nothing
	c.Directives.Hook_queryRoleExtInput = nothing
	c.Directives.Hook_addRoleExtInput = nothing
	c.Directives.Hook_updateRoleExtInput = setContextWithID // used by @unique
	c.Directives.Hook_deleteRoleExtInput = nothing
	// --
	c.Directives.Hook_addRoleExt = addNodeArtefactHook
	c.Directives.Hook_updateRoleExt = updateNodeArtefactHook
	c.Directives.Hook_deleteRoleExt = nothing
	// Label
	c.Directives.Hook_getLabelInput = nothing
	c.Directives.Hook_queryLabelInput = nothing
	c.Directives.Hook_addLabelInput = nothing
	c.Directives.Hook_updateLabelInput = setContextWithID // used by the @unique
	c.Directives.Hook_deleteLabelInput = nothing
	// --
	c.Directives.Hook_addLabel = addNodeArtefactHook
	c.Directives.Hook_updateLabel = updateNodeArtefactHook
	c.Directives.Hook_deleteLabel = nothing
	// TensionTemplate
	c.Directives.Hook_getTensionTemplateInput = nothing
	c.Directives.Hook_queryTensionTemplateInput = nothing
	c.Directives.Hook_addTensionTemplateInput = nothing
	c.Directives.Hook_updateTensionTemplateInput = setContextWithID // used by @unique
	c.Directives.Hook_deleteTensionTemplateInput = nothing
	// --
	c.Directives.Hook_addTensionTemplate = addNodeArtefactHook
	c.Directives.Hook_updateTensionTemplate = updateNodeArtefactHook
	c.Directives.Hook_deleteTensionTemplate = deleteNodeArtefactHook
	// ProjectTemplate
	c.Directives.Hook_getProjectTemplateInput = nothing
	c.Directives.Hook_queryProjectTemplateInput = nothing
	c.Directives.Hook_addProjectTemplateInput = nothing
	c.Directives.Hook_updateProjectTemplateInput = setContextWithID // used by @unique
	c.Directives.Hook_deleteProjectTemplateInput = nothing
	// --
	c.Directives.Hook_addProjectTemplate = addNodeArtefactHook
	c.Directives.Hook_updateProjectTemplate = updateNodeArtefactHook
	c.Directives.Hook_deleteProjectTemplate = deleteNodeArtefactHook
	// Project
	c.Directives.Hook_getProjectInput = nothing
	c.Directives.Hook_queryProjectInput = nothing
	c.Directives.Hook_addProjectInput = nothing
	c.Directives.Hook_updateProjectInput = setContextWithID // used by the @unique
	c.Directives.Hook_deleteProjectInput = nothing
	// --
	c.Directives.Hook_addProject = addNodeArtefactHook
	c.Directives.Hook_updateProject = updateNodeArtefactHook
	c.Directives.Hook_deleteProject = nothing
	// ProjectColumn
	c.Directives.Hook_getProjectColumnInput = nothing
	c.Directives.Hook_queryProjectColumnInput = nothing
	c.Directives.Hook_addProjectColumnInput = nothing
	c.Directives.Hook_updateProjectColumnInput = nothing
	c.Directives.Hook_deleteProjectColumnInput = nothing
	// --
	c.Directives.Hook_addProjectColumn = addProjectColumnHook
	c.Directives.Hook_updateProjectColumn = updateProjectColumnHook
	c.Directives.Hook_deleteProjectColumn = deleteProjectColumnHook
	// ProjectCard
	c.Directives.Hook_getProjectCardInput = nothing
	c.Directives.Hook_queryProjectCardInput = nothing
	c.Directives.Hook_addProjectCardInput = nothing
	c.Directives.Hook_updateProjectCardInput = nothing
	c.Directives.Hook_deleteProjectCardInput = nothing
	// --
	c.Directives.Hook_addProjectCard = addProjectCardHook
	c.Directives.Hook_updateProjectCard = updateProjectCardHook
	c.Directives.Hook_deleteProjectCard = deleteProjectCardHook
	// ProjectDraft
	c.Directives.Hook_getProjectDraftInput = nothing
	c.Directives.Hook_queryProjectDraftInput = nothing
	c.Directives.Hook_addProjectDraftInput = nothing
	c.Directives.Hook_updateProjectDraftInput = nothing
	c.Directives.Hook_deleteProjectDraftInput = nothing
	// --
	c.Directives.Hook_addProjectDraft = nothing
	c.Directives.Hook_updateProjectDraft = updateProjectDraftHook
	c.Directives.Hook_deleteProjectDraft = nothing
	// Tension
	c.Directives.Hook_getTensionInput = nothing
	c.Directives.Hook_queryTensionInput = nothing
	c.Directives.Hook_addTensionInput = nothing
	c.Directives.Hook_updateTensionInput = setUpdateContextInfo // for @hasEvent+@isOwner
	c.Directives.Hook_deleteTensionInput = nothing
	// --
	c.Directives.Hook_addTension = addTensionHook
	c.Directives.Hook_updateTension = updateTensionHook
	c.Directives.Hook_deleteTension = nothing
	// Comment
	c.Directives.Hook_getCommentInput = nothing
	c.Directives.Hook_queryCommentInput = nothing
	c.Directives.Hook_addCommentInput = nothing
	c.Directives.Hook_updateCommentInput = setContextWithID // used by @isOwner
	c.Directives.Hook_deleteCommentInput = nothing
	// --
	c.Directives.Hook_addComment = nothing
	c.Directives.Hook_updateComment = nothing
	c.Directives.Hook_deleteComment = nothing
	// Reaction
	c.Directives.Hook_getReactionInput = nothing
	c.Directives.Hook_queryReactionInput = nothing
	c.Directives.Hook_addReactionInput = addReactionInputHook
	c.Directives.Hook_updateReactionInput = nothing
	c.Directives.Hook_deleteReactionInput = nothing
	// --
	c.Directives.Hook_addReaction = nothing
	c.Directives.Hook_updateReaction = nothing
	c.Directives.Hook_deleteReaction = nothing
	// Contract
	c.Directives.Hook_getContractInput = nothing
	c.Directives.Hook_queryContractInput = nothing
	c.Directives.Hook_addContractInput = addContractInputHook
	c.Directives.Hook_updateContractInput = setContextWithID // used by @isOwner
	c.Directives.Hook_deleteContractInput = nothing
	// --
	c.Directives.Hook_addContract = addContractHook
	c.Directives.Hook_updateContract = updateContractHook
	c.Directives.Hook_deleteContract = deleteContractHook
	// Vote
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

func ExtractInputs[T any](ctx context.Context, inputs *[]T) {
	a := graphql.GetResolverContext(ctx).Args["input"]
	ExtractSlice(a, inputs)
}

func ExtractInput[T any](ctx context.Context, input *T) {
	a := graphql.GetResolverContext(ctx).Args["input"]
	*input = StructMap[T](a)
}

func ExtractFilter[T any](ctx context.Context, filter *T) {
	a := graphql.GetResolverContext(ctx).Args["filter"]
	*filter = StructMap[T](a)
}

// nothing is the no-op directive used as the default for hooks that don't
// need any pre/post processing. Kept here because it's the shared default
// referenced from Init().
//
// Reminder: Api to access to input query:
//
//	rc := graphql.GetResolverContext(ctx)
//	rqc := graphql.GetRequestContext(ctx)
//	cfc := graphql.CollectFieldsCtx(ctx, nil)
//	fc := graphql.GetFieldContext(ctx)
//	pc := graphql.GetPathContext(ctx) // .*.Field to get the field name
func nothing(ctx context.Context, obj any, next graphql.Resolver) (any, error) {
	return next(ctx)
}
