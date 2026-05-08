# garage — standalone Ansible role

Deploys [Garage](https://garagehq.deuxfleurs.fr/) as a single-node S3-compatible
object store on a Debian/Ubuntu host. Designed to back the Fractale file
attachment feature (see `docs/file-storage.md` in the fractal6.go repo).

> **Status**: standalone role, intended to be folded into the global Fractale
> Ansible inventory by a follow-up integration. Variables are namespaced
> `garage_*` so they should compose cleanly.

## What it does

1. Creates a dedicated `garage` system user + data/metadata directories.
2. Downloads the Garage binary (pinned version, sha256 verified) and installs
   it under `/usr/local/bin/garage`.
3. Renders `garage.toml` from a template (RPC + S3 + admin API endpoints,
   credentials read from variables/vault).
4. Installs a hardened systemd unit and starts the service.
5. (Optional) bootstraps a single-node cluster layout, an application bucket,
   and an S3 access key — emits the access/secret to a results file.

The role is **idempotent**: re-running won't re-bootstrap an existing cluster
or recreate a bucket that already exists.

## Required variables

| Variable | Default | Notes |
|---|---|---|
| `garage_rpc_secret` | — (required) | 32-byte hex secret. Generate with `openssl rand -hex 32`. Vault it. |
| `garage_admin_token` | — (required) | Token for the admin API. Vault it. |
| `garage_metrics_token` | — (required) | Token for the /metrics endpoint. Vault it. |
| `garage_bucket_name` | `fractale-storage` | Bucket created at bootstrap (if `garage_bootstrap_bucket: true`). |

## Common variables

See `defaults/main.yml` — every knob is documented inline. The defaults match
the `[storage]` block in `fractal6.go/config.toml`.

## Usage

```yaml
- hosts: garage_servers
  become: true
  roles:
    - role: garage
      vars:
        garage_rpc_secret:    "{{ vault_garage_rpc_secret }}"
        garage_admin_token:   "{{ vault_garage_admin_token }}"
        garage_metrics_token: "{{ vault_garage_metrics_token }}"
        garage_s3_public_host: "garage.fractale.co"   # the host name in the cert
```

## TLS

Garage's S3 API listens on plain HTTP and **expects to sit behind an
HTTPS-terminating reverse proxy** (nginx/caddy/traefik). This role does not
configure that proxy — `use_ssl = true` in fractal6.go's config refers to the
HTTPS endpoint your proxy exposes (e.g. `https://garage.fractale.co`).

A minimal nginx snippet is provided in `templates/nginx-garage.conf.j2` for
reference; wire it into your existing nginx role.

## Bootstrap output

When `garage_bootstrap_bucket: true`, the access key + secret created for the
bucket are written to `/etc/garage/bootstrap-result.yml` on the target host
(mode 0600, owned by root). Read it back via:

```sh
ansible <host> -b -m slurp -a 'src=/etc/garage/bootstrap-result.yml' | \
  awk '/content:/{print $2}' | base64 -d
```

Then plug the values into `fractal6.go/config.toml` `[storage]`.

## Version compatibility

Targets Garage **v2.x** (defaults pin `v2.1.0`). The config schema reflects
the v0.10 rename: `replication_mode` (string) became `replication_factor`
(int) plus a new required `consistency_mode`. If you need to target an older
Garage release, override `garage_version` *and* adjust the template — older
configs will be rejected at startup.

## References

- Upstream docs: https://garagehq.deuxfleurs.fr/documentation/
- Config reference: https://garagehq.deuxfleurs.fr/documentation/reference-manual/configuration/
- Downloads: https://garagehq.deuxfleurs.fr/download/
- Pinned version: see `garage_version` in `defaults/main.yml`.
