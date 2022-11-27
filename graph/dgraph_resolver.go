/*
 * Fractale - Self-organisation for humans.
 * Copyright (C) 2022 Fractale Co
 *
 * This file is part of Fractale.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as
 * published by the Free Software Foundation, either version 3 of the
 * License, or (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with Fractale.  If not, see <http://www.gnu.org/licenses/>.
 */

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
    "fractale/fractal6.go/web/auth"
)

var cache *sessions.Session

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
    gc := graphql.GetRequestContext(ctx)
    queryType, typeName, queryName, err := queryTypeFromGraphqlContext(ctx)
    if err != nil { return tools.LogErr("DgraphQueryResolver", err) }

    // Return error if jwt token error (particurly when has expired)
    if queryType == "add" || queryType == "update" || queryType == "delete" {
        _, _, err := auth.GetUserContext(ctx)
        if err != nil { return tools.LogErr("Access denied", err) }
    }

    // Remove some input
    rawQuery := gc.RawQuery
    variables := gc.Variables
    if ctx.Value("cut_history") != nil {
        // Go along PushHistory...
        // improve that hack with gqlgen #1144 issue
        // lazy (non-greedy) matching
        reg := regexp.MustCompile(`,?\s*history\s*:\s*\[("[^"]*"|[^\]])*?\]`)
        // If we remove completely history, it cause some "no data" box error on the frontend.
        rawQuery = reg.ReplaceAllString(rawQuery, "history:[]")

        // If Graphql variables are given...
        t := strings.ToLower(typeName)
        if variables[t] != nil && variables[t].(map[string]interface{})["set"] != nil {
            s := variables[t].(map[string]interface{})["set"].(map[string]interface{})
            s["history"] = nil
        }
    }

    variables_, _ := json.Marshal(variables)
    reqInput := map[string]string{
        "QueryName": queryName,
        // @warning: CleanString will lose format for text and mardown text data.
        //"RawQuery": tools.CleanString(gc.RawQuery, true),
        "RawQuery": tools.QuoteString(rawQuery),
        "Variables": string(variables_),
    }

    // Send request
    uctx := auth.GetUserContextOrEmpty(ctx)
    err = db.QueryGql(uctx, "rawQuery", reqInput, data)
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
        fmt.Println("Dgraph Error Ignored: ", err.Error())
        return nil
    } else if err != nil || data == nil {
        return err
    }

    // Post processing (@meta_patch) / Post hook operation.
    // If the query go trough the validation stack, execute.
    //if f, _ := cache.Do("GETDEL", uctx.Username + "meta_patch_f"); f != nil {
    //    k, _ := cache.Do("GETDEL", uctx.Username + "meta_patch_k")
    //    v, _ := cache.Do("GETDEL", uctx.Username + "meta_patch_v")
    //    maps := map[string]string{fmt.Sprintf("%s", k): fmt.Sprintf("%s", v)}
    //    db.Meta(fmt.Sprintf("%s", f), maps)
    //}
    if f, err := cache.GetDel(ctx, uctx.Username + "meta_patch_f").Result(); f != "" && err == nil {
        k, _ := cache.GetDel(ctx, uctx.Username + "meta_patch_k").Result()
        v, _ := cache.GetDel(ctx, uctx.Username + "meta_patch_v").Result()
        maps := map[string]string{k: v}
        db.Meta(f, maps)
    } else if err != nil {
        // ignore "redis: nil" error as it is always return because we call Result() ..
        //fmt.Println("Redis error: ", err)
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
//    queryGraph := GetQueryGraph(ctx)
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
//    uctx := auth.GetUserContextOrEmpty(ctx)
//    err = db.QueryGql(uctx, op, reqInput, data)
//    return err
//}
