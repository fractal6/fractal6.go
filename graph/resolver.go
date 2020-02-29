//go:generate go run github.com/99designs/gqlgen -v

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.
package graph

import (
    "fmt"
    "context"
    "github.com/99designs/gqlgen/graphql"
    "golang.org/x/crypto/bcrypt" 

    "zerogov/fractal6.go/graph/model"
    gen "zerogov/fractal6.go/graph/generated"
)

// Resolver is the interface to Dgraph backend
type Resolver struct {
    count int
    todos []*model.Todo
}

// Init initialize shema config and Directives...
func Init() gen.Config {
    c := gen.Config{Resolvers: &Resolver{}}
    c.Directives.HasRole = hasRoleMiddleware
    return c
}

/*
*
* Business Logic layer methods
*
*/

func hasRoleMiddleware (ctx context.Context, obj interface{}, next graphql.Resolver, role model.Role) (interface{}, error) {

    fmt.Println(ctx)
    //if !getCurrentUser(ctx).HasRole(role) {
    //    // block calling the next resolver
    //     fmt.Println(ctx)
    //    return nil, fmt.Errorf("Access denied")
    //}

    // or let it pass through
    return next(ctx)
}


// HashPassword generates a hash using the bcrypt.GenerateFromPassword
func HashPassword(password string) string {
    hash, err := bcrypt.GenerateFromPassword([]byte(password), 10)
    if err != nil {
        panic(err)
    }

    return string(hash)
}

// ComparePassword compares the hash
func ComparePassword(hash string, password string) bool {

    if len(password) == 0 || len(hash) == 0 {
        return false
    }

    err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
    return err == nil
}

func getCred(ctx context.Context, input model.InputCred) (model.Cred, error) {
    //cred := new(model.Cred)
    //if err := ctx.Bind(cred); err != nil {
    //    return nil, &echo.HTTPError{
    //        Code: http.StatusBadRequest,
    //        Message: "invalid email or password"
    //    }
    //}

    //hashedPassword = HashPassword(cred.Password)
    hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(input.Password), 10)
    cred := model.Cred{input.Username, string(hashedPassword)}
    return cred, nil
}

