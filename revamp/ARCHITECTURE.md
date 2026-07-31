# Architecture — BettercopeLK v2 (Ground-Up Revamp)

> Design rationale for the full rebuild. See `CHECKLIST.md` for build progress.
> This replaces the current `internal/` + `cmd/` structure entirely.

## Design principles

1. **Sources are data, not code.** A source is a YAML file in `sources/`. Adding one = drop file + restart. **No recompile, no Go knowledge required.** (Cardigann/Prowlarr prove this at 700+ definitions.)
2. **One generic engine, many configs.** A single `Scraper` implements the `Source` interface; each YAML config drives it. No per-site Go packages.
3. **Streaming by default.** Search streams via SSE (fast sources show first). Bulk downloads stream a single zip to the client — bounded memory, no temp files.
4. **Defense in depth.** Every user-supplied URL is SSRF-checked against a host allowlist derived from source configs. Every upstream zip is zip-bomb-guarded. Every endpoint is rate-limited.
5. **Standard library first.** `net/http`, `archive/zip`, `log/slog`, `golang.org/x/time/rate`, `golang.org/x/net/html`. goquery replaces the hand-rolled CSS engine.
6. **Thin handlers, fat services.** HTTP handlers only parse/validate/serialize; orchestration lives in the domain service.

## Layered architecture

```
┌─────────────────────────────────────────────────────────┐
│  cmd/bettercopelk/main.go   (wiring: config → deps → srv)│
├─────────────────────────────────────────────────────────┤
│  internal/httpapi   (handlers, router, SSE, middleware)  │
│        ↓  domain types                                   │
│  internal/subtitle  (service: search fan-out, download,  │
│        bulk zip orchestration)                           │
│        ↓  Source interface                               │
│  internal/source    (YAML loader + registry + Scraper)   │
│        ↓                                                 │
│  internal/httpclient (shared retrying client)            │
│  internal/archive   (zip stream + bomb guards)           │
│  internal/security  (SSRF guard, validation)             │
└─────────────────────────────────────────────────────────┘
        sources/*.yaml  ← the only thing a contributor edits
```

**Dependency direction is strictly downward.** `httpapi` depends on `subtitle`; `subtitle` depends on `source`; `source` depends on `httpclient`. No upward imports. `models` is leaf-shared.

## Target directory structure

```
bettercopelk/
├── cmd/
│   └── bettercopelk/
│       └── main.go                 # entrypoint: load config, wire, run server
├── internal/
│   ├── config/
│   │   └── config.go               # env + defaults; app-wide config struct
│   ├── models/
│   │   └── models.go               # SearchResult, DownloadRequest, BulkItem, events...
│   ├── source/
│   │   ├── source.go               # Source interface
│   │   ├── config.go               # YAML schema structs + Validate()
│   │   ├── loader.go               # glob sources/*.yaml → Registry
│   │   ├── registry.go             # lookup by name, enabled filter, availability cache
│   │   └── scraper.go              # generic engine: Search/Download via goquery + config
│   ├── httpclient/
│   │   └── client.go               # shared *http.Client: timeouts, pool, UA, retry w/ backoff
│   ├── subtitle/
│   │   └── service.go              # Search (fan-out+errgroup), Download, DownloadBulk
│   ├── archive/
│   │   ├── zipstream.go            # streaming zip.Writer → http.ResponseWriter
│   │   └── extract.go              # safely read a source zip, re-add entries, bomb guards
│   ├── security/
│   │   └── urlguard.go             # SSRF: host allowlist + private-IP rejection
│   ├── httpapi/
│   │   ├── server.go               # router + middleware wiring + http.Server
│   │   ├── handler_search.go       # GET /search, GET /search/stream
│   │   ├── handler_download.go     # GET /download, POST /download/bulk
│   │   ├── handler_sources.go      # GET /sources
│   │   ├── handler_health.go       # GET /healthz, /readyz
│   │   └── sse.go                  # typed SSE writer
│   ├── middleware/
│   │   ├── ratelimit.go            # per-IP token bucket, per-route limits
│   │   ├── requestid.go            # X-Request-ID / generate
│   │   ├── logging.go              # slog structured access log
│   │   ├── security.go             # security headers
│   │   ├── recovery.go             # panic → 500 + log
│   │   └── bodylimit.go            # http.MaxBytesReader for POST
│   └── web/
│       ├── embed.go                # //go:embed assets
│       └── assets/
│           ├── index.html
│           ├── script.js
│           └── style.css
├── sources/                        # ← THE pluggable part (no Go to add a source)
│   ├── baiscopelk.yaml
│   ├── cineru.yaml
│   ├── piratelk.yaml
│   └── zoomlk.yaml
├── deployments/
│   ├── Dockerfile                  # multi-stage, distroless/slim
│   ├── compose.yml
│   └── air.toml
├── docs/
│   └── source-definition.md        # contributor guide: how to add a source
├── go.mod
├── go.sum
├── Makefile
├── .github/workflows/deploy.yml
└── README.md
```

## Tech stack & versions

| Concern | Choice | Why |
|---|---|---|
| Language | Go 1.24+ | current; matches existing `go.mod` |
| HTTP routing | `net/http` ServeMux (Go 1.22+ pattern routing) | stdlib, `GET /api/v1/search` patterns — no framework needed |
| HTML parsing | **goquery v1.12.0** | jQuery-like CSS selectors via cascadia; replaces hand-rolled engine. Selectors come from YAML → goquery is the natural fit |
| HTTP client | `net/http` + `hashicorp/go-retryablehttp` | shared client, retry w/ backoff, sane pool defaults |
| YAML | `gopkg.in/yaml.v3` | de-facto standard |
| Rate limiting | `golang.org/x/time/rate` | token bucket, stdlib-adjacent, single-instance ready |
| Logging | `log/slog` | structured, stdlib |
| Concurrency | `golang.org/x/sync/errgroup` + semaphore | fan-out w/ limits + error propagation |
| Zip | `archive/zip` (stdlib) | streams to `io.Writer`; central dir at end |

Rejected: Go `plugin` (Linux-only, same-toolchain, fragile — kills the "easy source" goal); hand-rolled CSS engine (buggy, reinvents goquery); full web framework (overkill for 6 routes).

## Key decisions

| Decision | Choice | Rationale |
|---|---|---|
| Source definition format | YAML files in `sources/` | No recompile; proven by Cardigann/Prowlarr (700+ defs). All 4 current sources are expressible in pure YAML — no scripting needed. |
| Script escape hatch | Schema reserves a `script:` field, **not embedded yet** | Forward-compatible. Add Expr/Lua only if a future source needs it. YAGNI for now. |
| Search transport | SSE streaming (primary) + batch JSON (fallback) | Slow sources don't block fast ones; users see results as they arrive |
| Bulk transport | `POST` with JSON body | Supports item cap, body-limit middleware, avoids giant query strings |
| Bulk delivery | Single streamed zip | Fetch all in parallel, unzip nested zips, dedupe names, stream one archive. Manifest entry records success/errors |
| Availability | Cached health check w/ TTL (e.g. 60s) | Avoid HEAD-ing every source on every `/sources` call |
| Frontend | Vanilla JS (kept), embedded via `//go:embed` | No build step; single binary |

## Configuration (`internal/config`)

App config from env (sane defaults, 12-factor):

```
PORT=8080
SOURCES_DIR=./sources
MAX_BULK_ITEMS=20
MAX_FILE_BYTES=52428800        # 50MB per upstream file
MAX_BULK_TOTAL_BYTES=524288000 # 500MB total per bulk request
RATE_SEARCH_RPM=60            # per IP
RATE_DOWNLOAD_RPM=30          # per IP
RATE_BULK_RPM=10              # per IP
HTTP_TIMEOUT=30s
HTTP_IDLE_TIMEOUT=60s
SOURCE_AVAILABILITY_TTL=60s
```

## What gets thrown away from the current code

- `internal/htmlparser/` (hand-rolled CSS engine) → goquery
- `internal/sources/{baiscopelk,cineru,piratelk,zoomlk}/` (per-site Go packages) → YAML configs + one generic `Scraper`
- `internal/sse/writer.go` → rewritten typed SSE writer in `httpapi/sse.go`
- `cmd/server/main.go` + `cmd/bettercopelk/main.go` → single `cmd/bettercopelk/main.go`
- `internal/static/` → `internal/web/` with proper `http.FileServer` over `embed.FS`

**Preserved (the hard-won knowledge):** the exact CSS selectors, download button attributes, HTTP methods, and filename-extraction quirks for all 4 sites — encoded as YAML in `sources/*.yaml` (see `SOURCE-CONFIG.md`).
