# Adding a Source — Contributor Guide

Adding a new subtitle source to BettercopeLK requires **no Go code and no recompile**.
Drop a YAML file in `sources/`, restart the app, done.

## Prerequisites

- You know the subtitle site's search URL and HTML structure.
- You have basic familiarity with YAML and CSS selectors.

## Quick start

Copy an existing source definition and adjust:

```bash
cp sources/baiscopelk.yaml sources/mynewsite.yaml
```

Edit the fields documented below, then restart the server. The new source
appears in `GET /api/v1/sources` and is searchable immediately.

## Schema reference

```yaml
# sources/<id>.yaml
id: mysite               # stable API identifier (lowercase, no spaces)
name: My Site             # display name
baseURL: https://example.com
enabled: true             # set false to disable without deleting the file

search:
  path: /                 # URL path for search, usually /
  method: GET
  queryParam: s           # query parameter name (WordPress uses "s")
  results:
    selector: "div.post-item"       # CSS selector for each result row
    title:
      selector: "h3 a"              # selector for the title within a row
    link:
      selector: "h3 a"              # selector for the post URL
      attribute: href               # attribute containing the URL
  ignorePatterns:                    # optional: drop results matching these
    - "Collection"

download:
  postPage:
    button:
      selector: "a.download-link"   # selector for the download button
      attribute: href               # attribute with the file URL
  method: GET                        # GET or POST for the final file fetch
  headers:                           # optional extra headers
    X-Requested-With: XMLHttpRequest
```

## Field reference

| Field | Required | Description |
|---|---|---|
| `id` | yes | lowercase identifier, used in `?source=` |
| `name` | yes | human-readable name |
| `baseURL` | yes | scheme + host, no trailing slash |
| `enabled` | no | default `true`; set `false` to disable |
| `search.path` | yes | path for search (usually `/`) |
| `search.method` | yes | `GET` |
| `search.queryParam` | yes | query param name (WordPress uses `s`) |
| `search.results.selector` | yes | CSS selector for each result row |
| `search.results.title.selector` | yes | selector for title within a row; empty = use row element |
| `search.results.link.selector` | yes | selector for link within a row; empty = use row element |
| `search.results.link.attribute` | no | attribute for the URL (default `href`) |
| `search.ignorePatterns` | no | drop results whose title/url contain any substring |
| `download.postPage.button.selector` | yes | CSS selector for the download button |
| `download.postPage.button.attribute` | no | attribute for download URL (default `href`) |
| `download.method` | yes | `GET` or `POST` |

## Filename handling

The scraper tries these strategies in order:

1. A custom response header (e.g. `X-Dlm-File-Name`).
2. The `Content-Disposition` header.
3. The basename of the download URL.
4. A fallback like `<id>_subtitle_<timestamp>.zip`.

No configuration is needed for filename extraction.

## Worked example

Adding a new WordPress subtitle site `https://subtitles.example.com`:

```yaml
id: examplecom
name: Example Subtitles
baseURL: https://subtitles.example.com
enabled: true

search:
  path: /
  method: GET
  queryParam: s
  results:
    selector: "article.post"
    title:
      selector: "h2 a"
    link:
      selector: "h2 a"
      attribute: href
  ignorePatterns:
    - "Collection"

download:
  postPage:
    button:
      selector: "a.dl-btn"
      attribute: href
  method: GET
```

Save as `sources/example.yaml`, restart the server. Done.

## Troubleshooting

- **404 or empty results**: check the CSS selectors match the site's HTML.
- **Download fails**: check the download button selector and HTTP method.
- **Wrong filenames**: check whether the site uses a custom response header.
- **Source not appearing**: check the YAML is valid and `enabled: true`.
- **App won't start**: check the logs for validation errors with file and line.

For help, consult the full schema in `revamp/SOURCE-CONFIG.md` or open an issue.
