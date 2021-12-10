package graph

import (
	"context"
	"fmt"
	"strings"
	"github.com/99designs/gqlgen/graphql"
)

var FieldTransformFunc map[string]func(context.Context, graphql.Resolver) (interface{}, error)

func init() {

    FieldTransformFunc = map[string]func(context.Context, graphql.Resolver) (interface{}, error){
        "lower": lower,
    }

}


func lower(ctx context.Context, next graphql.Resolver) (interface{}, error) {
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

