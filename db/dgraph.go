package db

import (
    "fmt"
    "log"
    "bytes"
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
    // HTTP/Graphql and GPRC/Graphql+- client address
    gqlUrl string
    grpcUrl string

    // HTTP/Graphql and GPRC/Graphql+- client template
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

    // GPRC/Graphql+- Request Template
    gpmQueries := map[string]string{
        // Count objects
        "count": `{
            all(func: uid("{{.id}}")) {
                count({{.typeName}}.{{.fieldName}})
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
				c as Node.children
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
            all(func: eq({{.typeName}}.{{.fieldName}}, "{{.value}}")) {
                uid
            }
        }`,
        // Get single object
        "getFieldByEq": `{
            all(func: eq({{.typeName}}.{{.fieldid}}, "{{.objid}}")) {
                {{.typeName}}.{{.fieldName}}
            }
        }`,
        "getSubFieldById": `{
            all(func: uid("{{.id}}")) {
                {{.typeNameSource}}.{{.fieldNameSource}} {
                    {{.typeNameTarget}}.{{.fieldNameTarget}}
                }
            }
        }`,
        "getSubFieldByEq": `{
            all(func: eq({{.typeNameSource}}.{{.fieldid}}, "{{.value}}")) {
                {{.typeNameSource}}.{{.fieldNameSource}} {
                    {{.typeNameTarget}}.{{.fieldNameTarget}}
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
        "getTension": `{
            all(func: uid("{{.id}}"))
                {{.payload}}
        }`,
		// Get multiple objects
		"getAllChildren": `{
			var(func: eq(Node.{{.fieldid}}, "{{.objid}}")) @recurse {
				o as Node.children
			}

			all(func: uid(o)) {
				Node.{{.fieldid}}
			}
		}`,
        "getAllMembers": `{
            var(func: eq(Node.{{.fieldid}}, "{{.objid}}")) @recurse {
                o as Node.children
            }

            all(func: uid(o)) @filter(has(Node.role_type)) {
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
                }
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
// GraphQL/+- Interface
//


// QueryGpm runs a query on dgraph, using grpc channel (Graphql+-)
// Returns: data res
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
	if err != nil {
        return nil, err
	}
    return res, nil
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

	if _, err := txn.Do(ctx, req); err != nil {
		return err
	}
	return nil
}

//MutateGpm push a new object in the database with upsert operation
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
	obj, err := json.Marshal(object)
	if err != nil {
        return err
    }
    mu := &api.Mutation{SetJson: obj}

	req := &api.Request{
        Query: query,
		Mutations: []*api.Mutation{mu},
		CommitNow: true,
	}
	// User do instead of Mutate here ?
    _, err = txn.Do(ctx, req)
    if err != nil {
        return err
	}
    return nil
}

//MutateGpm push a new object in the database.
func (dg Dgraph) MutateGpm(object map[string]interface{}, dtype string) (string, error) {
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
	obj, err := json.Marshal(object)
    fmt.Println(obj)
	if err != nil {
        return "", err
    }

	mu := &api.Mutation{
		CommitNow: true,
		SetJson:   obj,
	}
    r, err := txn.Mutate(ctx, mu)
    if err != nil {
        return "", err
	}

	uid = r.Uids[uid.(string)]
    return uid.(string), nil
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
    if err != nil {
        return err
    }
    err = decoder.Decode(res.Data[queryName])
    if err != nil {
        return err
    }
    return nil
}


//
// Gprc/Graphql+- requests
//

// Count count the number of object in fieldName attribute for given type and id
// Returns: int or -1 if nothing is found.
func (dg Dgraph) Count(id string, typeName string, fieldName string) int {
    // Format Query
    maps := map[string]string{
       "id":id, "typeName": typeName, "fieldName":fieldName, 
    }
    // Send request
    res, err := dg.QueryGpm("count", maps)
	if err != nil {
        panic(err)
	}

    // Decode response
	var r GpmRespCount
	err = json.Unmarshal(res.Json, &r)
	if err != nil {
		panic(err)
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

func (dg Dgraph) GetNodeStats(nameid string) map[string]int {
    // Format Query
    maps := map[string]string{
        "nameid": nameid, 
    }
    // Send request
    res, err := dg.QueryGpm("getNodeStats", maps)
	if err != nil {
        panic(err)
	}

    // Decode response
	var r GpmRespCount
	err = json.Unmarshal(res.Json, &r)
	if err != nil {
		panic(err)
	}

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
func (dg Dgraph) Exists(typeName string, fieldName string, value string) (bool, error) {
    // Format Query
    maps := map[string]string{
        "typeName": typeName, "fieldName":fieldName, "value": value,
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

// Returns a field from objid
func (dg Dgraph) GetFieldByEq(typeName string, fieldid string, objid string, fieldName string) (interface{}, error) {
    // Format Query
    maps := map[string]string{
        "typeName":typeName,
        "fieldid": fieldid,
        "objid":objid,
        "fieldName":fieldName,
    }
    // Send request
    res, err := dg.QueryGpm("getFieldByEq", maps)
    if err != nil {
        return  nil, err
    }

    // Decode response
    var r GpmResp
	err = json.Unmarshal(res.Json, &r)
	if err != nil {
        return nil, err
	}

    if len(r.All) > 1 {
        return nil, fmt.Errorf("Got multiple in gpm query: %s %s", fieldName, objid)
    } else if len(r.All) == 1 {
        f1 := typeName +"."+ fieldName
        x := r.All[0][f1]
        return x, nil
    }
    return nil, err
}

// Returns a subfield from uid
func (dg Dgraph) GetSubFieldById(id string, typeNameSource string, fieldNameSource string, typeNameTarget string, fieldNameTarget string) (interface{}, error) {
    // Format Query
    maps := map[string]string{
        "id":id,
        "typeNameSource":typeNameSource,
        "fieldNameSource":fieldNameSource,
        "typeNameTarget":typeNameTarget,
        "fieldNameTarget":fieldNameTarget,
    }
    // Send request
    res, err := dg.QueryGpm("getSubFieldById", maps)
    if err != nil {
        return  nil, err
    }

    // Decode response
    var r GpmResp
	err = json.Unmarshal(res.Json, &r)
	if err != nil {
        return nil, err
	}

    if len(r.All) > 1 {
        return nil, fmt.Errorf("Got multiple in gpm query")
    } else if len(r.All) == 1 {
        f1 := typeNameSource +"."+ fieldNameSource
        f2 := typeNameTarget +"."+ fieldNameTarget
        x := r.All[0][f1].(model.JsonAtom)
        if x != nil {
            return x[f2], nil
        }
    }
    return nil, err
}

// Returns a subfield from Eq 
func (dg Dgraph) GetSubFieldByEq(fieldid string, value string, typeNameSource string, fieldNameSource string, typeNameTarget string, fieldNameTarget string) (interface{}, error) {
    // Format Query
    maps := map[string]string{
        "fieldid":fieldid,
        "value":value,
        "typeNameSource":typeNameSource,
        "fieldNameSource":fieldNameSource,
        "typeNameTarget":typeNameTarget,
        "fieldNameTarget":fieldNameTarget,
    }
    // Send request
    res, err := dg.QueryGpm("getSubFieldByEq", maps)
    if err != nil {
        return  nil, err
    }

    // Decode response
    var r GpmResp
	err = json.Unmarshal(res.Json, &r)
	if err != nil {
        return nil, err
	}

    if len(r.All) > 1 {
        return nil, fmt.Errorf("Got multiple in gpm query")
    } else if len(r.All) == 1 {
        f1 := typeNameSource +"."+ fieldNameSource
        f2 := typeNameTarget +"."+ fieldNameTarget
        x := r.All[0][f1].(model.JsonAtom)
        if x != nil {
            return x[f2], nil
        }
    }
    return nil, err
}

// Returns the user context
func (dg Dgraph) GetUser(fieldid string, userid string) (*model.UserCtx, error) {
    // Format Query
    maps := map[string]string{
        "fieldid":fieldid,
        "userid":userid,
        "payload": model.UserCtxPayloadDg,
    }
    // Send request
    res, err := dg.QueryGpm("getUser", maps)
    if err != nil {
        return  nil, err
    }

    // Decode response
    var r GpmResp
	err = json.Unmarshal(res.Json, &r)
	if err != nil {
        return nil, err
	}

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
        if err != nil {
            return nil, err
        }
        err = decoder.Decode(r.All[0])
        if err != nil {
            return nil, err
        }
    }
    return &user, nil
}

// Returns the node charac
func (dg Dgraph) GetNodeCharac(fieldid string, objid string) (*model.NodeCharac, error) {
    // Format Query
    maps := map[string]string{
        "fieldid":fieldid,
        "objid":objid,
        "payload": model.NodeCharacPayloadDg,
    }
    // Send request
    res, err := dg.QueryGpm("getNode", maps)
    if err != nil {
        return  nil, err
    }

    // Decode response
    var r GpmResp
	err = json.Unmarshal(res.Json, &r)
	if err != nil {
        return nil, err
	}

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
        if err != nil {
            return nil, err
        }
        err = decoder.Decode(r.All[0][model.NodeCharacNF])
        if err != nil {
            return nil, err
        }
    }
    return &obj, nil
}

// Returns the tension hook content
func (dg Dgraph) GetTensionHook(uid string) (*model.Tension, error) {
    // Format Query
    maps := map[string]string{
        "id":uid,
        "payload": model.TensionHookPayload,
    }
    // Send request
    res, err := dg.QueryGpm("getTension", maps)
    if err != nil {
        return  nil, err
    }

    // Decode response
    var r GpmResp
	err = json.Unmarshal(res.Json, &r)
	if err != nil {
        return nil, err
	}

    var obj model.Tension
    if len(r.All) > 1 {
        return nil, fmt.Errorf("Got multiple tension for @uid: %s", uid)
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
        if err != nil {
            return nil, err
        }
        err = decoder.Decode(r.All[0])
        if err != nil {
            return nil, err
        }
    }
    return &obj, nil
}

// Get all sub children
func (dg Dgraph) GetAllChildren(fieldid string, objid string) ([]model.NodeId, error) {
    // Format Query
    maps := map[string]string{
        "fieldid":fieldid,
        "objid":objid,
    }
    // Send request
    res, err := dg.QueryGpm("getAllChildren", maps)
    if err != nil {
        return  nil, err
    }

    // Decode response
    var r GpmResp
	err = json.Unmarshal(res.Json, &r)
	if err != nil {
        return nil, err
	}

    var data []model.NodeId
    //if len(r.All) > 1 {
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
        if err != nil {
            return nil, err
        }
        err = decoder.Decode(r.All)
        if err != nil {
            return nil, err
        }
    //}
    return data, nil
}

// Get all sub children
func (dg Dgraph) GetAllMembers(fieldid string, objid string) ([]model.MemberNode, error) {
    // Format Query
    maps := map[string]string{
        "fieldid":fieldid,
        "objid":objid,
    }
    // Send request
    res, err := dg.QueryGpm("getAllMembers", maps)
    if err != nil {
        return  nil, err
    }

    // Decode response
    var r GpmResp
	err = json.Unmarshal(res.Json, &r)
	if err != nil {
        return nil, err
	}

    var data []model.MemberNode
    //if len(r.All) > 1 {
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
        if err != nil {
            return nil, err
        }
        err = decoder.Decode(r.All)
        if err != nil {
            return nil, err
        }
    //}
    return data, nil
}

// Mutation

// UpdateRoleType update the role of a node given the nameid using upsert block.
func (dg Dgraph) UpgradeGuest(nameid string, roleType model.RoleType) error {
	query := fmt.Sprintf(`query {
		node as var(func: eq(Node.nameid, "%s"))
	}`, nameid)

    mu := fmt.Sprintf(`
        uid(node) <Node.role_type> "%s" .
        uid(node) <Node.name> "Member" .
    `, roleType)

	mutation := &api.Mutation{
		SetNquads: []byte(mu),
	}
	
    err := dg.MutateWithQueryGpm(query, mutation)
    if err != nil {
        return err
    }
    return nil
}

// UpdateRoleType update the role of a node given the nameid using upsert block.
func (dg Dgraph) SetPushedBlob(uid string, flag string) error {
	query := fmt.Sprintf(`query {
		obj as var(func: uid(%s))
	}`, uid)

    mu := fmt.Sprintf(`
        uid(obj) <Blob.pushedFlag> "%s" .
    `, flag)

	mutation := &api.Mutation{
		SetNquads: []byte(mu),
	}
	
    err := dg.MutateWithQueryGpm(query, mutation)
    if err != nil {
        return err
    }
    return nil
}


//
// Graphql requests
//

// AddUser add a new user
func (dg Dgraph) AddUser(input model.AddUserInput) error {
    queryName := "addUser"
    inputType := "AddUserInput"
    queryGraph := `user { id } `

    // Just One User
    inputs, _ := json.Marshal([]model.AddUserInput{input})

    // Build the string request
    reqInput := map[string]string{
        "QueryName": queryName, // function name (e.g addUser)
        "InputType": inputType, // input type name (e.g AddUserInput)
        "QueryGraph": tools.CleanString(queryGraph, true), // output data
        "InputPayload": string(inputs), // inputs data
    }

    // Send request
    payload := model.AddUserPayload{}
    err := dg.QueryGql("add", reqInput, &payload)
    if err != nil {
        return err
    }

    // Decode response
    //var user model.UserCtx
    //rRaw, err := json.Marshal(payload.User[0])
    //if err != nil {
    //    return nil, err
    //}
    //err = json.Unmarshal(rRaw, &user)
    //if err != nil {
    //    return nil, err
    //}
    //return &user, nil
    return nil
}


// AddNode add a new node
func (dg Dgraph) AddNode(input model.AddNodeInput) error {
    queryName := "addNode"
    inputType := "AddNodeInput"
    queryGraph := `node { id }`

    // Just One Node
    inputs, _ := json.Marshal([]model.AddNodeInput{input})

    // Build the string request
    reqInput := map[string]string{
        "QueryName": queryName, // function name (e.g addUser)
        "InputType": inputType, // input type name (e.g AddUserInput)
        "QueryGraph": tools.CleanString(queryGraph, true), // output data
        "InputPayload": string(inputs), // inputs data
    }

    // Send request
    payload := model.AddNodePayload{}
    err := dg.QueryGql("add", reqInput, &payload)
    if err != nil {
        return err
    }

    return nil
}

// AddTension add a new tension
func (dg Dgraph) AddTension(input model.AddTensionInput) (string, error) {
    queryName := "addTension"
    inputType := "AddTensionInput"
    queryGraph := `tension { id }`

    // Just One Tension
    inputs, _ := json.Marshal([]model.AddTensionInput{input})

    // Build the string request
    reqInput := map[string]string{
        "QueryName": queryName, // function name (e.g addUser)
        "InputType": inputType, // input type name (e.g AddUserInput)
        "QueryGraph": tools.CleanString(queryGraph, true), // output data
        "InputPayload": string(inputs), // inputs data
    }

    // Send request
    payload := model.AddTensionPayload{}
    err := dg.QueryGql("add", reqInput, &payload)
    if err != nil {
        return "", err
    }

    r := payload.Tension[0].ID
    return r, nil
}
