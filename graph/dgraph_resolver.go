package graph

import (
    "fmt"
    "context"
    "strings"
    "encoding/json"
    "github.com/99designs/gqlgen/graphql"

	"zerogov/fractal6.go/tools"
    "zerogov/fractal6.go/db"
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
    reqInput := map[string]string{
        "QueryName": queryName,
        "RawQuery": tools.CleanString(graphql.GetRequestContext(ctx).RawQuery, true),
    }

    // Send request
    err := db.QueryGql("rawQuery", reqInput, data)
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
    // in order to to get the struct in the scema.resolver caller.

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

    // Send request
    op := string(mutCtx.type_)
    err := db.QueryGql(op, reqInput, data)
    if err != nil {
        return err
    }
    return nil
}
