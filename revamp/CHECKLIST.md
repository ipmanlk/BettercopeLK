# BettercopeLK v2 — Revamp Checklist

> Ground-up rebuild complete. The current `internal/` + `cmd/` replaces the old
> structure entirely. New directory layout documented below.
>
| Doc | What's in it |
|---|---|
| [ARCHITECTURE.md](ARCHITECTURE.md) | Layering, dir structure, tech stack, key decisions |
| [SOURCE-CONFIG.md](SOURCE-CONFIG.md) | YAML schema + all 4 source definitions (knowledge preserved) |
| [API.md](API.md) | Endpoints, SSE events, bulk zip format, rate limits |
| [SECURITY.md](SECURITY.md) | SSRF, validation, zip-bomb guards, headers, logging, health |
| [TESTING.md](TESTING.md) | What to test (and what not to), fixtures, conventions |

Conventions: Go-standard comments only. Test behavior, not coverage.

---

## Phase 0 — Scaffolding

- [x] Create new module layout: `cmd/bettercopelk/`, `internal/{config,models,source,httpclient,subtitle,archive,security,httpapi,middleware,web}`
- [x] `go.mod` (module `ipmanlk/bettercopelk`), Go 1.25+
- [x] Add deps: `github.com/PuerkitoBio/goquery`, `gopkg.in/yaml.v3`, `github.com/hashicorp/go-retryablehttp`, `golang.org/x/net`, `golang.org/x/sync`
- [x] `Makefile` (build / run / test / test-race / fmt / vet)
- [x] `.gitignore` (binaries, `tmp/`, `*.log`)

## Phase 1 — Foundation packages

- [x] `internal/models/models.go` — `SearchResult`, `BulkItem`, SSE event types, error envelope, manifest
- [x] `internal/config/config.go` — env-driven config with defaults
- [x] `internal/httpclient/client.go` — shared `*http.Client` with retry, UA, timeouts
- [x] `internal/config/config_test.go` — config loading
- [x] `internal/httpclient/client_test.go` — client construction

## Phase 2 — Pluggable source engine (the core win)

- [x] `internal/source/source.go` — `Source` interface: `ID()`, `Name()`, `BaseURL()`, `Enabled()`, `Available()`, `Search()`, `Download()`
- [x] `internal/source/config.go` — YAML schema structs + `Validate()` (table-driven tested)
- [x] `internal/source/registry.go` — lookup by id, enabled filter, cached availability
- [x] `internal/source/loader.go` — glob `sources/*.yaml` → Registry; fail-fast on bad config
- [x] `internal/source/scraper.go` — generic engine via goquery: Search parses rows, Download fetches post page → finds button → fetches file (GET/POST per config); filename extraction
- [x] `sources/{baiscopelk,cineru,piratelk,zoomlk}.yaml` (from SOURCE-CONFIG.md)
- [x] `internal/source/config_test.go`, `scraper_test.go` — validation table, loader, httptest scraper tests

## Phase 3 — Archive & HTTP infrastructure

- [x] `internal/httpclient/client.go` — timeouts, conn pool, browser UA, retry w/ backoff (retryablehttp), context propagation
- [x] `internal/archive/zipstream.go` — streaming `zip.Writer` → `io.Writer`; add entries from `io.Reader`; write `manifest.json`
- [x] `internal/archive/extract.go` — read source zip, re-add entries to output, **bomb guards** (entry count ≤100, per-entry ≤50MB, total ≤500MB, ratio ≤100, depth-1 flat), name dedupe
- [x] `internal/archive/*_test.go` — round-trip, all 4 bomb guards, dedupe

## Phase 4 — Domain service

- [x] `internal/subtitle/service.go` — `Search` (errgroup fan-out, one failure ≠ all fail), `StreamSearch` (SSE channels), `Download`, `DownloadBulk` (parallel w/ semaphore, streaming zip + manifest, per-item errors recorded not fatal)
- [x] `internal/subtitle/service_test.go` with fake `Source` (errors, timeouts, ctx cancel, bulk partial-failure)

## Phase 5 — Security

- [x] `internal/security/urlguard.go` — host allowlist (from source baseURLs) + private-IP/loopback/link-local/metadata-IP rejection; injectable resolver for DNS-rebinding test
- [x] `internal/security/urlguard_test.go` — allowlist hits, all SSRF vectors, DNS-rebinding fake

## Phase 6 — HTTP layer

- [x] `internal/httpapi/sse.go` — typed SSE writer with Flusher
- [x] `internal/httpapi/handler_search.go` — `GET /search` (batch) + `GET /search/stream` (SSE)
- [x] `internal/httpapi/handler_download.go` — `GET /download` + `POST /download/bulk` (streamed zip)
- [x] `internal/httpapi/handler_sources.go` — `GET /sources` (cached availability)
- [x] `internal/httpapi/handler_health.go` — `/healthz`, `/readyz`
- [x] `internal/httpapi/server.go` — ServeMux wiring, `http.Server` timeouts, graceful shutdown
- [x] `internal/httpapi/helpers.go` — `writeJSON`, `writeError`
- [x] `internal/httpapi/httpapi_test.go` — statuses, content types, SSE ordering, item cap, body limit (fake service)

## Phase 7 — Middleware

- [x] `internal/middleware/requestid.go` — generate/propagate `X-Request-ID`
- [x] `internal/middleware/ratelimit.go` — per-IP `x/time/rate`, per-route limits, idle eviction, `Retry-After`
- [x] `internal/middleware/logging.go` — slog access log w/ request id
- [x] `internal/middleware/security.go` — security headers, CORS `*`
- [x] `internal/middleware/recovery.go` — panic → 500
- [x] `internal/middleware/bodylimit.go` — `MaxBytesReader`
- [x] `internal/middleware/middleware_test.go` — rate limit `429`, recovery, request-id propagation, body limit

## Phase 8 — Wiring, frontend & deploy (this phase)

- [x] `cmd/bettercopelk/main.go` — load config → build client → load sources → build guard → service → server; graceful shutdown
- [x] `internal/web/assets/index.html` — full UI: search bar, source dropdown, result list with checkboxes, bulk download, console log
- [x] `internal/web/assets/script.js` — vanilla JS: source loading, SSE search, single + bulk download, log panel
- [x] `internal/web/assets/style.css` — dark theme, accessible, responsive
- [x] `sources/{baiscopelk,cineru,piratelk,zoomlk}.yaml` — 4 source definitions
- [x] `deployments/Dockerfile` — multi-stage, alpine runtime
- [x] `deployments/compose.yml` — app service with port, restart, limits, healthcheck
- [x] `.github/workflows/deploy.yml` — SSH deploy workflow (preserved)
- [x] `docs/source-definition.md` — contributor guide
- [x] `README.md` — rewrite for v2

## Phase 9 — Verify

- [x] `go vet ./...` clean
- [x] `go test -race ./...` green
- [x] `go build ./cmd/bettercopelk` succeeds
- [x] Manual smoke: binary starts, /healthz and /readyz return 200, /api/v1/sources lists 4 sources, /api/v1/search returns real results
- [x] Confirm adding a source = drop YAML + restart (no recompile) — implemented via `sources/*.yaml` + registry loader

---

## Status

- [x] Plan complete (this doc + 5 linked specs)
- [x] All phases implemented
- [x] Manual smoke test completed
