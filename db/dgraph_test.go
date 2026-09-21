/*
 * Fractale - Self-organisation for humans.
 * Copyright (C) 2026 Fractale Co
 *
 * This file is part of Fractale.
 */

// Client lifecycle, txn shape, timeout, retry and reconnect behaviour of the
// shared gRPC/DQL client (dgraph.go), against an in-process fake Dgraph.

package db

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/dgraph-io/dgo/v200/protos/api"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// fakeDql is a minimal api.DgraphServer: records requests, can answer
// "Please retry" a given number of times and delay its answers.
type fakeDql struct {
	api.UnimplementedDgraphServer
	mu    sync.Mutex
	reqs  []*api.Request
	fails int
	delay time.Duration
}

func (f *fakeDql) Query(ctx context.Context, req *api.Request) (*api.Response, error) {
	f.mu.Lock()
	f.reqs = append(f.reqs, req)
	fail := f.fails > 0
	if fail {
		f.fails--
	}
	f.mu.Unlock()
	if fail {
		return nil, status.Error(codes.Aborted, "Transaction has been aborted. Please retry")
	}
	select {
	case <-time.After(f.delay):
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	return &api.Response{Json: []byte(`{"all":[]}`), Txn: &api.TxnContext{}}, nil
}

func (f *fakeDql) requests() []*api.Request {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]*api.Request(nil), f.reqs...)
}

// serveFakeDql serves f on addr ("" for a random port) until t ends or stop is called.
func serveFakeDql(t *testing.T, f *fakeDql, addr string) (string, func()) {
	t.Helper()
	if addr == "" {
		addr = "127.0.0.1:0"
	}
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	srv := grpc.NewServer()
	api.RegisterDgraphServer(srv, f)
	go srv.Serve(lis)
	t.Cleanup(srv.Stop)
	return lis.Addr().String(), srv.Stop
}

func TestDgraphClientLifecycle(t *testing.T) {
	dg := NewDgraph("http://localhost:8080/graphql", "localhost:9080")
	if dg.conn == nil || dg.dgc == nil {
		t.Fatal("expected a shared grpc client")
	}
	if cp := *dg; cp.dgc != dg.dgc {
		t.Error("client not shared across value receivers")
	}
	if err := dg.Close(); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}
	if dg.conn != nil || dg.dgc != nil {
		t.Error("Close should clear the client")
	}
	if err := dg.Close(); err != nil {
		t.Errorf("Close is not idempotent: %v", err)
	}
}

// fakeHealth serves Dgraph's /health endpoint.
func fakeHealth(t *testing.T) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/health" {
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return srv.URL + "/graphql"
}

func TestDgraphHttpOnlyClient(t *testing.T) {
	dg := NewDgraph(fakeHealth(t), "")
	if dg.dgc != nil {
		t.Fatal("expected no grpc client without a grpc address")
	}
	if _, err := dg.runDqlTxn("{ all(func: has(Node.nameid)) { uid } }", nil); err == nil {
		t.Error("expected an error on the DQL path without a grpc client")
	}
	if err := dg.Ping(context.Background()); err == nil || !strings.Contains(err.Error(), "grpc") {
		t.Errorf("Ping without grpc client = %v, want grpc error", err)
	}
}

func TestRunDqlTxn_TxnShape(t *testing.T) {
	f := &fakeDql{}
	addr, _ := serveFakeDql(t, f, "")
	dg := NewDgraph("", addr)
	t.Cleanup(func() { dg.Close() })

	if _, err := dg.runDqlTxn("{ all(func: uid(0x1)) { uid } }", nil); err != nil {
		t.Fatalf("read: %v", err)
	}
	mu := &api.Mutation{SetNquads: []byte(`<0x1> <Node.name> "x" .`)}
	if _, err := dg.runDqlTxn("{ q(func: uid(0x1)) { uid } }", []*api.Mutation{mu}); err != nil {
		t.Fatalf("write: %v", err)
	}

	reqs := f.requests()
	if len(reqs) != 2 {
		t.Fatalf("got %d requests, want 2", len(reqs))
	}
	if !reqs[0].ReadOnly || reqs[0].BestEffort || reqs[0].CommitNow {
		t.Errorf("read txn: ReadOnly=%v BestEffort=%v CommitNow=%v, want true/false/false", reqs[0].ReadOnly, reqs[0].BestEffort, reqs[0].CommitNow)
	}
	if reqs[1].ReadOnly || !reqs[1].CommitNow || len(reqs[1].Mutations) != 1 {
		t.Errorf("write txn: ReadOnly=%v CommitNow=%v mutations=%d, want false/true/1", reqs[1].ReadOnly, reqs[1].CommitNow, len(reqs[1].Mutations))
	}
}

func TestRunDqlTxn_Timeout(t *testing.T) {
	f := &fakeDql{delay: time.Second}
	addr, _ := serveFakeDql(t, f, "")
	dg := NewDgraph("", addr)
	t.Cleanup(func() { dg.Close() })

	old := dqlTimeout
	dqlTimeout = 50 * time.Millisecond
	t.Cleanup(func() { dqlTimeout = old })

	_, err := dg.runDqlTxn("{ all(func: uid(0x1)) { uid } }", nil)
	if status.Code(err) != codes.DeadlineExceeded {
		t.Fatalf("err = %v, want DeadlineExceeded", err)
	}
	if n := len(f.requests()); n != 1 {
		t.Errorf("timeout was retried: %d requests", n)
	}
}

func TestRunDqlTxn_RetriesOnConflict(t *testing.T) {
	f := &fakeDql{fails: 2}
	addr, _ := serveFakeDql(t, f, "")
	dg := NewDgraph("", addr)
	t.Cleanup(func() { dg.Close() })

	if _, err := dg.runDqlTxn("{ all(func: uid(0x1)) { uid } }", nil); err != nil {
		t.Fatalf("expected success after retries, got %v", err)
	}
	if n := len(f.requests()); n != 3 {
		t.Errorf("got %d attempts, want 3", n)
	}
}

func TestQueryGql_RetriesOnConflict(t *testing.T) {
	SetTestJWTKeys()
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls < 3 {
			w.Write([]byte(`{"errors":[{"message":"Transaction has been aborted. Please retry"}]}`))
			return
		}
		w.Write([]byte(`{"data":{"addNode":{"node":[{"id":"0x1"}]}}}`))
	}))
	t.Cleanup(srv.Close)
	dg := NewDgraph(srv.URL, "")

	var out struct {
		Node []struct{ ID string } `json:"node"`
	}
	err := dg.QueryGql(testUctx, "add", map[string]string{
		"QueryName": "addNode", "InputType": "AddNodeInput", "InputPayload": "[]", "QueryGraph": "node { id }",
	}, &out)
	if err != nil {
		t.Fatalf("expected success after retries, got %v", err)
	}
	if calls != 3 {
		t.Errorf("got %d calls, want 3", calls)
	}
	if len(out.Node) != 1 || out.Node[0].ID != "0x1" {
		t.Errorf("decoded %+v, want one node 0x1", out)
	}
}

func TestPing(t *testing.T) {
	addr, stop := serveFakeDql(t, &fakeDql{}, "")
	dg := NewDgraph(fakeHealth(t), addr)
	t.Cleanup(func() { dg.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := dg.Ping(ctx); err != nil {
		t.Fatalf("Ping on live fake: %v", err)
	}

	stop()
	ctx2, cancel2 := context.WithTimeout(context.Background(), time.Second)
	defer cancel2()
	err := dg.Ping(ctx2)
	if err == nil || !strings.Contains(err.Error(), "grpc") {
		t.Errorf("Ping with grpc down = %v, want grpc error", err)
	}
}

// The shared conn must recover once the alpha comes back on the same address.
func TestRunDqlTxn_Reconnect(t *testing.T) {
	f := &fakeDql{}
	addr, stop := serveFakeDql(t, f, "")
	dg := NewDgraph("", addr)
	t.Cleanup(func() { dg.Close() })

	if _, err := dg.runDqlTxn("{ all(func: uid(0x1)) { uid } }", nil); err != nil {
		t.Fatalf("initial query: %v", err)
	}
	stop()
	if _, err := dg.runDqlTxn("{ all(func: uid(0x1)) { uid } }", nil); err == nil {
		t.Fatal("expected failure while server is down")
	}
	serveFakeDql(t, f, addr)

	deadline := time.Now().Add(10 * time.Second)
	var err error
	for time.Now().Before(deadline) {
		if _, err = dg.runDqlTxn("{ all(func: uid(0x1)) { uid } }", nil); err == nil {
			return
		}
		if !errors.Is(err, context.DeadlineExceeded) && status.Code(err) != codes.Unavailable {
			t.Fatalf("unexpected error while reconnecting: %v", err)
		}
		time.Sleep(200 * time.Millisecond)
	}
	t.Fatalf("did not reconnect within 10s: %v", err)
}
