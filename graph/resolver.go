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
    // Query / Payload fields
    //

    // Fields directives
    c.Directives.Hidden = hidden
    c.Directives.Count = count
    c.Directives.Meta = meta
    c.Directives.IsContractValidator = isContractValidator

    //
    // Mutation / Input Fields
    //

    // isOwner
    c.Directives.Add_isOwner = isOwner
    c.Directives.Patch_isOwner = isOwner

    // Read-Only
    c.Directives.Alter_RO = readOnly
    c.Directives.Patch_RO = readOnly

    // Input validation
    c.Directives.Alter_minLength = inputMinLength
    c.Directives.Alter_maxLength = inputMaxLength
    c.Directives.Alter_oneByOne = oneByOne
    c.Directives.Alter_unique = unique

    // Input transformation
    c.Directives.Alter_toLower = toLower

    //
    // Hook
    //

    //Node
    c.Directives.Hook_getNodeInput = nothing
    c.Directives.Hook_queryNodeInput = nothing
    c.Directives.Hook_addNodeInput = nothing
    c.Directives.Hook_updateNodeInput = nothing
    c.Directives.Hook_deleteNodeInput = nothing
    // --
    c.Directives.Hook_addNode = nothing
    c.Directives.Hook_updateNode = nothing
    c.Directives.Hook_deleteNode = nothing
    //Label
    c.Directives.Hook_getLabelInput = nothing
    c.Directives.Hook_queryLabelInput = nothing
    c.Directives.Hook_addLabelInput = nothing
    c.Directives.Hook_updateLabelInput = setContextWithID
    c.Directives.Hook_deleteLabelInput = nothing
    // --
    c.Directives.Hook_addLabel = addLabelHook
    c.Directives.Hook_updateLabel = updateLabelHook
    c.Directives.Hook_deleteLabel = nothing
    //Tension
    c.Directives.Hook_getTensionInput = nothing
    c.Directives.Hook_queryTensionInput = nothing
    c.Directives.Hook_addTensionInput = nothing
    c.Directives.Hook_updateTensionInput = nothing
    c.Directives.Hook_deleteTensionInput = nothing
    // --
    c.Directives.Hook_addTension = addTensionHook
    c.Directives.Hook_updateTension = updateTensionHook
    c.Directives.Hook_deleteTension = nothing
    //Comment
    c.Directives.Hook_getCommentInput = nothing
    c.Directives.Hook_queryCommentInput = nothing
    c.Directives.Hook_addCommentInput = nothing
    c.Directives.Hook_updateCommentInput = nothing
    c.Directives.Hook_deleteCommentInput = nothing
    // --
    c.Directives.Hook_addComment = nothing
    c.Directives.Hook_updateComment = nothing
    c.Directives.Hook_deleteComment = nothing
    //Contract
    c.Directives.Hook_getContractInput = nothing
    c.Directives.Hook_queryContractInput = nothing
    c.Directives.Hook_addContractInput = nothing
    c.Directives.Hook_updateContractInput = nothing
    c.Directives.Hook_deleteContractInput = nothing
    // --
    c.Directives.Hook_addContract = addContractHook
    c.Directives.Hook_updateContract = updateContractHook
    c.Directives.Hook_deleteContract = deleteContractHook

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
// Query Fields
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

func meta(ctx context.Context, obj interface{}, next graphql.Resolver, f string, k string) (interface{}, error) {
    data, err:= next(ctx)
    if err != nil { return nil, err }

    // @debug: obj cast Doesnt work here why ?!
    //v := obj.(model.JsonAtom)[k]
    // Using reflexion
    v := reflect.ValueOf(obj).Elem().FieldByName(ToGoNameFormat(k)).String()
    if v == "" {
        rc := graphql.GetResolverContext(ctx)
        fieldName := rc.Field.Name
        err := fmt.Errorf("`%s' field is needed to query `%s'", k, fieldName)
        return nil, err
    }
    res := db.GetDB().Meta(k, v, f)
    if err != nil { return nil, err }
    err = Map2Struct(res, &data)
    return data, err

    // Rewrite graph result with reflection
    //n_guest := stats["n_guest"]
    //n_member := stats["n_member"]
    //stats_ := model.NodeStats{
    //    NGuest: &n_guest,
    //    NMember: &n_member,
    //}
    //reflect.ValueOf(obj).Elem().FieldByName("Stats"v).Set(reflect.ValueOf(&stats_))
    //
    //for k, v := range stats {
    //    goFieldfDef := ToGoNameFormat(k)
    //    reflect.ValueOf(obj).Elem().FieldByName(goFieldfDef).Set(reflect.ValueOf(&stats))
    //}
    //return next(ctx)
}

//
// Input Field
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

    field := *graphql.GetPathContext(ctx).Field
    if subfield != nil {
        // Extract the fieldname and type of the object queried
        qName :=  SplitCamelCase(graphql.GetResolverContext(ctx).Field.Name)
        if len(qName) != 2 { return nil, LogErr("@unique", fmt.Errorf("Unknow query name")) }
        t := qName[1]
        fieldName := t + "." + field
        filterName := t + "." + *subfield
        s := obj.(model.JsonAtom)[*subfield]
        if s != nil {
            //pass
        } else if ctx.Value("id") != nil {
            s, err = db.GetDB().GetFieldById(ctx.Value("id").(string), filterName)
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
    if len(InterfaceSlice(data)) != 1 {
        field := *graphql.GetPathContext(ctx).Field
        return nil, LogErr("@oneByOne error", fmt.Errorf("Only one object allowed in slice %s", field))
    }
    return data, err
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

// Only the onwer of the object can edit it.
func isOwner(ctx context.Context, obj interface{}, next graphql.Resolver, userField *string) (interface{}, error) {
    // Retrieve userCtx from token
    uctx, err := webauth.GetUserContext(ctx)
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

