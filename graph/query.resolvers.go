// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.
package graph

import (
	"fmt"
	"context"
	"strings"
    "encoding/json"

	"zerogov/fractal6.go/graph/generated"
	"zerogov/fractal6.go/graph/model"
	"zerogov/fractal6.go/tools"
	"zerogov/fractal6.go/tools/gql"
)


type GqlResponse struct {
	Adduser model.AddUserPayload `json:"addUser"`
}
type Res struct {
	Data   GqlResponse `json:"data"`
	Errors []gql.JsonAtom  `json:"errors"` // message, locations, path
}

var MutationQ, QueryQ gql.Query

func init() {

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

    QueryQ.Data = `{
        "query": "query {{.QueryName}} { 
            {{.QueryName}} {
                {{.QueryGraph}}
            } 
        }"
    }`

    QueryQ.Init()
    MutationQ.Init()
}


func (r *mutationResolver) AddUser(ctx context.Context, input []*model.AddUserInput) (*model.AddUserPayload, error) {

    /* Rebuild the Graphql request from this context */

    // Format inputs
    ipts := []gql.JsonAtom{}
    for _, x := range input {
        ipts = append(ipts, tools.StructToMap(x))
    }
    inputs, _ := json.Marshal(ipts)
	//reqq := ctx.Value("request_body").([]byte)
    //fmt.Println(string(reqq))
    
    // Format collected fields
    queryGraph := strings.Join(GetPreloads(ctx), " ")

    // Build the string request
    reqInput := gql.JsonAtom{
        "QueryName": "addUser", 
        "InputType": "AddUserInput", 
        "InputPayload": string(inputs),
        "QueryGraph": queryGraph,
    }
    req := MutationQ.Format(reqInput)

    /* Send the dgraph request and follow the results */

    // Dgraph request
	res := &Res{} // or new(Res)
	err := r.db.Request([]byte(req), res)
	if err != nil {
		panic(err)
	}

    // Get and returns the result
	//fmt.Println(string(req))
	//fmt.Println(res)
	data := &res.Data.Adduser
	errors := res.Errors
	if errors != nil {
		var msg []string
		for _, m := range res.Errors {
			msg = append(msg, m["message"].(string))
		}
		err := fmt.Errorf("%s", strings.Join(msg, " | "))
		return data, err
	}

	return data, nil
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

