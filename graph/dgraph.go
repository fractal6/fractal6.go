package graph

import (
    "fmt"
    "context"
    "strings"
    "encoding/json"
    "github.com/mitchellh/mapstructure"
    "github.com/99designs/gqlgen/graphql"

    "zerogov/fractal6.go/tools/gql"
)

/*
*
* Dgraph-Gqlgen bridge logic
*
*/

func (r *mutationResolver) Gqlgen2DgraphMutationResolver(ctx context.Context, ipts []gql.JsonAtom, data interface{}) (errors error) {
    /* Rebuild the Graphql inputs request from this context */
    ctxRslv := graphql.GetResolverContext(ctx)
    queryName := ctxRslv.Field.Name
    inputType := strings.Split(fmt.Sprintf("%T", ctxRslv.Args["input"]), ".")[1]
    //fmt.Println(string(ctx.Value("request_body").([]byte)))
    // Format inputs
    inputs, _ := json.Marshal(ipts)
    
    // Format collected fields
    queryGraph := strings.Join(GetPreloads(ctx), " ")

    // Build the string request
    reqInput := gql.JsonAtom{
        "QueryName": queryName, // function name (e.g addUser)
        "InputType": inputType, // input type name (e.g AddUserInput
        "InputPayload": string(inputs), // inputs data
        "QueryGraph": queryGraph, // output data
    }
    req := r.MutationQ.Format(reqInput)

    /* Send the dgraph request and follow the results */
    // Dgraph request
    res := &gql.Res{} // or new(Res)
    err := r.db.Request([]byte(req), res)
    //fmt.Println(string(req))
    //fmt.Println(res)
    if err != nil {
        panic(err)
    } else if res.Errors != nil {
        var msg []string
        for _, m := range res.Errors {
            msg = append(msg, m["message"].(string))
        }
        //DBUG: see gqlgen doc to returns erros as list.
        errors = fmt.Errorf("%s", strings.Join(msg, " | "))
        return errors
    }

    config := &mapstructure.DecoderConfig{TagName: "json", Result: data}
    decoder, err := mapstructure.NewDecoder(config)
    decoder.Decode(res.Data[queryName])
    if err != nil {
        panic(err)
    }
    return 
}

func (r *queryResolver) Gqlgen2DgraphQueryResolver(ctx context.Context, data interface{}) (errors error) {

    /* Rebuild the Graphql inputs request from this context */
    ctxRslv := graphql.GetResolverContext(ctx)
    queryName := ctxRslv.Field.Name
    //reqq := ctx.Value("request_body").([]byte)
    //fmt.Println(string(reqq))
    
    // Format collected fields
    queryGraph := strings.Join(GetPreloads(ctx), " ")

    // Build the string request
    reqInput := gql.JsonAtom{
        "QueryName": queryName, // function name (e.g addUser)
        "QueryGraph": queryGraph, // output data
    }
    req := r.QueryQ.Format(reqInput)

    /* Send the dgraph request and follow the results */
    // Dgraph request
    res := &gql.Res{} // or new(Res)
    err := r.db.Request([]byte(req), res)
    //fmt.Println(string(req))
    //fmt.Println(res)
    if err != nil {
        panic(err)
    } else if res.Errors != nil {
        var msg []string
        for _, m := range res.Errors {
            msg = append(msg, m["message"].(string))
        }
        //DBUG: see gqlgen doc to returns erros as list.
        errors = fmt.Errorf("%s", strings.Join(msg, " | "))
        return errors
    }

    config := &mapstructure.DecoderConfig{TagName: "json", Result: data}
    decoder, err := mapstructure.NewDecoder(config)
    decoder.Decode(res.Data[queryName])
    if err != nil {
        panic(err)
    }
    return 
}
