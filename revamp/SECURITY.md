# Security & Limits — BettercopeLK v2

Grounded in current Go best practice (Go 1.24+). Concrete values are starting points, all env-configurable.

## 1. SSRF protection (critical — the download endpoints fetch a URL upstream)

The download and bulk endpoints used to accept a user-supplied `url` that the server fetches upstream. They now accept only an opaque `path` token: the URL-safe base64 encoding of the post page's relative path (see `internal/token`). The server reconstructs the full URL from the source's `baseURL` and validates it through the guard before fetching.

**Defense layers:**

1. **Token validation** — the `path` token must decode to a relative path that starts with `/`, is not protocol-relative (`//host`), contains no scheme injection (`://`), no path-traversal segments (`..`), no backslashes or control characters, and is length-bounded (4 KB). A malformed token is rejected with `400`.
2. **Host allowlist** — the reconstructed URL's host always comes from a registered source's `baseURL` host, so a client can only ever ask us to fetch from the subtitle sites we already scrape.
3. **Private-IP rejection** — still enforced on the reconstructed URL, guarding against DNS rebinding (see below).

`internal/security/urlguard.go`:

```go
// Package security validates that a URL the server is about to fetch is safe.
// A URL is allowed only if its host matches a registered source's baseURL host,
// its scheme is http/https, and the resolved IP is not private/loopback/link-local.
package security

// Guard holds the set of allowed hosts (derived from enabled sources' baseURLs).
type Guard struct {
	allowedHosts map[string]struct{}
}

// NewGuard builds a Guard from the given host allowlist.
func NewGuard(allowedHosts []string) *Guard

// CheckURL validates a single URL. It rejects non-http(s) schemes, hosts not in
// the allowlist, and hosts that resolve to private/loopback/link-local IPs.
// The IP check guards against DNS-rebinding: we resolve the host and inspect
// the returned addresses, not just the hostname.
func (g *Guard) CheckURL(rawURL string) error
```

Checks:
1. Parse URL; scheme must be `http` or `https`.
2. Host (lowercased, port-stripped) must be in the allowlist (built from `sources/*.yaml` `baseURL` hosts at load time).
3. Resolve host via `net.DefaultResolver.LookupIPAddr`; reject if any returned IP is loopback / private / link-local-unicast / link-local-multicast / unspecified / multicast, or equals `169.254.169.254`. (`net.IP` has `IsLoopback`, `IsPrivate`, `IsLinkLocalUnicast`, `IsLinkLocalMulticast`, `IsUnspecified`, `IsMulticast`; explicitly block the cloud metadata IP too — it is link-local but be defensive.)

Because the allowlist is tied to source hosts, a user can only ever ask us to fetch from the subtitle sites we already scrape — the SSRF surface collapses to "trusted external sites".

## 2. Input validation

- `query`: trim, 1–128 runes (`utf8.RuneCountInString`). Reject empty / overlong.
- `source` param: must be in the registry; else `400`.
- `sources` list: each validated; unknown → `400`.
- bulk `items`: 1–`MAX_BULK_ITEMS` (default 20); dedupe identical `(url,source)` pairs.
- all request bodies: `http.MaxBytesReader` at `MAX_BODY_BYTES` (default 64KB — bulk payloads are tiny).

## 3. Response & file size limits (zip-bomb defense)

When reading an upstream file (the subtitle zip from a source), wrap the body reader:

```go
// limitedReader caps the bytes read from an upstream response. If the source
// claims a size larger than max, reject before reading.
body := io.LimitReader(resp.Body, maxFileBytes)
```

Guards on unzipping a source's zip before re-adding to the bulk archive (`internal/archive/extract.go`):

| Guard | Limit | Why |
|---|---|---|
| Max entries per source zip | 100 | zip bombs use thousands of tiny files |
| Max uncompressed bytes per entry | 50MB (`MAX_FILE_BYTES`) | single huge decompressed file |
| Max total uncompressed bytes per bulk | 500MB (`MAX_BULK_TOTAL_BYTES`) | overall cap |
| Compression ratio | reject if `uncompressed/compressed > 100` | classic bomb signature |
| Nested-zip depth | flatten depth 1 only; reject depth ≥2 | prevents recursive bombs |

Each `zip.File` has `UncompressedSize64`/`CompressedSize64` available **before** opening — check them in the loop, reject the archive if exceeded.

## 4. Concurrency & timeouts

- Bulk fetch uses `errgroup` with a semaphore (max 5 concurrent upstream fetches). A single slow source can't starve the others.
- Per-source HTTP timeout 30s (`HTTP_TIMEOUT`); per-request context cancelled on client disconnect.
- `http.Server` timeouts: `ReadHeaderTimeout=5s`, `ReadTimeout=10s`, `WriteTimeout=300s` (bulk zips can be slow), `IdleTimeout=60s`.

## 5. Rate limiting (per-IP token bucket)

`golang.org/x/time/rate`. A `*rate.Limiter` per client IP, keyed map with idle eviction (sweep every minute, drop IPs idle >5min). Routes have separate limits (see `API.md`). Behind a proxy, read `X-Forwarded-For` / `X-Real-IP` only if `TRUSTED_PROXIES` is configured.

## 6. Security headers & CORS

Headers middleware on all responses:
- `X-Content-Type-Options: nosniff`
- `Referrer-Policy: no-referrer`
- `X-Frame-Options: DENY` (we're not framed)
- `Content-Security-Policy: default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'` (static assets only)

CORS: current code uses `Access-Control-Allow-Origin: *`. **Keep `*`** — this is a public read-only subtitle API, no cookies/credentials are sent, so `*` is appropriate and maximizes usefulness. Add `Access-Control-Allow-Methods: GET, POST, OPTIONS`.

## 7. Logging

`log/slog` with JSON handler in prod, text in dev. Each request gets a request ID (middleware) flowing into logs and the `X-Request-ID` response header. Access log: method, path, status, duration, source(s) queried, bytes. Never log full URLs? URLs are fine to log here (they're public subtitle pages) — but don't log request bodies. Rate-limit hits are logged at `WARN`.

## 8. Health & readiness

- `/healthz` — process alive; `200` unless shutting down. No upstream calls.
- `/readyz` — `200` if ≥1 source available (cached availability), else `503`. Lets the load balancer pull the container out of rotation if all subtitle sites are down.
