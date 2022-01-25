package graph

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"
	"fmt"
	"zerogov/fractal6.go/graph/generated"
	"zerogov/fractal6.go/graph/model"
)

func (r *mutationResolver) AddNode(ctx context.Context, input []*model.AddNodeInput, upsert *bool) (data *model.AddNodePayload, errors error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *mutationResolver) UpdateNode(ctx context.Context, input model.UpdateNodeInput) (data *model.UpdateNodePayload, errors error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *mutationResolver) DeleteNode(ctx context.Context, filter model.NodeFilter) (data *model.DeleteNodePayload, errors error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *mutationResolver) AddNodeFragment(ctx context.Context, input []*model.AddNodeFragmentInput) (data *model.AddNodeFragmentPayload, errors error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *mutationResolver) UpdateNodeFragment(ctx context.Context, input model.UpdateNodeFragmentInput) (data *model.UpdateNodeFragmentPayload, errors error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *mutationResolver) DeleteNodeFragment(ctx context.Context, filter model.NodeFragmentFilter) (data *model.DeleteNodeFragmentPayload, errors error) {
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

func (r *mutationResolver) AddLabel(ctx context.Context, input []*model.AddLabelInput) (data *model.AddLabelPayload, errors error) {
	errors = r.Gqlgen2DgraphQueryResolver(ctx, &data)
	return data, errors
}

func (r *mutationResolver) UpdateLabel(ctx context.Context, input model.UpdateLabelInput) (data *model.UpdateLabelPayload, errors error) {
	errors = r.Gqlgen2DgraphQueryResolver(ctx, &data)
	return data, errors
}

func (r *mutationResolver) DeleteLabel(ctx context.Context, filter model.LabelFilter) (data *model.DeleteLabelPayload, errors error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *mutationResolver) AddRoleExt(ctx context.Context, input []*model.AddRoleExtInput) (data *model.AddRoleExtPayload, errors error) {
	errors = r.Gqlgen2DgraphQueryResolver(ctx, &data)
	return data, errors
}

func (r *mutationResolver) UpdateRoleExt(ctx context.Context, input model.UpdateRoleExtInput) (data *model.UpdateRoleExtPayload, errors error) {
	errors = r.Gqlgen2DgraphQueryResolver(ctx, &data)
	return data, errors
}

func (r *mutationResolver) DeleteRoleExt(ctx context.Context, filter model.RoleExtFilter) (data *model.DeleteRoleExtPayload, errors error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *mutationResolver) AddOrgaAgg(ctx context.Context, input []*model.AddOrgaAggInput) (data *model.AddOrgaAggPayload, errors error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *mutationResolver) UpdateOrgaAgg(ctx context.Context, input model.UpdateOrgaAggInput) (data *model.UpdateOrgaAggPayload, errors error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *mutationResolver) DeleteOrgaAgg(ctx context.Context, filter model.OrgaAggFilter) (data *model.DeleteOrgaAggPayload, errors error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *mutationResolver) UpdatePost(ctx context.Context, input model.UpdatePostInput) (data *model.UpdatePostPayload, errors error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *mutationResolver) DeletePost(ctx context.Context, filter model.PostFilter) (data *model.DeletePostPayload, errors error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *mutationResolver) AddTension(ctx context.Context, input []*model.AddTensionInput) (data *model.AddTensionPayload, errors error) {
	errors = r.Gqlgen2DgraphQueryResolver(ctx, &data)
	return data, errors
}

func (r *mutationResolver) UpdateTension(ctx context.Context, input model.UpdateTensionInput) (data *model.UpdateTensionPayload, errors error) {
	errors = r.Gqlgen2DgraphQueryResolver(ctx, &data)
	return data, errors
}

func (r *mutationResolver) DeleteTension(ctx context.Context, filter model.TensionFilter) (data *model.DeleteTensionPayload, errors error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *mutationResolver) AddComment(ctx context.Context, input []*model.AddCommentInput) (data *model.AddCommentPayload, errors error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *mutationResolver) UpdateComment(ctx context.Context, input model.UpdateCommentInput) (data *model.UpdateCommentPayload, errors error) {
	errors = r.Gqlgen2DgraphQueryResolver(ctx, &data)
	return data, errors
}

func (r *mutationResolver) DeleteComment(ctx context.Context, filter model.CommentFilter) (data *model.DeleteCommentPayload, errors error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *mutationResolver) AddBlob(ctx context.Context, input []*model.AddBlobInput) (data *model.AddBlobPayload, errors error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *mutationResolver) UpdateBlob(ctx context.Context, input model.UpdateBlobInput) (data *model.UpdateBlobPayload, errors error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *mutationResolver) DeleteBlob(ctx context.Context, filter model.BlobFilter) (data *model.DeleteBlobPayload, errors error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *mutationResolver) AddEvent(ctx context.Context, input []*model.AddEventInput) (data *model.AddEventPayload, errors error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *mutationResolver) UpdateEvent(ctx context.Context, input model.UpdateEventInput) (data *model.UpdateEventPayload, errors error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *mutationResolver) DeleteEvent(ctx context.Context, filter model.EventFilter) (data *model.DeleteEventPayload, errors error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *mutationResolver) AddEventFragment(ctx context.Context, input []*model.AddEventFragmentInput) (data *model.AddEventFragmentPayload, errors error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *mutationResolver) UpdateEventFragment(ctx context.Context, input model.UpdateEventFragmentInput) (data *model.UpdateEventFragmentPayload, errors error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *mutationResolver) DeleteEventFragment(ctx context.Context, filter model.EventFragmentFilter) (data *model.DeleteEventFragmentPayload, errors error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *mutationResolver) AddContract(ctx context.Context, input []*model.AddContractInput, upsert *bool) (data *model.AddContractPayload, errors error) {
	errors = r.Gqlgen2DgraphQueryResolver(ctx, &data)
	return data, errors
}

func (r *mutationResolver) UpdateContract(ctx context.Context, input model.UpdateContractInput) (data *model.UpdateContractPayload, errors error) {
	errors = r.Gqlgen2DgraphQueryResolver(ctx, &data)
	return data, errors
}

func (r *mutationResolver) DeleteContract(ctx context.Context, filter model.ContractFilter) (data *model.DeleteContractPayload, errors error) {
	errors = r.Gqlgen2DgraphQueryResolver(ctx, &data)
	return data, errors
}

func (r *mutationResolver) AddPendingUser(ctx context.Context, input []*model.AddPendingUserInput) (data *model.AddPendingUserPayload, errors error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *mutationResolver) UpdatePendingUser(ctx context.Context, input model.UpdatePendingUserInput) (data *model.UpdatePendingUserPayload, errors error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *mutationResolver) DeletePendingUser(ctx context.Context, filter model.PendingUserFilter) (data *model.DeletePendingUserPayload, errors error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *mutationResolver) AddVote(ctx context.Context, input []*model.AddVoteInput, upsert *bool) (data *model.AddVotePayload, errors error) {
	errors = r.Gqlgen2DgraphQueryResolver(ctx, &data)
	return data, errors
}

func (r *mutationResolver) UpdateVote(ctx context.Context, input model.UpdateVoteInput) (data *model.UpdateVotePayload, errors error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *mutationResolver) DeleteVote(ctx context.Context, filter model.VoteFilter) (data *model.DeleteVotePayload, errors error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *mutationResolver) AddUser(ctx context.Context, input []*model.AddUserInput, upsert *bool) (data *model.AddUserPayload, errors error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *mutationResolver) UpdateUser(ctx context.Context, input model.UpdateUserInput) (data *model.UpdateUserPayload, errors error) {
	errors = r.Gqlgen2DgraphQueryResolver(ctx, &data)
	return data, errors
}

func (r *mutationResolver) DeleteUser(ctx context.Context, filter model.UserFilter) (data *model.DeleteUserPayload, errors error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *mutationResolver) AddUserEvent(ctx context.Context, input []*model.AddUserEventInput) (data *model.AddUserEventPayload, errors error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *mutationResolver) UpdateUserEvent(ctx context.Context, input model.UpdateUserEventInput) (data *model.UpdateUserEventPayload, errors error) {
	errors = r.Gqlgen2DgraphQueryResolver(ctx, &data)
	return data, errors
}

func (r *mutationResolver) DeleteUserEvent(ctx context.Context, filter model.UserEventFilter) (data *model.DeleteUserEventPayload, errors error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *mutationResolver) AddUserRights(ctx context.Context, input []*model.AddUserRightsInput) (data *model.AddUserRightsPayload, errors error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *mutationResolver) UpdateUserRights(ctx context.Context, input model.UpdateUserRightsInput) (data *model.UpdateUserRightsPayload, errors error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *mutationResolver) DeleteUserRights(ctx context.Context, filter model.UserRightsFilter) (data *model.DeleteUserRightsPayload, errors error) {
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

func (r *queryResolver) AggregateNode(ctx context.Context, filter *model.NodeFilter) (data *model.NodeAggregateResult, errors error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *queryResolver) GetNodeFragment(ctx context.Context, id string) (data *model.NodeFragment, errors error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *queryResolver) QueryNodeFragment(ctx context.Context, filter *model.NodeFragmentFilter, order *model.NodeFragmentOrder, first *int, offset *int) (data []*model.NodeFragment, errors error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *queryResolver) AggregateNodeFragment(ctx context.Context, filter *model.NodeFragmentFilter) (data *model.NodeFragmentAggregateResult, errors error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *queryResolver) GetMandate(ctx context.Context, id string) (data *model.Mandate, errors error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *queryResolver) QueryMandate(ctx context.Context, filter *model.MandateFilter, order *model.MandateOrder, first *int, offset *int) (data []*model.Mandate, errors error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *queryResolver) AggregateMandate(ctx context.Context, filter *model.MandateFilter) (data *model.MandateAggregateResult, errors error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *queryResolver) GetLabel(ctx context.Context, id string) (data *model.Label, errors error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *queryResolver) QueryLabel(ctx context.Context, filter *model.LabelFilter, order *model.LabelOrder, first *int, offset *int) (data []*model.Label, errors error) {
	errors = r.Gqlgen2DgraphQueryResolver(ctx, &data)
	return data, errors
}

func (r *queryResolver) AggregateLabel(ctx context.Context, filter *model.LabelFilter) (data *model.LabelAggregateResult, errors error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *queryResolver) GetRoleExt(ctx context.Context, id string) (data *model.RoleExt, errors error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *queryResolver) QueryRoleExt(ctx context.Context, filter *model.RoleExtFilter, order *model.RoleExtOrder, first *int, offset *int) (data []*model.RoleExt, errors error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *queryResolver) AggregateRoleExt(ctx context.Context, filter *model.RoleExtFilter) (data *model.RoleExtAggregateResult, errors error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *queryResolver) QueryOrgaAgg(ctx context.Context, filter *model.OrgaAggFilter, order *model.OrgaAggOrder, first *int, offset *int) (data []*model.OrgaAgg, errors error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *queryResolver) AggregateOrgaAgg(ctx context.Context, filter *model.OrgaAggFilter) (data *model.OrgaAggAggregateResult, errors error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *queryResolver) GetPost(ctx context.Context, id string) (data *model.Post, errors error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *queryResolver) QueryPost(ctx context.Context, filter *model.PostFilter, order *model.PostOrder, first *int, offset *int) (data []*model.Post, errors error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *queryResolver) AggregatePost(ctx context.Context, filter *model.PostFilter) (data *model.PostAggregateResult, errors error) {
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

func (r *queryResolver) AggregateTension(ctx context.Context, filter *model.TensionFilter) (data *model.TensionAggregateResult, errors error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *queryResolver) GetComment(ctx context.Context, id string) (data *model.Comment, errors error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *queryResolver) QueryComment(ctx context.Context, filter *model.CommentFilter, order *model.CommentOrder, first *int, offset *int) (data []*model.Comment, errors error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *queryResolver) AggregateComment(ctx context.Context, filter *model.CommentFilter) (data *model.CommentAggregateResult, errors error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *queryResolver) GetBlob(ctx context.Context, id string) (data *model.Blob, errors error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *queryResolver) QueryBlob(ctx context.Context, filter *model.BlobFilter, order *model.BlobOrder, first *int, offset *int) (data []*model.Blob, errors error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *queryResolver) AggregateBlob(ctx context.Context, filter *model.BlobFilter) (data *model.BlobAggregateResult, errors error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *queryResolver) GetEvent(ctx context.Context, id string) (data *model.Event, errors error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *queryResolver) QueryEvent(ctx context.Context, filter *model.EventFilter, order *model.EventOrder, first *int, offset *int) (data []*model.Event, errors error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *queryResolver) AggregateEvent(ctx context.Context, filter *model.EventFilter) (data *model.EventAggregateResult, errors error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *queryResolver) QueryEventFragment(ctx context.Context, filter *model.EventFragmentFilter, order *model.EventFragmentOrder, first *int, offset *int) (data []*model.EventFragment, errors error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *queryResolver) AggregateEventFragment(ctx context.Context, filter *model.EventFragmentFilter) (data *model.EventFragmentAggregateResult, errors error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *queryResolver) GetContract(ctx context.Context, id *string, contractid *string) (data *model.Contract, errors error) {
	errors = r.Gqlgen2DgraphQueryResolver(ctx, &data)
	return data, errors
}

func (r *queryResolver) QueryContract(ctx context.Context, filter *model.ContractFilter, order *model.ContractOrder, first *int, offset *int) (data []*model.Contract, errors error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *queryResolver) AggregateContract(ctx context.Context, filter *model.ContractFilter) (data *model.ContractAggregateResult, errors error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *queryResolver) QueryPendingUser(ctx context.Context, filter *model.PendingUserFilter, order *model.PendingUserOrder, first *int, offset *int) (data []*model.PendingUser, errors error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *queryResolver) AggregatePendingUser(ctx context.Context, filter *model.PendingUserFilter) (data *model.PendingUserAggregateResult, errors error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *queryResolver) GetVote(ctx context.Context, id *string, voteid *string) (data *model.Vote, errors error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *queryResolver) QueryVote(ctx context.Context, filter *model.VoteFilter, order *model.VoteOrder, first *int, offset *int) (data []*model.Vote, errors error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *queryResolver) AggregateVote(ctx context.Context, filter *model.VoteFilter) (data *model.VoteAggregateResult, errors error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *queryResolver) GetUser(ctx context.Context, id *string, username *string) (data *model.User, errors error) {
	errors = r.Gqlgen2DgraphQueryResolver(ctx, &data)
	return data, errors
}

func (r *queryResolver) QueryUser(ctx context.Context, filter *model.UserFilter, order *model.UserOrder, first *int, offset *int) (data []*model.User, errors error) {
	errors = r.Gqlgen2DgraphQueryResolver(ctx, &data)
	return data, errors
}

func (r *queryResolver) AggregateUser(ctx context.Context, filter *model.UserFilter) (data *model.UserAggregateResult, errors error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *queryResolver) GetUserEvent(ctx context.Context, id string) (data *model.UserEvent, errors error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *queryResolver) QueryUserEvent(ctx context.Context, filter *model.UserEventFilter, order *model.UserEventOrder, first *int, offset *int) (data []*model.UserEvent, errors error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *queryResolver) AggregateUserEvent(ctx context.Context, filter *model.UserEventFilter) (data *model.UserEventAggregateResult, errors error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *queryResolver) QueryUserRights(ctx context.Context, filter *model.UserRightsFilter, order *model.UserRightsOrder, first *int, offset *int) (data []*model.UserRights, errors error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *queryResolver) AggregateUserRights(ctx context.Context, filter *model.UserRightsFilter) (data *model.UserRightsAggregateResult, errors error) {
	panic(fmt.Errorf("not implemented"))
}

// Mutation returns generated.MutationResolver implementation.
func (r *Resolver) Mutation() generated.MutationResolver { return &mutationResolver{r} }

// Query returns generated.QueryResolver implementation.
func (r *Resolver) Query() generated.QueryResolver { return &queryResolver{r} }

type mutationResolver struct{ *Resolver }
type queryResolver struct{ *Resolver }
