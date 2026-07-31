# BettercopeLK

Search and download Sinhala subtitles from multiple sources — one API, one UI.

## Features

- **Search** across multiple subtitle sites simultaneously, streamed via SSE
- **Single download** — get one subtitle file with one click
- **Bulk download** — select any number of results, download them all as a single zip
- **No recompile to add sources** — sources are YAML files in `sources/`. Drop one, restart, done
- **SSRF-safe** — the server will only fetch from registered source hosts
- **Rate-limited** — per-IP token bucket on all endpoints
- **Embedded frontend** — single binary, no build step for the UI

## Currently supported sources

- [Baiscope.lk](https://baiscopelk.com/)
- [Cineru.lk](https://cineru.lk/)
- [Piratelk.com](https://piratelk.com/)
- [Zoom.lk](https://zoom.lk/)

[Adding a new source](docs/source-definition.md) requires no Go code — just a YAML file.

## Quick start

```bash
# Build
go build -o bin/bettercopelk ./cmd/bettercopelk

# Run
./bin/bettercopelk
# Server starts on :8080 — open http://localhost:8080
```

## Development

```bash
# Run with hot reload (requires air)
make dev

# Run tests
make test

# Run tests with race detector
make test-race
```

The full `Makefile`:
- `build` — compile the binary
- `run` — `go run ./cmd/bettercopelk`
- `dev` — `air` (hot reload)
- `test` / `test-race` — run tests
- `fmt` / `vet` — format and lint
- `clean` — remove build artifacts

## Docker

```bash
# Build
docker compose -f deployments/compose.yml build

# Run
docker compose -f deployments/compose.yml up -d
```

## Configuration

All configuration is through environment variables:

| Variable | Default | Description |
|---|---|---|
| `PORT` | `8080` | HTTP listen port |
| `SOURCES_DIR` | `./sources` | Directory containing source YAML files |
| `MAX_BULK_ITEMS` | `20` | Max items per bulk download |
| `MAX_FILE_BYTES` | `52428800` (50 MB) | Max upstream file size |
| `MAX_BULK_TOTAL_BYTES` | `524288000` (500 MB) | Max total bulk zip size |
| `MAX_BODY_BYTES` | `65536` (64 KB) | Max request body size |
| `RATE_SEARCH_RPM` | `60` | Per-IP search rate limit |
| `RATE_DOWNLOAD_RPM` | `30` | Per-IP download rate limit |
| `RATE_BULK_RPM` | `10` | Per-IP bulk rate limit |
| `RATE_GLOBAL_RPM` | `120` | Global per-IP rate limit |
| `HTTP_TIMEOUT` | `30s` | Upstream request timeout |
| `READ_HEADER_TIMEOUT` | `5s` | Server read-header timeout |
| `READ_TIMEOUT` | `10s` | Server read timeout |
| `WRITE_TIMEOUT` | `300s` | Server write timeout (bulk zips) |
| `SOURCE_AVAILABILITY_TTL` | `60s` | Source health check cache TTL |
| `TRUSTED_PROXIES` | — | Comma-separated CIDRs for `X-Forwarded-For` |
| `LOG_FORMAT` | `text` | Set to `json` for structured JSON logs |

## API

**Base URL:** `/api/v1`

| Method | Path | Description |
|---|---|---|
| `GET` | `/api/v1/sources` | List available sources |
| `GET` | `/api/v1/search?query=...&sources=...` | Batch search (JSON) |
| `GET` | `/api/v1/search/stream?query=...&sources=...` | Streamed search (SSE) |
| `GET` | `/api/v1/download?url=...&source=...` | Single subtitle download |
| `POST` | `/api/v1/download/bulk` | Bulk download (streamed zip) |
| `GET` | `/healthz` | Liveness check |
| `GET` | `/readyz` | Readiness check |

See `revamp/API.md` for full details.

## Project structure

```
cmd/bettercopelk/      # Entrypoint
internal/
  config/              # Environment-based configuration
  models/              # Domain types
  source/              # YAML-driven pluggable source engine
  httpclient/          # Shared retrying HTTP client
  subtitle/            # Service orchestration (search, download, bulk)
  archive/             # Zip streaming and bomb protection
  security/            # SSRF guard
  httpapi/             # HTTP handlers, SSE, server lifecycle
  middleware/          # Rate limiting, logging, recovery, security headers
  web/                 # Embedded frontend assets
sources/               # Source YAML definitions — add new sources here
deployments/           # Dockerfile, compose.yml
docs/                  # Contributor documentation
```

## Security

- **SSRF protection**: every user-supplied URL is validated against a host
  allowlist derived from source configs. Resolved IPs are checked for
  private/loopback/link-local addresses.
- **Zip bomb protection**: entry count, per-entry size, total uncompressed
  size, and compression ratio are all capped.
- **Rate limiting**: per-IP token bucket with separate limits per endpoint.

## Building from source

Requires Go 1.24+. The current `go.mod` specifies Go 1.25, which is backward
compatible with 1.24.

```bash
git clone https://github.com/ipmanlk/BettercopeLK
cd BettercopeLK
go build -o bin/bettercopelk ./cmd/bettercopelk
```

## Disclaimer

This project is an experiment. It offers Sinhala subtitle search and download
features, but functionality and availability may change without notice. Not
intended for production use without review.
