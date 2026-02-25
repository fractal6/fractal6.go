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
	"log"
	"strings"

	"github.com/dgraph-io/dgo/v200/protos/api"

	"fractale/fractal6.go/graph/model"
	. "fractale/fractal6.go/internal/tools"
)

// @refactor: modularize generic function (GetFilterBy*) (returns (any, error}
//            as Traverse([list of key to traver], [list of payload to get])
//  Use Meta() for all other queries... (return []any)

// @uture: with Go.18 rewrite this module with generics
//
// make QueryMut and Queries/Mutations {
//      Q string // query blocks
//      S string // mutations block (for dql mutaitons)
//      T expected type/
// }
//
// rewrite Meta and Meta_patch for as the main generic function to uses the librairies of queries,
// replacing all the singular functions here.

//
// Gprc/DQL requests
//

// Count count the number of object in fieldName attribute for given type and id
// Returns: int or -1 if nothing is found.
func (dg Dgraph) Count(id string, fieldName string) int {
	// Format Query
	maps := map[string]string{
		"id": id, "fieldName": fieldName,
	}
	// Send request
	res, err := dg.QueryDql("count", maps)
	if err != nil {
		log.Printf("Error in db.Count: %v", err)
		return -1
	}

	// Decode response
	var r DqlRespCount
	err = json.Unmarshal(res.Json, &r)
	if err != nil {
		log.Printf("Error in db.Count: %v", err)
		return -1
	}

	// Extract result
	if len(r.All) == 0 {
		return -1
	}

	values := make([]int, 0, len(r.All[0]))
	for _, v := range r.All[0] {
		values = append(values, v)
	}

	return values[0]
}

func (dg Dgraph) Count2(f1, v1, f2, v2, fieldName string) int {
	// Format Query
	maps := map[string]string{
		"f1":        f1,
		"v1":        v1,
		"f2":        f2,
		"v2":        v2,
		"fieldName": fieldName,
	}
	// Send request
	res, err := dg.QueryDql("count2", maps)
	if err != nil {
		log.Printf("Error in db.Count2: %v", err)
		return -1
	}

	// Decode response
	var r DqlRespCount
	err = json.Unmarshal(res.Json, &r)
	if err != nil {
		log.Printf("Error in db.Count2: %v", err)
		return -1
	}

	// Extract result
	if len(r.All) == 0 {
		return -1
	}

	values := make([]int, 0, len(r.All[0]))
	for _, v := range r.All[0] {
		values = append(values, v)
	}

	return values[0]
}

func (dg Dgraph) CountHas(fieldName string) int {
	// Format Query
	maps := map[string]string{
		"fieldName": fieldName,
	}
	// Send request
	res, err := dg.QueryDql("countHas", maps)
	if err != nil {
		log.Printf("Error in db.CountHas: %v", err)
		return -1
	}

	// Decode response
	var r DqlRespCount
	err = json.Unmarshal(res.Json, &r)
	if err != nil {
		log.Printf("Error in db.CountHas: %v", err)
		return -1
	}

	// Extract result
	if len(r.All) == 0 {
		return -1
	}

	values := make([]int, 0, len(r.All[0]))
	for _, v := range r.All[0] {
		values = append(values, v)
	}

	return values[0]
}

func (dg Dgraph) CountHas2(fieldName, f2, v2 string) int {
	// Format Query
	maps := map[string]string{
		"fieldName": fieldName,
		"f2":        f2,
		"v2":        v2,
	}
	// Send request
	res, err := dg.QueryDql("countHas2", maps)
	if err != nil {
		log.Printf("Error in db.CountHas2: %v", err)
		return -1
	}

	// Decode response
	var r DqlRespCount
	err = json.Unmarshal(res.Json, &r)
	if err != nil {
		log.Printf("Error in db.CountHas2: %v", err)
		return -1
	}

	// Extract result
	if len(r.All) == 0 {
		return -1
	}

	values := make([]int, 0, len(r.All[0]))
	for _, v := range r.All[0] {
		values = append(values, v)
	}

	return values[0]
}

func (dg Dgraph) Meta(f string, maps map[string]string) ([]map[string]any, error) {
	// Execute a DQL request from the given defined query template
	// Returns: array
	var res *api.Response
	var err error

	if _, ok := dqlQueries[f]; ok { // Query Case
		// Send request
		res, err = dg.QueryDql(f, maps)
	} else if _, ok := dqlMutations[f]; ok { // Mutation Case
		// Send request
		// @codefactor: unify api...
		res, err = dg.MutateWithQueryDql3(dqlMutations[f], maps)
	} else {
		err = fmt.Errorf("Unknown DQL query")
	}

	if err != nil {
		return nil, err
	}

	return decodeDqlResp(res)
}

func (dg Dgraph) Gamma(q QueryMut, maps map[string]string) ([]map[string]any, error) {
	// Send Custom DQL request
	// Returns: array
	var res *api.Response
	var err error

	res, err = dg.MutateWithQueryDql3(q, maps)
	if res == nil {
		return nil, err
	}

	return decodeDqlResp(res)
}

// Probe if an object exists.
func (dg Dgraph) Exists(fieldName string, value string, filter *string) (bool, error) {
	// Format Query
	maps := map[string]string{
		"fieldName": fieldName,
		"value":     value,
		"filter":    "",
	}
	if filter != nil {
		maps["filter"] = fmt.Sprintf(`@filter(%s)`, *filter)
	}
	// Send request
	res, err := dg.QueryDql("exists", maps)
	if err != nil {
		return false, err
	}
	// Decode response
	var r DqlResp
	err = json.Unmarshal(res.Json, &r)
	if err != nil {
		return false, err
	}
	return len(r.All) > 0, nil
}

// IsChild returns true is a node parent has the given child.
func (dg Dgraph) IsChild(parent, child string) (bool, error) {
	// Format Query
	maps := map[string]string{
		"parent": parent,
		"child":  child,
	}
	// Send request
	res, err := dg.QueryDql("isChild", maps)
	if err != nil {
		return false, err
	}
	// Decode response
	var r DqlResp
	err = json.Unmarshal(res.Json, &r)
	if err != nil {
		return false, err
	}
	return len(r.All) > 0, nil
}

// Returns the uids of the objects if found.
func (dg Dgraph) GetIDs(fieldName string, value string, filterName, filterValue *string) ([]string, error) {
	result := []string{}
	// Format Query
	maps := map[string]string{
		"fieldName": fieldName,
		"value":     value,
		"filter":    "",
	}
	if filterName != nil {
		maps["filter"] = fmt.Sprintf(`@filter(eq(%s, "%s"))`, *filterName, *filterValue)
	}
	// Send request
	res, err := dg.QueryDql("getID", maps)
	if err != nil {
		return result, err
	}
	// Decode response
	var r DqlResp
	err = json.Unmarshal(res.Json, &r)
	if err != nil {
		return result, err
	}
	for _, x := range r.All {
		result = append(result, x["uid"].(string))
	}
	return result, nil
}

// Returns a field from id
func (dg Dgraph) GetFieldById(id string, fieldName string) (any, error) {
	results, err := dg.Meta("getFieldById", map[string]string{
		"id":        id,
		"fieldName": fieldName,
	})
	if err != nil {
		return nil, err
	}
	return DecodeField(results, fieldName)
}

// Returns a field from objid
func (dg Dgraph) GetFieldByEq(fieldid string, objid string, fieldName string) (any, error) {
	results, err := dg.Meta("getFieldByEq", map[string]string{
		"fieldid":   fieldid,
		"value":     objid,
		"fieldName": fieldName,
	})
	if err != nil {
		return nil, err
	}
	return DecodeField(results, fieldName)
}

// Returns a subfield from uid
func (dg Dgraph) GetSubFieldById(id string, fieldNameSource string, fieldNameTarget string) (any, error) {
	results, err := dg.Meta("getSubFieldById", map[string]string{
		"id":              id,
		"fieldNameSource": fieldNameSource,
		"fieldNameTarget": fieldNameTarget,
	})
	if err != nil {
		return nil, err
	}
	return DecodeSubField(results, fieldNameSource, fieldNameTarget)
}

// Returns a subfield from Eq
func (dg Dgraph) GetSubFieldByEq(fieldid string, value string, fieldNameSource string, fieldNameTarget string) (any, error) {
	results, err := dg.Meta("getSubFieldByEq", map[string]string{
		"fieldid":         fieldid,
		"value":           value,
		"fieldNameSource": fieldNameSource,
		"fieldNameTarget": fieldNameTarget,
	})
	if err != nil {
		return nil, err
	}
	return DecodeSubField(results, fieldNameSource, fieldNameTarget)
}

func (dg Dgraph) GetSubFieldByEq2(fieldid, value, f2, v2, fieldNameSource, fieldNameTarget string) (any, error) {
	results, err := dg.Meta("getSubFieldByEq2", map[string]string{
		"fieldid":         fieldid,
		"value":           value,
		"f2":              f2,
		"v2":              v2,
		"fieldNameSource": fieldNameSource,
		"fieldNameTarget": fieldNameTarget,
	})
	if err != nil {
		return nil, err
	}
	return DecodeSubField(results, fieldNameSource, fieldNameTarget)
}

// Returns a subsubfield from uid
func (dg Dgraph) GetSubSubFieldById(id string, fieldNameSource string, fieldNameTarget string, subFieldNameTarget string) (any, error) {
	results, err := dg.Meta("getSubSubFieldById", map[string]string{
		"id":                 id,
		"fieldNameSource":    fieldNameSource,
		"fieldNameTarget":    fieldNameTarget,
		"subFieldNameTarget": subFieldNameTarget,
	})
	if err != nil {
		return nil, err
	}
	return DecodeSubSubField(results, fieldNameSource, fieldNameTarget, subFieldNameTarget)
}

// Returns a subsubfield from Eq
func (dg Dgraph) GetSubSubFieldByEq(fieldid string, value string, fieldNameSource string, fieldNameTarget string, subFieldNameTarget string) (any, error) {
	results, err := dg.Meta("getSubSubFieldByEq", map[string]string{
		"fieldid":            fieldid,
		"value":              value,
		"fieldNameSource":    fieldNameSource,
		"fieldNameTarget":    fieldNameTarget,
		"subFieldNameTarget": subFieldNameTarget,
	})
	if err != nil {
		return nil, err
	}
	return DecodeSubSubField(results, fieldNameSource, fieldNameTarget, subFieldNameTarget)
}

func (dg Dgraph) GetShortestPath(from string, to string) (float64, error) {
	weight := 0.0
	// Format Query
	maps := map[string]string{
		"from": from,
		"to":   to,
	}
	// Send request
	res, err := dg.QueryDql("getShortestPath", maps)
	if err != nil {
		return weight, err
	}

	// Decode response
	var r DqlResp
	err = json.Unmarshal(res.Json, &r)
	if err != nil {
		return weight, err
	}

	if len(r.All) > 1 {
		return weight, fmt.Errorf("Got multiple in DQL query")
	} else if len(r.All) == 1 {
		weight, ok := r.All[0]["weight"].(float64)
		if !ok {
			return weight, fmt.Errorf("Cannot extract weight from shortest path query")
		} else {
			return weight, err
		}
	}
	return weight, err
}

// Returns the user context
func (dg Dgraph) GetUctxFull(fieldid string, userid string) (*model.UserCtx, error) {
	// Format Query
	maps := map[string]string{
		"fieldid": fieldid,
		"userid":  userid,
		"payload": userCtxPayload,
	}
	// Send request
	res, err := dg.QueryDql("getUser", maps)
	if err != nil {
		return nil, err
	}

	// Decode response
	var r DqlResp
	err = json.Unmarshal(res.Json, &r)
	if err != nil {
		return nil, err
	}

	var user model.UserCtx
	if len(r.All) > 1 {
		return nil, fmt.Errorf("Got multiple user with same @id: %s, %s", fieldid, userid)
	} else if len(r.All) == 1 {
		user, err = DecodeDql[model.UserCtx](r.All[0])
		if err != nil {
			return nil, err
		}
	}
	user.Hit++ // Avoid reloading user during the session context
	return &user, err
}

// Returns matching User. Never return nil user without an error.
func (dg Dgraph) GetUctx(fieldid string, userid string) (*model.UserCtx, error) {
	user, err := dg.GetUctxFull(fieldid, userid)
	if err != nil {
		return user, nil
	}
	if user == nil || user.Username == "" {
		return nil, fmt.Errorf("User not found for '%s': %s", fieldid, userid)
	}
	// @note: special role are filtered out in web/auth
	return user, err
}

// Returns the matching nodes
func (dg Dgraph) GetNodes(regex string, isRoot bool) ([]model.Node, error) {
	// Format Query
	maps := map[string]string{
		"regex": regex,
		"payload": `{
            Node.nameid
            Node.visibility
        }`,
	}

	// Send request
	var res *api.Response
	var err error
	if isRoot {
		res, err = dg.QueryDql("getNodesRoot", maps)
	} else {
		res, err = dg.QueryDql("getNodes", maps)
	}
	if err != nil {
		return nil, err
	}

	// Decode response
	var r DqlResp
	err = json.Unmarshal(res.Json, &r)
	if err != nil {
		return nil, err
	}

	data, err := DecodeDql[[]model.Node](r.All)
	return data, err
}

// Returns the tension hook content
func (dg Dgraph) GetTensionHook(tid string, withBlob bool, bid *string) (*model.Tension, error) {
	// Format Query
	var maps map[string]string
	if withBlob {
		var blobFilter string
		if bid == nil {
			blobFilter = "(orderdesc: Post.createdAt, first: 1)"
		} else {
			blobFilter = fmt.Sprintf(`@filter(uid(%s))`, *bid)
		}
		maps = map[string]string{
			"id":      tid,
			"payload": "{" + tensionHookPayload + fmt.Sprintf(tensionBlobHookPayload, blobFilter) + "}",
		}
	} else {
		maps = map[string]string{
			"id":      tid,
			"payload": "{" + tensionHookPayload + "}",
		}
	}

	// Send request
	res, err := dg.QueryDql("getTensionHook", maps)
	if err != nil {
		return nil, err
	}

	// Decode response
	var r DqlResp
	err = json.Unmarshal(res.Json, &r)
	if err != nil {
		return nil, err
	}

	var obj model.Tension
	if len(r.All) > 1 {
		return nil, fmt.Errorf("Got multiple tension for @uid: %s", tid)
	} else if len(r.All) == 1 {
		obj, err = DecodeDql[model.Tension](r.All[0])
		if err != nil {
			return nil, err
		}
	}

	// Assume that tension does not exists if receiver is empty
	// This is because DQL returns an object with t uid even is non existent.
	if obj.Receiver == nil {
		return nil, err
	}
	return &obj, err
}

// Returns the contract hook content
func (dg Dgraph) GetContractHook(cid string) (*model.Contract, error) {
	// Format Query
	maps := map[string]string{
		"id":      cid,
		"payload": "{" + contractHookPayload + "}",
	}

	// Send request
	var q string
	if strings.Contains(cid, "#") {
		q = "getContractHook2"
	} else {
		q = "getContractHook"
	}
	res, err := dg.QueryDql(q, maps)
	if err != nil {
		return nil, err
	}

	// Decode response
	var r DqlResp
	err = json.Unmarshal(res.Json, &r)
	if err != nil {
		return nil, err
	}

	var obj model.Contract
	if len(r.All) > 1 {
		return nil, fmt.Errorf("Got multiple contract for @uid: %s", cid)
	} else if len(r.All) == 1 {
		obj, err = DecodeDql[model.Contract](r.All[0])
		if err != nil {
			return nil, err
		}
	}
	return &obj, err
}

// excludeSelfFlag returns "true" (truthy in Go templates) when the target
// node should be excluded, or "" (falsy) when it should be included.
func excludeSelfFlag(includeSelf bool) string {
	if !includeSelf {
		return "true"
	}
	return ""
}

// Get all sub children
func (dg Dgraph) GetSubNodes(fieldid string, objid string, includeSelf bool) ([]model.Node, error) {
	results, err := dg.Meta("getSubNodes", map[string]string{
		"fieldid":     fieldid,
		"objid":       objid,
		"excludeSelf": excludeSelfFlag(includeSelf),
	})
	if err != nil {
		return nil, err
	}
	return DecodeDql[[]model.Node](results)
}

// Get all sub members
func (dg Dgraph) GetSubMembers(fieldid, objid, user_payload string, includeSelf bool) ([]model.Node, error) {
	results, err := dg.Meta("getSubMembers", map[string]string{
		"fieldid":      fieldid,
		"objid":        objid,
		"user_payload": user_payload,
		"excludeSelf":  excludeSelfFlag(includeSelf),
	})
	if err != nil {
		return nil, err
	}
	return DecodeDql[[]model.Node](results)
}

// Get all top labels
func (dg Dgraph) GetTopLabels(fieldid string, objid string, includeSelf bool) ([]model.Label, error) {
	results, err := dg.Meta("getTopLabels", map[string]string{
		"fieldid":     fieldid,
		"objid":       objid,
		"excludeSelf": excludeSelfFlag(includeSelf),
	})
	if err != nil {
		return nil, err
	}

	data, err := DecodeDql[[]model.Label](results)
	if err != nil {
		return nil, err
	}
	return Dedupe(data, func(l model.Label) string { return l.Name }), nil
}

// Get all sub labels
func (dg Dgraph) GetSubLabels(fieldid string, objid string, includeSelf bool) ([]model.Label, error) {
	results, err := dg.Meta("getSubLabels", map[string]string{
		"fieldid":     fieldid,
		"objid":       objid,
		"excludeSelf": excludeSelfFlag(includeSelf),
	})
	if err != nil {
		return nil, err
	}

	data, err := DecodeDql[[]model.Label](results)
	if err != nil {
		return nil, err
	}
	return Dedupe(data, func(l model.Label) string { return l.Name }), nil
}

// Get all top roles
func (dg Dgraph) GetTopRoles(fieldid string, objid string, includeSelf bool) ([]model.RoleExt, error) {
	results, err := dg.Meta("getTopRoles", map[string]string{
		"fieldid":     fieldid,
		"objid":       objid,
		"excludeSelf": excludeSelfFlag(includeSelf),
	})
	if err != nil {
		return nil, err
	}

	data, err := DecodeDql[[]model.RoleExt](results)
	if err != nil {
		return nil, err
	}
	return Dedupe(data, func(r model.RoleExt) string { return r.Name }), nil
}

// Get all sub roles
func (dg Dgraph) GetSubRoles(fieldid string, objid string, includeSelf bool) ([]model.RoleExt, error) {
	results, err := dg.Meta("getSubRoles", map[string]string{
		"fieldid":     fieldid,
		"objid":       objid,
		"excludeSelf": excludeSelfFlag(includeSelf),
	})
	if err != nil {
		return nil, err
	}

	data, err := DecodeDql[[]model.RoleExt](results)
	if err != nil {
		return nil, err
	}
	return Dedupe(data, func(r model.RoleExt) string { return r.Name }), nil
}

// ProjectFull is a lightweight project representation for the sub_projects endpoint.
type ProjectFull struct {
	ID            string        `json:"id"`
	UpdatedAt     string        `json:"updatedAt,omitempty"`
	Name          string        `json:"name"`
	Description   *string       `json:"description,omitempty"`
	Nodes         []*model.Node `json:"nodes,omitempty"`
	Collaborators []*model.User `json:"collaborators,omitempty"`
}

// Get all sub projects
func (dg Dgraph) GetSubProjects(fieldid string, objid string, includeSelf bool) ([]ProjectFull, error) {
	results, err := dg.Meta("getSubProjects", map[string]string{
		"fieldid":     fieldid,
		"objid":       objid,
		"excludeSelf": excludeSelfFlag(includeSelf),
	})
	if err != nil {
		return nil, err
	}
	return DecodeDql[[]ProjectFull](results)
}

func (dg Dgraph) GetTensions(q TensionQuery, type_ string) ([]model.TensionRef, error) {
	// Format Query
	maps, err := FormatTensionIntExtMap(q)
	if err != nil {
		return nil, err
	}
	// Send request
	var op string
	var isLight bool = false
	var payload string
	if type_ == "light" {
		op = "getTensionInt"
		isLight = true
	} else if type_ == "int" {
		op = "getTensionInt"
	} else if type_ == "ext" {
		op = "getTensionExt"
	} else if type_ == "all" {
		op = "getTensionAll"
	} else {
		panic("Unknow type (tension query)")
	}

	payload = `uid
        Post.createdAt
        Post.createdBy { User.username }
        Tension.receiver { Node.nameid Node.name Node.role_type }
        Tension.emitter { Node.nameid Node.name Node.role_type }
        Tension.title
        Tension.status
        Tension.type_
        Tension.action
        Tension.labels { uid Label.name Label.color }
        n_comments: count(Tension.comments)`

	if isLight {
		payload = `uid
            Tension.title
            Tension.status
            Tension.type_
            Tension.labels { uid Label.name Label.color }`
	}

	if type_ == "all" {
		payload += `
           Tension.assignees { User.username User.name }`
	}

	(*maps)["payload"] = payload
	res, err := dg.QueryDql(op, *maps)
	if err != nil {
		return nil, err
	}

	// Decode response
	var r DqlResp
	err = json.Unmarshal(res.Json, &r)
	if err != nil {
		return nil, err
	}

	data, err := DecodeDql[[]model.TensionRef](r.All)
	return data, err
}

func (dg Dgraph) GetTensionsCount(q TensionQuery) (map[string]int, error) {
	// Format Query
	maps, err := FormatTensionIntExtMap(q)
	if err != nil {
		return nil, err
	}
	// Send request
	res, err := dg.QueryDql("getTensionCount", *maps)
	if err != nil {
		return nil, err
	}

	// Decode response
	var r DqlRespCount
	err = json.Unmarshal(res.Json, &r)
	if err != nil {
		return nil, err
	}

	if len(r.All) > 0 && len(r.All2) > 0 {
		v := map[string]int{"open": r.All[0]["count"], "closed": r.All2[0]["count"]}
		return v, err
	}
	return nil, err
}

// tensionBlobRef is a decode target for GetLastBlobId DQL results.
type tensionBlobRef struct {
	Blobs []struct {
		ID string `json:"id"`
	} `json:"blobs"`
}

func (dg Dgraph) GetLastBlobId(tid string) *string {
	maps := map[string]string{"tid": tid}
	// Send request
	res, err := dg.QueryDql("getLastBlobId", maps)
	if err != nil {
		return nil
	}

	// Decode response
	var r DqlResp
	err = json.Unmarshal(res.Json, &r)
	if err != nil {
		return nil
	}

	if len(r.All) != 1 {
		return nil
	}

	data, err := DecodeDql[tensionBlobRef](r.All[0])
	if err != nil {
		return nil
	}

	var bid string
	if len(data.Blobs) > 0 {
		bid = data.Blobs[0].ID
	}
	return &bid
}

// nodeChildRefs is a decode target for DQL queries returning children with uid only.
type nodeChildRefs struct {
	Children []struct {
		ID string `json:"id"`
	} `json:"children"`
}

// Get all coordo roles in the given circle with an user linked.
func (dg Dgraph) HasCoordos(nameid string) bool {
	// Format Query
	maps := map[string]string{
		"nameid": nameid,
	}
	// Send request
	res, err := dg.QueryDql("getCoordos", maps)
	if err != nil {
		return false
	}

	// Decode response
	var r DqlResp
	err = json.Unmarshal(res.Json, &r)
	if err != nil {
		return false
	}

	if len(r.All) != 1 {
		return false
	}

	data, err := DecodeDql[nodeChildRefs](r.All[0])
	if err != nil {
		return false
	}
	return len(data.Children) > 0
}

// nodeChildNameids is a decode target for GetChildren DQL results.
type nodeChildNameids struct {
	Children []struct {
		Nameid string `json:"nameid"`
	} `json:"children"`
}

// Get children
func (dg Dgraph) GetChildren(nameid string) ([]string, error) {
	// Format Query
	maps := map[string]string{
		"nameid": nameid,
	}
	// Send request
	res, err := dg.QueryDql("getChildren", maps)
	if err != nil {
		return nil, err
	}

	// Decode response
	var r DqlResp
	err = json.Unmarshal(res.Json, &r)
	if err != nil {
		return nil, err
	}

	if len(r.All) > 1 {
		return nil, fmt.Errorf("Got multiple object for term: %s", nameid)
	}
	if len(r.All) != 1 {
		return nil, nil
	}

	data, err := DecodeDql[nodeChildNameids](r.All[0])
	if err != nil {
		return nil, err
	}

	result := make([]string, 0, len(data.Children))
	for _, c := range data.Children {
		result = append(result, c.Nameid)
	}
	return result, nil
}

// nodeParentNameids is a decode target for GetParents DQL results.
// The @recurse + @normalize query may return nameid as a string (single parent)
// or []any (multiple ancestors), so Nameid is typed as any.
type nodeParentNameids struct {
	Parent []struct {
		Nameid any `json:"nameid"`
	} `json:"parent"`
}

// Get path to root
func (dg Dgraph) GetParents(nameid string) ([]string, error) {
	res, err := dg.QueryDql("getParents", map[string]string{
		"nameid": nameid,
	})
	if err != nil {
		return nil, err
	}

	var r DqlResp
	if err = json.Unmarshal(res.Json, &r); err != nil {
		return nil, err
	}

	if len(r.All) > 1 {
		return nil, fmt.Errorf("Got multiple object for term: %s", nameid)
	}
	if len(r.All) != 1 {
		return nil, nil
	}

	data, err := DecodeDql[nodeParentNameids](r.All[0])
	if err != nil {
		return nil, err
	}
	if len(data.Parent) == 0 {
		return nil, nil
	}

	var result []string
	switch v := data.Parent[0].Nameid.(type) {
	case string:
		result = append(result, v)
	case []any:
		for _, x := range v {
			if s, ok := x.(string); ok {
				result = append(result, s)
			}
		}
	}
	return result, nil
}

// tensionSearchData is a decode target for GetTensionSearchData DQL results.
type tensionSearchData struct {
	Labels []struct {
		Name string `json:"name"`
	} `json:"labels"`
	Comments []struct {
		Message string `json:"message"`
	} `json:"comments"`
}

// GetTensionSearchData fetches label names and the first comment text for a tension.
// Used to build the denormalized Post.message search index.
func (dg Dgraph) GetTensionSearchData(tid string) ([]string, string, error) {
	maps := map[string]string{"tid": tid}
	res, err := dg.QueryDql("getTensionSearchData", maps)
	if err != nil {
		return nil, "", err
	}

	var r DqlResp
	err = json.Unmarshal(res.Json, &r)
	if err != nil {
		return nil, "", err
	}

	if len(r.All) != 1 {
		return nil, "", nil
	}

	data, err := DecodeDql[tensionSearchData](r.All[0])
	if err != nil {
		return nil, "", err
	}

	labels := make([]string, 0, len(data.Labels))
	for _, l := range data.Labels {
		labels = append(labels, l.Name)
	}

	var firstComment string
	if len(data.Comments) > 0 {
		firstComment = data.Comments[0].Message
	}

	return labels, firstComment, nil
}

// DQL Mutations

// SetFieldById set a predicate for the given node in the DB
func (dg Dgraph) SetFieldById(objid, predicate, val string) error {
	_, err := dg.Meta("setFieldById", map[string]string{"id": objid, "predicate": predicate, "value": val})
	return err
}

// SetFieldByEq set a predicate for the given node in the DB
func (dg Dgraph) SetFieldByEq(fieldid, objid, predicate, val string) error {
	_, err := dg.Meta("setFieldByEq", map[string]string{"fieldid": fieldid, "objid": objid, "predicate": predicate, "value": val})
	return err
}

// SetPushedFlagBlob sets the blob pushedFlag and the tension action
func (dg Dgraph) SetPushedFlagBlob(bid, flag, tid string, action model.TensionAction) error {
	_, err := dg.Meta("setPushedFlagBlob", map[string]string{"bid": bid, "flag": flag, "tid": tid, "action": string(action)})
	return err
}

// SetChildrenRoleVisibility updates visibility on all child roles
func (dg Dgraph) SetChildrenRoleVisibility(nameid, value string) error {
	_, err := dg.Meta("setChildrenRoleVisibility", map[string]string{"nameid": nameid, "value": value})
	return err
}

// UpgradeMember updates the role type of a member node
func (dg Dgraph) UpgradeMember(nameid string, roleType model.RoleType) error {
	_, err := dg.Meta("upgradeMember", map[string]string{"nameid": nameid, "roleType": string(roleType)})
	return err
}

// Deletions

// DeepDelete delete edges recursively for type {t} and id {id}.
// Reverse edges need to be deleted manually since they are defined in graphql and not in DQL.
// Note: If reverse are forgotten, empty redisual nodes will accumulates.
func (dg Dgraph) DeepDelete(t string, id string) error {
	var reverse string
	var query string

	switch t {
	case "tension":
		reverse = fmt.Sprintf(`
            uid(rid_emitter) <Node.tensions_out> uid(id) .
            uid(rid_receiver) <Node.tensions_in> uid(id) .
        `)
	case "contract":
		reverse = fmt.Sprintf(`
            uid(rid) <Tension.contracts> uid(id) .
            uid(candidates) <User.contracts> uid(id) .
            uid(user_pending) <PendingUser.contracts> uid(id) .
            uid(nodes) <Node.contracts> uid(votes) .
            uid(members) <User.events> uid(desync_events) .
            uid(desync_events) * * .
        `)
	default:
		return fmt.Errorf("delete query not implemented for this type %s", t)
	}

	maps := map[string]string{"id": id}
	query = dg.getDqlQuery("delete"+strings.Title(t), maps)
	mu := fmt.Sprintf(`
        %s
        uid(all_ids) * * .
    `, reverse)

	mutation := &api.Mutation{
		DelNquads: []byte(mu),
	}

	err := dg.MutateWithQueryDql(query, mutation)
	return err
}

//
// Generic DQL helpers
//

// decodeDqlResp decodes an api.Response into cleaned DQL result maps.
func decodeDqlResp(res *api.Response) ([]map[string]any, error) {
	if res == nil {
		return nil, nil
	}
	var r DqlResp
	if err := json.Unmarshal(res.Json, &r); err != nil {
		return nil, err
	}
	out := make([]map[string]any, len(r.All))
	for i, m := range r.All {
		out[i] = CleanDqlMap(m)
	}
	return out, nil
}

// Meta executes a named DQL query/mutation and decodes results into typed values.
func Meta[T any](f string, maps map[string]string) ([]T, error) {
	results, err := GetDB().Meta(f, maps)
	if err != nil {
		return nil, err
	}
	return DecodeDql[[]T](results)
}

// Gamma executes a custom DQL query/mutation and decodes results into typed values.
func Gamma[T any](q QueryMut, maps map[string]string) ([]T, error) {
	results, err := GetDB().Gamma(q, maps)
	if err != nil {
		return nil, err
	}
	return DecodeDql[[]T](results)
}
