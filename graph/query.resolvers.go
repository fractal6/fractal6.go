// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.
package graph

import (
	//"fmt"
	"context"

	"zerogov/fractal6.go/tools"
	"zerogov/fractal6.go/tools/gql"
	"zerogov/fractal6.go/graph/model"
	"zerogov/fractal6.go/graph/generated"
)


func (r *mutationResolver) AddUser(ctx context.Context, input []*model.AddUserInput) (data *model.AddUserPayload, errors error) {
    // Format inputs
    var ipts []gql.JsonAtom
    for _, ipt := range input {
        ipts = append(ipts, tools.StructToMap(ipt))
    }

    errors = r.Gqlgen2DgraphMutationResolver(ctx, ipts, &data)
	return data, errors
}

func (r *mutationResolver) AddTension(ctx context.Context, input []*model.AddTensionInput) (data *model.AddTensionPayload, errors error) {
    // Format inputs
    var ipts []gql.JsonAtom
    for _, ipt := range input {
        ipts = append(ipts, tools.StructToMap(ipt))
    }

    errors = r.Gqlgen2DgraphMutationResolver(ctx, ipts, &data)
	return data, errors
}

func (r *queryResolver) QueryUser(ctx context.Context) (data []*model.User, errors  error) {
    errors = r.Gqlgen2DgraphQueryResolver(ctx, &data)
	return data, errors
}

func (r *queryResolver) QueryMandate(ctx context.Context) (data []*model.Mandate, errors error) {
    errors = r.Gqlgen2DgraphQueryResolver(ctx, &data)
	return data, errors
}

func (r *queryResolver) QueryTension(ctx context.Context) (data []*model.Tension, errors error) {
    errors = r.Gqlgen2DgraphQueryResolver(ctx, &data)
	return data, errors
}

func (r *Resolver) Mutation() generated.MutationResolver { return &mutationResolver{r} }
func (r *Resolver) Query() generated.QueryResolver       { return &queryResolver{r} }

type mutationResolver struct{ *Resolver }
type queryResolver struct{ *Resolver }

