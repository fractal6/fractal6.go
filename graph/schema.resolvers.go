// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.
package graph

import (
	"context"
	"fmt"

	"zerogov/fractal6.go/graph/generated"
	"zerogov/fractal6.go/graph/model"
)

func (r *mutationResolver) UpdateNode(ctx context.Context, input model.UpdateNodeInput) (*model.UpdateNodePayload, error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *mutationResolver) DeleteNode(ctx context.Context, filter model.NodeFilter) (*model.DeleteNodePayload, error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *mutationResolver) AddCircle(ctx context.Context, input []*model.AddCircleInput) (*model.AddCirclePayload, error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *mutationResolver) UpdateCircle(ctx context.Context, input model.UpdateCircleInput) (*model.UpdateCirclePayload, error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *mutationResolver) DeleteCircle(ctx context.Context, filter model.CircleFilter) (*model.DeleteCirclePayload, error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *mutationResolver) AddRole(ctx context.Context, input []*model.AddRoleInput) (*model.AddRolePayload, error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *mutationResolver) UpdateRole(ctx context.Context, input model.UpdateRoleInput) (*model.UpdateRolePayload, error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *mutationResolver) DeleteRole(ctx context.Context, filter model.RoleFilter) (*model.DeleteRolePayload, error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *mutationResolver) UpdatePost(ctx context.Context, input model.UpdatePostInput) (*model.UpdatePostPayload, error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *mutationResolver) DeletePost(ctx context.Context, filter model.PostFilter) (*model.DeletePostPayload, error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *mutationResolver) AddTension(ctx context.Context, input []*model.AddTensionInput) (*model.AddTensionPayload, error) {
	// Format inputs
	var ipts []gql.JsonAtom
	for _, ipt := range input {
		ipts = append(ipts, tools.StructToMap(ipt))
	}

	errors = r.Gqlgen2DgraphMutationResolver(ctx, ipts, &data)
	return data, errors
}

func (r *mutationResolver) UpdateTension(ctx context.Context, input model.UpdateTensionInput) (*model.UpdateTensionPayload, error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *mutationResolver) DeleteTension(ctx context.Context, filter model.TensionFilter) (*model.DeleteTensionPayload, error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *mutationResolver) AddMandate(ctx context.Context, input []*model.AddMandateInput) (*model.AddMandatePayload, error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *mutationResolver) UpdateMandate(ctx context.Context, input model.UpdateMandateInput) (*model.UpdateMandatePayload, error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *mutationResolver) DeleteMandate(ctx context.Context, filter model.MandateFilter) (*model.DeleteMandatePayload, error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *mutationResolver) AddUser(ctx context.Context, input []*model.AddUserInput) (*model.AddUserPayload, error) {
	// Format inputs
	var ipts []gql.JsonAtom
	for _, ipt := range input {
		ipts = append(ipts, tools.StructToMap(ipt))
	}

	errors = r.Gqlgen2DgraphMutationResolver(ctx, ipts, &data)
	return data, errors
}

func (r *mutationResolver) UpdateUser(ctx context.Context, input model.UpdateUserInput) (*model.UpdateUserPayload, error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *mutationResolver) DeleteUser(ctx context.Context, filter model.UserFilter) (*model.DeleteUserPayload, error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *queryResolver) GetNode(ctx context.Context, id *string, nameid *string) (model.Node, error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *queryResolver) QueryNode(ctx context.Context, filter *model.NodeFilter, order *model.NodeOrder, first *int, offset *int) ([]model.Node, error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *queryResolver) GetCircle(ctx context.Context, id *string, nameid *string) (*model.Circle, error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *queryResolver) QueryCircle(ctx context.Context, filter *model.CircleFilter, order *model.CircleOrder, first *int, offset *int) ([]*model.Circle, error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *queryResolver) GetRole(ctx context.Context, id *string, nameid *string) (*model.Role, error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *queryResolver) QueryRole(ctx context.Context, filter *model.RoleFilter, order *model.RoleOrder, first *int, offset *int) ([]*model.Role, error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *queryResolver) GetPost(ctx context.Context, id string) (model.Post, error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *queryResolver) QueryPost(ctx context.Context, filter *model.PostFilter, order *model.PostOrder, first *int, offset *int) ([]model.Post, error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *queryResolver) GetTension(ctx context.Context, id string) (*model.Tension, error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *queryResolver) QueryTension(ctx context.Context, filter *model.TensionFilter, order *model.TensionOrder, first *int, offset *int) ([]*model.Tension, error) {
	errors = r.Gqlgen2DgraphQueryResolver(ctx, &data)
	return data, errors
}

func (r *queryResolver) GetMandate(ctx context.Context, id string) (*model.Mandate, error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *queryResolver) QueryMandate(ctx context.Context, filter *model.MandateFilter, order *model.MandateOrder, first *int, offset *int) ([]*model.Mandate, error) {
	errors = r.Gqlgen2DgraphQueryResolver(ctx, &data)
	return data, errors
}

func (r *queryResolver) GetUser(ctx context.Context, id *string, username *string) (*model.User, error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *queryResolver) QueryUser(ctx context.Context, filter *model.UserFilter, order *model.UserOrder, first *int, offset *int) ([]*model.User, error) {
	errors = r.Gqlgen2DgraphQueryResolver(ctx, &data)
	return data, errors
}

func (r *Resolver) Mutation() generated.MutationResolver { return &mutationResolver{r} }
func (r *Resolver) Query() generated.QueryResolver       { return &queryResolver{r} }

type mutationResolver struct{ *Resolver }
type queryResolver struct{ *Resolver }

