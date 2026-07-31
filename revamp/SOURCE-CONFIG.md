# Source Definition Spec — `sources/*.yaml`

> How a contributor adds a new subtitle source. **No Go code, no recompile.** Drop a `.yaml` file in `sources/`, restart, done.

## Why YAML (not code / not plugins)

Researched against how similar projects handle source definitions:

| Project | Approach | Scale |
|---|---|---|
| **Prowlarr / Cardigann** | YAML definitions | 700+ community indexers |
| **Jackett** | YAML-like JSON cards + C# fallback | 500+ indexers |
| **yt-dlp** | Python classes | 1800+ extractors |
| Go `plugin` pkg | compiled `.so` | rejected — Linux-only, same-toolchain, fragile |

The Cardigann pattern wins for our case because **all 4 current sources vary only in**: base URL, search-result CSS selectors, download-button selector/attribute, and download HTTP method. That's 100% expressible in pure YAML. The schema reserves a `script:` field for future escape-hatch needs (CSRF tokens, redirect chains) but we embed **no scripting engine** until a source actually needs it — YAGNI.

## Schema

```yaml
# sources/<name>.yaml
id: baiscopelk                 # stable identifier used in API/source param (lowercase, no spaces)
name: BaiscopeLK                # display name
baseURL: https://www.baiscope.lk
enabled: true                   # set false to disable without deleting file

search:
  path: /                       # appended to baseURL; query param added here
  method: GET
  queryParam: s                 # baseURL + path + ?<queryParam>=<urlencoded query>
  # Optional: extra static params (e.g. post_type=subtitle)
  params: {}
  results:
    selector: "article.elementor-post"      # each search-result row (goquery selector)
    title:
      selector: "h5.elementor-post__title a"  # within row
    link:
      selector: "h5.elementor-post__title a"  # within row
      attribute: href                          # omit → uses element text
  # Optional: drop results whose title/url contain any of these (case-sensitive substring)
  ignorePatterns: ["Collection"]

download:
  # Two-step: fetch the post page, find the download button, then fetch the file.
  postPage:
    button:
      selector: "a[data-e-disable-page-transition=true]"
      attribute: href          # what to extract from the button element
  method: GET                   # GET or POST for the final file fetch
  headers:                      # optional, sent on the final file fetch
    User-Agent: "Mozilla/5.0 (...)
  # Filename extraction precedence (used automatically, not in config):
  #   1. X-Dlm-File-Name header
  #   2. Content-Disposition: filename=
  #   3. URL path basename
  #   4. fallback "<id>_subtitle_<unix>.zip"

  # Forward-compat escape hatch (NOT embedded yet — reserved field):
  # script: |
  #   # future Lua/Expr snippet for CSRF/redirect flows
```

### Field reference

| Field | Required | Description |
|---|---|---|
| `id` | yes | lowercase identifier, used in `?source=` and `/sources` response |
| `name` | yes | human-readable display name |
| `baseURL` | yes | scheme + host, no trailing slash |
| `enabled` | no (default true) | skip this source entirely |
| `search.path` | yes | path appended to baseURL (usually `/`) |
| `search.method` | yes | GET (POST not needed by current sites) |
| `search.queryParam` | yes | query string param name (all 4 sites use `s`) |
| `search.params` | no | extra static query params |
| `search.results.selector` | yes | goquery selector for each result row |
| `search.results.title.selector` | yes | selector within row for title text |
| `search.results.link.selector` | yes | selector within row for the post URL |
| `search.results.link.attribute` | no | attr holding URL (default: `href`); omit to use text |
| `search.ignorePatterns` | no | drop results whose title/url contains any string |
| `download.postPage.button.selector` | yes | selector for download button on post page |
| `download.postPage.button.attribute` | no | attr holding the file URL (default: `href`) |
| `download.method` | yes | GET or POST for the final file fetch |
| `download.headers` | no | extra headers on the final fetch |

## The 4 sources (knowledge preserved from current code)

Selectors verified against current `internal/sources/*` Go implementations.

### `sources/baiscopelk.yaml`
```yaml
id: baiscopelk
name: BaiscopeLK
baseURL: https://www.baiscope.lk
enabled: true
search:
  path: /
  method: GET
  queryParam: s
  results:
    selector: "article.elementor-post"
    title:
      selector: "h5.elementor-post__title a"
    link:
      selector: "h5.elementor-post__title a"
      attribute: href
  ignorePatterns: ["Collection"]
download:
  postPage:
    button:
      selector: "a[data-e-disable-page-transition=true]"
      attribute: href
  method: POST
  headers:
    Content-Type: "application/x-www-form-urlencoded"
```
> Note: baiscopelk uses a **POST** with empty body for the final download, and exposes filename via `X-Dlm-File-Name` header. The scraper honors both automatically.

### `sources/cineru.yaml`
```yaml
id: cineru
name: CineRu
baseURL: https://cineru.lk
enabled: true
search:
  path: /
  method: GET
  queryParam: s
  results:
    selector: ".item-list .post-box-title a"
    title:
      selector: ""        # the row element itself holds the title text
    link:
      selector: ""
      attribute: href
  ignorePatterns: ["Collection", "tv_series"]
download:
  postPage:
    button:
      selector: "#btn-download"
      attribute: data-link   # cineru uses a data attribute, not href
  method: GET
```
> Note: cineru stores the download URL in a `data-link` attribute on `#btn-download` — not `href`. This is exactly the kind of variation the config exists to express.

### `sources/piratelk.yaml`
```yaml
id: piratelk
name: PirateLK
baseURL: https://piratelk.com
enabled: true
search:
  path: /
  method: GET
  queryParam: s
  results:
    selector: ".item-list .post-box-title a"
    title:
      selector: ""
    link:
      selector: ""
      attribute: href
  ignorePatterns: ["Collection"]
download:
  postPage:
    button:
      selector: ".download-button"
      attribute: href
  method: GET
```

### `sources/zoomlk.yaml`
```yaml
id: zoomlk
name: ZoomLK
baseURL: https://zoom.lk
enabled: true
search:
  path: /
  method: GET
  queryParam: s
  results:
    selector: ".td-ss-main-content .item-details .entry-title a"
    title:
      selector: ""
    link:
      selector: ""
      attribute: href
  ignorePatterns: ["Collection"]
download:
  postPage:
    button:
      selector: ".download-button"
      attribute: href
  method: GET
```

## Empty-selector convention

For cineru/piratelk/zoomlk, the search-result selector already targets the `<a>` element directly (`.item-list .post-box-title a`), so the title/link `selector` is `""` — meaning "use the row element itself". The scraper interprets empty selector as the current row node. This avoids forcing every source to nest a child selector.

## Validation at startup

`source.Config.Validate()` enforces at load time (fail fast, app won't start on bad config):
- `id`, `name`, `baseURL` non-empty; `id` matches `^[a-z0-9-]+$`
- `baseURL` parses, scheme is http/https
- `search.results.selector` non-empty
- at least one of `title.selector` or row-text present
- `download.postPage.button.selector` non-empty
- `download.method` ∈ {GET, POST}
- duplicate `id` across files → error

## Contributor workflow

1. Copy `sources/baiscopelk.yaml` → `sources/mynewsource.yaml`
2. Edit `id`, `name`, `baseURL`, selectors, download method
3. Restart the app (or trigger reload)
4. New source appears in `GET /sources` and is searchable

No `go build`. No PR to Go code. That's the whole point.
