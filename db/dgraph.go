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
	crand "crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
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
	"google.golang.org/grpc/backoff"
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

// dqlTimeout bounds every DQL attempt (var so tests can shorten it).
var dqlTimeout = 30 * time.Second

// Draph database clients
type Dgraph struct {
	// HTTP/Graphql and GPRC/DQL client address
	gqlAddr  string
	grpcAddr string
	// Shared gRPC connection and dgo client, reused across operations.
	// Both are nil for HTTP-only instances (grpcAddr empty).
	conn *grpc.ClientConn
	dgc  *dgo.Dgraph
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

// Err returns the GraphQL errors as a single error, or nil.
func (r *GqlRes) Err() error {
	if r.Errors == nil {
		return nil
	}
	b, _ := json.Marshal(r.Errors)
	return &GraphQLError{string(b)}
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

// NewDgraph build a client for the given endpoints. The gRPC connection is
// created once and reused (grpc.NewClient is lazy: no I/O until the first RPC).
func NewDgraph(gqlAddr, grpcAddr string) *Dgraph {
	dg := &Dgraph{
		gqlAddr:  gqlAddr,
		grpcAddr: grpcAddr,
	}
	if grpcAddr != "" {
		conn, err := grpc.NewClient(grpcAddr,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			// Cap reconnect backoff (default 120s) so an alpha restart is picked up quickly.
			grpc.WithConnectParams(grpc.ConnectParams{Backoff: backoff.Config{
				BaseDelay: time.Second, Multiplier: 1.6, Jitter: 0.2, MaxDelay: 5 * time.Second,
			}}),
		)
		if err != nil {
			log.Fatal("While trying to dial gRPC: ", err)
		}
		dg.conn = conn
		dg.dgc = dgo.NewDgraphClient(api.NewDgraphClient(conn))
	}
	return dg
}

// Close releases the shared gRPC connection.
func (dg *Dgraph) Close() error {
	if dg.conn == nil {
		return nil
	}
	conn := dg.conn
	dg.conn, dg.dgc = nil, nil
	return conn.Close()
}

// SetTestDB overrides the global db_dg singleton for integration tests.
func SetTestDB(gqlAddr, grpcAddr string) {
	db_dg.Close()
	db_dg = NewDgraph(gqlAddr, grpcAddr)
}

// SetTestJWTKeys installs an ephemeral keypair when no Dgraph JWT key is
// configured, so tests can query a fake Dgraph endpoint. No-op otherwise.
func SetTestJWTKeys() {
	if dgraphPrivateKey != nil && dgraphPublicKey != nil {
		return
	}
	key, err := rsa.GenerateKey(crand.Reader, 2048)
	if err != nil {
		panic(err)
	}
	dgraphPrivateKey, dgraphPublicKey = key, &key.PublicKey
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

	return NewDgraph(dgraphApiAddr, grpcAddr)
}

// Ping checks the Dgraph alpha is up: /health on the HTTP/GraphQL port, plus a
// trivial read-only DQL query over gRPC. Startup healthcheck, see cmd/health.go.
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

	if dg.dgc == nil {
		return fmt.Errorf("dgraph: no grpc client configured")
	}
	if _, err := dg.dgc.NewReadOnlyTxn().Query(ctx, "{ q(func: uid(0x1)) { uid } }"); err != nil {
		return fmt.Errorf("dgraph: grpc %s unreachable: %w", dg.grpcAddr, err)
	}
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

// runDqlTxn executes a query block plus optional mutations in a fresh txn:
// read-only (no Zero ts allocation) when there is no mutation, auto-commit
// otherwise. Each attempt is bounded by dqlTimeout and retried on transient
// conflicts via dgraphRetry. Same retry contract as QueryGql for the GraphQL path.
func (dg Dgraph) runDqlTxn(query string, mutations []*api.Mutation) (*api.Response, error) {
	if dg.dgc == nil {
		return nil, fmt.Errorf("dgraph: no grpc client configured")
	}
	return Retry(dgraphRetry, func() (*api.Response, error) {
		ctx, cancel := context.WithTimeout(context.Background(), dqlTimeout)
		defer cancel()
		if len(mutations) == 0 {
			return dg.dgc.NewReadOnlyTxn().Query(ctx, query)
		}
		txn := dg.dgc.NewTxn()
		defer txn.Discard(ctx)
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

// dgraphRetry: up to 10 attempts with a 10-100ms jittered backoff on isDgraphConflict. Shared by the DQL and GraphQL paths.
var dgraphRetry = RetryPolicy{
	Name:     "dgraph",
	Attempts: 10,
	Delay:    func(int) time.Duration { return time.Duration(10+rand.Intn(91)) * time.Millisecond },
	RetryIf:  isDgraphConflict,
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

	// Send the dgraph request. Transaction aborts surface as GraphQL errors
	// (res.Err()), which dgraphRetry treats like any other conflict.
	// @DEBUG: see https://discuss.hypermode.com/t/transactions-in-graphql/6861/10
	res, err := Retry(dgraphRetry, func() (*GqlRes, error) {
		res := &GqlRes{}
		// fmt.Println("request ->", string(q))
		if err := dg.postql(uctx, []byte(q), res); err != nil {
			return nil, err
		}
		// fmt.Println("response ->", res)
		return res, res.Err()
	})
	if res == nil {
		return err // transport error
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

	return err // GraphQL errors, if any
}
