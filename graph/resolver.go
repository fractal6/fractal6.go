//go:generate go run github.com/99designs/gqlgen -v

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.
package graph

import (
    "fmt"
    "context"
    "github.com/spf13/viper"
    "github.com/99designs/gqlgen/graphql"
    //"golang.org/x/crypto/bcrypt" 

    "zerogov/fractal6.go/tools"
    "zerogov/fractal6.go/tools/gql"
    "zerogov/fractal6.go/graph/model"
    gen "zerogov/fractal6.go/graph/generated"
    //"golang.org/x/crypto/bcrypt" 
)

/*
*
* Data structures initialisation
*
*/

type Resolver struct{
    // I/O objects
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

    // Dgraph directives
    c := gen.Config{Resolvers: &r}
    c.Directives.Id = nothing
    c.Directives.HasInverse = nothing2
    c.Directives.Search = nothing3

    // User defined directives
    c.Directives.MaxLength = maxLength
    //c.Directives.HasRole = hasRoleMiddleware
    return c
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

func maxLength (ctx context.Context, obj interface{}, next graphql.Resolver, max *int) (interface{}, error) {
    rc := graphql.GetResolverContext(ctx)
    fmt.Println(rc.Field.Name)
    fmt.Println(obj.(*model.User).Username)
    fmt.Println(*max)
    return next(ctx)
}

//func hasRoleMiddleware (ctx context.Context, obj interface{}, next graphql.Resolver, role model.Role) (interface{}, error) {
//
//    fmt.Println(ctx)
//    //if !getCurrentUser(ctx).HasRole(role) {
//    //    // block calling the next resolver
//    //     fmt.Println(ctx)
//    //    return nil, fmt.Errorf("Access denied")
//    //}
//
//    // or let it pass through
//    return next(ctx)
//}
//
//
//// HashPassword generates a hash using the bcrypt.GenerateFromPassword
//func HashPassword(password string) string {
//    hash, err := bcrypt.GenerateFromPassword([]byte(password), 10)
//    if err != nil {
//        panic(err)
//    }
//
//    return string(hash)
//}
//
//// ComparePassword compares the hash
//func ComparePassword(hash string, password string) bool {
//
//    if len(password) == 0 || len(hash) == 0 {
//        return false
//    }
//
//    err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
//    return err == nil
//}
//
//func getCred(ctx context.Context, input model.InputCred) (model.Cred, error) {
//    //cred := new(model.Cred)
//    //if err := ctx.Bind(cred); err != nil {
//    //    return nil, &echo.HTTPError{
//    //        Code: http.StatusBadRequest,
//    //        Message: "invalid email or password"
//    //    }
//    //}
//
//    //hashedPassword = HashPassword(cred.Password)
//    hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(input.Password), 10)
//    cred := model.Cred{input.Username, string(hashedPassword)}
//    return cred, nil
//}
//
