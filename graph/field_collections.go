package graph

import (
    "context"
    "github.com/99designs/gqlgen/graphql"
)

func GetPreloads(ctx context.Context) []string {
    return GetNestedPreloads(
	graphql.GetRequestContext(ctx),
	graphql.CollectFieldsCtx(ctx, nil),
	"",
	0,
    )
}

func GetNestedPreloads(ctx *graphql.RequestContext, fields []graphql.CollectedField, prefix string, level int) (preloads []string) {
    for _, column := range fields {
	prefixColumn := GetPreloadString(prefix, column.Name)
	preloads = append(preloads, prefixColumn)
	if len(column.SelectionSet) > 0 {
	    preloads = append(preloads, "{")
	    preloads = append(preloads, GetNestedPreloads(ctx, graphql.CollectFields(ctx, column.SelectionSet, nil), prefixColumn, level+1 )...)
	    //preloads = append(preloads, GetNestedPreloads(ctx, graphql.CollectFields(ctx, column.Selections, nil), prefixColumn)...)
	    preloads = append(preloads, "}")
	}
    }
    return
}

func GetPreloadString(prefix, name string) string {
    var fname string = name
    return fname
}

