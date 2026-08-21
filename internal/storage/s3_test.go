package storage

import (
	"context"
	"net/url"
	"testing"
	"time"
)

// Presigned URLs must carry the public host (SigV4 signs Host), never the
// private data-plane endpoint.
func TestPresignGetUsesPublicHost(t *testing.T) {
	cfg := Config{
		Endpoint:        "127.0.0.1:3900",
		Region:          "garage",
		Bucket:          "fractale-storage",
		AccessKey:       "GKtest",
		SecretKey:       "secret",
		PublicURLPrefix: "https://files.example.org",
	}
	c, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := c.PresignGet(context.Background(), "orgas/x/f.pdf", time.Hour, "inline")
	if err != nil {
		t.Fatal(err)
	}
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if u.Host != "files.example.org" || u.Scheme != "https" {
		t.Fatalf("got %s://%s, want https://files.example.org", u.Scheme, u.Host)
	}
	if u.Query().Get("X-Amz-Signature") == "" {
		t.Fatal("missing signature")
	}

	cfg.PublicURLPrefix = ""
	c, _ = New(cfg)
	raw, err = c.PresignGet(context.Background(), "orgas/x/f.pdf", time.Hour, "")
	if err != nil {
		t.Fatal(err)
	}
	if u, _ := url.Parse(raw); u.Host != "127.0.0.1:3900" {
		t.Fatalf("no prefix: got host %s", u.Host)
	}

	cfg.PublicURLPrefix = "not a url"
	if _, err := New(cfg); err == nil {
		t.Fatal("want error on malformed public_url_prefix")
	}
}
