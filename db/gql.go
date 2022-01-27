package db

import (
	"fmt"
	"encoding/json"
	"strings"

	"fractale/fractal6.go/graph/model"
	. "fractale/fractal6.go/tools"
)

// HTTP/Graphql Request Template
var gqlQueries map[string]string = map[string]string{
    // QUERIES
    "rawQuery": `{
        "query": "{{.RawQuery}}",
        "variables": {{.Variables}}
    }`,
    "query": `{
        "query": "query {{.QueryName}} {
            {{.QueryName}} ({{.Args}}) {
                {{.QueryGraph}}
            }
        }"
    }`,
    "get": `{
        "query": "query {{.QueryName}} {
            {{.QueryName}} ({{.key}}: \"{{.value}}\") {
                {{.QueryGraph}}
            }
        }"
    }`,

    // MUTATIONS
    "add": `{
        "query": "mutation {{.QueryName}}($input:[{{.InputType}}!]!) {
            {{.QueryName}}(input: $input) {
                {{.QueryGraph}}
            }
        }",
        "variables": {
            "input": {{.InputPayload}}
        }
    }`,
    "update": `{
        "query": "mutation {{.QueryName}}($input:{{.InputType}}!) {
            {{.QueryName}}(input: $input) {
                {{.QueryGraph}}
            }
        }",
        "variables": {
            "input": {{.InputPayload}}
        }
    }`,
    "delete": `{
        "query": "mutation {{.QueryName}}($input:{{.InputType}}!) {
            {{.QueryName}}(filter: $input) {
                {{.QueryGraph}}
            }
        }",
        "variables": {
            "input": {{.InputPayload}}
        }
    }`,
}

//
// Graphql requests
//

// Get a new vertex
func (dg Dgraph) Get(uctx model.UserCtx, vertex string, input map[string]string, graph string) (interface{}, error) {
    Vertex := strings.Title(vertex)
    queryName := "get" + Vertex
    queryGraph := graph

    // Build the string request
    reqInput := map[string]string{
        "QueryName": queryName, // function name (e.g addUser)
        "QueryGraph": CleanString(queryGraph, true), // output data
        "key": input["key"],
        "value": input["value"],
    }

    // Send request
    payload := make(model.JsonAtom, 1)
    err := dg.QueryGql(uctx, "get", reqInput, payload)
    if err != nil { return "", err }
    // Extract id result
    if payload[queryName] == nil {
        return "", fmt.Errorf("Unauthorized request. Possibly, name already exists.")
    }
    res := payload[queryName]
    return res, err
}

// Add a new vertex
func (dg Dgraph) Add(uctx model.UserCtx, vertex string, input interface{}) (string, error) {
    Vertex := strings.Title(vertex)
    queryName := "add" + Vertex
    inputType := "Add" + Vertex + "Input"
    queryGraph := vertex + ` { id }`

    // Build the string request
    inputs, _ := MarshalWithoutNil(input)
    reqInput := map[string]string{
        "QueryName": queryName, // function name (e.g addUser)
        "InputType": inputType, // input type name (e.g AddUserInput)
        "InputPayload": "["+string(inputs)+"]", // inputs data -- Just one node
        "QueryGraph": CleanString(queryGraph, true), // output data
    }

    // Send request
    payload := make(model.JsonAtom, 1)
    err := dg.QueryGql(uctx, "add", reqInput, payload)
    if err != nil { return "", err }
    // Extract id result
    if payload[queryName] == nil {
        return "", fmt.Errorf("Unauthorized request. Possibly, name already exists.")
    }
    res := payload[queryName].(model.JsonAtom)[vertex].([]interface{})[0].(model.JsonAtom)["id"]
    return res.(string), err
}

// Update a vertex
func (dg Dgraph) Update(uctx model.UserCtx, vertex string, input interface{}) error {
    Vertex := strings.Title(vertex)
    queryName := "update" + Vertex
    inputType := "Update" + Vertex + "Input"
    queryGraph := vertex + ` { id }`

    // Build the string request
    inputs, _ := MarshalWithoutNil(input)
    reqInput := map[string]string{
        "QueryName": queryName, // function name (e.g addUser)
        "InputType": inputType, // input type name (e.g AddUserInput)
        "InputPayload": string(inputs), // inputs data
        "QueryGraph": CleanString(queryGraph, true), // output data
    }

    // Send request
    payload := make(model.JsonAtom, 1)
    err := dg.QueryGql(uctx, "update", reqInput, payload)
    if payload[queryName] == nil && err == nil {
        return fmt.Errorf("Unauthorized request. Possibly, name already exists.")
    }
    return err
}

// Delete a vertex
func (dg Dgraph) Delete(uctx model.UserCtx, vertex string, input interface{}) error {
    Vertex := strings.Title(vertex)
    queryName := "delete" + Vertex
    inputType :=  Vertex + "Filter"
    queryGraph := vertex + ` { id }`

    // Build the string request
    inputs, _ := MarshalWithoutNil(input)
    reqInput := map[string]string{
        "QueryName": queryName, // function name (e.g addUser)
        "InputType": inputType, // input type name (e.g AddUserInput)
        "InputPayload": string(inputs), // inputs data
        "QueryGraph": CleanString(queryGraph, true), // output data
    }

    // Send request
    payload := make(model.JsonAtom, 1)
    err := dg.QueryGql(uctx, "delete", reqInput, payload)
    if payload[queryName] == nil && err == nil {
        return fmt.Errorf("Unauthorized request.")
    }
    return err
}

//
// GQL utility
//


// Add multiple new vertex
func (dg Dgraph) AddMany(uctx model.UserCtx, vertex string, input interface{}) ([]string, error) {
    Vertex := strings.Title(vertex)
    queryName := "add" + Vertex
    inputType := "Add" + Vertex + "Input"
    queryGraph := vertex + ` { id }`

    // Build the string request
    slice, ok := InterfaceSlice(input)
    if !ok {
        return []string{}, fmt.Errorf("Input must be a slice")
    }
    var inputs []string
    for _, x := range slice {
        s, _ := MarshalWithoutNil(x)
        inputs = append(inputs, string(s))
    }
    reqInput := map[string]string{
        "QueryName": queryName, // function name (e.g addUser)
        "InputType": inputType, // input type name (e.g AddUserInput)
        "InputPayload": "[" + strings.Join(inputs, ",") + "]", // inputs data
        "QueryGraph": CleanString(queryGraph, true), // output data
    }

    // Send request
    payload := make(model.JsonAtom, 1)
    err := dg.QueryGql(uctx, "add", reqInput, payload)
    if err != nil { return []string{}, err }
    // Extract id result
    if payload[queryName] == nil {
        return []string{}, fmt.Errorf("Unauthorized request. Possibly, name already exists.")
    }
    var l []string
    res := payload[queryName].(model.JsonAtom)[vertex].([]interface{})
    for _, r := range res {
        l = append(l, r.(model.JsonAtom)["id"].(string))
    }
    return l, err
}

// No way to dynamically build the type ?
func (dg Dgraph) UpdateValue(uctx model.UserCtx, vertex string, id, k, v string) error {
    var input model.UpdateTensionInput
    var filter model.TensionFilter
    var set model.TensionPatch

    switch vertex {
    case "tension":
        //field := ToGoNameFormat(k)
        // pass

    default:
        return fmt.Errorf("unknown vertex '%s'", vertex)
    }

    f := fmt.Sprintf(`{"%s":"%s"}`, k, v)
    err := json.Unmarshal([]byte(f), &set)
    if err != nil { return err }
    input.Filter = &filter
    input.Set = &set
    filter.ID = []string{id}
    err = dg.Update(uctx, vertex, input)
    return err
}

// Like Update but with a given payload
func (dg Dgraph) UpdateExtra(uctx model.UserCtx, vertex string, input interface{}, qgraph string) (interface{}, error) {
    Vertex := strings.Title(vertex)
    queryName := "update" + Vertex
    inputType := "Update" + Vertex + "Input"
    queryGraph := vertex + " {" + qgraph + "}"

    // Build the string request
    inputs, _ := MarshalWithoutNil(input)
    reqInput := map[string]string{
        "QueryName": queryName, // function name (e.g addUser)
        "InputType": inputType, // input type name (e.g AddUserInput)
        "InputPayload": string(inputs), // inputs data
        "QueryGraph": CleanString(queryGraph, true), // output data
    }

    // Send request
    payload := make(model.JsonAtom, 1)
    err := dg.QueryGql(uctx, "update", reqInput, payload)
    if payload[queryName] == nil && err == nil {
        return nil, fmt.Errorf("Unauthorized request. Possibly, name already exists.")
    }
    return payload[queryName], err
}

//
// @auth
//

//QueryAuthFilter Get only the authorized node
func (dg Dgraph) QueryAuthFilter(uctx model.UserCtx, vertex string, k string, values []string) ([]string, error) {
    Vertex := strings.Title(vertex)
    queryName := "query" + Vertex
    queryGraph := k

    var i int
    var n, args string
    var res []map[string]string
    final := []string{}

    // Build query arguments
    for i, n = range values {
        if i == 0 {
            args += fmt.Sprintf(`%s: {eq:"%s"},`, k, n)
        } else {
            args += fmt.Sprintf(`or: {%s: {eq: "%s"},`, k, n)
        }
    }
    args += strings.Repeat("},", i)

    // Build query
    input := map[string]string{
        "QueryName": queryName,
        "QueryGraph": queryGraph,
        "Args": CleanString("filter: {"+args+"}", true),
    }

    // send query
    err := dg.QueryGql(uctx, "query", input, &res)
    if err != nil { return final, err }

    for _, x := range res {
        final = append(final, x[k])
    }
    return final, nil
}

//
// Private User methods
//

// AddUserRole add a role to the user roles list
func (dg Dgraph) AddUserRole(username, nameid string) error {
    userInput := model.UpdateUserInput{
        Filter: &model.UserFilter{ Username: &model.StringHashFilterStringRegExpFilter{ Eq: &username } },
        Set: &model.UserPatch{
            Roles: []*model.NodeRef{ &model.NodeRef{ Nameid: &nameid }},
        },
    }
    err := dg.Update(dg.GetRootUctx(), "user", userInput)
    return err
}

// RemoveUserRole remove a  role to the user roles list
func (dg Dgraph) RemoveUserRole(username, nameid string) error {
    userInput := model.UpdateUserInput{
        Filter: &model.UserFilter{ Username: &model.StringHashFilterStringRegExpFilter{ Eq: &username } },
        Remove: &model.UserPatch{
            Roles: []*model.NodeRef{ &model.NodeRef{ Nameid: &nameid }},
        },
    }
    err := dg.Update(dg.GetRootUctx(), "user", userInput)
    return err
}
