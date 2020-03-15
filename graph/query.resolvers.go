// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.
package graph

import (
	"context"
	"fmt"
	"strings"
    "bytes"
    "encoding/json"
	"text/template"
    "regexp"

	"zerogov/fractal6.go/graph/generated"
	"zerogov/fractal6.go/graph/model"
	"zerogov/fractal6.go/tools"
)

type JsonAtom map[string]interface{}

type GqlResponse struct {
	Adduser model.AddUserPayload `json:"addUser"`
}
type Res struct {
	Data   GqlResponse `json:"data"`
	Errors []JsonAtom  `json:"errors"` // message, locations, path
}

var MutationQT *template.Template

func init() {

    var MutationQ string = `{
        "query": "mutation {{.QueryName}}($input:{{.InputType}}!) { 
            {{.QueryName}}( input: $input) {} 
        }",
        "variables": {
            "input": {{.InputPayload}}
        }
    }`
    MutationQ = strings.Replace(MutationQ, `\n`, "", -1)
    MutationQ = strings.Replace(MutationQ, "\n", "", -1)
    space := regexp.MustCompile(`\s+`)
    MutationQ = space.ReplaceAllString(MutationQ, " ")

	MutationQT = template.Must(template.New("mutationGQL").Parse(MutationQ))
}


func (r *mutationResolver) AddUser(ctx context.Context, input []*model.AddUserInput) (*model.AddUserPayload, error) {

    // Format request
	reqq := ctx.Value("request_body").([]byte)
    fmt.Println(string(reqq))
    ipts := []JsonAtom{}
    for _, x := range input {
        ipts = append(ipts, tools.StructToMap(x))
    }
    inputs, _ := json.Marshal(ipts)
    fieldCollections := GetPreloads(ctx)
    fmt. Println(fieldCollections)
    reqInput := map[string]interface{}{
        "QueryName": "addUser", 
        "InputType": "AddUserInput", 
        "InputPayload":string(inputs),
    }
    buf := bytes.Buffer{}
    MutationQT.Execute(&buf, reqInput)
    req := buf.String()

    // Make dgraph request
	res := &Res{} // or new(Res)
	err := r.db.Request([]byte(req), res)
	if err != nil {
		panic(err)
	}

	fmt.Println(string(req))
	fmt.Println(res)

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

