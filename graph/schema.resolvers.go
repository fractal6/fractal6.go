package graph

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"
	"fmt"

	"zerogov/fractal6.go/graph/generated"
	"zerogov/fractal6.go/graph/model"
)

func (r *mutationResolver) AddNode(ctx context.Context, input []*model.AddNodeInput) (data *model.AddNodePayload, errors error) {
	ctx = context.WithValue(ctx, "mutation_context", MutationContext{type_: AddMut, argName: "input"})
	errors = r.Gqlgen2DgraphMutationResolver(ctx, &data, input)
	return data, errors
}

func (r *mutationResolver) UpdateNode(ctx context.Context, input model.UpdateNodeInput) (data *model.UpdateNodePayload, errors error) {
	ctx = context.WithValue(ctx, "mutation_context", MutationContext{type_: UpdateMut, argName: "input"})
	errors = r.Gqlgen2DgraphMutationResolver(ctx, &data, input)
	return data, errors
}

func (r *mutationResolver) DeleteNode(ctx context.Context, filter model.NodeFilter) (data *model.DeleteNodePayload, errors error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *mutationResolver) AddNodeFragment(ctx context.Context, input []*model.AddNodeFragmentInput) (data *model.AddNodeFragmentPayload, errors error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *mutationResolver) AddNodeCharac(ctx context.Context, input []*model.AddNodeCharacInput) (data *model.AddNodeCharacPayload, errors error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *mutationResolver) UpdateNodeCharac(ctx context.Context, input model.UpdateNodeCharacInput) (data *model.UpdateNodeCharacPayload, errors error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *mutationResolver) DeleteNodeCharac(ctx context.Context, filter model.NodeCharacFilter) (data *model.DeleteNodeCharacPayload, errors error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *mutationResolver) AddNodeStats(ctx context.Context, input []*model.AddNodeStatsInput) (data *model.AddNodeStatsPayload, errors error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *mutationResolver) UpdatePost(ctx context.Context, input model.UpdatePostInput) (data *model.UpdatePostPayload, errors error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *mutationResolver) DeletePost(ctx context.Context, filter model.PostFilter) (data *model.DeletePostPayload, errors error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *mutationResolver) AddTension(ctx context.Context, input []*model.AddTensionInput) (data *model.AddTensionPayload, errors error) {
	ctx = context.WithValue(ctx, "mutation_context", MutationContext{type_: AddMut, argName: "input"})
	errors = r.Gqlgen2DgraphMutationResolver(ctx, &data, input)
	return data, errors
}

func (r *mutationResolver) UpdateTension(ctx context.Context, input model.UpdateTensionInput) (data *model.UpdateTensionPayload, errors error) {
	ctx = context.WithValue(ctx, "mutation_context", MutationContext{type_: UpdateMut, argName: "input"})
	errors = r.Gqlgen2DgraphMutationResolver(ctx, &data, input)
	return data, errors
}

func (r *mutationResolver) DeleteTension(ctx context.Context, filter model.TensionFilter) (data *model.DeleteTensionPayload, errors error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *mutationResolver) AddComment(ctx context.Context, input []*model.AddCommentInput) (data *model.AddCommentPayload, errors error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *mutationResolver) UpdateComment(ctx context.Context, input model.UpdateCommentInput) (data *model.UpdateCommentPayload, errors error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *mutationResolver) DeleteComment(ctx context.Context, filter model.CommentFilter) (data *model.DeleteCommentPayload, errors error) {
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
	panic(fmt.Errorf("not implemented"))
}

func (r *mutationResolver) UpdateUser(ctx context.Context, input model.UpdateUserInput) (data *model.UpdateUserPayload, errors error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *mutationResolver) DeleteUser(ctx context.Context, filter model.UserFilter) (data *model.DeleteUserPayload, errors error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *mutationResolver) AddUserRights(ctx context.Context, input []*model.AddUserRightsInput) (data *model.AddUserRightsPayload, errors error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *mutationResolver) AddLabel(ctx context.Context, input []*model.AddLabelInput) (data *model.AddLabelPayload, errors error) {
	panic(fmt.Errorf("not implemented"))
	//ctx = context.WithValue(ctx, "mutation_context", MutationContext{type_: AddMut, argName: "input"})
	//errors = r.Gqlgen2DgraphMutationResolver(ctx, &data, input)
	//return data, errors
}

func (r *mutationResolver) UpdateLabel(ctx context.Context, input model.UpdateLabelInput) (data *model.UpdateLabelPayload, errors error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *mutationResolver) DeleteLabel(ctx context.Context, filter model.LabelFilter) (data *model.DeleteLabelPayload, errors error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *queryResolver) GetNode(ctx context.Context, id *string, nameid *string) (data *model.Node, errors error) {
	errors = r.Gqlgen2DgraphQueryResolver(ctx, &data)
	return data, errors
}

func (r *queryResolver) QueryNode(ctx context.Context, filter *model.NodeFilter, order *model.NodeOrder, first *int, offset *int) (data []*model.Node, errors error) {
	errors = r.Gqlgen2DgraphQueryResolver(ctx, &data)
	return data, errors
}

func (r *queryResolver) QueryNodeFragment(ctx context.Context, order *model.NodeFragmentOrder, first *int, offset *int) (data []*model.NodeFragment, errors error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *queryResolver) GetNodeCharac(ctx context.Context, id string) (data *model.NodeCharac, errors error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *queryResolver) QueryNodeCharac(ctx context.Context, filter *model.NodeCharacFilter, first *int, offset *int) (data []*model.NodeCharac, errors error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *queryResolver) QueryNodeStats(ctx context.Context, order *model.NodeStatsOrder, first *int, offset *int) (data []*model.NodeStats, errors error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *queryResolver) GetPost(ctx context.Context, id string) (data *model.Post, errors error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *queryResolver) QueryPost(ctx context.Context, filter *model.PostFilter, order *model.PostOrder, first *int, offset *int) (data []*model.Post, errors error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *queryResolver) GetTension(ctx context.Context, id string) (data *model.Tension, errors error) {
	errors = r.Gqlgen2DgraphQueryResolver(ctx, &data)
	return data, errors
}

func (r *queryResolver) QueryTension(ctx context.Context, filter *model.TensionFilter, order *model.TensionOrder, first *int, offset *int) (data []*model.Tension, errors error) {
	errors = r.Gqlgen2DgraphQueryResolver(ctx, &data)
	return data, errors
}

func (r *queryResolver) GetComment(ctx context.Context, id string) (data *model.Comment, errors error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *queryResolver) QueryComment(ctx context.Context, filter *model.CommentFilter, order *model.CommentOrder, first *int, offset *int) (data []*model.Comment, errors error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *queryResolver) GetMandate(ctx context.Context, id string) (data *model.Mandate, errors error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *queryResolver) QueryMandate(ctx context.Context, filter *model.MandateFilter, order *model.MandateOrder, first *int, offset *int) (data []*model.Mandate, errors error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *queryResolver) GetUser(ctx context.Context, id *string, username *string) (data *model.User, errors error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *queryResolver) QueryUser(ctx context.Context, filter *model.UserFilter, order *model.UserOrder, first *int, offset *int) (data []*model.User, errors error) {
	errors = r.Gqlgen2DgraphQueryResolver(ctx, &data)
	return data, errors
}

func (r *queryResolver) QueryUserRights(ctx context.Context, first *int, offset *int) (data []*model.UserRights, errors error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *queryResolver) GetLabel(ctx context.Context, id *string, name *string) (data *model.Label, errors error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *queryResolver) QueryLabel(ctx context.Context, filter *model.LabelFilter, order *model.LabelOrder, first *int, offset *int) (data []*model.Label, errors error) {
	errors = r.Gqlgen2DgraphQueryResolver(ctx, &data)
	return data, errors
}

// Mutation returns generated.MutationResolver implementation.
func (r *Resolver) Mutation() generated.MutationResolver { return &mutationResolver{r} }

// Query returns generated.QueryResolver implementation.
func (r *Resolver) Query() generated.QueryResolver { return &queryResolver{r} }

type mutationResolver struct{ *Resolver }
type queryResolver struct{ *Resolver }
