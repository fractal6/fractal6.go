/*
 * Fractale - Self-organisation for humans.
 * Copyright (C) 2026 Fractale Co
 *
 * This file is part of Fractale.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as
 * published by the Free Software Foundation, either version 3 of the
 * License, or (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with Fractale.  If not, see <http://www.gnu.org/licenses/>.
 */

package db

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"fractale/fractal6.go/graph/model"
	. "fractale/fractal6.go/internal/tools"
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

	// MUTATIONS - @todo merge with bridger query ?
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
	// Extra - Bridge
	"queryExtra": `{
        "query": "query {{.QueryName}}($filter:{{.FilterType}}, $order:{{.OrderType}}, $first:Int, $offset:Int) {
            {{.QueryName}}{{.QueryInput}} {{.Directives}} {
                {{.QueryGraph}}
            }
        }",
        "variables": {{.VarMap}}
    }`,
	"addExtra": `{
        "query": "mutation {{.QueryName}}($input:[{{.InputType}}!]!) {
            {{.QueryName}}{{.QueryInput}} {
                {{.QueryGraph}}
            }
        }",
        "variables": {
            "input": {{.InputPayload}}
        }
    }`,
	"mutationExtra": `{
        "query": "mutation {{.QueryName}}($input:{{.InputType}}!) {
            {{.QueryName}}{{.QueryInput}} {
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

// Query data using GQL dgraph API. @auth rules will apply.
func (dg Dgraph) Query(uctx model.UserCtx, vertex string, k string, values []string, queryGraph string) ([]map[string]string, error) {
	Vertex := strings.Title(vertex)
	queryName := "query" + Vertex

	var i int
	var n, args string
	var res []map[string]string

	// Build query arguments
	if k == "id" {
		ids_formated, _ := json.Marshal(values)
		args = fmt.Sprintf(`id: %s`, ids_formated)
	} else {
		for i, n = range values {
			if i == 0 {
				args += fmt.Sprintf(`%s: {eq:"%s"},`, k, n)
			} else {
				args += fmt.Sprintf(`or: {%s: {eq: "%s"},`, k, n)
			}
		}
		args += strings.Repeat("},", i)
	}

	// Build query
	input := map[string]string{
		"QueryName":  queryName,
		"QueryGraph": queryGraph,
		"Args":       CleanString("filter: {"+args+"}", true),
	}

	// send query
	err := dg.QueryGql(uctx, "query", input, &res)
	if err != nil {
		return res, err
	}

	return res, nil
}

// Get a new vertex (NOT USED YET...)
func (dg Dgraph) Get(uctx model.UserCtx, vertex string, input map[string]string, graph string) (any, error) {
	Vertex := strings.Title(vertex)
	queryName := "get" + Vertex
	queryGraph := graph

	// Build the string request
	reqInput := map[string]string{
		"QueryName":  queryName,                     // function name (e.g addUser)
		"QueryGraph": CleanString(queryGraph, true), // output data
		"key":        input["key"],
		"value":      input["value"],
	}

	// Send request
	payload := make(model.JsonAtom, 1)
	err := dg.QueryGql(uctx, "get", reqInput, payload)
	if err != nil {
		return "", err
	}
	// Extract id result
	if payload[queryName] == nil {
		return "", fmt.Errorf("Unauthorized request. Possibly, name already exists.")
	}
	res := payload[queryName]
	return res, err
}

// Add a new vertex
func (dg Dgraph) Add(uctx model.UserCtx, vertex string, input any) (string, error) {
	Vertex := strings.Title(vertex)
	queryName := "add" + Vertex
	inputType := "Add" + Vertex + "Input"
	queryGraph := vertex + ` { id }`

	// Build the string request
	inputs, _ := MarshalWithoutNil(input)
	reqInput := map[string]string{
		"QueryName":    queryName,                     // function name (e.g addUser)
		"InputType":    inputType,                     // input type name (e.g AddUserInput)
		"InputPayload": "[" + string(inputs) + "]",    // inputs data -- Just one node
		"QueryGraph":   CleanString(queryGraph, true), // output data
	}

	// Send request
	payload := make(model.JsonAtom, 1)
	err := dg.QueryGql(uctx, "add", reqInput, payload)
	if err != nil {
		return "", err
	}
	// Extract id result
	if payload[queryName] == nil {
		return "", fmt.Errorf("Unauthorized request. Possibly, name already exists.")
	}
	res := payload[queryName].(model.JsonAtom)[vertex].([]any)[0].(model.JsonAtom)["id"]
	return res.(string), err
}

// Update a vertex
func (dg Dgraph) Update(uctx model.UserCtx, vertex string, input any) error {
	Vertex := strings.Title(vertex)
	queryName := "update" + Vertex
	inputType := "Update" + Vertex + "Input"
	queryGraph := vertex + ` { id }`

	// Build the string request
	inputs, _ := MarshalWithoutNil(input)
	reqInput := map[string]string{
		"QueryName":    queryName,                     // function name (e.g addUser)
		"InputType":    inputType,                     // input type name (e.g AddUserInput)
		"InputPayload": string(inputs),                // inputs data
		"QueryGraph":   CleanString(queryGraph, true), // output data
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
func (dg Dgraph) Delete(uctx model.UserCtx, vertex string, input any) error {
	Vertex := strings.Title(vertex)
	queryName := "delete" + Vertex
	inputType := Vertex + "Filter"
	queryGraph := vertex + ` { id }`

	// Build the string request
	inputs, _ := MarshalWithoutNil(input)
	reqInput := map[string]string{
		"QueryName":    queryName,                     // function name (e.g addUser)
		"InputType":    inputType,                     // input type name (e.g AddUserInput)
		"InputPayload": string(inputs),                // inputs data
		"QueryGraph":   CleanString(queryGraph, true), // output data
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
func (dg Dgraph) AddMany(uctx model.UserCtx, vertex string, input any) ([]string, error) {
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
		"QueryName":    queryName,                             // function name (e.g addUser)
		"InputType":    inputType,                             // input type name (e.g AddUserInput)
		"InputPayload": "[" + strings.Join(inputs, ",") + "]", // inputs data
		"QueryGraph":   CleanString(queryGraph, true),         // output data
	}

	// Send request
	payload := make(model.JsonAtom, 1)
	err := dg.QueryGql(uctx, "add", reqInput, payload)
	if err != nil {
		return []string{}, err
	}
	// Extract id result
	if payload[queryName] == nil {
		return []string{}, fmt.Errorf("Unauthorized request. Possibly, name already exists.")
	}
	var l []string
	res := payload[queryName].(model.JsonAtom)[vertex].([]any)
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
		// field := ToGoNameFormat(k)
		// pass

	default:
		return fmt.Errorf("unknown vertex '%s'", vertex)
	}

	f := fmt.Sprintf(`{"%s":"%s"}`, k, QuoteString(v))
	err := json.Unmarshal([]byte(f), &set)
	if err != nil {
		return err
	}
	input.Filter = &filter
	input.Set = &set
	filter.ID = []string{id}
	err = dg.Update(uctx, vertex, input)
	return err
}

/*
 *  Bridge queries
 */

// GetDirectives return the list of directives to apply to the given query
// by looking for the pressence of special attributes in the payload graph.
func GetDirectives(pg string) (string, string) {
	directives := []string{}
	words := strings.Fields(pg)
	if slices.Contains(words, "cascade_directive") {
		directives = append(directives, "@cascade")
		pg = strings.ReplaceAll(pg, "cascade_directive", "")
	}
	return pg, strings.Join(directives, " ")
}

// Query codec, to be used in the resolver functions
func (dg Dgraph) QueryExtra(uctx model.UserCtx, vertex string, filter any, order any, first *int, offset *int, qg string, data any) error {
	Vertex := strings.Title(vertex)
	queryName := "query" + Vertex
	filterType := Vertex + "Filter"
	orderType := Vertex + "Order"

	// Build the string request
	var queryInput string
	queryInput = `(filter: $filter, order: $order, first: $first, offset: $offset)`

	// Marshal the inputs
	filter_ := struct {
		Filter any  `json:"filter"`
		Order  any  `json:"order"`
		First  *int `json:"first"`
		Offset *int `json:"offset"`
	}{filter, order, first, offset}
	varmap, _ := MarshalWithoutNil(filter_)

	qg, directives := GetDirectives(qg)

	// Build the request template map
	reqInput := map[string]string{
		"QueryName":  queryName,               // Query name (e.g addUser)
		"FilterType": filterType,              // input type name (e.g AddUserInput)
		"OrderType":  orderType,               // input type name (e.g AddUserInput)
		"QueryInput": QuoteString(queryInput), // inputs data
		"QueryGraph": CleanString(qg, true),   // output data
		"VarMap":     string(varmap),          // inputs data
		"Directives": directives,
	}

	// Send request
	err := dg.QueryGql(uctx, "queryExtra", reqInput, data)
	return err
}

// Add codec, to be used in the resolver functions
func (dg Dgraph) AddExtra(uctx model.UserCtx, vertex string, input any, upsert *bool, qg string, data any) error {
	Vertex := strings.Title(vertex)
	queryName := "add" + Vertex
	inputType := "Add" + Vertex + "Input"
	// queryGraph := vertex + " {" + qgraph + "}"

	// Build the string request
	var queryInput string
	if upsert != nil {
		queryInput = fmt.Sprintf(`(input: $input, upsert: %t)`, *upsert)
	} else {
		queryInput = `(input: $input)`
	}

	var inputs string
	slice, ok := InterfaceSlice(input)
	if ok {
		var ipts []string
		for _, x := range slice {
			s, _ := MarshalWithoutNil(x)
			ipts = append(ipts, string(s))
		}
		inputs = "[" + strings.Join(ipts, ",") + "]"
	} else {
		x, _ := MarshalWithoutNil(input)
		inputs = string(x)
	}

	reqInput := map[string]string{
		"QueryName":    queryName,               // Query name (e.g addUser)
		"InputType":    inputType,               // input type name (e.g AddUserInput)
		"QueryInput":   QuoteString(queryInput), // inputs data
		"QueryGraph":   CleanString(qg, true),   // output data
		"InputPayload": string(inputs),          // inputs data
	}

	// Send request
	err := dg.QueryGql(uctx, "addExtra", reqInput, data)
	return err
}

// Update codec, to be used in the resolver functions
func (dg Dgraph) UpdateExtra(uctx model.UserCtx, vertex string, input any, qg string, data any) error {
	Vertex := strings.Title(vertex)
	queryName := "update" + Vertex
	inputType := "Update" + Vertex + "Input"
	// queryGraph := vertex + " {" + qgraph + "}"

	// Build the string request
	var queryInput string = "(input: $input)"
	var inputs string
	slice, ok := InterfaceSlice(input)
	if ok {
		var ipts []string
		for _, x := range slice {
			s, _ := MarshalWithoutNil(x)
			ipts = append(ipts, string(s))
		}
		inputs = "[" + strings.Join(ipts, ",") + "]"
	} else {
		x, _ := MarshalWithoutNil(input)
		inputs = string(x)
	}

	reqInput := map[string]string{
		"QueryName":    queryName,               // Query name (e.g addUser)
		"InputType":    inputType,               // input type name (e.g AddUserInput)
		"QueryInput":   QuoteString(queryInput), // inputs data
		"QueryGraph":   CleanString(qg, true),   // output data
		"InputPayload": string(inputs),          // inputs data
	}

	// Send request
	err := dg.QueryGql(uctx, "mutationExtra", reqInput, data)
	return err
}

// Delete codec, to be used in the resolver functions
func (dg Dgraph) DeleteExtra(uctx model.UserCtx, vertex string, input any, qg string, data any) error {
	Vertex := strings.Title(vertex)
	queryName := "delete" + Vertex
	inputType := Vertex + "Filter"
	// queryGraph := vertex + " {" + qgraph + "}"

	// Build the string request
	var queryInput string = "(filter: $input)"
	var inputs string
	slice, ok := InterfaceSlice(input)
	if ok {
		var ipts []string
		for _, x := range slice {
			s, _ := MarshalWithoutNil(x)
			ipts = append(ipts, string(s))
		}
		inputs = "[" + strings.Join(ipts, ",") + "]"
	} else {
		x, _ := MarshalWithoutNil(input)
		inputs = string(x)
	}

	reqInput := map[string]string{
		"QueryName":    queryName,               // Query name (e.g addUser)
		"InputType":    inputType,               // input type name (e.g AddUserInput)
		"QueryInput":   QuoteString(queryInput), // inputs data
		"QueryGraph":   CleanString(qg, true),   // output data
		"InputPayload": string(inputs),          // inputs data
	}

	// Send request
	err := dg.QueryGql(uctx, "mutationExtra", reqInput, data)
	return err
}

//
// Private User methods
//

// AddUserRole add a role to the user roles list
func (dg Dgraph) AddUserRole(username, nameid string) error {
	userInput := model.UpdateUserInput{
		Filter: &model.UserFilter{Username: &model.StringHashFilterStringRegExpFilter{Eq: &username}},
		Set: &model.UserPatch{
			Roles: []*model.NodeRef{{Nameid: &nameid}},
		},
	}
	err := dg.Update(dg.GetRootUctx(), "user", userInput)
	return err
}

// RemoveUserRole remove a  role to the user roles list
func (dg Dgraph) RemoveUserRole(username, nameid string) error {
	userInput := model.UpdateUserInput{
		Filter: &model.UserFilter{Username: &model.StringHashFilterStringRegExpFilter{Eq: &username}},
		Remove: &model.UserPatch{
			Roles: []*model.NodeRef{{Nameid: &nameid}},
		},
	}
	err := dg.Update(dg.GetRootUctx(), "user", userInput)
	return err
}
