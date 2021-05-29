//go:generate go run github.com/99designs/gqlgen -v

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.
package graph

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/99designs/gqlgen/graphql"

	"zerogov/fractal6.go/db"
	"zerogov/fractal6.go/graph/auth"
	"zerogov/fractal6.go/graph/codec"
	gen "zerogov/fractal6.go/graph/generated"
	"zerogov/fractal6.go/graph/model"
	. "zerogov/fractal6.go/tools"
	webauth "zerogov/fractal6.go/web/auth"
)

//
// Resolver initialisation
//

// Mutation type Enum
type mutationType string
const (
    AddMut mutationType = "add"
    UpdateMut mutationType = "update"
    DelMut mutationType = "delete"
)
type MutationContext struct  {
    type_ mutationType
    argName string
}

type Resolver struct{
    // Pointer on Dgraph client
    db *db.Dgraph
}

// Init initialize shema config and Directives...
func Init() gen.Config {
    r := Resolver{
        db:db.GetDB(),
    }

    // Dgraph directives
    c := gen.Config{Resolvers: &r}
    c.Directives.Id = nothing
    c.Directives.HasInverse = nothing2
    c.Directives.Search = nothing3
    c.Directives.Auth = nothing4

    //
    // Query
    //

    // Fields directives
    c.Directives.Hidden = hidden
    c.Directives.Count = count
    c.Directives.Meta_getNodeStats = getNodeStats

    //
    // Mutation
    //

    // Add inputs directives
    c.Directives.Add_isOwner = isOwner

    // Update or Remove inputs directives
    c.Directives.Patch_isOwner = isOwner
    c.Directives.Patch_RO = readOnly
    c.Directives.Patch_hasRole = hasRole

    // Add, Update and Remove inputs directives
    c.Directives.Alter_unique = unique
    c.Directives.Alter_oneByOne = oneByOne
    c.Directives.Alter_toLower = toLower
    c.Directives.Alter_minLength = inputMinLength
    c.Directives.Alter_maxLength = inputMaxLength
    c.Directives.Alter_hasRole = hasRole
    c.Directives.Alter_hasRoot = hasRoot

    //
    // Hook
    //

    //Node
    c.Directives.Hook_getNode = nothing
    c.Directives.Hook_queryNode = nothing
    c.Directives.Hook_addNode = nothing
    c.Directives.Hook_addNodePost = nothing
    c.Directives.Hook_updateNode = updateNodeHook
    c.Directives.Hook_updateNodePost = nothing
    c.Directives.Hook_deleteNode = nothing
    c.Directives.Hook_deleteNodePost = nothing
    //Label
    c.Directives.Hook_getLabel = nothing
    c.Directives.Hook_queryLabel = nothing
    c.Directives.Hook_addLabel = addLabelHook
    c.Directives.Hook_addLabelPost = nothing
    c.Directives.Hook_updateLabel = updateLabelHook
    c.Directives.Hook_updateLabelPost = nothing
    c.Directives.Hook_deleteLabel = nothing
    c.Directives.Hook_deleteLabelPost = nothing
    //Tension
    c.Directives.Hook_getTension = nothing
    c.Directives.Hook_queryTension = nothing
    c.Directives.Hook_addTension = addTensionHook
    c.Directives.Hook_addTensionPost = addTensionPostHook
    c.Directives.Hook_updateTension = updateTensionHook
    c.Directives.Hook_updateTensionPost = updateTensionPostHook
    c.Directives.Hook_deleteTension = nothing
    c.Directives.Hook_deleteTensionPost = nothing
    //Comment
    c.Directives.Hook_getComment = nothing
    c.Directives.Hook_queryComment = nothing
    c.Directives.Hook_addComment = nothing
    c.Directives.Hook_addCommentPost = nothing
    c.Directives.Hook_updateComment = updateCommentHook
    c.Directives.Hook_updateCommentPost = nothing
    c.Directives.Hook_deleteComment = nothing
    c.Directives.Hook_deleteCommentPost = nothing
    //Contract
    c.Directives.Hook_getContract = nothing
    c.Directives.Hook_queryContract = nothing
    c.Directives.Hook_addContract = nothing
    c.Directives.Hook_addContractPost = nothing
    c.Directives.Hook_updateContract = nothing
    c.Directives.Hook_updateContractPost = nothing
    c.Directives.Hook_deleteContract = nothing
    c.Directives.Hook_deleteContractPost = deleteContractHookPost

    return c
}


/*
*
* Business logic layer methods
*
*/

func nothing(ctx context.Context, obj interface{}, next graphql.Resolver) (interface{}, error) {
    return next(ctx)
}

func nothing2(ctx context.Context, obj interface{}, next graphql.Resolver, key string) (interface{}, error) {
    return next(ctx)
}

func nothing3(ctx context.Context, obj interface{}, next graphql.Resolver, idx []model.DgraphIndex) (interface{}, error) {
    return next(ctx)
}

func nothing4(ctx context.Context, obj interface {}, next graphql.Resolver, idx *model.AuthRule, r1 *model.AuthRule, r2 *model.AuthRule, r3 *model.AuthRule, r4 *model.AuthRule) (interface{}, error) {
    return next(ctx)
}


// To document. Api to access to input query:
//  rc := graphql.GetResolverContext(ctx)
//  rqc := graphql.GetRequestContext(ctx)
//  cfc := graphql.CollectFieldsCtx(ctx, nil)
//  fc := graphql.GetFieldContext(ctx)
//  pc := graphql.GetPathContext(ctx) // .*.Field to get the field name

//
// Query
//

func hidden(ctx context.Context, obj interface{}, next graphql.Resolver) (interface{}, error) {
    rc := graphql.GetResolverContext(ctx)
    fieldName := rc.Field.Name
    return nil, fmt.Errorf("`%s' field is hidden", fieldName)
}

func count(ctx context.Context, obj interface{}, next graphql.Resolver, field string) (interface{}, error) {
    rc := graphql.GetResolverContext(ctx)
    fieldName := rc.Field.Name
    goFieldfDef := ToGoNameFormat(fieldName)

    // Reflect to get obj data info
    // DEBUG: use type switch instead ? (less modular but faster?)
    id := reflect.ValueOf(obj).Elem().FieldByName("ID").String()
    if id == "" {
        err := fmt.Errorf("`id' field is needed to query `%s'", fieldName)
        return nil, err
    }
    typeName := ToTypeName(reflect.TypeOf(obj).String())
    v := db.GetDB().Count(id, typeName+"."+field)
    if v >= 0 {
        reflect.ValueOf(obj).Elem().FieldByName(goFieldfDef).Set(reflect.ValueOf(&v))
    }
    return next(ctx)
}

func getNodeStats(ctx context.Context, obj interface{}, next graphql.Resolver) (interface{}, error) {
    rc := graphql.GetResolverContext(ctx)
    fieldName := rc.Field.Name

    // Reflect to get obj data info
    // DEBUG: use type switch instead ? (less modular but faster?)
    nameid := reflect.ValueOf(obj).Elem().FieldByName("Nameid").String()
    if nameid == "" {
        err := fmt.Errorf("`nameid' field is needed to query `%s'", fieldName)
        return nil, err
    }
    stats := db.GetDB().GetNodeStats(nameid)
    n_guest := stats["n_guest"]
    n_member := stats["n_member"]
    stats_ := model.NodeStats{
        NGuest: &n_guest,
        NMember: &n_member,
    }
    reflect.ValueOf(obj).Elem().FieldByName("Stats").Set(reflect.ValueOf(&stats_))
    //for k, v := range stats {
    //    goFieldfDef := ToGoNameFormat(k)
    //    reflect.ValueOf(obj).Elem().FieldByName(goFieldfDef).Set(reflect.ValueOf(&stats))
    //}
    return next(ctx)
}

//
// Field utils
//

// Check uniqueness (@DEBUG follow @unique dgraph field iplementation)
func unique(ctx context.Context, obj interface{}, next graphql.Resolver, subfield *string) (interface{}, error) {
    data, err := next(ctx)
    var v string
    switch d := data.(type) {
    case *string:
        v = *d
    case string:
        v = d
    }

    // Extract the fieldname and type of the object queried
    field := *graphql.GetPathContext(ctx).Field
    s :=  SplitCamelCase(graphql.GetResolverContext(ctx).Field.Name)
    if len(s) != 2 { return nil, LogErr("@unique", fmt.Errorf("Unknow query name")) }
    t := s[1]

    fieldName := t + "." + field
    id := ctx.Value("id")
    if subfield != nil {
        filterName := t + "." + *subfield
        s := obj.(model.JsonAtom)[*subfield]
        if s != nil {
            //pass
        } else if id != nil {
            s, err = db.GetDB().GetFieldById(id.(string), filterName)
            if err != nil || s == nil { return nil, LogErr("Internal error", err) }
        } else {
            return nil, LogErr("Value Error", fmt.Errorf("%s or id is required.", *subfield))
        }
        filterValue := s.(string)
        ex, err :=  db.GetDB().Exists(fieldName, v, &filterName, &filterValue)
        if err != nil { return nil, LogErr("Internal error", err) }
        if !ex {
            return data, err
        }
    } else {
        return nil, fmt.Errorf("@unique alone not implemented.")
    }

    return data, LogErr("Duplicate error", fmt.Errorf("%s is already taken", field))
}

//oneByOne ensure that mutation on the given field should contains set least one element.
func oneByOne(ctx context.Context, obj interface{}, next graphql.Resolver) (interface{}, error) {
    data, err := next(ctx)
    if len(InterfaceSlice(data)) == 1 {
        return data, err
    }
    field := *graphql.GetPathContext(ctx).Field
    return nil, LogErr("@oneByOne error", fmt.Errorf("Only one object allowed in slice %s", field))
}

func toLower(ctx context.Context, obj interface{}, next graphql.Resolver) (interface{}, error) {
    data, err := next(ctx)
    switch d := data.(type) {
    case *string:
        v := strings.ToLower(*d)
        return &v, err
    case string:
        v := strings.ToLower(d)
        return v, err
    }
    field := *graphql.GetPathContext(ctx).Field
    return nil, fmt.Errorf("Type unknwown for field %s", field)
}

func inputMinLength(ctx context.Context, obj interface{}, next graphql.Resolver, min int) (interface{}, error) {
    var l int
    data, err := next(ctx)
    switch d := data.(type) {
    case *string:
        l = len(*d)
    case string:
        l = len(d)
    default:
        field := *graphql.GetPathContext(ctx).Field
        return nil, fmt.Errorf("Type unknwown for field %s", field)
    }
    if l < min {
        field := *graphql.GetPathContext(ctx).Field
        return nil, fmt.Errorf("`%s' to short. Minimum length is %d", field, min)
    }
    return data, err
}

func inputMaxLength(ctx context.Context, obj interface{}, next graphql.Resolver, max int) (interface{}, error) {
    var l int
    data, err := next(ctx)
    switch d := data.(type) {
    case *string:
        l = len(*d)
    case string:
        l = len(d)
    default:
        field := *graphql.GetPathContext(ctx).Field
        return nil, fmt.Errorf("Type unknwown for field %s", field)
    }
    if l > max {
        field := *graphql.GetPathContext(ctx).Field
        return nil, fmt.Errorf("`%s' to short. Maximum length is %d", field, max)
    }
    return data, err
}

//
// Auth directives
//

// HasRole check the user has the authorisation to update a ressource by checking if it satisfies at least one of
// 1. user owner
// 1. user rights (n field)
// 3. check assignees
// 4. check residual
func hasRole(ctx context.Context, obj interface{}, next graphql.Resolver, nFields []string, uField *string, assignee *int) (interface{}, error) {
    // Retrieve userCtx from token
    uctx, err := webauth.UserCtxFromContext(ctx)
    if err != nil { return nil, LogErr("Access denied", err) }

    var ok bool
    if uField != nil { // Check if the user is the creator of the ressource
        ok, err = CheckUserOwnership(ctx, uctx, *uField, obj)
        if err != nil { return nil, LogErr("Access denied", err) }
        if ok { return next(ctx) }
    }

    if assignee != nil { // Check if the user is an assignee of the curent tension
        ok, err = CheckAssignees(ctx, uctx, obj)
        if err != nil { return nil, LogErr("Access denied", err) }
        if ok { return next(ctx) }
    }

    for _, nField := range nFields { // Check if the user has the given (nested) role on the asked node
        nameid, err := extractNameid(ctx, nField, obj)
        if err != nil { return nil, LogErr("Internal error", err) }

        // Check user rights
        ok, err := CheckUserRights(uctx, nameid, nil)
        if err != nil { return nil, LogErr("Internal error", err) }
        if ok { return next(ctx) }
    }

    // Check if user has rights in any parents if the node has no Coordo role.
    if !ok && ctx.Value("nameid") != nil && !db.GetDB().HasCoordos(ctx.Value("nameid").(string)) { // is a Node
        ok, err = CheckUpperRights(uctx, ctx.Value("nameid").(string), nil)
        if err != nil { return nil, LogErr("Internal error", err) }
        if ok { return next(ctx) }
    }

    return nil, LogErr("Access denied", fmt.Errorf("Contact a coordinator to access this ressource."))
}

// HasRoot check the list of node to check if the user has root node in common.
func hasRoot(ctx context.Context, obj interface{}, next graphql.Resolver, nodeFields []string) (interface{}, error) {
    // Retrieve userCtx from token
    uctx, err := webauth.UserCtxFromContext(ctx)
    if err != nil { return nil, LogErr("Access denied", err) }

    // Check that user has the given role on the asked node
    var rootnameid string
    for _, nodeField := range nodeFields {
        rootnameid, err = extractRootnameid(ctx, nodeField, obj)
        if err != nil { return nil, LogErr("Internal error", err) }
        if auth.UserIsMember(uctx, rootnameid) >= 0 { return next(ctx) }
    }

    e := LogErr("Access denied", fmt.Errorf("Contact a coordinator to access this ressource."))

    // Check for bot access
    nameid_ := obj.(model.JsonAtom)["emitterid"]
    if nameid_ == nil { return nil, e }
    nameid := nameid_.(string)
    rid, err := codec.Nid2rootid(nameid)
    if err != nil { return nil, LogErr("Internal error", err) }
    if rid == rootnameid {
        r_, err := db.GetDB().GetFieldByEq("Node.nameid", nameid, "Node.role_type")
        if err != nil { return nil, LogErr("Internal error", err) }
        isArchived, err := db.GetDB().GetFieldByEq("Node.nameid", nameid, "Node.isArchived")
        if err != nil { return nil, LogErr("Internal error", err) }
        if r_ == nil { return nil, e }
        if model.RoleType(r_.(string)) == model.RoleTypeBot && !isArchived.(bool) {
            return next(ctx)
        }
    }

    return nil, e
}

// Only the onwer of the object can edit it.
func isOwner(ctx context.Context, obj interface{}, next graphql.Resolver, userField *string) (interface{}, error) {
    // Retrieve userCtx from token
    uctx, err := webauth.UserCtxFromContext(ctx)
    if err != nil { return nil, LogErr("Access denied", err) }

    // Get attributes and check everything is ok
    var userObj model.JsonAtom
    var f string
    if userField == nil {
        f = "user"
        userObj[f] = obj
    } else {
        f = *userField
        userObj = obj.(model.JsonAtom)
    }

    ok, err := CheckUserOwnership(ctx, uctx, f, userObj)
    if err != nil { return nil, LogErr("Access denied", err) }
    if ok { return next(ctx) }

    return nil, LogErr("Access Denied", fmt.Errorf("bad ownership."))
}

func readOnly(ctx context.Context, obj interface{}, next graphql.Resolver) (interface{}, error) {
    rc := graphql.GetResolverContext(ctx)
    fieldName := rc.Field.Name
    return nil, LogErr("Forbiden", fmt.Errorf("Read only field on `%s'", fieldName))
}


//
// Private auth methods
//

func extractRootnameid(ctx context.Context, nodeField string, nodeObj interface{}) (string, error) {
    // Check that nodes are present
    var rootnameid string
    var err error
    nodeTarget_ := getNestedObj(nodeObj, nodeField)
    if nodeTarget_ == nil {
        // Tension here
        id := ctx.Value("id").(string)
        if id == "" {
            return rootnameid, fmt.Errorf("node target unknown(id), need a database request here...")
        }
        rootnameid_, err := db.GetDB().GetSubFieldById(id, "Tension."+nodeField, "Node.rootnameid")
        if err != nil { return rootnameid, err }
        if rootnameid_ != nil {
            rootnameid = rootnameid_.(string)
        }
    } else {
        // Node here
        nameid_ := nodeTarget_.(model.JsonAtom)["nameid"]
        if nameid_ == nil {
            return rootnameid, fmt.Errorf("node target unknown (nameid), need a database request here...")
        }
        rootnameid, err = codec.Nid2rootid(nameid_.(string))
        if err != nil {
            panic(err.Error())
        }
    }

    return rootnameid, err
}

func extractNameid(ctx context.Context, nodeField string, nodeObj interface{}) (string, error) {
    // Check that nodes are present
    var nameid string
    var err error
    node := nodeObj.(model.JsonAtom)[nodeField]
    id_ := ctx.Value("id")
    nameid_ := ctx.Value("nameid")
    if id_ != nil {
        // Tension here
        // Request the database to get the field
        nameid_, err = db.GetDB().GetSubFieldById(id_.(string), "Tension."+nodeField, "Node.nameid")
        if err != nil { return nameid, err }
        nameid = nameid_.(string)
    } else if (nameid_ != nil) {
        // Node Here
        if nodeField == "__self__" { return nameid_.(string), err }
        nameid_, err := db.GetDB().GetSubFieldByEq("Node.nameid", nameid_.(string), "Node."+nodeField, "Node.nameid")
        if err != nil { return nameid, err }
        if nameid_ == nil {
            // Assume root node
            return nameid, fmt.Errorf("Root node updates are not implemented yet...")
        }
        nameid = nameid_.(string)
    } else if (node != nil && node.(model.JsonAtom)["nameid"] != nil) {
        nameid = node.(model.JsonAtom)["nameid"].(string)
    } else {
        return nameid, fmt.Errorf("node target unknown, need a database request here...")
    }

    return nameid, err
}
