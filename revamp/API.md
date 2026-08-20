# API Spec — BettercopeLK v2

> Base URL: `/api/v1`. All endpoints JSON unless noted. See `SECURITY.md` for limits/validation.

## Design notes

- Search streams via **SSE** so fast sources show before slow ones finish. A non-streaming JSON batch endpoint also exists as fallback.
- **Full URLs never leave the server.** Search results carry a `path` field — the URL-safe base64-encoded relative path of the post page (see `internal/token`). The server reconstructs the full URL at download time using the source's `baseURL`. This keeps payloads small and the origin host server-side.
- Single download is `GET` (works as a direct `<a href>` link, browser-handled).
- Bulk download is `POST` with a JSON body — supports item cap, body-limit middleware, and avoids oversized query strings. Returns a **single streamed zip**.
- Bulk zip contains a `manifest.json` recording per-item success/error, plus the fetched subtitle files (unzipped from any source zips, names deduped).

## `GET /api/v1/sources`

List enabled sources. Availability is cached with TTL (`SOURCE_AVAILABILITY_TTL`, default 60s) — does not HEAD every source per request.

Response `200`:
```json
{ "sources": [ { "id": "baiscopelk", "name": "BaiscopeLK", "available": true } ] }
```

## `GET /api/v1/search?query=...&sources=baiscopelk,cineru`

Batch search across selected (or all) sources. Blocks until all finish. Prefer `/search/stream`.

Params:
- `query` (required, 1–128 chars)
- `sources` (optional, comma-separated `id`s; omitted = all enabled)

Response `200`:
```json
{ "results": [ { "title": "...", "path": "L2luY2VwdGlvbi1zaW5oYWxh...", "source": "baiscopelk" } ] }
```
Errors: `400` missing/overlong query or unknown source; `429` rate limited; `504` if all sources time out.

## `GET /api/v1/search/stream?query=...&sources=...` (SSE)

Streams results as each source resolves. Event types:

- `result` — one subtitle hit
- `source-complete` — a source finished (with count and optional error)
- `done` — terminal, sent once

```
event: result
data: {"title":"Inception (2010)","path":"L2luY2VwdGlvbi1zaW5oYWxh...","source":"baiscopelk"}

event: source-complete
data: {"source":"baiscopelk","count":3}

event: done
data: {"total":3}
```

Client disconnect cancels in-flight upstream fetches (context propagation).

## `GET /api/v1/download?path=...&source=...`

Single subtitle download. `path` is the base64url-encoded relative path from a search result; the server reconstructs the full URL from the source's `baseURL` and validates it through the SSRF guard.

Response: the raw file (`application/zip` or `text/plain` for `.srt`) with `Content-Disposition: attachment`.

Errors: `400` missing params / invalid token / unknown source; `404` source unavailable; `502` upstream fetch failed; `429` rate limited.

## `POST /api/v1/download/bulk`

Fetch multiple subtitles in parallel, unzip any nested zips, dedupe filenames, stream a single zip.

Request body (JSON):
```json
{
  "items": [
    { "path": "L2luY2VwdGlvbi1zaW5oYWxh...", "source": "baiscopelk" },
    { "path": "L2luY2VwdGlvbi1zaW5oYWxh...", "source": "cineru" }
  ]
}
```

Constraints:
- `items` length 1–`MAX_BULK_ITEMS` (default 20)
- each `path` must be a valid base64url token that decodes to a safe relative path
- per-file size ≤ `MAX_FILE_BYTES` (default 50MB); total ≤ `MAX_BULK_TOTAL_BYTES` (default 500MB)

Response `200` — `application/zip` streamed directly to the response (no buffering). Contains:

```
manifest.json
Inception.zip/            # if source zip; contents flattened in
  Inception.srt
Inception (cineru).srt   # name collisions suffixed
```

`manifest.json`:
```json
{
  "items": [
    { "path": "L2luY2VwdGlvbi1zaW5oYWxh...", "source": "baiscopelk", "ok": true, "files": ["Inception.srt"], "bytes": 42000 },
    { "path": "L2luY2VwdGlvbi1zaW5oYWxh...", "source": "cineru", "ok": false, "error": "upstream 502" }
  ]
}
```

Errors: `400` invalid body / too many items / invalid token; `413` body too large; `429` rate limited.

> A partially-failed bulk still streams a 200 zip with the manifest recording which items failed — better UX than failing the whole request because one source is down.

## `GET /healthz` & `GET /readyz`

- `/healthz` — process liveness; always `200 { "status":"ok" }` unless shutting down.
- `/readyz` — at least one source available, else `503`. Used for container readiness probes.

## Error envelope

All `4xx`/`5xx` JSON:
```json
{ "error": "query parameter is required", "request_id": "abc123" }
```

## Rate limits (per IP)

| Route | Limit | Burst |
|---|---|---|
| `/search`, `/search/stream` | 60/min | 10 |
| `/download` | 30/min | 5 |
| `/download/bulk` | 10/min | 2 |
| global (all routes) | 120/min | 20 |

`429 Too Many Requests` with `Retry-After` header.
