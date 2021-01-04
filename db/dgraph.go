package db

import (
    "fmt"
    "log"
    "bytes"
    "strings"
    "context"
    "reflect"
    "net/http"
    "encoding/json"
    "text/template"
    "github.com/spf13/viper"
    "github.com/mitchellh/mapstructure"
    //"github.com/vektah/gqlparser/v2/gqlerror"
    "github.com/dgraph-io/dgo/v200"
    "github.com/dgraph-io/dgo/v200/protos/api"
    "google.golang.org/grpc"

    "zerogov/fractal6.go/graph/model"
    "zerogov/fractal6.go/tools"
)

// Draph database clients
type Dgraph struct {
    // HTTP/Graphql and GPRC/DQL client address
    gqlUrl string
    grpcUrl string

    // HTTP/Graphql and GPRC/DQL client template
    gqlTemplates map[string]*QueryString
    gpmTemplates map[string]*QueryString
}

//
// GRPC/Graphql+- response
//

type GpmResp struct {
    All []map[string]interface{} `json:"all"`
}

type GpmRespCount struct {
    All []map[string]int `json:"all"`
}

//
// HTTP/Graphql response
//

type GqlRes struct {
    Data   model.JsonAtom `json:"data"`
    Errors []model.JsonAtom `json:"errors"` // message, locations, path, extensions
}

type GraphQLError struct {
    msg string
}

func (e *GraphQLError) Error() string {
    return fmt.Sprintf("%s", e.msg)
}

//
// Query String Interface
//

type QueryString struct {
    Q string
    Template *template.Template
}

// Init clean the query to be compatible in application/json format.
func (q *QueryString) Init() {
    d := q.Q
    q.Q = tools.CleanString(d, false)
    // Load the template @DEBUG: Do we need a template name ?
    q.Template = template.Must(template.New("graphql").Parse(q.Q))
}

func (q QueryString) Format(maps map[string]string) string {
    buf := bytes.Buffer{}
    q.Template.Execute(&buf, maps)
    return buf.String()
}

//
// Initialization
//

// Database client
var DB *Dgraph

func init () {
    DB = initDB()
}

func GetDB() *Dgraph {
    return DB
}

func initDB() *Dgraph {
    tools.InitViper()
    HOSTDB := viper.GetString("db.host")
    PORTDB := viper.GetString("db.port_graphql")
    PORTGRPC := viper.GetString("db.port_grpc")
    APIDB := viper.GetString("db.api")
    dgraphApiUrl := "http://"+HOSTDB+":"+PORTDB+"/"+APIDB
    grpcUrl := HOSTDB+":"+PORTGRPC

    if HOSTDB == "" {
        panic("Viper error: not host found")
    } else {
        fmt.Println("Dgraph Graphql addr:", dgraphApiUrl)
        fmt.Println("Dgraph Grpc addr:", grpcUrl)
    }

    // GPRC/DQL Request Template
    gpmQueries := map[string]string{
        // Count objects
        "count": `{
            all(func: uid("{{.id}}")) {
                count({{.fieldName}})
            }
        }`,
        "getNodeStats": `{
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

            var(func: eq(Node.nameid, "{{.nameid}}")) @recurse {
                c as Node.children @filter(NOT eq(Node.isArchived, true))
            }
            var(func: uid(c)) @filter(eq(Node.type_, "Circle")) {
                circle as count(uid)
            }
            var(func: uid(c)) @filter(eq(Node.type_, "Role")) {
                role as count(uid)
            }

            all() {
                n_member: sum(val(member))
                n_guest: sum(val(guest))
                n_role: sum(val(role))
                n_circle: sum(val(circle))
            }
        }`,
        // Query existance
        "exists": `{
            all(func: eq({{.fieldName}}, "{{.value}}")) {{.filter}} { uid }
        }`,
        "getID": `{
            all(func: eq({{.fieldName}}, "{{.value}}")) {{.filter}} { uid }
        }`,
        // Get single object
        "getFieldById": `{
            all(func: uid("{{.id}}")) {
                {{.fieldName}}
            }
        }`,
        "getFieldByEq": `{
            all(func: eq({{.fieldid}}, "{{.objid}}")) {
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
        "getTension": `{
            all(func: uid("{{.id}}"))
            {{.payload}}
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
        "getAllChildren": `{
            var(func: eq(Node.{{.fieldid}}, "{{.objid}}")) @recurse {
                o as Node.children
            }

            all(func: uid(o)) @filter(NOT eq(Node.isArchived, true)) {
                Node.{{.fieldid}}
                Node.isPrivate
            }
        }`,
        "getAllMembers": `{
            var(func: eq(Node.{{.fieldid}}, "{{.objid}}")) @recurse {
                o as Node.children
            }

            all(func: uid(o)) @filter(has(Node.role_type) AND NOT eq(Node.isArchived, true)) {
                Node.createdAt
                Node.name
                Node.nameid
                Node.rootnameid
                Node.role_type
                Node.first_link {
                    User.username
                    User.name
                }
                Node.parent {
                    Node.nameid
                    Node.isPrivate
                }
                Node.isPrivate
            }
        }`,
        "getCoordos": `{
            all(func: eq(Node.nameid, "{{.nameid}}")) {
                Node.children @filter(eq(Node.role_type, "Coordinator") AND NOT eq(Node.isArchived, true)) { uid }
            }
        }`,
        // Mutations
    }

    // HTTP/Graphql Request Template
    gqlQueries := map[string]string{
        // QUERIES
        "query": `{
            "query": "query {{.Args}} {{.QueryName}} {
                {{.QueryName}} {
                    {{.QueryGraph}}
                }
            }"
        }`,
        "rawQuery": `{
            "query": "{{.RawQuery}}"
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

    gpmT := map[string]*QueryString{}
    gqlT := map[string]*QueryString{}

    for op, q := range(gpmQueries) {
        gpmT[op] = &QueryString{Q:q}
        gpmT[op].Init()
    }
    for op, q := range(gqlQueries) {
        gqlT[op] = &QueryString{Q:q}
        gqlT[op].Init()
    }

    return &Dgraph{
        gqlUrl: dgraphApiUrl,
        grpcUrl: grpcUrl,
        gpmTemplates: gpmT,
        gqlTemplates: gqlT,
    }
}

//
// Internals
//

// Get the grpc Dgraph client.
func (dg Dgraph) getDgraphClient() (dgClient *dgo.Dgraph, cancelFunc func()) {
    conn, err := grpc.Dial(dg.grpcUrl, grpc.WithInsecure())
    if err != nil {
        log.Fatal("While trying to dial gRPC")
    }

    dgClient = dgo.NewDgraphClient(api.NewDgraphClient(conn))
    //ctx := context.Background()

    //// Perform login call. If the Dgraph cluster does not have ACL and
    //// enterprise features enabled, this call should be skipped.
    //for {
    //	// Keep retrying until we succeed or receive a non-retriable error.
    //	err = dgClient.Login(ctx, "groot", "password")
    //	if err == nil || !strings.Contains(err.Error(), "Please retry") {
    //		break
    //	}
    //	time.Sleep(time.Second)
    //}
    //if err != nil {
    //	log.Fatalf("While trying to login %v", err.Error())
    //}

    cancelFunc =  func() {
        if err := conn.Close(); err != nil {
            log.Printf("Error while closing connection:%v", err)
        }
    }
    return
}

// Post send a post request to the Graphql client.
func (dg Dgraph) post(data []byte, res interface{}) error {
    req, err := http.NewRequest("POST", dg.gqlUrl, bytes.NewBuffer(data))
    req.Header.Set("Content-Type", "application/json")

    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    return json.NewDecoder(resp.Body).Decode(res)
}


//
// DQL (ex GraphQL+-) Interface
//


// QueryGpm runs a query on dgraph (...QueryDql)
func (dg Dgraph) QueryGpm(op string, maps map[string]string) (*api.Response, error) {
    // init client
    dgc, cancel := dg.getDgraphClient()
    defer cancel()
    ctx := context.Background()
    txn := dgc.NewTxn()
    defer txn.Discard(ctx)

    // Get the Query
    q := dg.gpmTemplates[op].Format(maps)
    // Send Request
    //fmt.Println(string(q))
    res, err := txn.Query(ctx, q)
    //fmt.Println(res)
    return res, err
}

//MutateWithQueryGpm runs an upsert block mutation by first querying query
//and then mutate based on the result.
func (dg Dgraph) MutateWithQueryGpm(query string, mu *api.Mutation) (error) {
    // init client
    dgc, cancel := dg.getDgraphClient()
    defer cancel()
    ctx := context.Background()
    txn := dgc.NewTxn()
    defer txn.Discard(ctx)

    req := &api.Request{
        Query: query,
        Mutations: []*api.Mutation{mu},
        CommitNow: true,
    }

    _, err := txn.Do(ctx, req)
    return err
}

//MutateUpsertGpm adds a new object in the database if it doesn't exist
func (dg Dgraph) MutateUpsertGpm(object map[string]interface{}, dtype string, upsertField string, upsertVal string) error {
    // init client
    dgc, cancel := dg.getDgraphClient()
    defer cancel()
    ctx := context.Background()
    txn := dgc.NewTxn()
    defer txn.Discard(ctx)

    // make the template here.
    template := template.Must(template.New("graphql").Parse(`{
        all(func: eq({{.dtype}}.{{.upsertField}}, "{{.upserVal}}")) {
            v as uid
        }
    }`))
    buf := bytes.Buffer{}
    template.Execute(&buf, map[string]string{"dtype":dtype, "upsertField":upsertField, "upsertVal":upsertVal})
    query := buf.String()

    object["dgraph.type"] = []string{dtype}
    object["uid"] = "uid(v)"
    js, err := json.Marshal(object)
    if err != nil { return err }
    mu := &api.Mutation{SetJson: js}

    req := &api.Request{
        Query: query,
        Mutations: []*api.Mutation{mu},
        CommitNow: true,
    }
    // User do instead of Mutate here ?
    _, err = txn.Do(ctx, req)
    return err
}

//Push adds a new object in the database.
func (dg Dgraph) Push(object map[string]interface{}, dtype string) (string, error) {
    // init client
    dgc, cancel := dg.getDgraphClient()
    defer cancel()
    ctx := context.Background()
    txn := dgc.NewTxn()
    defer txn.Discard(ctx)

    object["dgraph.type"] = []string{dtype}
    uid, ok := object["uid"]
    if !ok {
        uid = "_:new_obj"
        object["uid"] = uid
    }
    js, err := json.Marshal(object)
    if err != nil { return "", err }

    mu := &api.Mutation{
        CommitNow: true,
        SetJson: js,
    }
    r, err := txn.Mutate(ctx, mu)
    if err != nil { return "", err }

    uid = r.Uids[uid.(string)]
    return uid.(string), nil
}

//DeleteNodes Delete nodes from theirs uid and their edges in dgraph
func (dg Dgraph) DeleteNodes(uids ...string) (error) {
    // init client
    dgc, cancel := dg.getDgraphClient()
    defer cancel()
    ctx := context.Background()
    txn := dgc.NewTxn()
    defer txn.Discard(ctx)

    d :=  map[string]string{}
    for _, uid := range uids { d["uid"] = uid }
    js, err := json.Marshal(d)
    if err != nil { return err }

    mu := &api.Mutation{
        CommitNow: true,
        DeleteJson: js,
    }

    _, err = txn.Mutate(ctx, mu)
    return err
}

//DeleteEdges Delete edges from their uid
func (dg Dgraph) DeleteEdges(key string, value string, delMap map[string]interface{}) (error) {
    // init client
    dgc, cancel := dg.getDgraphClient()
    defer cancel()
    ctx := context.Background()
    txn := dgc.NewTxn()
    defer txn.Discard(ctx)

    query := fmt.Sprintf(`query {
        o as var(func: eq(%s, "%s"))
    }`, key, value)

    var mu string
    for k, _ := range delMap {
        mu = mu + fmt.Sprintf(`uid(o) <%s> * .`, k) + "\n"
    }

    mutation := &api.Mutation{
        DelNquads: []byte(mu),
    }

    err := dg.MutateWithQueryGpm(query, mutation)
    return err
}

//
// GraphQL Interface
//

// QueryGql query the Dgraph Graphql endpoint by following a http request.
// It map the result in to given data structure
func (dg Dgraph) QueryGql(op string, reqInput map[string]string, data interface{}) error {
    // Get the query
    queryName := reqInput["QueryName"]
    var q string
    if _q, ok := dg.gqlTemplates[op]; ok {
        q = _q.Format(reqInput)
    } else {
        panic("unknonw QueryGql op: " + op)
    }

    // Send the dgraph request and follow the results
    res := &GqlRes{}
    //fmt.Println("request ->", string(q))
    err := dg.post([]byte(q), res)
    //fmt.Println("response ->", res)
    if err != nil {
        return err
    } else if res.Errors != nil {
        err, _ := json.Marshal(res.Errors)
        //return fmt.Errorf(string(err))
        return &GraphQLError{string(err)}
    }


    switch v := data.(type) {
    case model.JsonAtom:
        for k, val := range res.Data {
            v[k] = val
        }
    default: // Payload data type
        var config *mapstructure.DecoderConfig
        if op == "query" || op == "rawQuery" {
            // Decoder config to handle aliased request
            // @DEBUG: see bug #3c3f1f7
            config = &mapstructure.DecoderConfig{
                Result: data,
                TagName: "json",
                DecodeHook: func(from, to reflect.Kind, v interface{}) (interface{}, error) {
                    if to == reflect.Struct {
                        nv := tools.CleanAliasedMap(v.(map[string]interface{}))
                        return nv, nil
                    }
                    return v, nil
                },
            }
        } else {
            config = &mapstructure.DecoderConfig{TagName: "json", Result: data}
        }

        decoder, err := mapstructure.NewDecoder(config)
        if err != nil { return err }
        err = decoder.Decode(res.Data[queryName])
    }

    return err
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
    res, err := dg.QueryGpm("count", maps)
    if err != nil { panic(err) }

    // Decode response
    var r GpmRespCount
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

func (dg Dgraph) GetNodeStats(nameid string) map[string]int {
    // Format Query
    maps := map[string]string{
        "nameid": nameid,
    }
    // Send request
    res, err := dg.QueryGpm("getNodeStats", maps)
    if err != nil { panic(err) }

    // Decode response
    var r GpmRespCount
    err = json.Unmarshal(res.Json, &r)
    if err != nil { panic(err) }

    // Extract result
    if len(r.All) == 0 {
        panic("no stats for: "+ nameid)
    }
    stats := make(map[string]int, len(r.All))
    for _, s := range r.All {
        for k, v := range s {
            stats[k] = v
        }
    }

    return stats
}

// Probe if an object exists.
func (dg Dgraph) Exists(fieldName string, value string, filterName, filterValue *string) (bool, error) {
    // Format Query
    maps := map[string]string{
        "fieldName":fieldName, "value": value, "filter": "",
    }
    if filterName != nil {
        maps["filter"] = fmt.Sprintf(`@filter(eq(%s, %s))`, *filterName, *filterValue )
    }
    // Send request
    res, err := dg.QueryGpm("exists", maps)
    if err != nil {
        return false, err
    }

    // Decode response
    var r GpmResp
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
        "fieldName":fieldName, "value": value, "filter": "",
    }
    if filterName != nil {
        maps["filter"] = fmt.Sprintf(`@filter(eq(%s, %s))`, *filterName, *filterValue )
    }
    // Send request
    res, err := dg.QueryGpm("getID", maps)
    if err != nil {
        return result, err
    }

    // Decode response
    var r GpmResp
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
func (dg Dgraph) GetFieldById(id string, fieldName string) (interface{}, error) {
    // Format Query
    maps := map[string]string{
        "id": id,
        "fieldName": fieldName,
    }
    // Send request
    res, err := dg.QueryGpm("getFieldById", maps)
    if err != nil { return nil, err }

    // Decode response
    var r GpmResp
    err = json.Unmarshal(res.Json, &r)
    if err != nil { return nil, err }

    if len(r.All) > 1 {
        return nil, fmt.Errorf("Got multiple in gpm query: %s %s", fieldName, id)
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
        "objid": objid,
        "fieldName": fieldName,
    }
    // Send request
    res, err := dg.QueryGpm("getFieldByEq", maps)
    if err != nil { return nil, err }

    // Decode response
    var r GpmResp
    err = json.Unmarshal(res.Json, &r)
    if err != nil { return nil, err }

    if len(r.All) > 1 {
        return nil, fmt.Errorf("Got multiple in gpm query: %s %s", fieldName, objid)
    } else if len(r.All) == 1 {
        x := r.All[0][fieldName]
        return x, nil
    }
    return nil, err
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
    res, err := dg.QueryGpm("getSubFieldById", maps)
    if err != nil { return nil, err }

    // Decode response
    var r GpmResp
    err = json.Unmarshal(res.Json, &r)
    if err != nil { return nil, err }

    if len(r.All) > 1 {
        return nil, fmt.Errorf("Got multiple in gpm query")
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
    res, err := dg.QueryGpm("getSubFieldByEq", maps)
    if err != nil { return nil, err }

    // Decode response
    var r GpmResp
    err = json.Unmarshal(res.Json, &r)
    if err != nil { return nil, err }

    if len(r.All) > 1 {
        return nil, fmt.Errorf("Got multiple in gpm query")
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
    res, err := dg.QueryGpm("getSubSubFieldByEq", maps)
    if err != nil { return nil, err }

    // Decode response
    var r GpmResp
    err = json.Unmarshal(res.Json, &r)
    if err != nil { return nil, err }

    if len(r.All) > 1 {
        return nil, fmt.Errorf("Got multiple in gpm query")
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
    res, err := dg.QueryGpm("getUser", maps)
    if err != nil { return nil, err }

    // Decode response
    var r GpmResp
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
        if user.Roles[i].RoleType == model.RoleTypeRetired {
            user.Roles = append(user.Roles[:i], user.Roles[i+1:]...)
            i--
        }
    }
    return &user, err
}

// Returns the node charac
func (dg Dgraph) GetNodeCharac(fieldid string, objid string) (*model.NodeCharac, error) {
    // Format Query
    maps := map[string]string{
        "fieldid": fieldid,
        "objid": objid,
        "payload": model.NodeCharacPayloadDg,
    }
    // Send request
    res, err := dg.QueryGpm("getNode", maps)
    if err != nil { return nil, err }

    // Decode response
    var r GpmResp
    err = json.Unmarshal(res.Json, &r)
    if err != nil { return nil, err }

    var obj model.NodeCharac
    if len(r.All) > 1 {
        return nil, fmt.Errorf("Got multiple node charac for @id: %s, %s", fieldid, objid)
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
        err = decoder.Decode(r.All[0][model.NodeCharacNF])
        if err != nil { return nil, err }
    }
    return &obj, err
}

// Returns the matching nodes
func (dg Dgraph) GetNodes(regex string, isRoot bool) ([]model.NodeId, error) {
    // Format Query
    maps := map[string]string{
        "regex": regex,
        "payload": model.NodeIdPayloadDg,
    }

    // Send request
    var res *api.Response
    var err error
    if isRoot {
        res, err = dg.QueryGpm("getNodesRoot", maps)
    } else {
        res, err = dg.QueryGpm("getNodes", maps)
    }
    if err != nil { return nil, err }

    // Decode response
    var r GpmResp
    err = json.Unmarshal(res.Json, &r)
    if err != nil { return nil, err }

    var data []model.NodeId
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
func (dg Dgraph) GetTensionHook(tid string, bid *string) (*model.Tension, error) {
    // Format Query
    var blobFilter string
    if bid == nil {
        blobFilter = "(orderdesc: Post.createdAt, first: 1)"
    } else {
        blobFilter = fmt.Sprintf(`@filter(uid(%s))`, *bid)
    }
    maps := map[string]string{
        "id": tid,
        "payload": fmt.Sprintf(model.TensionHookPayload, blobFilter),
    }
    // Send request
    res, err := dg.QueryGpm("getTension", maps)
    if err != nil { return nil, err }

    // Decode response
    var r GpmResp
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
    return &obj, err
}


// Get all sub children
func (dg Dgraph) GetAllChildren(fieldid string, objid string) ([]model.NodeId, error) {
    // Format Query
    maps := map[string]string{
        "fieldid": fieldid,
        "objid": objid,
    }
    // Send request
    res, err := dg.QueryGpm("getAllChildren", maps)
    if err != nil { return nil, err }

    // Decode response
    var r GpmResp
    err = json.Unmarshal(res.Json, &r)
    if err != nil { return nil, err }

    var data []model.NodeId
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
    var r GpmResp
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
// Get all sub children
func (dg Dgraph) GetAllMembers(fieldid string, objid string) ([]model.MemberNode, error) {
    // Format Query
    maps := map[string]string{
        "fieldid": fieldid,
        "objid": objid,
    }
    // Send request
    res, err := dg.QueryGpm("getAllMembers", maps)
    if err != nil { return nil, err }

    // Decode response
    var r GpmResp
    err = json.Unmarshal(res.Json, &r)
    if err != nil { return nil, err }

    var data []model.MemberNode
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

// Get all coordo roles
func (dg Dgraph) HasCoordos(nameid string) (bool) {
    // Format Query
    maps := map[string]string{
        "nameid": nameid,
    }
    // Send request
    res, err := dg.QueryGpm("getCoordos", maps)
    if err != nil { return false }

    // Decode response
    var r GpmResp
    err = json.Unmarshal(res.Json, &r)
    if err != nil { return false }

    //var data []model.NodeId
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
    res, err := dg.QueryGpm("getChildren", maps)
    if err != nil { return nil, err }

    // Decode response
    var r GpmResp
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
    res, err := dg.QueryGpm("getParents", maps)
    if err != nil { return nil, err }

    // Decode response
    var r GpmResp
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

    err := dg.MutateWithQueryGpm(query, mutation)
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

    err := dg.MutateWithQueryGpm(query, mutation)
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

    err := dg.MutateWithQueryGpm(query, mutation)
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

    err := dg.MutateWithQueryGpm(query, mutation)
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

    err := dg.MutateWithQueryGpm(query, mutation)
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

    err := dg.MutateWithQueryGpm(query, mutation)
    return err
}

//
// Graphql requests
//

// Add a new vertex
func (dg Dgraph) Add(vertex string, input interface{}) (string, error) {
    Vertex := strings.Title(vertex)
    queryName := "add" + Vertex
    inputType := "Add" + Vertex + "Input"
    queryGraph := vertex + ` { id }`

    // Just One Node
    var ifaces []interface{} = make([]interface{}, 1)
    ifaces[0] = input
    inputs, _ := json.Marshal(ifaces)

    // Build the string request
    reqInput := map[string]string{
        "QueryName": queryName, // function name (e.g addUser)
        "InputType": inputType, // input type name (e.g AddUserInput)
        "QueryGraph": tools.CleanString(queryGraph, true), // output data
        "InputPayload": string(inputs), // inputs data
    }

    // Send request
    payload := make(model.JsonAtom, 1)
    err := dg.QueryGql("add", reqInput, payload)
    if err != nil { return "", err }
    // Extract id result
    res := payload[queryName].(model.JsonAtom)[vertex].([]interface{})[0].(model.JsonAtom)["id"]
    return res.(string), err
}

// Update a vertex
func (dg Dgraph) Update(vertex string, input interface{}) error {
    Vertex := strings.Title(vertex)
    queryName := "update" + Vertex
    inputType := "Update" + Vertex + "Input"
    queryGraph := vertex + ` { id }`

    // Just One Node
    inputs, _ := json.Marshal(input)

    // Build the string request
    reqInput := map[string]string{
        "QueryName": queryName, // function name (e.g addUser)
        "InputType": inputType, // input type name (e.g AddUserInput)
        "QueryGraph": tools.CleanString(queryGraph, true), // output data
        "InputPayload": string(inputs), // inputs data
    }

    // Send request
    payload := make(model.JsonAtom, 1)
    err := dg.QueryGql("update", reqInput, payload)
    return err
}

// Delete a vertex
func (dg Dgraph) Delete(vertex string, input interface{}) error {
    Vertex := strings.Title(vertex)
    queryName := "delete" + Vertex
    inputType :=  Vertex + "Filter"
    queryGraph := vertex + ` { id }`

    // Just One Node
    inputs, _ := json.Marshal(input)

    // Build the string request
    reqInput := map[string]string{
        "QueryName": queryName, // function name (e.g addUser)
        "InputType": inputType, // input type name (e.g AddUserInput)
        "QueryGraph": tools.CleanString(queryGraph, true), // output data
        "InputPayload": string(inputs), // inputs data
    }

    // Send request
    payload := make(model.JsonAtom, 1)
    err := dg.QueryGql("delete", reqInput, payload)
    return err
}

