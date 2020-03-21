//go:generate go run github.com/99designs/gqlgen -v

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.
package graph

import (
    "fmt"
    "context"
	"strings"
    "encoding/json"
    "github.com/mitchellh/mapstructure"
    "github.com/spf13/viper"
    "github.com/99designs/gqlgen/graphql"
    //"golang.org/x/crypto/bcrypt" 

    //"zerogov/fractal6.go/graph/model"
	"zerogov/fractal6.go/tools"
	"zerogov/fractal6.go/tools/gql"
    "zerogov/fractal6.go/graph/model"
    gen "zerogov/fractal6.go/graph/generated"
)

/*
*
* Data structures initialisation
*
*/

type Resolver struct{
    MutationQ gql.Query
    QueryQ gql.Query
    // pointer on dgraph
    db tools.Dgraph
}

// Init initialize shema config and Directives...
func Init() gen.Config {
    var MutationQ, QueryQ gql.Query
    HOSTDB := viper.GetString("db.host")
    PORTDB := viper.GetString("db.port")
    APIDB := viper.GetString("db.api")
    dgraphApiUrl := "http://"+HOSTDB+":"+PORTDB+"/"+APIDB

    MutationQ.Data = `{
        "query": "mutation {{.QueryName}}($input:[{{.InputType}}!]!) { 
            {{.QueryName}}( input: $input) {
                {{.QueryGraph}}
            } 
        }",
        "variables": {
            "input": {{.InputPayload}}
        }
    }`
    MutationQ.Init()

    QueryQ.Data = `{
        "query": "query {{.QueryName}} { 
            {{.QueryName}} {
                {{.QueryGraph}}
            } 
        }"
    }`
    QueryQ.Init()


    r := Resolver{
        db:tools.Dgraph{
            Url: dgraphApiUrl,
        },
        QueryQ: QueryQ,
        MutationQ: MutationQ,
    }

    c := gen.Config{Resolvers: &r}
    //c.Directives.HasRole = hasRoleMiddleware
    c.Directives.Id = nothing
    c.Directives.HasInverse = nothing2
    c.Directives.Search = nothing3
    return c
}

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

    err = mapstructure.Decode(res.Data[queryName], data)
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

    err = mapstructure.Decode(res.Data[queryName], data)
    if err != nil {
        panic(err)
    }
    return 
}

/*
*
* Business logic layer methods
*
*/

func nothing (ctx context.Context, obj interface{}, next graphql.Resolver) (interface{}, error) {
    return next(ctx)
}

func nothing2 (ctx context.Context, obj interface{}, next graphql.Resolver, key string) (interface{}, error) {
    return next(ctx)
}

func nothing3 (ctx context.Context, obj interface{}, next graphql.Resolver, idx []model.DgraphIndex) (interface{}, error) {
    return next(ctx)
}
