/*
 * Fractale - Self-organisation for humans.
 * Copyright (C) 2024 Fractale Co
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
	"bytes"
	"context"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"github.com/go-chi/jwtauth/v5"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"text/template"
	"time"
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
var DOMAIN string

// Database client
var DB *Dgraph

// Draph database clients
type Dgraph struct {
	// HTTP/Graphql and GPRC/DQL client address
	gqlAddr  string
	grpcAddr string
}

type DgraphClaims struct {
	Username string         `json:"USERNAME"`
	UserType model.UserType `json:"USERTYPE"`
	// Rootnameid where user is Member
	Rootids []string `json:"ROOTIDS"`
	// Rootnameid where user is Owner
	Ownids []string `json:"OWNIDS"`
}

//
// DQL response
//

type DqlResp struct {
	All []map[string]interface{} `json:"all"`
}

type DqlRespCount struct {
	All  []map[string]int `json:"all"`
	All2 []map[string]int `json:"all2"`
}

//
// GQL response
//

type GqlRes struct {
	Data   model.JsonAtom   `json:"data"`
	Errors []model.JsonAtom `json:"errors"` // message, locations, path, extensions
}

type GraphQLError struct {
	msg string
}

func (e *GraphQLError) Error() string {
	return fmt.Sprintf("%s", e.msg)
}

//
// Initialization
//

func init() {
	InitViper()
	DOMAIN = viper.GetString("server.domain")
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
	} else if os.Getenv("DGRAPH_PUBLIC_KEY") != "" {
		pub_key = os.Getenv("DGRAPH_PUBLIC_KEY")
	}
	// Get Jwt private key
	if fn := viper.GetString("db.dgraph_private_key"); fn != "" {
		if content, err := ioutil.ReadFile(fn); err != nil {
			log.Fatal(err)
		} else {
			priv_key = string(content)
		}
	} else if os.Getenv("DGRAPH_PRIVATE_KEY") != "" {
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
	HOSTDB := viper.GetString("db.hostname")
	PORTDB := viper.GetString("db.port_graphql")
	PORTGRPC := viper.GetString("db.port_grpc")
	APIDB := viper.GetString("db.api")
	dgraphApiAddr := "http://" + HOSTDB + ":" + PORTDB + "/" + APIDB
	grpcAddr := HOSTDB + ":" + PORTGRPC

	if HOSTDB == "" {
		panic("Viper error: not host found")
	} else {
		// @DEBUG: log level, Viper!
		//fmt.Println("Dgraph Graphql addr:", dgraphApiAddr)
		//fmt.Println("Dgraph Grpc addr:", grpcAddr)
	}

	return &Dgraph{
		gqlAddr:  dgraphApiAddr,
		grpcAddr: grpcAddr,
	}
}

func RawFormat(q string, maps map[string]string) string {
	template := template.Must(template.New("graphql").Parse(q))
	buf := bytes.Buffer{}
	template.Execute(&buf, maps)
	return buf.String()
}

//
// Internals
//

func (dg Dgraph) getGqlQuery(op string, m map[string]string) string {
	var q string
	if _q, ok := gqlQueries[op]; ok {
		q = RawFormat(CleanString(_q, false), m)
	} else {
		panic("unknonw GQL query op: " + op)
	}
	return q
}

func (dg Dgraph) getDqlQuery(op string, m map[string]string) string {
	var q string
	if _q, ok := dqlQueries[op]; ok {
		q = RawFormat(CleanString(_q, false), m)
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

	cancelFunc = func() {
		if err := conn.Close(); err != nil {
			log.Printf("Error while closing connection:%v", err)
		}
	}
	return
}

func (dg Dgraph) GetRootUctx() model.UserCtx {
	return model.UserCtx{
		Username: "root",
		Rights:   model.UserRights{CanLogin: false, CanCreateRoot: true, Type: model.UserTypeRoot},
		Hit:      1,
	}
}

func (dg Dgraph) BuildGqlToken(uctx model.UserCtx, t time.Duration) string {
	// Get unique rootnameid
	var rootids []string
	var ownids []string
	check := make(map[string]bool)
	for _, d := range uctx.Roles {
		if d.RoleType == nil {
			// Happens if a user get assigned first link of a circle...
			continue
		}
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
	if len(rootids) == 0 {
		rootids = append(rootids, "")
	}
	if len(ownids) == 0 {
		ownids = append(ownids, "")
	}

	// Build claims
	dgClaims := DgraphClaims{
		Username: uctx.Username,
		UserType: uctx.Rights.Type,
		Rootids:  rootids,
		Ownids:   ownids,
	}
	claims := map[string]interface{}{
		"https://" + DOMAIN + "/jwt/claims": dgClaims,
	}
	jwtauth.SetIssuedNow(claims)
	jwtauth.SetExpiry(claims, time.Now().UTC().Add(t))

	// Create token
	tkm := jwtauth.New("RS256", dgraphPrivateKey, dgraphPublicKey)
	//tkm := jwtauth.New("HS256", []byte("checkJwkToken_or_pubkey"), []byte("checkJwkToken_or_pubkey"))
	_, token, err := tkm.Encode(claims)
	if err != nil {
		panic("Dgraph JWT error: " + err.Error())
	}
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
	if err != nil {
		return err
	}
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
	if viper.GetString("rootCmd") == "api" && !strings.HasPrefix(op, "count") && buildMode == "DEV" {
		// @DEBUG LEVEL
		fmt.Println(op)
	}
	//fmt.Println(string(q))
	res, err := txn.Query(ctx, q)
	//fmt.Println(res)
	return res, err
}

// MutateWithQueryDql runs an upsert block mutation by first querying query
// and then mutate based on the result.
func (dg Dgraph) MutateWithQueryDql(query string, mu *api.Mutation) error {
	// init client
	dgc, cancel := dg.getDgraphClient()
	defer cancel()
	ctx := context.Background()
	txn := dgc.NewTxn()
	defer txn.Discard(ctx)

	req := &api.Request{
		Query:     query,
		Mutations: []*api.Mutation{mu},
		CommitNow: true,
	}

	_, err := txn.Do(ctx, req)
	return err
}

// MutateWithQueryDql3 runs an upsert block mutations by first querying query
// and then mutate based on the result. Accepte conditions.
func (dg Dgraph) MutateWithQueryDql3(q QueryMut, maps map[string]string) (*api.Response, error) {
	// init client
	dgc, cancel := dg.getDgraphClient()
	defer cancel()
	ctx := context.Background()
	txn := dgc.NewTxn()
	defer txn.Discard(ctx)

	query := RawFormat(q.Q, maps)
	mutations := []*api.Mutation{}
	for _, m := range q.M {
		mu := api.Mutation{}
		muSet := RawFormat(m.S, maps)
		muDel := RawFormat(m.D, maps)
		cond := RawFormat(m.C, maps)
		if muSet != "" {
			mu.SetNquads = []byte(muSet)
		}
		if muDel != "" {
			mu.DelNquads = []byte(muDel)
		}
		if cond != "" {
			mu.Cond = cond
		}
		mutations = append(mutations, &mu)
	}

	//fmt.Println(query)
	//fmt.Println(mutations)

	if len(q.M) == 0 {
		return txn.Query(ctx, query)
	}

	req := &api.Request{
		Query:     query,
		Mutations: mutations,
		CommitNow: true,
	}

	return txn.Do(ctx, req)
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
	if err != nil {
		return err
	}

	switch v := data.(type) {
	case model.JsonAtom:
		for k, val := range res.Data {
			v[k] = val
		}
	default: // Interface{} data type (Payload)
		config := &mapstructure.DecoderConfig{
			Result:  data,
			TagName: "json",
			DecodeHook: mapstructure.ComposeDecodeHookFunc(
				// Decoder config to handle aliased request
				// @DEBUG: see bug #3c3f1f7
				// Not needed since version 5.0.10 of elm-graphql that do not used hashes by defaut.
				// Not that alias won be supported since we know to handle it with gqlgen resolver.
				//CleanAliasedMapHook(),
				ToUnionHookFunc(),
			),
		}

		decoder, err := mapstructure.NewDecoder(config)
		if err != nil {
			return err
		}
		if err = decoder.Decode(res.Data[queryName]); err != nil {
			return err
		}
	}

	if res.Errors != nil {
		err, _ := json.Marshal(res.Errors)
		//return fmt.Errorf(string(err))
		return &GraphQLError{string(err)}
	}
	return err
}
