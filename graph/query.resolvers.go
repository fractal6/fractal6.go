// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.
package graph

import (
	"context"
	"fmt"

	"zerogov/fractal6.go/graph/generated"
	"zerogov/fractal6.go/graph/model"
)

func (r *mutationResolver) AddUser(ctx context.Context, input []*model.AddUserInput) (*model.AddUserPayload, error) {

    body := ctx.Value("body").([]byte)
    res := r.db.Request(body)

    fmt.Println(string(body))
    fmt.Println(string(res))

    var users []*model.User
	for _, ipt := range input {
        user := &model.User{
            Username: ipt.Username, 
            Password: ipt.Password,
        }
        users = append(users, user)
	}

    up := &model.AddUserPayload{
        User: users,
    }

    return up, nil
}

func (r *mutationResolver) AddTension(ctx context.Context, input []*model.AddTensionInput) (*model.AddTensionPayload, error) {
	//fmt.Println(input.Title)
	panic(fmt.Errorf("not implemented"))
}

func (r *queryResolver) QueryUser(ctx context.Context) ([]*model.User, error) {
	fmt.Println(ctx)
	panic(fmt.Errorf("not implemented"))
}

func (r *queryResolver) QueryMandate(ctx context.Context) ([]*model.Mandate, error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *queryResolver) QueryTension(ctx context.Context) ([]*model.Tension, error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *Resolver) Mutation() generated.MutationResolver { return &mutationResolver{r} }
func (r *Resolver) Query() generated.QueryResolver       { return &queryResolver{r} }

type mutationResolver struct{ *Resolver }
type queryResolver struct{ *Resolver }
