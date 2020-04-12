package graph

import (
    "fmt"
    "context"
    "strings"
    "encoding/json"
    "github.com/mitchellh/mapstructure"
    "github.com/99designs/gqlgen/graphql"

	"zerogov/fractal6.go/tools"
)

/*
*
* Dgraph-Gqlgen bridge logic
*
*/

func (r *mutationResolver) Gqlgen2DgraphMutationResolver(ctx context.Context, data interface{}, ipts interface{}) (errors error) {
    mutCtx := ctx.Value("mutation_context").(MutationContext)

    /* Rebuild the Graphql inputs request from this context */
    rslvCtx := graphql.GetResolverContext(ctx)
    queryName := rslvCtx.Field.Name

    // Format inputs
    inputs, _ := json.Marshal(ipts)
    // If inputs needs to get modified, see tools.StructToMap() usage 
    // in order to to get the struct in the scema.resolver caller.

    // Format collected fields
    inputType := strings.Split(fmt.Sprintf("%T", rslvCtx.Args[mutCtx.argName]), ".")[1]
    queryGraph := strings.Join(GetPreloads(ctx), " ")
    // Build the string request
    reqInput := JsonAtom{
        "QueryName": queryName, // function name (e.g addUser)
        "InputType": inputType, // input type name (e.g AddUserInput
        "QueryGraph": queryGraph, // output data
        "InputPayload": string(inputs), // inputs data
    }
    var req string
    switch mutCtx.type_ {
    case AddMut:
        req = r.AddMutationQ.Format(reqInput)
    case UpdateMut:
        req = r.UpdateMutationQ.Format(reqInput)
    case DelMut:
        req = r.DelMutationQ.Format(reqInput)
    default:
        panic("Not implemented mutation type:" + mutCtx.type_)
    }

    /* Send the dgraph request and follow the results */
    // Dgraph request
    res := &GqlRes{} // or new(Res)
    //fmt.Println("request ->", string(req))
    err := r.db.Request([]byte(req), res)
    //fmt.Println("response ->", res)
    if err != nil {
        panic(err)
    } else if res.Errors != nil {
        // see gqlgen doc to returns erros as list.
        e, _ := json.Marshal(res.Errors)
        return fmt.Errorf(string(e))
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
    rslvCtx := graphql.GetResolverContext(ctx)
    queryName := rslvCtx.Field.Name

    // How to get the query args ? https://github.com/99designs/gqlgen/issues/1144
    ////var queryArgs string
    ////var args []string
    ////if len(rslvCtx.Args) > 0 {
    ////    for k, a := range rslvCtx.Args {
    ////        var v string
    ////        switch _v := a.(type) {
    ////        case *int:
    ////            if _v != nil {
    ////                v = strconv.Itoa(*_v)
    ////            }
    ////        case *string:
    ////            if _v != nil {
    ////                v = *_v
    ////            }
    ////        default:
    ////            if _v != nil && !reflect.ValueOf(_v).IsNil() {
    ////                fmt.Println(k, a, fmt.Sprintf("%T", a))
    ////                vv, _ := json.Marshal(_v)
    ////                v = string(vv)
    ////            }
    ////        }
    ////        if len(v) > 0 {
    ////            args = append(args, k + ":" + v)
    ////        }
    ////    }
    ////}
    ////queryArgs = "(" + strings.Join(args, ",") + ")"
    ////fmt.Println(queryArgs)

    ////reqq := ctx.Value("request_body").([]byte)
    ////fmt.Println(string(reqq))
    //
    //// Format collected fields
    //queryGraph := strings.Join(GetPreloads(ctx), " ")

    //// Build the string request
    //reqInput := JsonAtom{
    //    "QueryName": queryName, // function name (e.g addUser)
    //    "QueryGraph": queryGraph, // output data
    //    "Args": queryArgs,       // query argument (e.g. filtering/pagination, etc)
    //}
    //req := r.QueryQ.Format(reqInput)
    reqInput := JsonAtom{
        "RawQuery": tools.CleanString(graphql.GetRequestContext(ctx).RawQuery),
    }
    req := r.RawQueryQ.Format(reqInput)

    /* Send the dgraph request and follow the results */
    // Dgraph request
    res := &GqlRes{} // or new(Res)
    //fmt.Println("request ->", string(req))
    err := r.db.Request([]byte(req), res)
    //fmt.Println("response ->", res)
    if err != nil {
        panic(err)
    } else if res.Errors != nil {
        // see gqlgen doc to returns erros as list.
        e, _ := json.Marshal(res.Errors)
        return fmt.Errorf(string(e))
    }

    config := &mapstructure.DecoderConfig{TagName: "json", Result: data}
    decoder, err := mapstructure.NewDecoder(config)
    decoder.Decode(res.Data[queryName])
    if err != nil {
        panic(err)
    }
    return 
}
