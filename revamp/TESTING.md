# Testing Strategy — BettercopeLK v2

Test what matters; skip what's trivial. No coverage targets — behavior over percentages.

## Principles

1. **Test behavior, not implementation.** Tests assert *what* the code does for given inputs, not *how*. If a refactor breaks tests that don't change behavior, the tests were wrong.
2. **Table-driven tests** for anything with input variants (validation, URL parsing, filename extraction).
3. **Dependency injection via interfaces** so tests substitute fakes — no real network, no real filesystem in unit tests.
4. **`testdata/`** for fixture HTML/zip files. Real captured samples, trimmed to the minimum that exercises the selector.
5. **`httptest.Server`** for integration tests that need a real HTTP round-trip.
6. **Skip obvious stuff.** Don't test getters, struct literals, or stdlib wrappers with no logic. Don't unit-test `net/http`.

## What to test, per package

### `internal/source` — the highest-value tests

The scraper is the heart of the app and the most failure-prone (sites change). This is where most test effort goes.

- `config_test.go` — `Config.Validate()` table-driven: missing `id`, bad scheme, unknown `method`, duplicate `id`, valid case. Assert exact error messages.
- `loader_test.go` — load from `testdata/sources/*.yaml` (valid + malformed). Assert registry contents, error on bad file, `enabled:false` skipped.
- `scraper_test.go` — **fixture-driven**. For each of the 4 sites, store a trimmed real HTML page in `testdata/html/<site>_search.html` and `<site>_post.html`. Assert:
  - `Search` extracts the expected title + URL count from the fixture.
  - `Download` finds the right button URL from the post fixture.
  - Empty selector convention (`selector: ""` → use row node) works.
  - `ignorePatterns` filters the right rows.
  - Use `httptest.Server` to serve the fixture and point `baseURL` at it — exercises the full client→parse path without real internet.

> Fixtures are committed. If a site changes, update the fixture + selectors together — the failing test tells you exactly what broke.

### `internal/archive` — zip streaming & bomb guards

- `zipstream_test.go` — write 3 files via the streaming writer into a `bytes.Buffer`, `zip.NewReader` it back, assert names + contents. Confirm `manifest.json` is present.
- `extract_test.go` — **bomb guards are the critical tests here**:
  - A source zip with an entry claiming `UncompressedSize64 > MAX_FILE_BYTES` → rejected before decompression.
  - A zip with >100 entries → rejected.
  - A zip with compression ratio >100 → rejected.
  - A normal source zip → entries flattened into the output, names deduped on collision (`name.srt`, `name (1).srt`).
  - Use `archive/zip` to construct fixtures programmatically (no binary blobs committed).
- Integration: fetch-from-fake-server → unzip → re-zip → read back, assert round-trip.

### `internal/security` — SSRF guard

Table-driven `CheckURL`:
- allowed: `https://www.baiscope.lk/inception`
- rejected: `http://169.254.169.254/`, `http://localhost:8080/`, `http://10.0.0.1/`, `ftp://...`, `http://evil.com/` (host not in allowlist).
- DNS-rebinding: use a fake resolver returning a private IP for an allowed-looking host → rejected. (Inject the resolver via an interface so the test doesn't touch real DNS.)

### `internal/subtitle` — service orchestration

- `service_test.go` with a **fake `Source`** (`type fakeSource struct{}` implementing the interface with configurable results/errors/timeouts):
  - `Search` fans out across sources, aggregates results from all.
  - One source erroring doesn't fail the whole search; its results are just absent.
  - Context cancellation stops in-flight sources.
  - `DownloadBulk` fetches all in parallel, collects per-item success/error into the manifest, a failing item doesn't abort the rest.

### `internal/httpapi` — handlers

- Use `httptest.NewRecorder` + a fake service. Assert status codes, content types, `Content-Disposition`, error envelope shape, request ID propagation.
- SSE handler: assert event ordering (`result`...`source-complete`...`end`).
- Bulk handler: assert item cap enforced (`400` at `MAX_BULK_ITEMS+1`), body limit enforced (`413`).
- Don't re-test service logic here — only the HTTP layer.

### `internal/middleware` — rate limit & recovery

- Rate limit: fire N+1 requests at a fake handler with a tight limiter, assert the (N+1)th returns `429` and `Retry-After`.
- Recovery: handler panics → `500`, panic value logged, other requests still served.

### What we do NOT test

- `net/http` ServeMux routing (stdlib).
- `goquery` selector matching (third-party, its own tests).
- `golang.org/x/time/rate` internals.
- Trivial accessors / struct construction.
- The frontend JS (out of scope for the Go plan; could be a later addition).

## Layout

```
internal/source/
  scraper.go
  scraper_test.go
  testdata/
    sources/
      valid.yaml
      missing_id.yaml
    html/
      baiscopelk_search.html
      baiscopelk_post.html
      cineru_search.html
      cineru_post.html
      ...
internal/archive/
  extract_test.go
  testdata/              # built programmatically in TestMain if needed
```

## Running

- `go test ./...` — all unit + integration tests.
- `go test -run Integration ./internal/source` — only the httptest-based scraper tests (tagged via build tag `integration` if they get slow; currently fast enough to always run).
- CI: `go test -race ./...` — race detector on every run (concurrency is central to this app).
- `go vet ./...` + `gofmt -l .` clean in CI.

## Benchmarking (optional, only if needed)

If a source search or zip stream ever needs perf validation, add a `BenchmarkScraperSearch` in the relevant `_test.go`. Not required up front.
