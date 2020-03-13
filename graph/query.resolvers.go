// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.
package graph

import (
	"context"
	"fmt"
    "strings"

	"zerogov/fractal6.go/graph/generated"
	"zerogov/fractal6.go/graph/model"
)

type JsonAtom map[string]interface{}

type GqlResponse struct {
	Adduser model.AddUserPayload `json:"addUser"`
}
type Res struct {
	Data GqlResponse `json:"data"`
	Errors []JsonAtom `json:"errors"` // message, locations, path
}

func (r *mutationResolver) AddUser(ctx context.Context, input []model.AddUserInput) (*model.AddUserPayload, error) {



	body := ctx.Value("request_body").([]byte)
	res := &Res{} // or new(Res)
	err := r.db.Request(body, res)
	if err != nil {
		panic(err)
	}
	//fmt.Println(string(body))
	//fmt.Println(res)

	data := res.Data
    d := &data.Adduser
	errors := res.Errors

	if errors != nil {
        var msg []string
        for _, m := range res.Errors {
            msg = append(msg, m["message"].(string))
        }
		err := fmt.Errorf("%s", strings.Join(msg, " | "))
        return d, err
    }

	return d, nil
}

func (r *mutationResolver) AddTension(ctx context.Context, input []model.AddTensionInput) (*model.AddTensionPayload, error) {
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

