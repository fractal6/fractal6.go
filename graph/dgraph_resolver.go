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

/* Raw bridges pass the raw query from the request context to Dgraph.
 * @Warning: It looses transformation that eventually happen in the resolvers/directives.
 */

func (r *queryResolver) DgraphBridgeRaw(ctx context.Context, data interface{}) error {
    err := DgraphQueryResolverRaw(ctx, r.db, data)
    return postGqlProcess(ctx, r.db, data, err)
}

func (r *mutationResolver) DgraphBridgeRaw(ctx context.Context, data interface{}) error {
    err := DgraphQueryResolverRaw(ctx, r.db, data)
    return postGqlProcess(ctx, r.db, data, err)
}

/* Those bridges rebuild the query from the request context preloads, and uses the input
 * parameters from the gqlgen resolvers whuch reflext the modifications in the resolvers/directives.
 * @Warning: It looses eventual directive in the query graph (@cascade, @skip etc)
 */

func (r *queryResolver) DgraphGetBridge(ctx context.Context, maps map[string]interface{}, data interface{}) error {
	panic(fmt.Errorf("not implemented"))
}

func (r *queryResolver) DgraphQueryBridge(ctx context.Context, maps map[string]interface{}, data interface{}) error {
	panic(fmt.Errorf("not implemented"))
}

func (r *mutationResolver) DgraphAddBridge(ctx context.Context, input interface{}, upsert *bool, data interface{}) error {
    err := DgraphAddResolver(ctx, r.db, input, upsert, data)
    return postGqlProcess(ctx, r.db, data, err)
}

func (r *mutationResolver) DgraphUpdateBridge(ctx context.Context, input interface{}, data interface{}) error {
    err := DgraphUpdateResolver(ctx, r.db, input, data)
    return postGqlProcess(ctx, r.db, data, err)
}

func (r *mutationResolver) DgraphDeleteBridge(ctx context.Context, filter interface{}, data interface{}) error {
    err := DgraphDeleteResolver(ctx, r.db, filter, data)
    return postGqlProcess(ctx, r.db, data, err)
}

/*
*
* Dgraph-Gqlgen bridge logic
*
*/

func DgraphAddResolver(ctx context.Context, db *db.Dgraph, input interface{}, upsert *bool,  data interface{}) error {
	_, uctx, err := auth.GetUserContext(ctx)
	if err != nil { return tools.LogErr("Access denied", err) }

    _, typeName, _, err := queryTypeFromGraphqlContext(ctx)
    if err != nil { return tools.LogErr("DgraphQueryResolver", err) }

    err = db.AddExtra(*uctx, typeName, input, upsert, GetQueryGraph(ctx), data)
	return err
}

func DgraphUpdateResolver(ctx context.Context, db *db.Dgraph, input interface{}, data interface{}) error {
	_, uctx, err := auth.GetUserContext(ctx)
	if err != nil { return tools.LogErr("Access denied", err) }

    _, typeName, _, err := queryTypeFromGraphqlContext(ctx)
    if err != nil { return tools.LogErr("DgraphQueryResolver", err) }

    err = db.UpdateExtra(*uctx, typeName, input, GetQueryGraph(ctx), data)
	return err
}

func DgraphDeleteResolver(ctx context.Context, db *db.Dgraph, input interface{}, data interface{}) error {
	_, uctx, err := auth.GetUserContext(ctx)
	if err != nil { return tools.LogErr("Access denied", err) }

    _, typeName, _, err := queryTypeFromGraphqlContext(ctx)
    if err != nil { return tools.LogErr("DgraphQueryResolver", err) }

    err = db.DeleteExtra(*uctx, typeName, input, GetQueryGraph(ctx), data)
	return err
}

// @deprecated: Follow the Gql request to Dgraph.
// This use raw query from the request context and thus won't propage change
// of the input that may happend in the resolvers.
func DgraphQueryResolverRaw(ctx context.Context, db *db.Dgraph, data interface{}) error {
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
        t := strings.ToLower(typeName) // @DEBUG: only the first letter must lowered ?!
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
    return err
}

func postGqlProcess(ctx context.Context, db *db.Dgraph, data interface{}, errors error) error {
    if data != nil && errors != nil {
        // Gqlgen ignore the data if there is an error returned
        // see https://github.com/99designs/gqlgen/issues/1191
        //graphql.AddErrorf(ctx, errors.Error())

        // Nodes query can return null field if Node are hidden
        // but children are not. The source ends up to be a tension where
        // the receiver is the parent wich is hidden ;)
        //
        d, _ := json.Marshal(data)
        if (string(d) == "null") {
            // If there is really no data, show the graphql error
            // otherwise, fail silently.
            return errors
        }
        fmt.Println("Dgraph Error Ignored: ", errors.Error())
        return nil
    } else if errors != nil || data == nil {
        return errors
    }

    uctx := auth.GetUserContextOrEmpty(ctx)
    if uctx.Username == "" { return errors }
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

    return errors
}
