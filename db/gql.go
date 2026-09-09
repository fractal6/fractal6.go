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
        "query": "query {{.QueryName}}{{.VarDecl}} {
            {{.QueryName}}{{.QueryInput}} {{.Directives}} {
                {{.QueryGraph}}
            }
        }",
        "variables": {{.VarMap}}
    }`,

	// MUTATIONS
	"add": `{
        "query": "mutation {{.QueryName}}($input:[{{.InputType}}!]!) {
            {{.QueryName}}{{.QueryInput}} {
                {{.QueryGraph}}
            }
        }",
        "variables": {
            "input": {{.InputPayload}}
        }
    }`,
	// Update and Delete, which both take a single input object.
	"mutation": `{
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
// The *Graph methods take the requested payload graph (qg) and decode the
// result into data: they back the gqlgen bridges (graph/dgraph_resolver.go).
// The shortcuts below wrap them for internal callers that only need ids.
//

// capitalize upper the first letter, e.g "node" -> "Node".
func capitalize(s string) string {
	return strings.ToUpper(s[:1]) + s[1:]
}

// marshalInputList marshal an input as a GQL list: the arity is set by the
// template (only `add` takes a list), a single object is wrapped.
func marshalInputList(input any) string {
	slice, ok := InterfaceSlice(input)
	if !ok {
		slice = []any{input}
	}
	var inputs []string
	for _, x := range slice {
		s, _ := MarshalWithoutNil(x)
		inputs = append(inputs, string(s))
	}
	return "[" + strings.Join(inputs, ",") + "]"
}

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

// QueryGraph query a vertex, with the given payload graph.
func (dg Dgraph) QueryGraph(uctx model.UserCtx, vertex string, filter any, order any, first *int, offset *int, qg string, data any) error {
	Vertex := capitalize(vertex)

	// Marshal the inputs, dropping the nil ones.
	args := CleanNilMap(StructMap[map[string]any](struct {
		Filter any  `json:"filter"`
		Order  any  `json:"order"`
		First  *int `json:"first"`
		Offset *int `json:"offset"`
	}{filter, order, first, offset}))
	varmap, _ := json.Marshal(args)

	// Declare only the arguments given: not every vertex has an <Vertex>Order input type.
	var decls, inputs []string
	for _, a := range [][2]string{
		{"filter", Vertex + "Filter"},
		{"order", Vertex + "Order"},
		{"first", "Int"},
		{"offset", "Int"},
	} {
		if _, ok := args[a[0]]; !ok {
			continue
		}
		decls = append(decls, "$"+a[0]+":"+a[1])
		inputs = append(inputs, a[0]+": $"+a[0])
	}
	var varDecl, queryInput string
	if len(decls) > 0 {
		varDecl = "(" + strings.Join(decls, ", ") + ")"
		queryInput = "(" + strings.Join(inputs, ", ") + ")"
	}

	qg, directives := GetDirectives(qg)

	// Build the request template map
	reqInput := map[string]string{
		"QueryName":  "query" + Vertex, // Query name (e.g queryUser)
		"VarDecl":    QuoteString(varDecl),
		"QueryInput": QuoteString(queryInput),
		"QueryGraph": CleanString(qg, true), // output data
		"VarMap":     string(varmap),        // inputs data
		"Directives": directives,
	}

	return dg.QueryGql(uctx, "query", reqInput, data)
}

// AddGraph add one or many vertex, with the given payload graph.
func (dg Dgraph) AddGraph(uctx model.UserCtx, vertex string, input any, upsert *bool, qg string, data any) error {
	Vertex := capitalize(vertex)

	queryInput := `(input: $input)`
	if upsert != nil {
		queryInput = fmt.Sprintf(`(input: $input, upsert: %t)`, *upsert)
	}

	reqInput := map[string]string{
		"QueryName":    "add" + Vertex,           // Query name (e.g addUser)
		"InputType":    "Add" + Vertex + "Input", // input type name (e.g AddUserInput)
		"QueryInput":   QuoteString(queryInput),
		"InputPayload": marshalInputList(input), // inputs data
		"QueryGraph":   CleanString(qg, true),   // output data
	}

	return dg.QueryGql(uctx, "add", reqInput, data)
}

// UpdateGraph update a vertex, with the given payload graph.
func (dg Dgraph) UpdateGraph(uctx model.UserCtx, vertex string, input any, qg string, data any) error {
	Vertex := capitalize(vertex)
	payload, _ := MarshalWithoutNil(input)

	reqInput := map[string]string{
		"QueryName":    "update" + Vertex,           // Query name (e.g updateUser)
		"InputType":    "Update" + Vertex + "Input", // input type name (e.g UpdateUserInput)
		"QueryInput":   QuoteString(`(input: $input)`),
		"InputPayload": string(payload),       // inputs data
		"QueryGraph":   CleanString(qg, true), // output data
	}

	return dg.QueryGql(uctx, "mutation", reqInput, data)
}

// DeleteGraph delete a vertex, with the given payload graph.
func (dg Dgraph) DeleteGraph(uctx model.UserCtx, vertex string, filter any, qg string, data any) error {
	Vertex := capitalize(vertex)
	payload, _ := MarshalWithoutNil(filter)

	reqInput := map[string]string{
		"QueryName":    "delete" + Vertex, // Query name (e.g deleteUser)
		"InputType":    Vertex + "Filter", // input type name (e.g UserFilter)
		"QueryInput":   QuoteString(`(filter: $input)`),
		"InputPayload": string(payload),       // inputs data
		"QueryGraph":   CleanString(qg, true), // output data
	}

	return dg.QueryGql(uctx, "mutation", reqInput, data)
}

//
// Shortcuts, for callers that only need the vertex ids.
//

// Query data using GQL dgraph API. @auth rules will apply.
// k must be "id" or a field that supports {in: […]} (i.e. @id or @search(by:[hash])).
func (dg Dgraph) Query(uctx model.UserCtx, vertex string, k string, values []string, queryGraph string) ([]map[string]string, error) {
	var filter any = map[string]any{k: map[string]any{"in": values}}
	if k == "id" {
		filter = map[string]any{"id": values}
	}

	var res []map[string]string
	err := dg.QueryGraph(uctx, vertex, filter, nil, nil, nil, queryGraph, &res)
	return res, err
}

// Add a new vertex and return its id.
func (dg Dgraph) Add(uctx model.UserCtx, vertex string, input any) (string, error) {
	ids, err := dg.AddMany(uctx, vertex, []any{input})
	if err != nil {
		return "", err
	}
	if len(ids) == 0 {
		return "", fmt.Errorf("Unauthorized request. Possibly, name already exists.")
	}
	return ids[0], nil
}

// AddMany add multiple new vertex and return their ids.
func (dg Dgraph) AddMany(uctx model.UserCtx, vertex string, input any) ([]string, error) {
	payload := make(model.JsonAtom, 1)
	if err := dg.AddGraph(uctx, vertex, input, nil, vertex+` { id }`, payload); err != nil {
		return nil, err
	}

	// Extract id results. The payload can be empty when @auth rules filter
	// out the created vertex: that's not an error, only a missing payload is.
	res, ok := payload["add"+capitalize(vertex)].(model.JsonAtom)
	if !ok {
		return nil, fmt.Errorf("Unauthorized request. Possibly, name already exists.")
	}
	nodes, _ := res[vertex].([]any)
	ids := make([]string, 0, len(nodes))
	for _, n := range nodes {
		node, _ := n.(model.JsonAtom)
		id, _ := node["id"].(string)
		ids = append(ids, id)
	}
	return ids, nil
}

// Update a vertex
func (dg Dgraph) Update(uctx model.UserCtx, vertex string, input any) error {
	payload := make(model.JsonAtom, 1)
	err := dg.UpdateGraph(uctx, vertex, input, vertex+` { id }`, payload)
	if payload["update"+capitalize(vertex)] == nil && err == nil {
		return fmt.Errorf("Unauthorized request. Possibly, name already exists.")
	}
	return err
}

// Delete a vertex
func (dg Dgraph) Delete(uctx model.UserCtx, vertex string, filter any) error {
	payload := make(model.JsonAtom, 1)
	err := dg.DeleteGraph(uctx, vertex, filter, vertex+` { id }`, payload)
	if payload["delete"+capitalize(vertex)] == nil && err == nil {
		return fmt.Errorf("Unauthorized request.")
	}
	return err
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

// RemoveUserRole remove a role to the user roles list
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
