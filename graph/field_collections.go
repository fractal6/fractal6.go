package graph

import (
    "fmt"
    "context"
    "strings"
    "reflect"
    "github.com/99designs/gqlgen/graphql"

	"fractale/fractal6.go/tools"
    "fractale/fractal6.go/graph/model"
)

//
// Misc field utils
//

func typeNameFromGraphqlContext(ctx context.Context) (string, error) {
    qName :=  tools.SplitCamelCase(graphql.GetResolverContext(ctx).Field.Name)
    if len(qName) < 2 {
        return "", fmt.Errorf("Unknow query type name")
    }
    typeName := strings.Join(qName[1:], "")
    return typeName, nil
}

// setContext add the {n} field in the context for further inspection in next resolvers.
// Its used in the hook_ resolvers for Update and Delete queries.
func setContextWith(ctx context.Context, obj interface{}, n string) (context.Context, string, error) {
    var val string
    var err error
    var filter model.JsonAtom

    if obj.(model.JsonAtom)["input"] != nil {
        if obj.(model.JsonAtom)["input"].(model.JsonAtom)["filter"] == nil { panic("add mutation not supported here.") }
        // Update mutation
        filter = obj.(model.JsonAtom)["input"].(model.JsonAtom)["filter"].(model.JsonAtom)
    } else if obj.(model.JsonAtom)["filter"] != nil {
        // Delete mutation
        filter = obj.(model.JsonAtom)["filter"].(model.JsonAtom)
    } else {
        return ctx, val, err
    }

    if filter[n] == nil {
        return ctx, val, err
    }

    switch n {
    case "nameid", "rootnameid", "username":
        v := filter[n].(model.JsonAtom)["eq"]
        if v != nil {
            val = v.(string)
        }
    case "id":
        ids := filter[n].([]interface{})
        if len(ids) != 1 {
            return ctx, val, fmt.Errorf("multiple ID is not allowed for this request.")
        }
        val = ids[0].(string)
    }

    ctx = context.WithValue(ctx, n, val)
    return ctx, val, err
}

func getNestedObj(obj interface{}, field string) interface{} {
    var source model.JsonAtom
    var target interface{}

    source =  obj.(model.JsonAtom)
    fields := strings.Split(field, ".")

    for i, f := range fields {
        target = source[f]
        if target == nil { return nil }
        if i < len(fields) -1 {
            source = target.(model.JsonAtom)
        }
    }

    return target
}


func get(obj model.JsonAtom, field string, deflt interface{}) interface{} {
    v := obj[field]
    if v == nil {
        return deflt
    }

    return v
}

//
// qqlgen code to extract fields
//

func GetPreloads(ctx context.Context) []string {
  return GetNestedPreloads(
    graphql.GetRequestContext(ctx),
    graphql.CollectFieldsCtx(ctx, nil),
    "", true,
  )
}

func GetNestedPreloads(ctx *graphql.RequestContext, fields []graphql.CollectedField, prefix string, first bool) (preloads []string) {
  //if first {
  //  fmt.Println(ctx.OperationName) // user define name of operation
  //  fmt.Println(ctx.Operation.Operation) // query|mutation|etc
  //  // @DEBUG: empty see: https://github.com/99designs/gqlgen/issues/1144
  //  fmt.Println("variables -> ", ctx.Variables, len(ctx.Variables)==0)
  //}
  for _, column := range fields {
    //prefixColumn := GetPreloadString(prefix, column.Name)
    prefixColumn := column.Name
    preloads = append(preloads, prefixColumn)
    if len(column.SelectionSet) > 0 {
      preloads = append(preloads, "{")
      preloads = append(preloads, GetNestedPreloads(ctx, graphql.CollectFields(ctx, column.SelectionSet, nil), prefixColumn, false)...)
      preloads = append(preloads, "}")
    }
  }
  return
}

func GetPreloadString(prefix, name string) string {
  var fname string = name
  return fname
}

// PayloadContains return true if the query payload contains the given field,
// by looking the raw json query (trough collected fields)
func PayloadContains(ctx context.Context, field string) bool {
    fields := graphql.CollectFieldsCtx(ctx, nil)[0]
    for _, c := range(graphql.CollectFields(graphql.GetRequestContext(ctx), fields.SelectionSet, nil)) {
        if c.Name == field {
            return true
        }
    }
    return false
}

// PayloadContains return true if the query payload contains the given field,
// by looking the reflected Go varaible. It is used for input hook when
// the payload is not available in the context.
func PayloadContainsGo(obj interface{}, field string) bool {
    n := reflect.ValueOf(obj).Elem().FieldByName(field).String()
    return n != ""
}
