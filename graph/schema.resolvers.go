// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.
package graph

import (
	"context"
	"fmt"
	"math/rand"

	"zerogov/fractal6.go/graph/generated"
	"zerogov/fractal6.go/graph/model"
)

func (r *mutationResolver) CreateTodo(ctx context.Context, input model.NewTodo) (*model.Todo, error) {
	todo := &model.Todo{
		Text:   input.Text,
		ID:     fmt.Sprintf("T%d", rand.Int()),
		UserId: input.UserID, // fix this line
		//User: &model.User{ID: input.UserID, Name: "user " + input.UserID},
	}
	r.todos = append(r.todos, todo)
	return todo, nil
}

func (r *mutationResolver) SignUp(ctx context.Context, input model.InputCred) (*model.Cred, error) {

	//c, err := internal.RouterContextFromContext(ctx)
    //fmt.Println(c)
    fmt.Println(ctx)
    cred, err := getCred(ctx, input)
    if err != nil {
        fmt.Println(err, "dfzfefz ")
        return nil, err
    }

    return &cred, nil
}

func (r *queryResolver) Todos(ctx context.Context) ([]*model.Todo, error) {
	return r.todos, nil
}

func (r *todoResolver) Nth(ctx context.Context, obj *model.Todo) (*int, error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *todoResolver) User(ctx context.Context, obj *model.Todo) (*model.User, error) {
	n := "user " + obj.UserId
	return &model.User{ID: obj.UserId, Name: &n}, nil
}

func (r *Resolver) Mutation() generated.MutationResolver { return &mutationResolver{r} }
func (r *Resolver) Query() generated.QueryResolver       { return &queryResolver{r} }
func (r *Resolver) Todo() generated.TodoResolver         { return &todoResolver{r} }

type mutationResolver struct{ *Resolver }
type queryResolver struct{ *Resolver }
type todoResolver struct{ *Resolver }
