package graph

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/99designs/gqlgen/graphql"

	"zerogov/fractal6.go/db"
	"zerogov/fractal6.go/graph/model"
    webauth "zerogov/fractal6.go/web/auth"
	"zerogov/fractal6.go/tools"
)


func (r *queryResolver) Gqlgen2DgraphQueryResolver(ctx context.Context, data interface{}) error {
    return DgraphRawQueryResolver(ctx, data, r.db)
}

func (r *mutationResolver) Gqlgen2DgraphQueryResolver(ctx context.Context, data interface{}) error {
    return DgraphRawQueryResolver(ctx, data, r.db)
}

//func (r *mutationResolver) Gqlgen2DgraphMutationResolver(ctx context.Context, ipts interface{}, data interface{}) error {
//    return DgraphQueryResolver(ctx, ipts, data, r.db)
//}

/*
*
* Dgraph-Gqlgen bridge logic
*
*/

func DgraphRawQueryResolver(ctx context.Context, data interface{}, db *db.Dgraph) error {
    // How to get the query args ? https://github.com/99designs/gqlgen/issues/1144
    // for k, a := range rc.Args {

    /* Rebuild the Graphql inputs request from this context */
    rc := graphql.GetResolverContext(ctx)
    // rc.Field.Name
    queryName := rc.Path().String()


    // Build the graphql raw request
    gc := graphql.GetRequestContext(ctx)
    variables, _ := json.Marshal(gc.Variables)
    reqInput := map[string]string{
        "QueryName": queryName,
        "RawQuery": tools.QuoteString(gc.RawQuery),
        "Variables": string(variables),
    }

    // Send request
    uctx, err := webauth.GetUserContext(ctx)
    if err != nil { uctx = &model.UserCtx{} }
    err = db.QueryGql(*uctx, "rawQuery", reqInput, data)
    if err != nil {
        return err
    }
    return nil
}

// @Debug: GetPreloads loose subfilter in payload(in QueryGraph)
func DgraphQueryResolver(ctx context.Context, ipts interface{}, data interface{}, db *db.Dgraph) error {
    mutCtx := ctx.Value("mutation_context").(MutationContext)

    /* Rebuild the Graphql inputs request from this context */
    rc := graphql.GetResolverContext(ctx)
    queryName := rc.Field.Name

    // Format inputs
    inputs, _ := json.Marshal(ipts)
    // If inputs needs to get modified, see tools.StructToMap() usage
    // in order to to get the struct in the schema.resolver caller.

    // Format collected fields
    inputType := strings.Split(fmt.Sprintf("%T", rc.Args[mutCtx.argName]), ".")[1]
    queryGraph := strings.Join(GetPreloads(ctx), " ")

    // Build the graphql raw request
    reqInput := map[string]string{
        "QueryName": queryName, // function name (e.g addUser)
        "InputType": inputType, // input type name (e.g AddUserInput)
        "QueryGraph": queryGraph, // output data
        "InputPayload": string(inputs), // inputs data
    }

    op := string(mutCtx.type_)

    // Send request
    uctx, err := webauth.GetUserContext(ctx)
    if err != nil { uctx = &model.UserCtx{} }
    err = db.QueryGql(*uctx, op, reqInput, data)
    if err != nil {
        return err
    }
    return nil
}
