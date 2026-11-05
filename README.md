# synthetic-probe

[![ci](https://github.com/moveeeax/synthetic-probe/actions/workflows/ci.yml/badge.svg)](https://github.com/moveeeax/synthetic-probe/actions/workflows/ci.yml)
[![Go](https://img.shields.io/badge/go-1.26%20%7C%201.27-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![License: MIT](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)

> Blackbox-style HTTP / DNS / TLS checks without the operational weight — one static binary that speaks Prometheus.

`synthetic-probe` periodically probes a list of targets and exposes the results
at `/metrics`. It answers the three questions you actually page on: *is the
endpoint up*, *how slow is it*, and *when does the TLS certificate expire*. The
config hot-reloads, so you add or drop targets without a restart.

## What it does

- **HTTP(S)** — requests a URL and marks it up when the status matches
  `expect_status` (default `200`); records the observed status and latency.
- **DNS** — resolves a host and counts the returned addresses.
- **TLS** — connects and reports the days remaining on the leaf certificate,
  even if it is self-signed or already expired.
- Exposes everything as Prometheus gauges under the `synthetic_probe_` namespace.
- Builds into a single static binary; a `Dockerfile` for a `distroless` image
  is included (no prebuilt image is published).

## How it works

Each target gets its own goroutine on its own interval. A result is a small
struct (`up`, `duration`, `http_status`, `dns_records`, `cert_expiry`) that the
metrics layer maps onto gauge vectors. The config file is polled for changes on
a fixed cadence; when its mtime moves, the runner reloads, resets the metric
vectors so stale series from removed targets disappear, and restarts the probe
goroutines against the new config. A bad edit is logged and ignored — the last
good config keeps running.

## Install

```bash
go install github.com/moveeeax/synthetic-probe@latest
```

Or build from source:

```bash
git clone https://github.com/moveeeax/synthetic-probe
cd synthetic-probe
go build -o synthetic-probe .
```

## Usage

```bash
synthetic-probe --config examples/targets.yaml --listen :9115
```

Flags:

| Flag | Default | Meaning |
| --- | --- | --- |
| `--config` | `targets.yaml` | Path to the YAML target file |
| `--listen` | `:9115` | Address for the metrics server |
| `--reload-interval` | `10s` | How often the config file is checked for changes |
| `--version` | | Print version and exit |

A minimal config:

```yaml
default_interval: 15s
targets:
  - name: example-http
    type: http
    url: https://example.com
  - name: example-dns
    type: dns
    host: example.com
  - name: example-tls
    type: tls
    address: example.com:443
```

Scrape it:

```
$ curl -s localhost:9115/metrics | grep synthetic_probe
synthetic_probe_up{target="example-http",type="http"} 1
synthetic_probe_duration_seconds{target="example-http",type="http"} 0.081
synthetic_probe_http_status_code{target="example-http"} 200
synthetic_probe_ssl_cert_expiry_days{target="example-tls"} 61
```

### Metrics

| Metric | Labels | Meaning |
| --- | --- | --- |
| `synthetic_probe_up` | `target`, `type` | 1 if the last check passed |
| `synthetic_probe_duration_seconds` | `target`, `type` | Last probe latency |
| `synthetic_probe_http_status_code` | `target` | Observed HTTP status |
| `synthetic_probe_dns_records` | `target` | Addresses returned by DNS |
| `synthetic_probe_ssl_cert_expiry_days` | `target` | Days until cert expiry |
| `synthetic_probe_ssl_cert_expiry_timestamp_seconds` | `target` | Cert expiry as a Unix time |

Example Prometheus scrape config and alert rules (down / slow / cert expiring)
live in [`examples/prometheus`](examples/prometheus).

## Container

There is no published container image. Build it yourself from the
`Dockerfile` (Go 1.27 builder, `distroless/static:nonroot` runtime pinned by
digest):

```bash
docker build -t synthetic-probe .
docker run -p 9115:9115 -v $PWD/examples/targets.yaml:/etc/synthetic-probe/targets.yaml synthetic-probe
```

## Development

```bash
go test ./...
go vet ./...
```

## License

MIT — see [LICENSE](LICENSE).

---

**Archived on 2026-11-05.** blackbox_exporter already does HTTP, DNS and TLS probes with cert expiry, regex assertions and much more. A self-scheduled target list with hot reload is a thin angle. Keep only if DNS divergence across resolvers (#3) becomes its identity.
