package graph

import (
	"context"
	"fmt"
	"github.com/99designs/gqlgen/graphql"

	. "zerogov/fractal6.go/tools"
	"zerogov/fractal6.go/db"
	"zerogov/fractal6.go/graph/model"
	webauth "zerogov/fractal6.go/web/auth"
)

var FieldAuthorizationFunc map[string]func(context.Context, interface{}, graphql.Resolver, *string, *int) (interface{}, error)

func init() {

    FieldAuthorizationFunc = map[string]func(context.Context, interface{}, graphql.Resolver, *string, *int) (interface{}, error){
        "isOwner": isOwner,
        "unique": unique,
        "oneByOne": oneByOne,
        "minLen": minLength,
        "maxLen": maxLength,
    }

}



//isOwner Check that object is own by the user.
// If user(u) field is empty, assume a user object, else field should match the user(u) credential.
func isOwner(ctx context.Context, obj interface{}, next graphql.Resolver, f *string, n *int) (interface{}, error) {
    // Retrieve userCtx from token
    uctx, err := webauth.GetUserContext(ctx)
    if err != nil { return nil, LogErr("Access denied", err) }

    // Get attributes and check everything is ok
    var userObj model.JsonAtom
    var userField string
    if f == nil {
        userField = "user"
        userObj[userField] = obj
    } else {
        userField = *f
        userObj = obj.(model.JsonAtom)
    }

    ok, err := CheckUserOwnership(ctx, uctx, userField, userObj)
    if err != nil { return nil, LogErr("Access denied", err) }
    if ok { return next(ctx) }

    return nil, LogErr("Access Denied", fmt.Errorf("bad ownership."))
}

//unique Check uniqueness (@DEBUG follow @unique dgraph field iplementation)
// Ensure the field value is unique. If a field is given, it check the uniqueness on a subset of the parent type.
func unique(ctx context.Context, obj interface{}, next graphql.Resolver, f *string, n *int) (interface{}, error) {
    data, err := next(ctx)
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
        qName :=  SplitCamelCase(graphql.GetResolverContext(ctx).Field.Name)
        if len(qName) != 2 { return nil, LogErr("@unique", fmt.Errorf("Unknow query name")) }
        t := qName[1]
        fieldName := t + "." + field
        filterName := t + "." + *f
        s := obj.(model.JsonAtom)[*f]
        if s != nil {
            //pass
        } else if ctx.Value("id") != nil {
            s, err = db.GetDB().GetFieldById(ctx.Value("id").(string), filterName)
            if err != nil || s == nil { return nil, LogErr("Internal error", err) }
        } else {
            return nil, LogErr("Value Error", fmt.Errorf("'%s' or id is required.", *f))
        }
        filterValue := s.(string)

        // Check existence
        ex, err :=  db.GetDB().Exists(fieldName, v, &filterName, &filterValue)
        if err != nil { return nil, LogErr("Internal error", err) }
        if !ex {
            return data, err
        }
    } else {
        return nil, fmt.Errorf("@unique alone not implemented.")
    }

    return data, LogErr("Duplicate error", fmt.Errorf("'%s' is already taken", field))
}

//oneByOne ensure that the mutation on the given field should contains at least one element.
func oneByOne(ctx context.Context, obj interface{}, next graphql.Resolver, f *string, n *int) (interface{}, error) {
    data, err := next(ctx)
    if len(InterfaceSlice(data)) > 1 {
        field := *graphql.GetPathContext(ctx).Field
        return nil, LogErr("@oneByOne error", fmt.Errorf("Only one object allowed in slice '%s'", field))
    }
    return data, err
}

//inputMinLength the that the size of the field is stricly lesser than the given value
func minLength(ctx context.Context, obj interface{}, next graphql.Resolver, f *string, n *int) (interface{}, error) {
    var l int
    data, err := next(ctx)
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

//inputMaxLength the that the size of the field is stricly greater than the given value
func maxLength(ctx context.Context, obj interface{}, next graphql.Resolver, f *string, n *int) (interface{}, error) {
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
    if l > *n {
        field := *graphql.GetPathContext(ctx).Field
        return nil, fmt.Errorf("`%s' to short. Maximum length is %d", field, *n)
    }
    return data, err
}
