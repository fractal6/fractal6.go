package db

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/dgraph-io/dgo/v200/protos/api"
	"github.com/mitchellh/mapstructure"

	"zerogov/fractal6.go/graph/model"
	"zerogov/fractal6.go/tools"
)


var tensionHookPayload string = `
  uid
  Post.createdBy { User.username }
  Tension.action
  Tension.emitter {
    Node.nameid
    Node.role_type
    Node.rights
  }
  Tension.receiver {
    Node.nameid
    Node.visibility
    Node.mode
    Node.userCanJoin
  }
`
var tensionBlobHookPayload string = `
  Tension.blobs %s {
    uid
    Blob.blob_type
    Blob.md
    Blob.node {
      uid
      NodeFragment.name
      NodeFragment.nameid
      NodeFragment.type_
      NodeFragment.about
      NodeFragment.mandate {
        expand(_all_)
      }

      NodeFragment.first_link
      NodeFragment.second_link
      NodeFragment.skills
      NodeFragment.role_type

      NodeFragment.children {
        NodeFragment.first_link
        NodeFragment.role_type
      }
    }

  }
`

var contractHookPayload string = `{
  uid
  Contract.tension { uid }
  Contract.status
  Contract.contract_type
  Contract.event {
    EventFragment.event_type
    EventFragment.old
    EventFragment.new
  }
  Contract.candidates { User.username }
  Contract.participants { Vote.data Vote.Node { Node.nameid } }
}`

// GPRC/DQL Request Template
var dqlQueries map[string]string = map[string]string{
    // Count objects
    "count": `{
        all(func: uid("{{.id}}")) {
            count({{.fieldName}})
        }
    }`,
    "getOrgaAgg": `{
        var(func: eq(Node.nameid, "{{.nameid}}"))  {
            Node.children @filter(eq(Node.role_type, "Guest")) {
                guest as count(uid)
            }
        }
        var(func: eq(Node.nameid, "{{.nameid}}"))  {
            Node.children @filter(eq(Node.role_type, "Member")) {
                member as count(uid)
            }
        }
        all() {
            n_members: sum(val(member))
            n_guests: sum(val(guest))
        }
    }`,
    // Get the total number of roles and circle recursively
    //    var(func: eq(Node.nameid, "{{.nameid}}")) @recurse {
    //        c as Node.children @filter(NOT eq(Node.isArchived, true))
    //    }
    //    var(func: uid(c)) @filter(eq(Node.type_, "Circle")) {
    //        circle as count(uid)
    //    }
    //    var(func: uid(c)) @filter(eq(Node.type_, "Role")) {
    //        role as count(uid)
    //    }

    //    all() {
    //        n_member: sum(val(member))
    //        n_guest: sum(val(guest))
    //        n_role: sum(val(role))
    //        n_circle: sum(val(circle))
    //    }
    //}`,
    // Get literal value
    "getID": `{
        all(func: eq({{.fieldName}}, "{{.value}}")) {{.filter}} { uid }
    }`,
    "getFieldById": `{
        all(func: uid("{{.id}}")) {
            {{.fieldName}}
        }
    }`,
    "getFieldByEq": `{
        all(func: eq({{.fieldid}}, "{{.value}}")) {
            {{.fieldName}}
        }
    }`,
    "getSubFieldById": `{
        all(func: uid("{{.id}}")) {
            {{.fieldNameSource}} {
                {{.fieldNameTarget}}
            }
        }
    }`,
    "getSubFieldByEq": `{
        all(func: eq({{.fieldid}}, "{{.value}}")) {
            {{.fieldNameSource}} {
                {{.fieldNameTarget}}
            }
        }
    }`,
    "getSubSubFieldById": `{
        all(func: uid({{.id}})) {
            {{.fieldNameSource}} {
                {{.fieldNameTarget}} {
                    {{.subFieldNameTarget}}
                }
            }
        }
    }`,
    "getSubSubFieldByEq": `{
        all(func: eq({{.fieldid}}, "{{.value}}")) {
            {{.fieldNameSource}} {
                {{.fieldNameTarget}} {
                    {{.subFieldNameTarget}}
                }
            }
        }
    }`,
    "getUser": `{
        all(func: eq(User.{{.fieldid}}, "{{.userid}}"))
        {{.payload}}
    }`,
    "getNode": `{
        all(func: eq(Node.{{.fieldid}}, "{{.objid}}"))
        {{.payload}}
    }`,
    "getNodes": `{
        all(func: regexp(Node.nameid, /{{.regex}}/))
        {{.payload}}
    }`,
    "getNodesRoot": `{
        all(func: regexp(Node.nameid, /{{.regex}}/)) @filter(eq(Node.isRoot, true))
        {{.payload}}
    }`,
    "getTensionHook": `{
        all(func: uid("{{.id}}"))
        {{.payload}}
    }`,
    "getContractHook": `{
        all(func: eq(Contract.contractid, "{{.id}}"))
        {{.payload}}
    }`,
    // Boolean
    "exists": `{
        all(func: eq({{.fieldName}}, "{{.value}}")) {{.filter}} { uid }
    }`,
    "isChild": `{
        var(func: eq(Node.nameid, "{{.parent}}")) @recurse {
            uid
            u as Node.children @filter(eq(Node.nameid, "{{.child}}"))
        }
        all(func: uid(u)) { uid }
    }`,
    // Get multiple objects
    "getChildren": `{
        all(func: eq(Node.nameid, "{{.nameid}}"))  {
            Node.children @filter(NOT eq(Node.isArchived, true)) {
                Node.nameid
            }
        }
    }`,
    "getParents": `{
        all(func: eq(Node.nameid, "{{.nameid}}")) @recurse {
            Node.parent @normalize
            Node.nameid
        }
    }`,
    "getSubNodes": `{
        var(func: eq(Node.{{.fieldid}}, "{{.objid}}")) @recurse {
            o as Node.children
        }

        all(func: uid(o)) @filter(NOT eq(Node.isArchived, true)) {
            Node.{{.fieldid}}
        }
    }`,
    "getSubMembers": `{
        var(func: eq(Node.{{.fieldid}}, "{{.objid}}")) @recurse {
            o as Node.children
        }

        all(func: uid(o)) @filter(has(Node.role_type) AND NOT eq(Node.isArchived, true)) {
            Node.createdAt
            Node.name
            Node.nameid
            Node.role_type
            Node.first_link {
                User.username
                User.name
            }
            Node.parent {
                Node.nameid
            }
        }
    }`,
    "getTopLabels": `{
        var(func: eq(Node.{{.fieldid}}, "{{.objid}}")) @recurse {
            o as uid
            Node.parent @normalize
        }

        var(func: uid(o)) @filter(NOT eq(Node.isArchived, true) AND NOT eq(Node.{{.fieldid}}, "{{.objid}}")) {
            l as Node.labels
        }

        all(func: uid(l)){
            uid
            Label.name
            Label.color
            Label.description
            n_nodes: count(Label.nodes)
        }
    }`,
    "getSubLabels": `{
        var(func: eq(Node.{{.fieldid}}, "{{.objid}}")) @recurse {
            o as Node.children
        }

        var(func: uid(o)) @filter(NOT eq(Node.isArchived, true)) {
            l as Node.labels
        }

        all(func: uid(l)){
            uid
            Label.name
            Label.color
            Label.description
            n_nodes: count(Label.nodes)
        }
    }`,
    "getTensionInt": `{
        var(func: eq(Node.rootnameid, "{{.rootnameid}}")) @filter({{.nameids}}) {
            tensions as Node.tensions_in {{.tensionFilter}} @cascade {
                Tension.emitter @filter({{.nameids}})
                {{.authorsFilter}}
                {{.labelsFilter}}
            }
        }

        all(func: uid(tensions), first:{{.first}}, offset:{{.offset}}, orderdesc: Post.createdAt) {
            uid
            Post.createdAt
            Post.createdBy { User.username }
            Tension.receiver { Node.nameid Node.name Node.role_type }
            Tension.emitter { Node.nameid Node.name Node.role_type }
            Tension.title
            Tension.status
            Tension.type_
            Tension.action
            Tension.labels { uid Label.name Label.color }
            n_comments: count(Tension.comments)
        }
    }`,
    "getTensionExt": `{
        var(func: eq(Node.rootnameid, "{{.rootnameid}}")) @filter({{.nameids}}) {
            tensions_in as Node.tensions_in {{.tensionFilter}} @cascade {
                Tension.emitter @filter(NOT ({{.nameids}}))
                {{.authorsFilter}}
                {{.labelsFilter}}
            }
            tensions_out as Node.tensions_out {{.tensionFilter}} @cascade {
                Tension.receiver @filter(NOT ({{.nameids}}))
                {{.authorsFilter}}
                {{.labelsFilter}}
            }
        }

        all(func: uid(tensions_in, tensions_out), first:{{.first}}, offset:{{.offset}}, orderdesc: Post.createdAt) {
            uid
            Post.createdAt
            Post.createdBy { User.username }
            Tension.receiver { Node.nameid Node.name Node.role_type }
            Tension.emitter { Node.nameid Node.name Node.role_type }
            Tension.title
            Tension.status
            Tension.type_
            Tension.action
            Tension.labels { uid Label.name Label.color }
            n_comments: count(Tension.comments)
        }
    }`,
    "getTensionAll": `{
        var(func: eq(Node.rootnameid, "{{.rootnameid}}")) @filter({{.nameids}}) {
            tensions_in as Node.tensions_in {{.tensionFilter}} @cascade {
                uid
                {{.authorsFilter}}
                {{.labelsFilter}}
            }
            tensions_out as Node.tensions_out {{.tensionFilter}} @cascade {
                uid
                {{.authorsFilter}}
                {{.labelsFilter}}
            }
        }

        all(func: uid(tensions_in, tensions_out), first:{{.first}}, offset:{{.offset}}, orderdesc: Post.createdAt) {
            uid
            Post.createdAt
            Post.createdBy { User.username }
            Tension.receiver { Node.nameid Node.name Node.role_type }
            Tension.emitter { Node.nameid Node.name Node.role_type }
            Tension.title
            Tension.status
            Tension.type_
            Tension.action
            Tension.labels { uid Label.name Label.color }
            n_comments: count(Tension.comments)
        }
    }`,
    "getTensionCount": `{
        var(func: eq(Node.rootnameid, "{{.rootnameid}}")) @filter({{.nameids}}) {
            tensions_in as Node.tensions_in {{.tensionFilter}} @cascade {
                uid
                {{.authorsFilter}}
                {{.labelsFilter}}
            }
            tensions_out as Node.tensions_out {{.tensionFilter}} @cascade {
                uid
                {{.authorsFilter}}
                {{.labelsFilter}}
            }
        }

        all(func: uid(tensions_in, tensions_out)) @filter(eq(Tension.status, "Open")) {
            count: count(uid)
        }
        all2(func: uid(tensions_in, tensions_out)) @filter(eq(Tension.status, "Closed")) {
            count: count(uid)
        }
    }`,
    "getCoordos": `{
        all(func: eq(Node.nameid, "{{.nameid}}")) {
            Node.children @filter(eq(Node.role_type, "Coordinator") AND NOT eq(Node.isArchived, true)) { uid }
        }
    }`,
    // Mutations
    "deleteTension": `{
        id as var(func: uid({{.id}})) {
          rid_emitter as Tension.emitter
          rid_receiver as Tension.receiver
          a as Tension.comments
          b as Tension.blobs {
              bb as Blob.node {
                  bb1 as NodeFragment.children
                  bb2 as NodeFragment.mandate
              }
          }
          c as Tension.contracts
          d as Tension.history
        }
        all(func: uid(id,a,b,c,d,bb,bb1,bb2)) {
            all_ids as uid
        }
    }`,
    "deleteContract": `{
        id as var(func: uid({{.id}})) {
          rid as Contract.tension
          a as Contract.event
          b as Contract.participants
          c as Contract.comments
        }
        all(func: uid(id,a,b,c)) {
            all_ids as uid
        }
    }`,
}

//
// Gprc/DQL requests
//

// Count count the number of object in fieldName attribute for given type and id
// Returns: int or -1 if nothing is found.
func (dg Dgraph) Count(id string, fieldName string) int {
    // Format Query
    maps := map[string]string{
        "id":id, "fieldName":fieldName,
    }
    // Send request
    res, err := dg.QueryDql("count", maps)
    if err != nil { panic(err) }

    // Decode response
    var r DqlRespCount
    err = json.Unmarshal(res.Json, &r)
    if err != nil { panic(err) }

    // Extract result
    if len(r.All) == 0 { return -1 }

    values := make([]int, 0, len(r.All[0]))
    for _, v := range r.All[0] {
        values = append(values, v)
    }

    return values[0]
}

func (dg Dgraph) Meta(k, v, f string) map[string]interface{} {
    // Format Query
    maps := map[string]string{k: v}
    // Send request
    res, err := dg.QueryDql(f, maps)
    if err != nil { panic(err) }

    // Decode response
    var r DqlResp
    err = json.Unmarshal(res.Json, &r)
    if err != nil { panic(err) }

    // Extract result
    if len(r.All) == 0 {
        panic("no result for: "+ k)
    }
    agg := make(map[string]interface{}, len(r.All))
    for _, s := range r.All {
        for n, m := range s {
            agg[n] = m
        }
    }
    return agg
}

// Probe if an object exists.
func (dg Dgraph) Exists(fieldName string, value string, filterName, filterValue *string) (bool, error) {
    // Format Query
    maps := map[string]string{
        "fieldName": fieldName,
        "value": value,
        "filter": "",
    }
    if filterName != nil {
        maps["filter"] = fmt.Sprintf(`@filter(eq(%s, "%s"))`, *filterName, *filterValue )
    }
    // Send request
    res, err := dg.QueryDql("exists", maps)
    if err != nil { return false, err }
    // Decode response
    var r DqlResp
    err = json.Unmarshal(res.Json, &r)
    if err != nil { return false, err }
    return len(r.All) > 0, nil
}

//IsChild returns true is a node parent has the given child.
func (dg Dgraph) IsChild(parent, child string) (bool, error) {
    // Format Query
    maps := map[string]string{
        "parent": parent,
        "child": child,
    }
    // Send request
    res, err := dg.QueryDql("isChild", maps)
    if err != nil { return false, err }
    // Decode response
    var r DqlResp
    err = json.Unmarshal(res.Json, &r)
    if err != nil { return false, err }
    return len(r.All) > 0, nil
}

// Returns the uids of the objects if found.
func (dg Dgraph) GetIDs(fieldName string, value string, filterName, filterValue *string) ([]string, error) {
    result := []string{}
    // Format Query
    maps := map[string]string{
        "fieldName":fieldName,
        "value": value,
        "filter": "",
    }
    if filterName != nil {
        maps["filter"] = fmt.Sprintf(`@filter(eq(%s, "%s"))`, *filterName, *filterValue )
    }
    // Send request
    res, err := dg.QueryDql("getID", maps)
    if err != nil { return result, err }
    // Decode response
    var r DqlResp
    err = json.Unmarshal(res.Json, &r)
    if err != nil { return result, err }
    for _, x := range r.All {
        result = append(result, x["uid"].(string))
    }
    return result, nil
}

// Returns a field from id
func (dg Dgraph) GetFieldById(id string, fieldName string) (interface{}, error) {
    // Format Query
    maps := map[string]string{
        "id": id,
        "fieldName": fieldName,
    }
    // Send request
    res, err := dg.QueryDql("getFieldById", maps)
    if err != nil { return nil, err }

    // Decode response
    var r DqlResp
    err = json.Unmarshal(res.Json, &r)
    if err != nil { return nil, err }

    if len(r.All) > 1 {
        return nil, fmt.Errorf("Got multiple in DQL query: %s %s", fieldName, id)
    } else if len(r.All) == 1 {
        x := r.All[0][fieldName]
        return x, nil
    }
    return nil, err
}

// Returns a field from objid
func (dg Dgraph) GetFieldByEq(fieldid string, objid string, fieldName string) (interface{}, error) {
    // Format Query
    maps := map[string]string{
        "fieldid": fieldid,
        "value": objid,
        "fieldName": fieldName,
    }
    // Send request
    res, err := dg.QueryDql("getFieldByEq", maps)
    if err != nil { return nil, err }

    // Decode response
    var r DqlResp
    err = json.Unmarshal(res.Json, &r)
    if err != nil { return nil, err }

    if len(r.All) == 0 {
        return nil, err
    } else if len(r.All) > 1 {
        return nil, fmt.Errorf("Only one resuts allowed for DQL query: %s %s", fieldName, objid)
    }
    x := r.All[0][fieldName]
    return x, err
}

// Returns a subfield from uid
func (dg Dgraph) GetSubFieldById(id string, fieldNameSource string, fieldNameTarget string) (interface{}, error) {
    // Format Query
    maps := map[string]string{
        "id": id,
        "fieldNameSource": fieldNameSource,
        "fieldNameTarget": fieldNameTarget,
    }
    // Send request
    res, err := dg.QueryDql("getSubFieldById", maps)
    if err != nil { return nil, err }

    // Decode response
    var r DqlResp
    err = json.Unmarshal(res.Json, &r)
    if err != nil { return nil, err }

    if len(r.All) > 1 {
        return nil, fmt.Errorf("Got multiple in DQL query")
    } else if len(r.All) == 1 {
        switch x := r.All[0][fieldNameSource].(type) {
        case model.JsonAtom:
            if x != nil {
                return x[fieldNameTarget], nil
            }
        case []interface{}:
            if x != nil {
                var y []interface{}
                for _, v := range(x) {
                    y = append(y, v.(model.JsonAtom)[fieldNameTarget])
                }
                return y, nil
            }
        default:
            return nil, fmt.Errorf("Decode type unknonwn: %T", x)
        }
    }
    return nil, err
}

// Returns a subfield from Eq
func (dg Dgraph) GetSubFieldByEq(fieldid string, value string, fieldNameSource string, fieldNameTarget string) (interface{}, error) {
    // Format Query
    maps := map[string]string{
        "fieldid":fieldid,
        "value":value,
        "fieldNameSource":fieldNameSource,
        "fieldNameTarget":fieldNameTarget,
    }
    // Send request
    res, err := dg.QueryDql("getSubFieldByEq", maps)
    if err != nil { return nil, err }

    // Decode response
    var r DqlResp
    err = json.Unmarshal(res.Json, &r)
    if err != nil { return nil, err }

    if len(r.All) > 1 {
        return nil, fmt.Errorf("Got multiple in DQL query")
    } else if len(r.All) == 1 {
        switch x := r.All[0][fieldNameSource].(type) {
        case model.JsonAtom:
            if x != nil {
                return x[fieldNameTarget], nil
            }
        case []interface{}:
            if x != nil {
                var y []interface{}
                for _, v := range(x) {
                    y = append(y, v.(model.JsonAtom)[fieldNameTarget])
                }
                return y, nil
            }
        default:
            return nil, fmt.Errorf("Decode type unknonwn: %T", x)
        }
    }
    return nil, err
}

// Returns a subsubfield from uid
func (dg Dgraph) GetSubSubFieldById(id string, fieldNameSource string, fieldNameTarget string, subFieldNameTarget string) (interface{}, error) {
    // Format Query
    maps := map[string]string{
        "id": id,
        "fieldNameSource": fieldNameSource,
        "fieldNameTarget": fieldNameTarget,
        "subFieldNameTarget": subFieldNameTarget,
    }
    // Send request
    res, err := dg.QueryDql("getSubSubFieldById", maps)
    if err != nil { return nil, err }

    // Decode response
    var r DqlResp
    err = json.Unmarshal(res.Json, &r)
    if err != nil { return nil, err }

    if len(r.All) > 1 {
        return nil, fmt.Errorf("Got multiple in DQL query")
    } else if len(r.All) == 1 {
        x := r.All[0][fieldNameSource].(model.JsonAtom)
        if x != nil {
            y := x[fieldNameTarget].(model.JsonAtom)
            if y != nil {
                return y[subFieldNameTarget], nil
            }
        }
    }
    return nil, err
}

// Returns a subsubfield from Eq
func (dg Dgraph) GetSubSubFieldByEq(fieldid string, value string, fieldNameSource string, fieldNameTarget string, subFieldNameTarget string) (interface{}, error) {
    // Format Query
    maps := map[string]string{
        "fieldid": fieldid,
        "value": value,
        "fieldNameSource": fieldNameSource,
        "fieldNameTarget": fieldNameTarget,
        "subFieldNameTarget": subFieldNameTarget,
    }
    // Send request
    res, err := dg.QueryDql("getSubSubFieldByEq", maps)
    if err != nil { return nil, err }

    // Decode response
    var r DqlResp
    err = json.Unmarshal(res.Json, &r)
    if err != nil { return nil, err }

    if len(r.All) > 1 {
        return nil, fmt.Errorf("Got multiple in DQL query")
    } else if len(r.All) == 1 {
        x := r.All[0][fieldNameSource].(model.JsonAtom)
        if x != nil {
            y := x[fieldNameTarget].(model.JsonAtom)
            if y != nil {
                return y[subFieldNameTarget], nil
            }
        }
    }
    return nil, err
}

// Returns the user context
func (dg Dgraph) GetUser(fieldid string, userid string) (*model.UserCtx, error) {
    // Format Query
    maps := map[string]string{
        "fieldid": fieldid,
        "userid": userid,
        "payload": model.UserCtxPayloadDg,
    }
    // Send request
    res, err := dg.QueryDql("getUser", maps)
    if err != nil { return nil, err }

    // Decode response
    var r DqlResp
    err = json.Unmarshal(res.Json, &r)
    if err != nil { return nil, err }

    var user model.UserCtx
    if len(r.All) > 1 {
        return nil, fmt.Errorf("Got multiple user with same @id: %s, %s", fieldid, userid)
    } else if len(r.All) == 1 {
        config := &mapstructure.DecoderConfig{
            Result: &user,
            TagName: "json",
            DecodeHook: func(from, to reflect.Kind, v interface{}) (interface{}, error) {
                if to == reflect.Struct {
                    nv := tools.CleanCompositeName(v.(map[string]interface{}))
                    return nv, nil
                }
                return v, nil
            },
        }
        decoder, err := mapstructure.NewDecoder(config)
        if err != nil { return nil, err }
        err = decoder.Decode(r.All[0])
        if err != nil { return nil, err }
    }
    // Filter special roles
    for i := 0; i < len(user.Roles); i++ {
        if *user.Roles[i].RoleType == model.RoleTypeRetired ||
        *user.Roles[i].RoleType == model.RoleTypePending {
            user.Roles = append(user.Roles[:i], user.Roles[i+1:]...)
            i--
        }
    }
    return &user, err
}

// Returns the matching nodes
func (dg Dgraph) GetNodes(regex string, isRoot bool) ([]model.Node, error) {
    // Format Query
    maps := map[string]string{
        "regex": regex,
        "payload": `{
            Node.nameid
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
    if err != nil { return nil, err }

    // Decode response
    var r DqlResp
    err = json.Unmarshal(res.Json, &r)
    if err != nil { return nil, err }

    var data []model.Node
    config := &mapstructure.DecoderConfig{
        Result: &data,
        TagName: "json",
        DecodeHook: func(from, to reflect.Kind, v interface{}) (interface{}, error) {
            if to == reflect.Struct {
                nv := tools.CleanCompositeName(v.(map[string]interface{}))
                return nv, nil
            }
            return v, nil
        },
    }
    decoder, err := mapstructure.NewDecoder(config)
    if err != nil { return nil, err }
    err = decoder.Decode(r.All)
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
            "id": tid,
            "payload": "{" + tensionHookPayload + fmt.Sprintf(tensionBlobHookPayload, blobFilter) + "}",
        }
    } else {
        maps = map[string]string{
            "id": tid,
            "payload": "{" + tensionHookPayload + "}",
        }
    }

    // Send request
    res, err := dg.QueryDql("getTensionHook", maps)
    if err != nil { return nil, err }

    // Decode response
    var r DqlResp
    err = json.Unmarshal(res.Json, &r)
    if err != nil { return nil, err }

    var obj model.Tension
    if len(r.All) > 1 {
        return nil, fmt.Errorf("Got multiple tension for @uid: %s", tid)
    } else if len(r.All) == 1 {
        config := &mapstructure.DecoderConfig{
            Result: &obj,
            TagName: "json",
            DecodeHook: func(from, to reflect.Kind, v interface{}) (interface{}, error) {
                if to == reflect.Struct {
                    nv := tools.CleanCompositeName(v.(map[string]interface{}))
                    return nv, nil
                }
                return v, nil
            },
        }
        decoder, err := mapstructure.NewDecoder(config)
        if err != nil { return nil, err }
        err = decoder.Decode(r.All[0])
        if err != nil { return nil, err }
    }

    // Assume that tension does not exists if receiver is empty
    // This is becausae DQL returns an object with t uid even is non existent.
    if obj.Receiver == nil { return nil, err }
    return &obj, err
}

// Returns the contract hook content
func (dg Dgraph) GetContractHook(cid string) (*model.Contract, error) {
    // Format Query
    maps := map[string]string{
        "id": cid,
        "payload": "{" + contractHookPayload + "}",
    }

    // Send request
    res, err := dg.QueryDql("getContractHook", maps)
    if err != nil { return nil, err }

    // Decode response
    var r DqlResp
    err = json.Unmarshal(res.Json, &r)
    if err != nil { return nil, err }

    var obj model.Contract
    if len(r.All) > 1 {
        return nil, fmt.Errorf("Got multiple contract for @uid: %s", cid)
    } else if len(r.All) == 1 {
        config := &mapstructure.DecoderConfig{
            Result: &obj,
            TagName: "json",
            DecodeHook: func(from, to reflect.Kind, v interface{}) (interface{}, error) {
                if to == reflect.Struct {
                    nv := tools.CleanCompositeName(v.(map[string]interface{}))
                    return nv, nil
                }
                return v, nil
            },
        }
        decoder, err := mapstructure.NewDecoder(config)
        if err != nil { return nil, err }
        err = decoder.Decode(r.All[0])
        if err != nil { return nil, err }
    }
    return &obj, err
}

// Get all sub children
func (dg Dgraph) GetSubNodes(fieldid string, objid string) ([]model.Node, error) {
    // Format Query
    maps := map[string]string{
        "fieldid": fieldid,
        "objid": objid,
    }
    // Send request
    res, err := dg.QueryDql("getSubNodes", maps)
    if err != nil { return nil, err }

    // Decode response
    var r DqlResp
    err = json.Unmarshal(res.Json, &r)
    if err != nil { return nil, err }

    var data []model.Node
    config := &mapstructure.DecoderConfig{
        Result: &data,
        TagName: "json",
        DecodeHook: func(from, to reflect.Kind, v interface{}) (interface{}, error) {
            if to == reflect.Struct {
                nv := tools.CleanCompositeName(v.(map[string]interface{}))
                return nv, nil
            }
            return v, nil
        },
    }
    decoder, err := mapstructure.NewDecoder(config)
    if err != nil { return nil, err }
    err = decoder.Decode(r.All)
    return data, err
}

// Get all sub members
func (dg Dgraph) GetSubMembers(fieldid string, objid string) ([]model.Node, error) {
    // Format Query
    maps := map[string]string{
        "fieldid": fieldid,
        "objid": objid,
    }
    // Send request
    res, err := dg.QueryDql("getSubMembers", maps)
    if err != nil { return nil, err }

    // Decode response
    var r DqlResp
    err = json.Unmarshal(res.Json, &r)
    if err != nil { return nil, err }

    var data []model.Node
    config := &mapstructure.DecoderConfig{
        Result: &data,
        TagName: "json",
        DecodeHook: func(from, to reflect.Kind, v interface{}) (interface{}, error) {
            if to == reflect.Struct {
                nv := tools.CleanCompositeName(v.(map[string]interface{}))
                return nv, nil
            }
            return v, nil
        },
    }
    decoder, err := mapstructure.NewDecoder(config)
    if err != nil { return nil, err }
    err = decoder.Decode(r.All)
    return data, err
}

// Get all top labels
func (dg Dgraph) GetTopLabels(fieldid string, objid string) ([]model.Label, error) {
    // Format Query
    maps := map[string]string{
        "fieldid": fieldid,
        "objid": objid,
    }
    // Send request
    res, err := dg.QueryDql("getTopLabels", maps)
    if err != nil { return nil, err }

    // Decode response
    var r DqlResp
    err = json.Unmarshal(res.Json, &r)
    if err != nil { return nil, err }

    var data_dup []model.Label
    config := &mapstructure.DecoderConfig{
        Result: &data_dup,
        TagName: "json",
        DecodeHook: func(from, to reflect.Kind, v interface{}) (interface{}, error) {
            if to == reflect.Struct {
                nv := tools.CleanCompositeName(v.(map[string]interface{}))
                return nv, nil
            }
            return v, nil
        },
    }
    decoder, err := mapstructure.NewDecoder(config)
    if err != nil { return nil, err }
    err = decoder.Decode(r.All)
    // Remove duplicate based on Label.name
    data := []model.Label{}
    check := make(map[string]bool)
    for _, d := range data_dup {
        if _, v := check[d.Name]; !v {
            check[d.Name] = true
            data = append(data, d)
        }
    }
    return data, err
}

// Get all sub labels
func (dg Dgraph) GetSubLabels(fieldid string, objid string) ([]model.Label, error) {
    // Format Query
    maps := map[string]string{
        "fieldid": fieldid,
        "objid": objid,
    }
    // Send request
    res, err := dg.QueryDql("getSubLabels", maps)
    if err != nil { return nil, err }

    // Decode response
    var r DqlResp
    err = json.Unmarshal(res.Json, &r)
    if err != nil { return nil, err }

    var data_dup []model.Label
    config := &mapstructure.DecoderConfig{
        Result: &data_dup,
        TagName: "json",
        DecodeHook: func(from, to reflect.Kind, v interface{}) (interface{}, error) {
            if to == reflect.Struct {
                nv := tools.CleanCompositeName(v.(map[string]interface{}))
                return nv, nil
            }
            return v, nil
        },
    }
    decoder, err := mapstructure.NewDecoder(config)
    if err != nil { return nil, err }
    err = decoder.Decode(r.All)
    // Remove duplicate based on Label.name
    data := []model.Label{}
    check := make(map[string]bool)
    for _, d := range data_dup {
        if _, v := check[d.Name]; !v {
            check[d.Name] = true
            data = append(data, d)
        }
    }
    return data, err
}

func (dg Dgraph) GetTensions(q TensionQuery, type_ string) ([]model.Tension, error) {
    // Format Query
    maps, err := FormatTensionIntExtMap(q)
    if err != nil { return nil, err }
    // Send request
    var op string
    if type_ == "int" {
        op = "getTensionInt"
    } else if type_ == "ext" {
        op = "getTensionExt"
    } else if type_ == "all" {
        op = "getTensionAll"
    } else {
        panic("Unknow type (tension query)")
    }
    res, err := dg.QueryDql(op, *maps)
    if err != nil { return nil, err }

    // Decode response
    var r DqlResp
    err = json.Unmarshal(res.Json, &r)
    if err != nil { return nil, err }

    var data []model.Tension
    config := &mapstructure.DecoderConfig{
        Result: &data,
        TagName: "json",
        DecodeHook: func(from, to reflect.Kind, v interface{}) (interface{}, error) {
            if to == reflect.Struct {
                nv := tools.CleanCompositeName(v.(map[string]interface{}))
                return nv, nil
            }
            return v, nil
        },
    }
    decoder, err := mapstructure.NewDecoder(config)
    if err != nil { return nil, err }
    err = decoder.Decode(r.All)
    return data, err
}

func (dg Dgraph) GetTensionsCount(q TensionQuery) (map[string]int, error) {
    // Format Query
    maps, err := FormatTensionIntExtMap(q)
    if err != nil { return nil, err }
    // Send request
    res, err := dg.QueryDql("getTensionCount", *maps)
    if err != nil { return nil, err }

    // Decode response
    var r DqlRespCount
    err = json.Unmarshal(res.Json, &r)
    if err != nil { return nil, err }

    if len(r.All) > 0 && len(r.All2) > 0 {
        v := map[string]int{"open":r.All[0]["count"], "closed":r.All2[0]["count"]}
        return v, err
    }
    return nil, err
}

func (dg Dgraph) GetLastBlobId(tid string) (*string) {
    // init client
    dgc, cancel := dg.getDgraphClient()
    defer cancel()
    ctx := context.Background()
    txn := dgc.NewTxn()
    defer txn.Discard(ctx)

    q := fmt.Sprintf(`{all(func: uid(%s))  {
        Tension.blobs (orderdesc: Post.createdAt, first: 1) { uid }
    }}`, tid)
    // Send request
    res, err := txn.Query(ctx, q)
    if err != nil { return nil}

    // Decode response
    var r DqlResp
    err = json.Unmarshal(res.Json, &r)
    if err != nil { return nil}

    var bid string
    if len(r.All) > 1 {
        return nil
    } else if len(r.All) == 1 {
        blobs := r.All[0]["Tension.blobs"].([]interface{})
        if len(blobs) > 0 {
            bid = blobs[0].(model.JsonAtom)["uid"].(string)
        }
    }

    return &bid
}

// Get all coordo roles
func (dg Dgraph) HasCoordos(nameid string) (bool) {
    // Format Query
    maps := map[string]string{
        "nameid": nameid,
    }
    // Send request
    res, err := dg.QueryDql("getCoordos", maps)
    if err != nil { return false }

    // Decode response
    var r DqlResp
    err = json.Unmarshal(res.Json, &r)
    if err != nil { return false }

    var ok bool = false
    if len(r.All) > 1 {
        return ok
    } else if len(r.All) == 1 {
        c := r.All[0]["Node.children"]
        if c != nil && len(c.([]interface{})) > 0 {
            ok = true
        }
    }
    return ok
}

// Get children
func (dg Dgraph) GetChildren(nameid string) ([]string, error) {
    // Format Query
    maps := map[string]string{
        "nameid" :nameid,
    }
    // Send request
    res, err := dg.QueryDql("getChildren", maps)
    if err != nil { return nil, err }

    // Decode response
    var r DqlResp
    err = json.Unmarshal(res.Json, &r)
    if err != nil { return nil, err }

    var data []string
    if len(r.All) > 1 {
        return nil, fmt.Errorf("Got multiple object for term: %s", nameid)
    } else if len(r.All) == 1 {
        c := r.All[0]["Node.children"].([]interface{})
        for _, x := range(c) {
            data = append(data, x.(model.JsonAtom)["Node.nameid"].(string))
        }
    }
    return data, err
}

// Get path to root
func (dg Dgraph) GetParents(nameid string) ([]string, error) {
    // Format Query
    maps := map[string]string{
        "nameid" :nameid,
    }
    // Send request
    res, err := dg.QueryDql("getParents", maps)
    if err != nil { return nil, err }

    // Decode response
    var r DqlResp
    err = json.Unmarshal(res.Json, &r)
    if err != nil { return nil, err }

    var data []string
    if len(r.All) > 1 {
        return nil, fmt.Errorf("Got multiple object for term: %s", nameid)
    } else if len(r.All) == 1 {
        // f%$*µ%ing decoding
        parents := r.All[0]["Node.parent"]
        if parents == nil {return data, err}
        switch p := parents.([]interface{})[0].(model.JsonAtom)["Node.nameid"].(type) {
        case []interface{}:
            for _, x := range(p) {
                data = append(data, x.(string))
            }
        case string:
            data = append(data, p)
        }
    }
    return data, err
}

// DQL Mutations

// SetFieldById set a predicate for the given node in the DB
func (dg Dgraph) SetFieldById(objid string, predicate string, val string) error {
    query := fmt.Sprintf(`query {
        node as var(func: uid(%s))
    }`, objid)

    mu := fmt.Sprintf(`
        uid(node) <%s> "%s" .
    `, predicate, val)

    mutation := &api.Mutation{
        SetNquads: []byte(mu),
    }

    err := dg.MutateWithQueryDql(query, mutation)
    return err
}
// SetFieldByEq set a predicate for the given node in the DB
func (dg Dgraph) SetFieldByEq(fieldid string, objid string, predicate string, val string) error {
    query := fmt.Sprintf(`query {
        node as var(func: eq(%s, "%s"))
    }`, fieldid, objid)

    mu := fmt.Sprintf(`
        uid(node) <%s> "%s" .
    `, predicate, val)

    mutation := &api.Mutation{
        SetNquads: []byte(mu),
    }

    err := dg.MutateWithQueryDql(query, mutation)
    return err
}
//SetSubFieldByEq set a predicate for the given node in the DB
func (dg Dgraph) SetSubFieldByEq(fieldid string, objid string, predicate1, predicate2 string, val string) error {
    query := fmt.Sprintf(`query {
        var(func: eq(%s, "%s")) {
            x as %s
        }
    }`, fieldid, objid, predicate1)

    mu := fmt.Sprintf(`
        uid(x) <%s> "%s" .
    `, predicate2, val)

    mutation := &api.Mutation{
        SetNquads: []byte(mu),
    }

    err := dg.MutateWithQueryDql(query, mutation)
    return err
}

// UpdateRoleType update the role of a node given the nameid using upsert block.
func (dg Dgraph) UpgradeMember(nameid string, roleType model.RoleType) error {
    query := fmt.Sprintf(`query {
        node as var(func: eq(Node.nameid, "%s"))
    }`, nameid)

    mu := fmt.Sprintf(`
        uid(node) <Node.role_type> "%s" .
        uid(node) <Node.name> "%s" .
    `, roleType, roleType)

    mutation := &api.Mutation{
        SetNquads: []byte(mu),
    }

    err := dg.MutateWithQueryDql(query, mutation)
    return err
}

// Remove the user link in the last blob if user match
func (dg Dgraph) MaybeDeleteFirstLink(tid, username string) error {
    query := fmt.Sprintf(`query {
        var(func: uid(%s)) {
          Tension.blobs (orderdesc: Post.createdAt, first: 1) {
            n as Blob.node @filter(eq(NodeFragment.first_link, "%s"))
          }
        }
    }`, tid, username)

    muDel := `uid(n) <NodeFragment.first_link> * .`

    mutation := &api.Mutation{
        DelNquads: []byte(muDel),
    }

    err := dg.MutateWithQueryDql(query, mutation)
    return err
}

// Set the blob pushedFlag and the tension action
func (dg Dgraph) SetPushedFlagBlob(bid string, flag string, tid string, action model.TensionAction) error {
    query := fmt.Sprintf(`query {
        obj as var(func: uid(%s))
    }`, bid)

    mu := fmt.Sprintf(`
        uid(obj) <Blob.pushedFlag> "%s" .
        <%s> <Tension.action> "%s" .
    `, flag, tid, action)
    muDel := `uid(obj) <Blob.archivedFlag> * . `

    mutation := &api.Mutation{
        SetNquads: []byte(mu),
        DelNquads: []byte(muDel),
    }

    err := dg.MutateWithQueryDql(query, mutation)
    return err
}

// Set the blob pushedFlag and the tension action
func (dg Dgraph) SetArchivedFlagBlob(bid string, flag string, tid string, action model.TensionAction) error {
    query := fmt.Sprintf(`query {
        obj as var(func: uid(%s))
    }`, bid)

    mu := fmt.Sprintf(`
        uid(obj) <Blob.archivedFlag> "%s" .
        <%s> <Tension.action> "%s" .
    `, flag, tid, action)

    mutation := &api.Mutation{
        SetNquads: []byte(mu),
    }

    err := dg.MutateWithQueryDql(query, mutation)
    return err
}

func (dg Dgraph) SetNodeSource(nameid string, bid string) error {
    query := fmt.Sprintf(`query {
        node as var(func: eq(Node.nameid, "%s"))
    }`, nameid)

    mu := fmt.Sprintf(`
        uid(node) <Node.source> <%s> .
    `, bid)

    mutation := &api.Mutation{
        SetNquads: []byte(mu),
    }

    err := dg.MutateWithQueryDql(query, mutation)
    return err
}

func (dg Dgraph) PatchNameid(nameid_old string, nameid_new string) error {
    query := fmt.Sprintf(`query {
        node as var(func: eq(Node.nameid, "%s")) {
            tin as Node.tensions_in
            tout as Node.tensions_out
        }
    }`, nameid_old)

    mu := fmt.Sprintf(`
        uid(node) <Node.nameid> "%s" .
        uid(tin) <Tension.receiverid> "%s" .
        uid(tout) <Tension.emitterid> "%s" .
    `, nameid_new, nameid_new, nameid_new)

    mutation := &api.Mutation{
        SetNquads: []byte(mu),
    }

    err := dg.MutateWithQueryDql(query, mutation)
    return err
}


// Deletions

// DeepDelete delete edges recursively for type {t} and id {id}.
// If {rid] is given, it represents the reverse node that should be cleaned up.
func (dg Dgraph) DeepDelete(t string, id string) (error) {
    var q string = "delete"
    var reverse string
    var query string

    switch t {
    case "tension":
        reverse = fmt.Sprintf(`
            uid(rid_emitter) <Node.tensions_out> uid(id) .
            uid(rid_receiver) <Node.tensions_in> uid(id) .
        `)
    case "contract":
        reverse = fmt.Sprintf(`uid(rid) <Tension.contracts> uid(id) .`)
    default:
        return fmt.Errorf("delete query not implemented for this type %s", t)
    }

    q = q + strings.Title(t)
    maps := map[string]string{"id": id}
    query = dg.getDqlQuery(q, maps)
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


