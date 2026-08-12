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
	"bytes"
	"context"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"text/template"
	"time"

	"github.com/go-chi/jwtauth/v5"
	"github.com/spf13/viper"

	//"github.com/vektah/gqlparser/v2/gqlerror"
	"github.com/dgraph-io/dgo/v200"
	"github.com/dgraph-io/dgo/v200/protos/api"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"fractale/fractal6.go/graph/codec"
	"fractale/fractal6.go/graph/model"
	. "fractale/fractal6.go/internal/tools"
)

var (
	dgraphPrivateKey *rsa.PrivateKey
	dgraphPublicKey  *rsa.PublicKey
	buildMode        string
	DOMAIN           string
)

// Database client
var db_dg *Dgraph

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
	All []map[string]any `json:"all"`
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
	return e.msg
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
	// Get Jwt public key: try config file first, then env var fallback.
	if fn := viper.GetString("db.dgraph_public_key"); fn != "" {
		if content, err := os.ReadFile(fn); err != nil {
			log.Printf("Warning: %v", err)
		} else {
			pub_key = string(content)
		}
	}
	if pub_key == "" && os.Getenv("DGRAPH_PUBLIC_KEY") != "" {
		pub_key = os.Getenv("DGRAPH_PUBLIC_KEY")
	}
	// Get Jwt private key: try config file first, then env var fallback.
	if fn := viper.GetString("db.dgraph_private_key"); fn != "" {
		if content, err := os.ReadFile(fn); err != nil {
			log.Printf("Warning: %v", err)
		} else {
			priv_key = string(content)
		}
	}
	if priv_key == "" && os.Getenv("DGRAPH_PRIVATE_KEY") != "" {
		priv_key = os.Getenv("DGRAPH_PRIVATE_KEY")
	}

	if pub_key != "" && priv_key != "" {
		dgraphPublicKey = ParseRsaPublic(pub_key)
		dgraphPrivateKey = ParseRsaPrivate(priv_key)
	} else {
		log.Println("Warning: DGRAPH_PRIVATE_KEY or DGRAPH_PUBLIC_KEY not found. JWT signing disabled.")
	}

	db_dg = initDB()
}

func GetDB() *Dgraph {
	return db_dg
}

// SetTestDB overrides the global db_dg singleton for integration tests.
func SetTestDB(gqlAddr, grpcAddr string) {
	db_dg = &Dgraph{
		gqlAddr:  gqlAddr,
		grpcAddr: grpcAddr,
	}
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
		// fmt.Println("Dgraph Graphql addr:", dgraphApiAddr)
		// fmt.Println("Dgraph Grpc addr:", grpcAddr)
	}

	return &Dgraph{
		gqlAddr:  dgraphApiAddr,
		grpcAddr: grpcAddr,
	}
}

// Ping checks the Dgraph alpha is up: /health on the HTTP/GraphQL port, plus a
// TCP dial on the gRPC port used for DQL. Startup healthcheck, see cmd/health.go.
func (dg Dgraph) Ping(ctx context.Context) error {
	u, err := url.Parse(dg.gqlAddr)
	if err != nil {
		return fmt.Errorf("dgraph: bad address %q: %w", dg.gqlAddr, err)
	}
	u.Path = "/health"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("dgraph: %s unreachable: %w", u.Host, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("dgraph: %s /health: %s", u.Host, resp.Status)
	}

	var d net.Dialer
	conn, err := d.DialContext(ctx, "tcp", dg.grpcAddr)
	if err != nil {
		return fmt.Errorf("dgraph: grpc %s unreachable: %w", dg.grpcAddr, err)
	}
	conn.Close()
	return nil
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
	conn, err := grpc.NewClient(dg.grpcAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal("While trying to dial gRPC: ", err)
	}

	dgClient = dgo.NewDgraphClient(api.NewDgraphClient(conn))
	// ctx := context.Background()

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
			// continue
		}
		if _, v := check[rid]; !v {
			// @DEBUG: if pending is not included here, invited user, or author of tension created with BOT
			// won't be able to see on tensins. But, authorizing it, make give a visibity hole for private circle
			// that can be seen by **self-invited** user.
			// if *d.RoleType != model.RoleTypePending && *d.RoleType != model.RoleTypeRetired {
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
	claims := map[string]any{
		"https://" + DOMAIN + "/jwt/claims": dgClaims,
	}
	jwtauth.SetIssuedNow(claims)
	jwtauth.SetExpiry(claims, time.Now().UTC().Add(t))

	// Create token
	if dgraphPrivateKey == nil || dgraphPublicKey == nil {
		panic("Dgraph JWT keys not loaded. Ensure public.pem and private.pem are available.")
	}
	tkm := jwtauth.New("RS256", dgraphPrivateKey, dgraphPublicKey)
	// tkm := jwtauth.New("HS256", []byte("checkJwkToken_or_pubkey"), []byte("checkJwkToken_or_pubkey"))
	_, token, err := tkm.Encode(claims)
	if err != nil {
		panic("Dgraph JWT error: " + err.Error())
	}
	return token
}

// Post send a post request to the Graphql client.
func (dg Dgraph) postql(uctx model.UserCtx, data []byte, res any) error {
	req, err := http.NewRequest("POST", dg.gqlAddr, bytes.NewBuffer(data))
	if err != nil {
		return err
	}
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

// QueryDql runs a read-only DQL query template. Routed through runDqlTxn so
// transient "Please retry" errors (replication lag, tablet rebalance) are
// retried transparently — same contract as the upsert path.
func (dg Dgraph) QueryDql(op string, maps map[string]string) (*api.Response, error) {
	q := dg.getDqlQuery(op, maps)
	if viper.GetString("rootCmd") == "api" && !strings.HasPrefix(op, "count") && buildMode == "DEV" {
		fmt.Println(op)
	}
	return dg.runDqlTxn(q, nil)
}

// runDqlTxn executes a query block plus optional mutations in a fresh
// auto-commit txn. Pass nil/empty mutations for a read-only query. Wrapped
// in withDqlRetry so transient conflicts/snapshot-staleness are invisible
// to callers. Same retry contract as QueryGql for the GraphQL path.
func (dg Dgraph) runDqlTxn(query string, mutations []*api.Mutation) (*api.Response, error) {
	return withDqlRetry(func() (*api.Response, error) {
		dgc, cancel := dg.getDgraphClient()
		defer cancel()
		ctx := context.Background()
		txn := dgc.NewTxn()
		defer txn.Discard(ctx)

		if len(mutations) == 0 {
			return txn.Query(ctx, query)
		}
		return txn.Do(ctx, &api.Request{
			Query:     query,
			Mutations: mutations,
			CommitNow: true,
		})
	})
}

// isDgraphConflict reports whether err is a transient Dgraph error that
// callers should retry against a fresh txn snapshot. Covers both write
// conflicts (CommitNow upserts losing the race) and read-side staleness
// (replication lag, tablet rebalance). Mirrors the marker used by Dgraph's
// dgo client and the existing GraphQL-path retry at QueryGql.
func isDgraphConflict(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "Please retry")
}

// withDqlRetry runs fn under the same retry policy as the GraphQL path:
// up to 10 attempts, 10-100ms jittered backoff, only on isDgraphConflict.
// Logs once when retries fire so contention shows up in observability
// instead of being silently swallowed.
func withDqlRetry(fn func() (*api.Response, error)) (*api.Response, error) {
	const maxAttempts = 10
	var (
		res *api.Response
		err error
	)
	for attempt := 0; attempt < maxAttempts; attempt++ {
		res, err = fn()
		if !isDgraphConflict(err) {
			if attempt > 0 {
				fmt.Printf("dql: succeeded after %d retries\n", attempt)
			}
			return res, err
		}
		time.Sleep(time.Duration(10+rand.Intn(91)) * time.Millisecond)
	}
	return res, err
}

// UpsertDql runs an upsert template (QueryMut + variable map): formats
// Q/M[i].S/D/C, then delegates to runDqlTxn. Retries on conflict.
func (dg Dgraph) UpsertDql(q QueryMut, maps map[string]string) (*api.Response, error) {
	query := RawFormat(q.Q, maps)
	mutations := make([]*api.Mutation, 0, len(q.M))
	for _, m := range q.M {
		mu := &api.Mutation{}
		if s := RawFormat(m.S, maps); s != "" {
			mu.SetNquads = []byte(s)
		}
		if d := RawFormat(m.D, maps); d != "" {
			mu.DelNquads = []byte(d)
		}
		if c := RawFormat(m.C, maps); c != "" {
			mu.Cond = c
		}
		mutations = append(mutations, mu)
	}
	return dg.runDqlTxn(query, mutations)
}

//
// GraphQL Interface
//

// QueryGql query the Dgraph Graphql endpoint by following a http request.
// It map the result in to given data structure
func (dg Dgraph) QueryGql(uctx model.UserCtx, op string, reqInput map[string]string, data any) error {
	// Get the query
	queryName := reqInput["QueryName"]
	q := dg.getGqlQuery(op, reqInput)

	// Send the dgraph request and follow the results
	res := &GqlRes{}
	// fmt.Println("request ->", string(q))
	err := dg.postql(uctx, []byte(q), res)
	// fmt.Println("response ->", res)

	// Check if error contains the transaction aborted message
	// @DEBUG: solve the issue https://discuss.hypermode.com/t/transactions-in-graphql/6861/10
	if res.Errors != nil {
		gqlErr, _ := json.Marshal(res.Errors)
		if strings.Contains(string(gqlErr), "Please retry") {
			// Retry up to 10 times
			for i := 0; i < 10; i++ {
				// Random sleep between 10 and 100 ms
				sleepTime := time.Duration(10+rand.Intn(91)) * time.Millisecond
				time.Sleep(sleepTime)

				// Retry the request
				res = &GqlRes{}
				err = dg.postql(uctx, []byte(q), res)

				// If success or different error, stop retrying
				if res.Errors == nil {
					break
				}
				gqlErr, _ = json.Marshal(res.Errors)
				if !strings.Contains(string(gqlErr), "Please retry") {
					break
				}
			}
		}
	}

	if err != nil {
		return err
	}

	switch v := data.(type) {
	case model.JsonAtom:
		for k, val := range res.Data {
			v[k] = val
		}
	default: // Interface{} data type (Payload)
		b, err := json.Marshal(res.Data[queryName])
		if err != nil {
			return err
		}
		if err = json.Unmarshal(b, data); err != nil {
			return err
		}
	}

	if res.Errors != nil {
		err, _ := json.Marshal(res.Errors)
		// return fmt.Errorf(string(err))
		return &GraphQLError{string(err)}
	}
	return err
}
