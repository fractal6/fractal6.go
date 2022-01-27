package graph

import (
    "fmt"
	"context"
    "regexp"
    "strings"
	"encoding/json"
	"github.com/99designs/gqlgen/graphql"

	"fractale/fractal6.go/db"
	"fractale/fractal6.go/tools"
	"fractale/fractal6.go/web/sessions"
    webauth "fractale/fractal6.go/web/auth"
)

var cache sessions.Session

func init() {
    cache = sessions.GetCache()
}


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
    gc := graphql.GetRequestContext(ctx)
    variables, _ := json.Marshal(gc.Variables)

    // Return error if jwt token error (particurly when has expired)
    if strings.HasPrefix(queryName, "update") || strings.HasPrefix(queryName, "delete") {
        _, err := webauth.GetUserContext(ctx)
        if err != nil { return tools.LogErr("Access denied", err) }
    }

    // Remove some input
    rawQuery := gc.RawQuery
    if ctx.Value("cut_history") != nil {
        // Go along PushHistory...
        reg := regexp.MustCompile(`,?\s*history\s*:\s*\[[^\]]*\]`)
        // If we remove completely history, it cause some "no data" box error on the frontend.
        rawQuery = reg.ReplaceAllString(rawQuery, "history:[]")
    }

    reqInput := map[string]string{
        "QueryName": queryName,
        // Warning: CleanString will lose format for text and mardown text data.
        //"RawQuery": tools.CleanString(gc.RawQuery, true),
        "RawQuery": tools.QuoteString(rawQuery),
        "Variables": string(variables),
    }

    // Send request
    uctx := webauth.GetUserContextOrEmpty(ctx)
    err := db.QueryGql(uctx, "rawQuery", reqInput, data)
    if data != nil && err != nil {
        // Gqlgen ignore the data if there is an error returned
        // see https://github.com/99designs/gqlgen/issues/1191
        //graphql.AddErrorf(ctx, err.Error())

        // Nodes query can return null field if Node are hidden
        // but children are not. The source ends up to be a tension where
        // the receiver is the parent wich is hidden ;)
        //
        d, _ := json.Marshal(data)
        if (string(d) == "null") {
            // If there is really no data, show the graphql error
            // otherwise, fail silently.
            return err
        }
        return nil
    } else if err != nil || data == nil {
        return err
    }

    // Post processing (@meta_patch)
    if f, _ := cache.Do("GETDEL", uctx.Username + "meta_patch_f"); f != nil {
        k, _ := cache.Do("GETDEL", uctx.Username + "meta_patch_k")
        v, _ := cache.Do("GETDEL", uctx.Username + "meta_patch_v")
        kk := fmt.Sprintf("%s", k)
        db.Meta(fmt.Sprintf("%s", f), fmt.Sprintf("%s", v), &kk)
    }

    return err
}

//// Mutation type Enum
//type mutationType string
//const (
//    AddMut mutationType = "add"
//    UpdateMut mutationType = "update"
//    DelMut mutationType = "delete"
//)
//type MutationContext struct  {
//    type_ mutationType
//    argName string
//}


//// @Debug: GetPreloads loose subfilter in payload(in QueryGraph)
//func DgraphQueryResolver(ctx context.Context, ipts interface{}, data interface{}, db *db.Dgraph) error {
//    mutCtx := ctx.Value("mutation_context").(MutationContext)
//
//    /* Rebuild the Graphql inputs request from this context */
//    rc := graphql.GetResolverContext(ctx)
//    queryName := rc.Field.Name
//
//    // Format inputs
//    inputs, _ := json.Marshal(ipts)
//    // If inputs needs to get modified, see tools.StructToMap() usage
//    // in order to to get the struct in the schema.resolver caller.
//
//    // Format collected fields
//    inputType := strings.Split(fmt.Sprintf("%T", rc.Args[mutCtx.argName]), ".")[1]
//    queryGraph := strings.Join(GetPreloads(ctx), " ")
//
//    // Build the graphql raw request
//    reqInput := map[string]string{
//        "QueryName": queryName, // function name (e.g addUser)
//        "InputType": inputType, // input type name (e.g AddUserInput)
//        "QueryGraph": queryGraph, // output data
//        "InputPayload": string(inputs), // inputs data
//    }
//
//    op := string(mutCtx.type_)
//
//    // Send request
//    uctx := webauth.GetUserContextOrEmpty(ctx)
//    err = db.QueryGql(uctx, op, reqInput, data)
//    return err
//}
