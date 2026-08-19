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
	"regexp"
	"strconv"
	"strings"

	"github.com/dgraph-io/dgo/v200/protos/api"

	"fractale/fractal6.go/graph/model"
	. "fractale/fractal6.go/internal/tools"
)

//
// DQL requests
//

// Count counts the number of objects in fieldName attribute for given type and id.
// Returns: int or -1 if nothing is found.
func (dg Dgraph) Count(id string, fieldName string) int {
	res, err := dg.QueryDql("count", map[string]string{
		"id": id, "fieldName": fieldName,
	})
	if err != nil {
		log.Printf("Error in db.Count: %v", err)
		return -1
	}
	return unmarshalCountResp(res)
}

func (dg Dgraph) Count2(f1, v1, f2, v2, fieldName string) int {
	res, err := dg.QueryDql("count2", map[string]string{
		"f1": f1, "v1": v1, "f2": f2, "v2": v2, "fieldName": fieldName,
	})
	if err != nil {
		log.Printf("Error in db.Count2: %v", err)
		return -1
	}
	return unmarshalCountResp(res)
}

func (dg Dgraph) CountHas(fieldName string) int {
	res, err := dg.QueryDql("countHas", map[string]string{
		"fieldName": fieldName,
	})
	if err != nil {
		log.Printf("Error in db.CountHas: %v", err)
		return -1
	}
	return unmarshalCountResp(res)
}

func (dg Dgraph) CountHas2(fieldName, f2, v2 string) int {
	res, err := dg.QueryDql("countHas2", map[string]string{
		"fieldName": fieldName, "f2": f2, "v2": v2,
	})
	if err != nil {
		log.Printf("Error in db.CountHas2: %v", err)
		return -1
	}
	return unmarshalCountResp(res)
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
		res, err = dg.UpsertDql(dqlMutations[f], maps)
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
	res, err := dg.UpsertDql(q, maps)
	if res == nil {
		return nil, err
	}
	return decodeDqlResp(res)
}

// Probe if an object exists.
func (dg Dgraph) Exists(fieldName string, value string, filter *string) (bool, error) {
	maps := map[string]string{
		"fieldName": fieldName,
		"value":     value,
		"filter":    "",
	}
	if filter != nil {
		maps["filter"] = fmt.Sprintf(`@filter(%s)`, *filter)
	}
	res, err := dg.QueryDql("exists", maps)
	if err != nil {
		return false, err
	}
	r, err := unmarshalDqlResp(res)
	if err != nil {
		return false, err
	}
	return len(r.All) > 0, nil
}

// GetNodeVisibilities returns a {nameid -> visibility} map for the given
// nameids using a DQL query that bypasses @auth. Callers must perform their
// own visibility / membership check.
func (dg Dgraph) GetNodeVisibilities(nameids []string) (map[string]model.NodeVisibility, error) {
	out := map[string]model.NodeVisibility{}
	if len(nameids) == 0 {
		return out, nil
	}
	res, err := dg.QueryDql("getNodeVisibilities", map[string]string{
		"nameids": quoteNameids(nameids),
	})
	if err != nil {
		return nil, err
	}
	return decodeNodeVisibilities(res)
}

// GetSubNodeVisibilities returns a {nameid -> visibility} map for every Circle
// in the subtree rooted at (fieldid, objid). Bypasses @auth — callers must
// classify visibility themselves (see auth.ClassifyVisibleNameids).
func (dg Dgraph) GetSubNodeVisibilities(fieldid, objid string, includeSelf bool) (map[string]model.NodeVisibility, error) {
	return dg.queryNodeVisibilities("getSubNodeVisibilities", fieldid, objid, includeSelf)
}

// GetTopNodeVisibilities returns a {nameid -> visibility} map for every Circle
// in the ancestor chain of (fieldid, objid). Bypasses @auth.
func (dg Dgraph) GetTopNodeVisibilities(fieldid, objid string, includeSelf bool) (map[string]model.NodeVisibility, error) {
	return dg.queryNodeVisibilities("getTopNodeVisibilities", fieldid, objid, includeSelf)
}

func (dg Dgraph) queryNodeVisibilities(query, fieldid, objid string, includeSelf bool) (map[string]model.NodeVisibility, error) {
	res, err := dg.QueryDql(query, map[string]string{
		"fieldid":     fieldid,
		"objid":       objid,
		"excludeSelf": excludeSelfFlag(includeSelf),
	})
	if err != nil {
		return nil, err
	}
	return decodeNodeVisibilities(res)
}

func decodeNodeVisibilities(res *api.Response) (map[string]model.NodeVisibility, error) {
	out := map[string]model.NodeVisibility{}
	r, err := unmarshalDqlResp(res)
	if err != nil {
		return nil, err
	}
	for _, m := range r.All {
		nameid, _ := m["Node.nameid"].(string)
		vis, _ := m["Node.visibility"].(string)
		if nameid != "" {
			out[nameid] = model.NodeVisibility(vis)
		}
	}
	return out, nil
}

func quoteNameids(nameids []string) string {
	quoted := make([]string, len(nameids))
	for i, n := range nameids {
		quoted[i] = strconv.Quote(n)
	}
	return strings.Join(quoted, ",")
}

// GetMembersIn returns members (Roles with first_link) attached to circles
// in the given visible-nameids list.
func (dg Dgraph) GetMembersIn(nameids []string, userPayload string) ([]model.Node, error) {
	if len(nameids) == 0 {
		return []model.Node{}, nil
	}
	results, err := dg.Meta("getMembersByNameids", map[string]string{
		"nameids":      quoteNameids(nameids),
		"user_payload": userPayload,
	})
	if err != nil {
		return nil, err
	}
	return DecodeDql[[]model.Node](results)
}

// fetchByNameids runs a Meta query parameterized only by the visible-nameids
// list and decodes the result into []T. If dedupeKey is non-nil, results are
// de-duplicated by its return value.
func fetchByNameids[T any](dg Dgraph, queryName string, nameids []string, dedupeKey func(T) string) ([]T, error) {
	if len(nameids) == 0 {
		return []T{}, nil
	}
	results, err := dg.Meta(queryName, map[string]string{
		"nameids": quoteNameids(nameids),
	})
	if err != nil {
		return nil, err
	}
	data, err := DecodeDql[[]T](results)
	if err != nil {
		return nil, err
	}
	if dedupeKey == nil {
		return data, nil
	}
	return Dedupe(data, dedupeKey), nil
}

// fetchTopByNameids is fetchByNameids for "top" queries that distinguish self
// vs ancestors via (fieldid, objid).
func fetchTopByNameids[T any](dg Dgraph, queryName string, nameids []string, objid string, dedupeKey func(T) string) ([]T, error) {
	if len(nameids) == 0 {
		return []T{}, nil
	}
	results, err := dg.Meta(queryName, map[string]string{
		"nameids": quoteNameids(nameids),
		"fieldid": "nameid",
		"objid":   objid,
	})
	if err != nil {
		return nil, err
	}
	data, err := DecodeDql[[]T](results)
	if err != nil {
		return nil, err
	}
	if dedupeKey == nil {
		return data, nil
	}
	return Dedupe(data, dedupeKey), nil
}

// GetLabelsIn returns labels attached to circles in the given visible-nameids list.
// Labels deduplicated by name. objid is unused; present to match ArtefactFetcher.
func (dg Dgraph) GetLabelsIn(nameids []string, _ string) ([]model.Label, error) {
	return fetchByNameids(dg, "getLabelsByNameids", nameids, func(l model.Label) string { return l.Name })
}

// GetRolesIn returns role templates attached to circles in the given visible-nameids list.
// Roles deduplicated by name. objid is unused; present to match ArtefactFetcher.
func (dg Dgraph) GetRolesIn(nameids []string, _ string) ([]model.RoleExt, error) {
	return fetchByNameids(dg, "getRolesByNameids", nameids, func(r model.RoleExt) string { return r.Name })
}

// GetTensionTemplatesIn returns all tension templates attached to circles in
// the given visible-nameids list. Used by /q/tension_templates/sub. objid is
// unused; present to match ArtefactFetcher.
func (dg Dgraph) GetTensionTemplatesIn(nameids []string, _ string) ([]model.TensionTemplate, error) {
	return fetchByNameids(dg, "getTensionTemplatesByNameids", nameids, func(t model.TensionTemplate) string { return t.ID })
}

// GetTopTensionTemplatesIn returns templates inherited from ancestors
// (is_recursive=true only) plus all templates from self (when self is in
// the visible set). Used by /q/tension_templates/top.
func (dg Dgraph) GetTopTensionTemplatesIn(nameids []string, objid string) ([]model.TensionTemplate, error) {
	return fetchTopByNameids(dg, "getTopTensionTemplatesByNameids", nameids, objid, func(t model.TensionTemplate) string { return t.ID })
}

// GetProjectTemplatesIn returns all project templates attached to circles in
// the given visible-nameids list. Used by /q/project_templates/sub. objid is
// unused; present to match ArtefactFetcher.
func (dg Dgraph) GetProjectTemplatesIn(nameids []string, _ string) ([]model.ProjectTemplate, error) {
	return fetchByNameids(dg, "getProjectTemplatesByNameids", nameids, func(t model.ProjectTemplate) string { return t.ID })
}

// GetTopProjectTemplatesIn returns templates inherited from ancestors
// (is_recursive=true only) plus all templates from self (when self is in
// the visible set). Used by /q/project_templates/top.
func (dg Dgraph) GetTopProjectTemplatesIn(nameids []string, objid string) ([]model.ProjectTemplate, error) {
	return fetchTopByNameids(dg, "getTopProjectTemplatesByNameids", nameids, objid, func(t model.ProjectTemplate) string { return t.ID })
}

// GetProjectsIn returns projects attached to circles in the given visible-nameids list.
// objid is unused; present to match ArtefactFetcher.
func (dg Dgraph) GetProjectsIn(nameids []string, _ string) ([]ProjectFull, error) {
	return fetchByNameids[ProjectFull](dg, "getProjectsByNameids", nameids, nil)
}

// IsChild returns true is a node parent has the given child.
func (dg Dgraph) IsChild(parent, child string) (bool, error) {
	res, err := dg.QueryDql("isChild", map[string]string{
		"parent": parent,
		"child":  child,
	})
	if err != nil {
		return false, err
	}
	r, err := unmarshalDqlResp(res)
	if err != nil {
		return false, err
	}
	return len(r.All) > 0, nil
}

// Returns the uids of the objects if found.
func (dg Dgraph) GetIDs(fieldName string, value string, filterName, filterValue *string) ([]string, error) {
	maps := map[string]string{
		"fieldName": fieldName,
		"value":     value,
		"filter":    "",
	}
	if filterName != nil {
		maps["filter"] = fmt.Sprintf(`@filter(eq(%s, "%s"))`, *filterName, *filterValue)
	}
	res, err := dg.QueryDql("getID", maps)
	if err != nil {
		return nil, err
	}
	r, err := unmarshalDqlResp(res)
	if err != nil {
		return nil, err
	}
	result := make([]string, 0, len(r.All))
	for _, x := range r.All {
		result = append(result, x["uid"].(string))
	}
	return result, nil
}

var uidRe = regexp.MustCompile(`^0x[0-9a-fA-F]+$`)

// ValidateUids rejects ids that are not well-formed Dgraph uids, so
// client-supplied ids never reach a DQL uid(...) root (parse error / injection).
func ValidateUids(ids ...string) error {
	for _, id := range ids {
		if !uidRe.MatchString(id) {
			return fmt.Errorf("invalid id: %q", id)
		}
	}
	return nil
}

// GetByUid fetches a value at `path` under the node with the given uid.
// Path elements use "Type.field" notation; the leaf may be a space-separated
// multi-field selection (e.g. "uid Tension.title"), in which case the parent
// map is returned in lieu of a scalar. Cardinality-many edges fan out the
// result into a slice.
func (dg Dgraph) GetByUid(uid string, path ...string) (any, error) {
	if err := ValidateUids(uid); err != nil {
		return nil, err
	}
	return dg.runPathQuery(fmt.Sprintf(`uid("%s")`, uid), "", path)
}

// GetByEq fetches a value at `path` under nodes matching predicate=value.
func (dg Dgraph) GetByEq(predicate, value string, path ...string) (any, error) {
	return dg.runPathQuery(fmt.Sprintf(`eq(%s, "%s")`, predicate, value), "", path)
}

// GetByEqFiltered is GetByEq with an extra @filter(eq(filterPred, filterValue)).
func (dg Dgraph) GetByEqFiltered(predicate, value, filterPred, filterValue string, path ...string) (any, error) {
	filter := fmt.Sprintf(`@filter(eq(%s, "%s"))`, filterPred, filterValue)
	return dg.runPathQuery(fmt.Sprintf(`eq(%s, "%s")`, predicate, value), filter, path)
}

func (dg Dgraph) runPathQuery(root, filter string, path []string) (any, error) {
	if len(path) == 0 {
		return nil, fmt.Errorf("path query: empty path")
	}
	res, err := dg.runDqlTxn(buildPathQuery(root, filter, path), nil)
	if err != nil {
		return nil, err
	}
	cleaned, err := decodeDqlResp(res)
	if err != nil {
		return nil, err
	}
	return DecodeAt(cleaned, path...)
}

// buildPathQuery renders `{ all(func: <root>) <filter>? { p0 { p1 { ... } } } }`.
func buildPathQuery(root, filter string, path []string) string {
	var b strings.Builder
	b.WriteString("{ all(func: ")
	b.WriteString(root)
	b.WriteString(") ")
	if filter != "" {
		b.WriteString(filter)
		b.WriteByte(' ')
	}
	b.WriteString("{ ")
	for i, p := range path {
		b.WriteString(p)
		if i < len(path)-1 {
			b.WriteString(" { ")
		}
	}
	for range path {
		b.WriteString(" }")
	}
	b.WriteString(" }")
	return b.String()
}

func (dg Dgraph) GetShortestPath(from string, to string) (float64, error) {
	res, err := dg.QueryDql("getShortestPath", map[string]string{
		"from": from, "to": to,
	})
	if err != nil {
		return 0, err
	}
	r, err := unmarshalDqlResp(res)
	if err != nil {
		return 0, err
	}

	if len(r.All) > 1 {
		return 0, fmt.Errorf("Got multiple in DQL query")
	}
	if len(r.All) == 1 {
		weight, ok := r.All[0]["weight"].(float64)
		if !ok {
			return 0, fmt.Errorf("Cannot extract weight from shortest path query")
		}
		return weight, nil
	}
	return 0, nil
}

// Returns the user context
func (dg Dgraph) GetUctxFull(fieldid string, userid string) (*model.UserCtx, error) {
	res, err := dg.QueryDql("getUser", map[string]string{
		"fieldid": fieldid, "userid": userid, "payload": userCtxPayload,
	})
	if err != nil {
		return nil, err
	}
	r, err := unmarshalDqlResp(res)
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
	maps := map[string]string{
		"regex": regex,
		"payload": `{
            Node.nameid
            Node.visibility
        }`,
	}

	var op string
	if isRoot {
		op = "getNodesRoot"
	} else {
		op = "getNodes"
	}
	res, err := dg.QueryDql(op, maps)
	if err != nil {
		return nil, err
	}
	r, err := unmarshalDqlResp(res)
	if err != nil {
		return nil, err
	}
	return DecodeDql[[]model.Node](r.All)
}

// Returns the tension hook content
func (dg Dgraph) GetTensionHook(tid string, withBlob bool, bid *string) (*model.Tension, error) {
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

	res, err := dg.QueryDql("getTensionHook", maps)
	if err != nil {
		return nil, err
	}
	r, err := unmarshalDqlResp(res)
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
	maps := map[string]string{
		"id":      cid,
		"payload": "{" + contractHookPayload + "}",
	}

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
	r, err := unmarshalDqlResp(res)
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

// GetSubMembers returns all members in the subtree of (fieldid, objid).
// Used by graph/notifications.go for system-level dispatch (no auth check).
// /q/members/sub uses the visibility-aware GetMembersIn instead.
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

// ProjectFull is a lightweight project representation for the /q/projects/sub endpoint.
type ProjectFull struct {
	ID            string        `json:"id"`
	UpdatedAt     string        `json:"updatedAt,omitempty"`
	Name          string        `json:"name"`
	Description   *string       `json:"description,omitempty"`
	Nodes         []*model.Node `json:"nodes,omitempty"`
	Collaborators []*model.User `json:"collaborators,omitempty"`
}

const tensionListPayload = `uid
        Post.createdAt
        Post.createdBy { User.username }
        Tension.receiver { Node.nameid Node.name Node.role_type }
        Tension.emitter { Node.nameid Node.name Node.role_type }
        Tension.title
        Tension.status
        Tension.type_
        Tension.labels { uid Label.name Label.color }
        Tension.governed_node { uid Node.nameid Node.type_ Node.isArchived }
        Tension.blobs (orderdesc: Post.createdAt, first: 1) {
            Blob.node { NodeFragment.type_ }
        }
        n_comments: count(Tension.comments)`

const tensionLightPayload = `uid
        Tension.title
        Tension.status
        Tension.type_
        Tension.labels { uid Label.name Label.color }
        Tension.receiver { Node.nameid Node.name Node.role_type }`

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
	switch type_ {
	case "light":
		op = "getTensionInt"
		isLight = true
	case "int":
		op = "getTensionInt"
	case "ext":
		op = "getTensionExt"
	case "all":
		op = "getTensionAll"
	default:
		panic("Unknow type (tension query)")
	}

	payload = tensionListPayload
	if isLight {
		payload = tensionLightPayload
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
	r, err := unmarshalDqlResp(res)
	if err != nil {
		return nil, err
	}
	return DecodeDql[[]model.TensionRef](r.All)
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
	res, err := dg.QueryDql("getLastBlobId", map[string]string{"tid": tid})
	if err != nil {
		return nil
	}
	r, err := unmarshalDqlResp(res)
	if err != nil || len(r.All) != 1 {
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
	res, err := dg.QueryDql("getCoordos", map[string]string{"nameid": nameid})
	if err != nil {
		return false
	}
	r, err := unmarshalDqlResp(res)
	if err != nil || len(r.All) != 1 {
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
	res, err := dg.QueryDql("getChildren", map[string]string{"nameid": nameid})
	if err != nil {
		return nil, err
	}
	r, err := unmarshalDqlResp(res)
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
	res, err := dg.QueryDql("getParents", map[string]string{"nameid": nameid})
	if err != nil {
		return nil, err
	}
	r, err := unmarshalDqlResp(res)
	if err != nil {
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
	res, err := dg.QueryDql("getTensionSearchData", map[string]string{"tid": tid})
	if err != nil {
		return nil, "", err
	}
	r, err := unmarshalDqlResp(res)
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

// SetPushedFlagBlob sets the blob pushed flag.
func (dg Dgraph) SetPushedFlagBlob(bid, flag string) error {
	if err := ValidateUids(bid); err != nil {
		return err
	}
	_, err := dg.Meta("setPushedFlagBlob", map[string]string{"bid": bid, "flag": flag})
	return err
}

// LinkGovernedNode atomically establishes Node.source and Tension.governed_node.
func (dg Dgraph) LinkGovernedNode(tid, nid, bid string) error {
	if err := ValidateUids(tid, nid, bid); err != nil {
		return err
	}
	_, err := dg.Meta("linkGovernedNode", map[string]string{"tid": tid, "nid": nid, "bid": bid})
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

//
// Generic DQL helpers
//

// unmarshalDqlResp unmarshals an api.Response into raw DQL result maps (no cleaning).
// Use this when passing results to DecodeDql which applies CleanDqlMap internally.
func unmarshalDqlResp(res *api.Response) (DqlResp, error) {
	var r DqlResp
	if res == nil {
		return r, nil
	}
	if err := json.Unmarshal(res.Json, &r); err != nil {
		return r, err
	}
	return r, nil
}

// unmarshalCountResp unmarshals an api.Response into a count result
// and returns the first count value, or -1 if empty/error.
func unmarshalCountResp(res *api.Response) int {
	var r DqlRespCount
	if err := json.Unmarshal(res.Json, &r); err != nil {
		return -1
	}
	if len(r.All) == 0 {
		return -1
	}
	for _, v := range r.All[0] {
		return v
	}
	return -1
}

// DecodeDqlBlock decodes a named result block of a DQL response directly into
// []T using T's json tags — no CleanDqlMap pass, no re-marshal. Use when the
// result has a fixed shape and you want to skip the generic-map path taken by
// Meta/Gamma. block defaults to "all" when empty. Returns (nil, nil) when the
// response is empty or the block is absent.
//
// Typical T uses Dgraph-native keys, e.g. `json:"File.storageKey"`.
func DecodeDqlBlock[T any](res *api.Response, block string) ([]T, error) {
	if res == nil || len(res.Json) == 0 {
		return nil, nil
	}
	if block == "" {
		block = "all"
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(res.Json, &raw); err != nil {
		return nil, err
	}
	blk, ok := raw[block]
	if !ok || len(blk) == 0 {
		return nil, nil
	}
	var out []T
	if err := json.Unmarshal(blk, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// decodeDqlResp decodes an api.Response into cleaned DQL result maps.
// Used by Meta()/Gamma() which return pre-cleaned maps.
func decodeDqlResp(res *api.Response) ([]map[string]any, error) {
	r, err := unmarshalDqlResp(res)
	if err != nil || len(r.All) == 0 {
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
