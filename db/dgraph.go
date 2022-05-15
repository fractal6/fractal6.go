package db

import (
	"fmt"
	"log"
	"os"
	"time"
	"bytes"
	"context"
	"strings"
	"encoding/json"
	"net/http"
	"text/template"
	"crypto/rsa"
    "io/ioutil"

	"github.com/go-chi/jwtauth/v5"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"

	//"github.com/vektah/gqlparser/v2/gqlerror"
	"github.com/dgraph-io/dgo/v200"
	"github.com/dgraph-io/dgo/v200/protos/api"
	"google.golang.org/grpc"

	"fractale/fractal6.go/graph/codec"
	"fractale/fractal6.go/graph/model"
	. "fractale/fractal6.go/tools"
)

var dgraphPrivateKey *rsa.PrivateKey
var dgraphPublicKey *rsa.PublicKey
var buildMode string

// Draph database clients
type Dgraph struct {
    // HTTP/Graphql and GPRC/DQL client address
    gqlAddr string
    grpcAddr string

    // HTTP/Graphql and GPRC/DQL client template
    gqlTemplates map[string]*QueryString
    dqlTemplates map[string]*QueryString
    dqlMutTemplates map[string]*QueryString
}

type DgraphClaims struct {
    Username string          `json:"USERNAME"`
    UserType model.UserType  `json:"USERTYPE"`
    // Rootnameid where user is Member
    Rootids []string        `json:"ROOTIDS"`
    // Rootnameid where user is Owner
    Ownids []string         `json:"OWNIDS"`
}

//
// GRPC/Graphql+-(DQL) response
//

type DqlResp struct {
    All []map[string]interface{} `json:"all"`
}

type DqlRespCount struct {
    All []map[string]int `json:"all"`
    All2 []map[string]int `json:"all2"`
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
    q.Q = CleanString(d, false)
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
    InitViper()
    // Get env mode
    if buildMode != "PROD" {
        buildMode = "DEV"
    }

    // @DEBUG: how to integrate it with cobra to execute other command without error ?
    var pub_key string
    var priv_key string
    // Get Jwt public key
    if fn := viper.GetString("db.dgraph_public_key"); fn != "" {
        if content, err := ioutil.ReadFile(fn); err != nil {
            log.Fatal(err)
        } else {
            pub_key = string(content)
        }
    } else if os.Getenv("DGRAPH_PUBLIC_KEY") != ""{
        pub_key = os.Getenv("DGRAPH_PUBLIC_KEY")
    }
    // Get Jwt private key
    if fn := viper.GetString("db.dgraph_private_key"); fn != "" {
        if content, err := ioutil.ReadFile(fn); err != nil {
            log.Fatal(err)
        } else {
            priv_key = string(content)
        }
    } else if os.Getenv("DGRAPH_PRIVATE_KEY") != ""{
        priv_key = os.Getenv("DGRAPH_PRIVATE_KEY")
    }

    if pub_key != "" && priv_key != "" {
        dgraphPublicKey = ParseRsaPublic(pub_key)
        dgraphPrivateKey = ParseRsaPrivate(priv_key)
    } else {
        log.Fatal("DGRAPH_PRIVATE_KEY or DGRAPH_PUBLIC_KEY not found")
    }

    DB = initDB()
}

func GetDB() *Dgraph {
    return DB
}

func initDB() *Dgraph {
    HOSTDB := viper.GetString("db.host")
    PORTDB := viper.GetString("db.port_graphql")
    PORTGRPC := viper.GetString("db.port_grpc")
    APIDB := viper.GetString("db.api")
    dgraphApiAddr := "http://"+HOSTDB+":"+PORTDB+"/"+APIDB
    grpcAddr := HOSTDB+":"+PORTGRPC

    if HOSTDB == "" {
        panic("Viper error: not host found")
    } else {
        fmt.Println("Dgraph Graphql addr:", dgraphApiAddr)
        fmt.Println("Dgraph Grpc addr:", grpcAddr)
    }

    gqlT := map[string]*QueryString{}
    dqlT := map[string]*QueryString{}
    dqlMutT := map[string]*QueryString{}

    for op, q := range(dqlQueries) {
        dqlT[op] = &QueryString{Q:q}
        dqlT[op].Init()
    }
    for op, q := range(dqlMutations) {
        dqlT[op] = &QueryString{Q:q.Q}
        dqlT[op].Init()
        dqlMutT[op] = &QueryString{Q:q.M}
        dqlMutT[op].Init()
    }
    for op, q := range(gqlQueries) {
        gqlT[op] = &QueryString{Q:q}
        gqlT[op].Init()
    }

    return &Dgraph{
        gqlAddr: dgraphApiAddr,
        grpcAddr: grpcAddr,
        gqlTemplates: gqlT,
        dqlTemplates: dqlT,
        dqlMutTemplates: dqlMutT,
    }
}

//
// Internals
//

func (dg Dgraph) getGqlQuery(op string, m map[string]string) string {
    var q string
    if _q, ok := dg.gqlTemplates[op]; ok {
        q = _q.Format(m)
    } else {
        panic("unknonw GQL query op: " + op)
    }
    return q
}

func (dg Dgraph) getDqlQuery(op string, m map[string]string) string {
    var q string
    if _q, ok := dg.dqlTemplates[op]; ok {
        q = _q.Format(m)
    } else {
        panic("unknonw DQL query op: " + op)
    }
    return q
}

func (dg Dgraph) getDqlMutQuery(op string, m map[string]string) string {
    var q string
    if _q, ok := dg.dqlMutTemplates[op]; ok {
        q = _q.Format(m)
    } else {
        panic("unknonw DQL query op: " + op)
    }
    return q
}

// Get the grpc Dgraph client.
func (dg Dgraph) getDgraphClient() (dgClient *dgo.Dgraph, cancelFunc func()) {
    conn, err := grpc.Dial(dg.grpcAddr, grpc.WithInsecure())
    if err != nil {
        log.Fatal("While trying to dial gRPC: ", err)
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

func (dg Dgraph) GetRootUctx() model.UserCtx {
    return model.UserCtx{
        Username: "root",
        Rights: model.UserRights{CanLogin:false, CanCreateRoot:true, Type:model.UserTypeRoot},
        Hit: 1,
    }
}
func (dg Dgraph) GetRegularUctx() model.UserCtx {
    return model.UserCtx{
        Username: "root",
        Rights: model.UserRights{CanLogin:false, CanCreateRoot:true, Type:model.UserTypeRegular},
        Hit: 1,
    }
}

func (dg Dgraph) BuildGqlToken(uctx model.UserCtx, t time.Duration) string {
    // Get unique rootnameid
    var rootids []string
    var ownids []string
    check := make(map[string]bool)
    for _, d := range uctx.Roles {
        rid, _ := codec.Nid2rootid(d.Nameid)
        if *d.RoleType == model.RoleTypeOwner {
            ownids = append(ownids, rid)
            // Owner is also a member !
            //continue
        }
        if _, v := check[rid]; !v {
            // @DEBUG: if pending is not included here, invited user, or author of tension created with BOT
            // won't be able to see on tensins. But, authorizing it, make give a visibity hole for private circle
            // that can be seen by **self-invited** user.
            //if *d.RoleType != model.RoleTypePending && *d.RoleType != model.RoleTypeRetired {
            if *d.RoleType != model.RoleTypeRetired {
                check[rid] = true
                rootids = append(rootids, rid)
            }
        }
    }

    // Dgraph failed to run the @auth query if the variable is null
    // see https://discuss.dgraph.io/t/auth-rule-with-or-condition-fail-if-an-empty-list-is-given-as-variable/16251
    if len(rootids) == 0 { rootids = append(rootids, "") }
    if len(ownids) == 0 { ownids = append(ownids, "") }

    // Build claims
    dgClaims := DgraphClaims{
        Username: uctx.Username,
        UserType: uctx.Rights.Type,
        Rootids: rootids,
        Ownids: ownids,
    }
    claims := map[string]interface{}{
        "https://fractale.co/jwt/claims": dgClaims,
    }
    jwtauth.SetIssuedNow(claims)
    jwtauth.SetExpiry(claims, time.Now().UTC().Add(t))

    // Create token
    tkm := jwtauth.New("RS256", dgraphPrivateKey, dgraphPublicKey)
    //tkm := jwtauth.New("HS256", []byte("checkJwkToken_or_pubkey"), []byte("checkJwkToken_or_pubkey"))
    _, token, err := tkm.Encode(claims)
    if err != nil { panic("Dgraph JWT error: " + err.Error()) }
    return token
}

// Post send a post request to the Graphql client.
func (dg Dgraph) postql(uctx model.UserCtx, data []byte, res interface{}) error {
    req, err := http.NewRequest("POST", dg.gqlAddr, bytes.NewBuffer(data))
    req.Header.Set("Content-Type", "application/json")

    // Set dgraph token
    gqlToken := dg.BuildGqlToken(uctx, time.Minute*10)
    req.Header.Set("X-Frac6-Auth", gqlToken)

    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil { return err }
    defer resp.Body.Close()

    return json.NewDecoder(resp.Body).Decode(res)
}

//
// DQL (ex GraphQL+-) Interface
//

// QueryDql runs a query on dgraph (...QueryDql)
func (dg Dgraph) QueryDql(op string, maps map[string]string) (*api.Response, error) {
    // init client
    dgc, cancel := dg.getDgraphClient()
    defer cancel()
    ctx := context.Background()
    txn := dgc.NewTxn()
    defer txn.Discard(ctx)

    // Get the Query
    q := dg.getDqlQuery(op, maps)
    // Send Request
    if !strings.HasPrefix(op, "countHas") && buildMode == "DEV" {
        fmt.Println(op)
    }
    //fmt.Println(string(q))
    res, err := txn.Query(ctx, q)
    //fmt.Println(res)
    return res, err
}

//MutateWithQueryDql runs an upsert block mutation by first querying query
//and then mutate based on the result.
func (dg Dgraph) MutateWithQueryDql(query string, mu *api.Mutation) (error) {
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

//MutateWithQueryDql2 runs an upsert block mutation by first querying query
//and then mutate based on the result.
func (dg Dgraph) MutateWithQueryDql2(op string, maps map[string]string) (*api.Response, error) {
    // init client
    dgc, cancel := dg.getDgraphClient()
    defer cancel()
    ctx := context.Background()
    txn := dgc.NewTxn()
    defer txn.Discard(ctx)

    query := dg.getDqlQuery(op, maps)
    muSet := dg.getDqlMutQuery(op, maps)

    req := &api.Request{
        Query: query,
        Mutations: []*api.Mutation{
            &api.Mutation{
                SetNquads: []byte(muSet),
            },
        },
        CommitNow: true,
    }

    res, err := txn.Do(ctx, req)
    return res, err
}

//MutateUpsertDql adds a new object in the database if it doesn't exist
func (dg Dgraph) MutateUpsertDql_(object map[string]interface{}, dtype string, upsertField string, upsertVal string) error {
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
func (dg Dgraph) Push_(object map[string]interface{}, dtype string) (string, error) {
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

//ClearNodes remove nodes from theirs uid and their edges in dgraph.
// refers to https://dgraph.io/docs/mutations/json-mutation-format/#deleting-edges
func (dg Dgraph) ClearNodes_(uids ...string) (error) {
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

//ClearEdges remove edges from their uid
func (dg Dgraph) ClearEdges_(key string, value string, delMap map[string]interface{}) (error) {
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

    err := dg.MutateWithQueryDql(query, mutation)
    return err
}

//
// GraphQL Interface
//

// QueryGql query the Dgraph Graphql endpoint by following a http request.
// It map the result in to given data structure
func (dg Dgraph) QueryGql(uctx model.UserCtx, op string, reqInput map[string]string, data interface{}) error {
    // Get the query
    queryName := reqInput["QueryName"]
    q := dg.getGqlQuery(op, reqInput)

    // Send the dgraph request and follow the results
    res := &GqlRes{}
    //fmt.Println("request ->", string(q))
    err := dg.postql(uctx, []byte(q), res)
    //fmt.Println("response ->", res)
    if err != nil { return err }

    switch v := data.(type) {
    case model.JsonAtom:
        for k, val := range res.Data {
            v[k] = val
        }
    default: // Interface{} data type (Payload)
        var config *mapstructure.DecoderConfig
        if op == "query" || op == "rawQuery" {
            // Decoder config to handle aliased request
            // @DEBUG: see bug #3c3f1f7
            config = &mapstructure.DecoderConfig{
                Result: data,
                TagName: "json",
                DecodeHook: mapstructure.ComposeDecodeHookFunc(
                    CleanAliasedMapHook(),
                    ToUnionHookFunc(),
                ),
            }
        } else {
            config = &mapstructure.DecoderConfig{TagName: "json", Result: data}
        }

        decoder, err := mapstructure.NewDecoder(config)
        if err != nil { return err }
        err = decoder.Decode(res.Data[queryName])
        if err != nil { return err }
    }

    if res.Errors != nil {
        err, _ := json.Marshal(res.Errors)
        //return fmt.Errorf(string(err))
        return &GraphQLError{string(err)}
    }
    return err
}

