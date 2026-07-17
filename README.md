# synthetic-probe

> Blackbox-style checks without the operational weight — one binary.

**Status:** 🚧 In development

## Overview

Lightweight periodic HTTP/DNS/TLS prober exporting Prometheus metrics (single binary).

## Features

- HTTP(S), DNS, and TLS-expiry checks
- Per-target latency, status, and cert-days-left metrics
- YAML target config with intervals
- Prometheus /metrics endpoint
- Tiny footprint, container-ready

## Stack

Go + prometheus/client_golang.

## Usage

```bash
synthetic-probe --config targets.yaml --listen :9115
```

## License

MIT
