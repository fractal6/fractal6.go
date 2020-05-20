package graph

import (
    "fmt"
    "context"
    "strings"
    "encoding/json"
    "github.com/99designs/gqlgen/graphql"

	"zerogov/fractal6.go/tools"
)

/*
*
* Dgraph-Gqlgen bridge logic
*
*/

func (r *queryResolver) Gqlgen2DgraphQueryResolver(ctx context.Context, data interface{}) error {
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
    op := "rawQuery"
    err := r.db.QueryGql(op, reqInput, data)
    if err != nil {
        //panic(err)
        return err
    }
    return nil
}

func (r *mutationResolver) Gqlgen2DgraphMutationResolver(ctx context.Context, data interface{}, ipts interface{}) error {
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
    err := r.db.QueryGql(op, reqInput, data)
    if err != nil {
        //panic(err)
        return err
    }
    return nil
}

