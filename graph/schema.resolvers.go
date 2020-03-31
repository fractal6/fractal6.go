// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.
package graph

import (
	"context"
	"fmt"

	"zerogov/fractal6.go/graph/generated"
	"zerogov/fractal6.go/graph/model"
	"zerogov/fractal6.go/tools"
	"zerogov/fractal6.go/tools/gql"
)

func (r *mutationResolver) UpdateNode(ctx context.Context, input model.UpdateNodeInput) (data *model.UpdateNodePayload, errors error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *mutationResolver) DeleteNode(ctx context.Context, filter model.NodeFilter) (data *model.DeleteNodePayload, errors error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *mutationResolver) AddCircle(ctx context.Context, input []*model.AddCircleInput) (data *model.AddCirclePayload, errors error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *mutationResolver) UpdateCircle(ctx context.Context, input model.UpdateCircleInput) (data *model.UpdateCirclePayload, errors error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *mutationResolver) DeleteCircle(ctx context.Context, filter model.CircleFilter) (data *model.DeleteCirclePayload, errors error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *mutationResolver) AddRole(ctx context.Context, input []*model.AddRoleInput) (data *model.AddRolePayload, errors error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *mutationResolver) UpdateRole(ctx context.Context, input model.UpdateRoleInput) (data *model.UpdateRolePayload, errors error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *mutationResolver) DeleteRole(ctx context.Context, filter model.RoleFilter) (data *model.DeleteRolePayload, errors error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *mutationResolver) UpdatePost(ctx context.Context, input model.UpdatePostInput) (data *model.UpdatePostPayload, errors error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *mutationResolver) DeletePost(ctx context.Context, filter model.PostFilter) (data *model.DeletePostPayload, errors error) {
	panic(fmt.Errorf("not implemented"))
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

func (r *mutationResolver) UpdateTension(ctx context.Context, input model.UpdateTensionInput) (data *model.UpdateTensionPayload, errors error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *mutationResolver) DeleteTension(ctx context.Context, filter model.TensionFilter) (data *model.DeleteTensionPayload, errors error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *mutationResolver) AddMandate(ctx context.Context, input []*model.AddMandateInput) (data *model.AddMandatePayload, errors error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *mutationResolver) UpdateMandate(ctx context.Context, input model.UpdateMandateInput) (data *model.UpdateMandatePayload, errors error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *mutationResolver) DeleteMandate(ctx context.Context, filter model.MandateFilter) (data *model.DeleteMandatePayload, errors error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *mutationResolver) AddUser(ctx context.Context, input []*model.AddUserInput) (data *model.AddUserPayload, errors error) {
	// Format inputs
	var ipts []gql.JsonAtom
	for _, ipt := range input {
		ipts = append(ipts, tools.StructToMap(ipt))
	}

	errors = r.Gqlgen2DgraphMutationResolver(ctx, ipts, &data)
	return data, errors
}

func (r *mutationResolver) UpdateUser(ctx context.Context, input model.UpdateUserInput) (data *model.UpdateUserPayload, errors error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *mutationResolver) DeleteUser(ctx context.Context, filter model.UserFilter) (data *model.DeleteUserPayload, errors error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *queryResolver) GetNode(ctx context.Context, id *string, nameid *string) (data model.Node, errors error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *queryResolver) QueryNode(ctx context.Context, filter *model.NodeFilter, order *model.NodeOrder, first *int, offset *int) (data []model.Node, errors error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *queryResolver) GetCircle(ctx context.Context, id *string, nameid *string) (data *model.Circle, errors error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *queryResolver) QueryCircle(ctx context.Context, filter *model.CircleFilter, order *model.CircleOrder, first *int, offset *int) (data []*model.Circle, errors error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *queryResolver) GetRole(ctx context.Context, id *string, nameid *string) (data *model.Role, errors error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *queryResolver) QueryRole(ctx context.Context, filter *model.RoleFilter, order *model.RoleOrder, first *int, offset *int) (data []*model.Role, errors error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *queryResolver) GetPost(ctx context.Context, id string) (data model.Post, errors error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *queryResolver) QueryPost(ctx context.Context, filter *model.PostFilter, order *model.PostOrder, first *int, offset *int) (data []model.Post, errors error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *queryResolver) GetTension(ctx context.Context, id string) (data *model.Tension, errors error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *queryResolver) QueryTension(ctx context.Context, filter *model.TensionFilter, order *model.TensionOrder, first *int, offset *int) (data []*model.Tension, errors error) {
	errors = r.Gqlgen2DgraphQueryResolver(ctx, &data)
	return data, errors
}

func (r *queryResolver) GetMandate(ctx context.Context, id string) (data *model.Mandate, errors error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *queryResolver) QueryMandate(ctx context.Context, filter *model.MandateFilter, order *model.MandateOrder, first *int, offset *int) (data []*model.Mandate, errors error) {
	errors = r.Gqlgen2DgraphQueryResolver(ctx, &data)
	return data, errors
}

func (r *queryResolver) GetUser(ctx context.Context, id *string, username *string) (data *model.User, errors error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *queryResolver) QueryUser(ctx context.Context, filter *model.UserFilter, order *model.UserOrder, first *int, offset *int) (data []*model.User, errors error) {
	errors = r.Gqlgen2DgraphQueryResolver(ctx, &data)
	return data, errors
}

func (r *Resolver) Mutation() generated.MutationResolver { return &mutationResolver{r} }
func (r *Resolver) Query() generated.QueryResolver       { return &queryResolver{r} }

type mutationResolver struct{ *Resolver }
type queryResolver struct{ *Resolver }

