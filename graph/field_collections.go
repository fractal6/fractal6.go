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
	)
}

func GetNestedPreloads(ctx *graphql.RequestContext, fields []graphql.CollectedField, prefix string) (preloads []string) {
	for i, column := range fields {
		prefixColumn := GetPreloadString(prefix, column.Name, i, len(fields))
		preloads = append(preloads, prefixColumn)
		preloads = append(preloads, GetNestedPreloads(ctx, graphql.CollectFields(ctx, column.SelectionSet, nil), prefixColumn)...)
		preloads = append(preloads, GetNestedPreloads(ctx, graphql.CollectFields(ctx, column.Selections, nil), prefixColumn)...)
	}
	return
}

func GetPreloadString(prefix, name string, i, N int) string {
	//if len(prefix) > 0 {
	//	return prefix + "." + name
	//}
    if i == 0 {
        return "{" + name
    } else if i == N-1 {
        return name + "}"
    } else {
        return name
    }
}

