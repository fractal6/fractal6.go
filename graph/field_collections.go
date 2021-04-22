package graph

import (
    //"fmt"
    "context"
    "github.com/99designs/gqlgen/graphql"
)

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

func PayloadContains(ctx context.Context, field string) bool {
    fields := graphql.CollectFieldsCtx(ctx, nil)[0]
    for _, c := range(graphql.CollectFields(graphql.GetRequestContext(ctx), fields.SelectionSet, nil)) {
        if c.Name == field {
            return true
        }
    }
    return false
}
