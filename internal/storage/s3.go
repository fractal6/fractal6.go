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

// Package storage wraps an S3-compatible object store (Garage or MinIO) used
// to persist file attachments referenced by Comment.files in the schema.
//
// All access from the rest of the codebase MUST go through this package; it
// keeps the SDK surface small and swappable, and centralises bucket/region
// concerns in one place.
//
// Bytes are never streamed back to clients through this package — callers
// (web/handlers/files.go) issue presigned URLs and let the storage backend
// serve the bytes directly. This keeps Fractale out of the data path and
// preserves the existing 4-layer auth model (auth runs in Go before the URL
// is ever minted).
package storage

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"sync"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/spf13/viper"
)

// Config holds the resolved [storage] section of config.toml.
type Config struct {
	Endpoint        string // e.g. "garage.fractale.co" or "127.0.0.1:3900"
	Region          string // e.g. "garage" (Garage default) or "us-east-1"
	Bucket          string // e.g. "fractale-attachments"
	AccessKey       string
	SecretKey       string
	UseSSL          bool
	PublicURLPrefix string // optional: rewrites presigned host to a public-facing CDN/proxy
}

// Client is the storage handle; wrap minio.Client and a resolved bucket.
type Client struct {
	cfg Config
	mc  *minio.Client
}

var (
	defaultClient *Client
	once          sync.Once
	initErr       error
)

// LoadConfig reads the [storage] section of config.toml via viper.
// Missing fields fall back to sensible defaults so dev environments can
// run a vanilla Garage with minimal config.
func LoadConfig() Config {
	return Config{
		Endpoint:        viper.GetString("storage.endpoint"),
		Region:          viper.GetString("storage.region"),
		Bucket:          viper.GetString("storage.bucket"),
		AccessKey:       viper.GetString("storage.access_key"),
		SecretKey:       viper.GetString("storage.secret_key"),
		UseSSL:          viper.GetBool("storage.use_ssl"),
		PublicURLPrefix: viper.GetString("storage.public_url_prefix"),
	}
}

// New builds a Client from the given config. Returns an error if the
// endpoint/credentials are missing — callers should treat storage as
// optional and disable upload features if init fails.
func New(cfg Config) (*Client, error) {
	if cfg.Endpoint == "" || cfg.Bucket == "" {
		return nil, fmt.Errorf("storage: endpoint and bucket are required")
	}
	if cfg.AccessKey == "" || cfg.SecretKey == "" {
		return nil, fmt.Errorf("storage: access_key and secret_key are required")
	}
	if cfg.Region == "" {
		cfg.Region = "garage" // Garage's default region name
	}

	mc, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
		Region: cfg.Region,
	})
	if err != nil {
		return nil, fmt.Errorf("storage: minio client init: %w", err)
	}

	return &Client{cfg: cfg, mc: mc}, nil
}

// GetDefault returns a process-wide singleton initialised from viper config.
// Initialisation is lazy + cached; subsequent calls reuse the same client.
func GetDefault() (*Client, error) {
	once.Do(func() {
		defaultClient, initErr = New(LoadConfig())
	})
	return defaultClient, initErr
}

// Bucket returns the configured bucket name (mainly for logging/diagnostics).
func (c *Client) Bucket() string { return c.cfg.Bucket }

// Put streams an object to the bucket. Size MUST be >= 0; pass -1 only for
// truly unknown sizes (forces multipart, slower). ContentType is mandatory
// because clients receive it via the presigned redirect (sets the Content-Type
// header on download).
func (c *Client) Put(ctx context.Context, key string, body io.Reader, size int64, contentType string) error {
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	_, err := c.mc.PutObject(ctx, c.cfg.Bucket, key, body, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	return err
}

// Delete removes an object. Idempotent: deleting a missing key is not an error.
func (c *Client) Delete(ctx context.Context, key string) error {
	return c.mc.RemoveObject(ctx, c.cfg.Bucket, key, minio.RemoveObjectOptions{})
}

// PresignGet returns a short-lived URL that the browser can hit directly.
// ttl SHOULD be a few minutes — long enough to render an image, short enough
// that a leaked URL has limited blast radius. See docs/file-attachments.md.
//
// contentDisposition, when non-empty, is forwarded as the S3
// `response-content-disposition` parameter so the storage backend stamps it
// on the served response. Callers compose the full header value (e.g.
// `inline; filename*=UTF-8''hello.png` or `attachment; filename=...`); see
// the inline-safe MIME allowlist in web/handlers/files.go for the policy.
func (c *Client) PresignGet(ctx context.Context, key string, ttl time.Duration, contentDisposition string) (string, error) {
	reqParams := url.Values{}
	if contentDisposition != "" {
		reqParams.Set("response-content-disposition", contentDisposition)
	}
	u, err := c.mc.PresignedGetObject(ctx, c.cfg.Bucket, key, ttl, reqParams)
	if err != nil {
		return "", err
	}
	out := u.String()
	// If a public-facing URL prefix is configured (e.g. CDN or reverse proxy
	// in front of Garage), rewrite the host portion so browsers don't try to
	// hit a private endpoint.
	if c.cfg.PublicURLPrefix != "" {
		out = c.cfg.PublicURLPrefix + u.RequestURI()
	}
	return out, nil
}

// EnsureBucket creates the bucket if it doesn't exist. Intended for first-run
// bootstrap (e.g. dev/CI); production setups should pre-create the bucket via
// Ansible or `garage bucket create`.
func (c *Client) EnsureBucket(ctx context.Context) error {
	exists, err := c.mc.BucketExists(ctx, c.cfg.Bucket)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	return c.mc.MakeBucket(ctx, c.cfg.Bucket, minio.MakeBucketOptions{Region: c.cfg.Region})
}
