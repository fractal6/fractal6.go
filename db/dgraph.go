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
        panic("viper error: not host found.")
    } else {
        fmt.Println("Dgraph Graphql addr:", dgraphApiUrl)
        fmt.Println("Dgraph Grpc addr:", grpcUrl)
    }

    // GPRC/Graphql+- Request Template
    gpmQueries := map[string]string{
        "count": `{
            all(func: uid("{{.id}}")) {
                count({{.typeName}}.{{.fieldName}})
            }
        }`,
        "exists": `{
            all(func: eq({{.typeName}}.{{.fieldName}}, "{{.value}}")) {
                uid
            }
        }`,
        // getUser with UserCtx payload
        "getUser": `{
            all(func: eq(User.{{.fieldid}}, "{{.userid}}")) 
                {{.payload}}
        }`,
        "getNodeCharac": `{
            all(func: eq(Node.{{.fieldid}}, "{{.objid}}")) 
                {{.payload}}
        }`,
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

	dc := api.NewDgraphClient(conn)
	dgClient = dgo.NewDgraphClient(dc)
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


// Run a query on dgraph, using grpc channel (Graphql+-)
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
	res, err := txn.Query(ctx, q)
	if err != nil {
        return nil, err
	}
    return res, nil
}

//
// GraphQL Interface
//

// Run a query Dgraph Graphql endpoing and map the result in the given data structure
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
        "typeName": typeName, "fieldName":fieldName, "id":id,
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
    res, err := dg.QueryGpm("getNodeCharac", maps)
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
    userDGql := model.AddUserPayload{}
    err := dg.QueryGql("add", reqInput, &userDGql)
    if err != nil {
        return err
    }

    // Decode response
    //var user model.UserCtx
    //rRaw, err := json.Marshal(userDGql.User[0])
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
    queryGraph := `node { id } `

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
    nodeDGql := model.AddNodePayload{}
    err := dg.QueryGql("add", reqInput, &nodeDGql)
    if err != nil {
        return err
    }

    return nil
}

