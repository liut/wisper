---
name: webpawm
description: Web search and web fetch tools. Use when you need to search the web or fetch web page content. Supports CLI and REST API modes.
---

# Webpawm - Web Search & Fetch Tools

Provides two capabilities: **web search** (multi-engine) and **web fetch** (HTML→Markdown).

## Engine Availability

| Engine | Always available? | Requires config |
|--------|-------------------|-----------------|
| `bingcn` | Yes | None |
| `arxiv` | Yes | None |
| `google` | No | `google_api_key` + `google_cx` |
| `bing` | No | `bing_api_key` |
| `brave` | No | `brave_api_key` |
| `searchxng` | No | `searchxng_url` |

Use `bingcn` or `arxiv` for instant testing. Other engines require API keys in `~/.config/webpawm/config.json`.

## Expected Latency

| Search depth | Engines | Typical time |
|-------------|---------|--------------|
| `quick` | 1 engine, 1 query | 1–5s |
| `normal` | 1 engine, 2 queries | 3–10s |
| `deep` | 1 engine, 3 queries | 5–20s |
| Multi-engine | 2–3 engines | 5–30s |

Fetch typically takes 1–10s depending on target page size and network. HTTP client timeout is 30s.

---

## Mode 1: CLI (local)

### Search

```bash
webpawm search "your search query" [flags]
```

| Flag | REST JSON equivalent | Description | Default |
|------|---------------------|-------------|---------|
| `-e, --engine` | `engine` | Single search engine | config default |
| `--engines` | `engines` | Multiple engines (comma-separated) | — |
| `-n, --max-results` | `max_results` | Max results to return | config value |
| `-l, --language` | `language` | Language code (`en`, `zh`, etc.) | — |
| `--arxiv-category` | `arxiv_category` | Arxiv category (`cs.AI`, `math.CO`, etc.) | — |
| `-d, --depth` | `search_depth` | Search depth: `quick`/`normal`/`deep` | `normal` |
| `--academic` | `include_academic` | Include academic papers from Arxiv | `false` |
| `--no-expand` | `auto_query_expand` | **Disable** auto query expansion (expansion is ON by default) | `false` |
| `--no-dedup` | `auto_deduplicate` | **Disable** auto result deduplication (dedup is ON by default) | `false` |

> **Boolean semantics:** `auto_query_expand` and `auto_deduplicate` default to `true`. The CLI uses inverted flags: `--no-expand` disables expansion, `--no-dedup` disables deduplication. Not passing these flags means the feature is ON. In REST, set `"auto_query_expand": false` to disable.

### Fetch

```bash
webpawm fetch "https://example.com" [flags]
```

| Flag | REST JSON equivalent | Description | Default |
|------|---------------------|-------------|---------|
| `-n, --max-length` | `max_length` | Max characters to return | `5000` |
| `-s, --start-index` | `start_index` | Start from this character index | `0` |
| `-r, --raw` | `raw` | Return raw HTML instead of Markdown | `false` |

### CLI Success Response: Search

```json
{
  "total_results": 3,
  "summary": {
    "total_raw_results": 3,
    "total_unique_results": 3,
    "original_query": "golang generics",
    "search_queries": ["golang generics", "golang generics latest news"],
    "engines_used": ["bingcn"],
    "search_depth": "normal"
  },
  "results": [
    {
      "index": 1,
      "title": "An Introduction To Generics - The Go Programming Language",
      "link": "https://go.dev/doc/tutorial/generics",
      "snippet": "This tutorial introduces the basics of generics in Go..."
    }
  ],
  "search_time": "2026-05-07T00:00:00+08:00"
}
```

### CLI Success Response: Fetch

```json
{
  "url": "https://example.com",
  "content": "# Example Domain\n\nThis domain is for use in illustrative examples...",
  "content_type": "markdown",
  "original_length": 1250,
  "truncated": false,
  "next_start": 0
}
```

When `truncated` is `true`, `next_start` contains the next `start_index` to pass for pagination.

When fetch fails (HTTP error, network issue), error details are in the `error` field with a 200 response:
```json
{"url": "https://broken.example.com", "content": "", "content_type": "", "original_length": 0, "truncated": false, "error": "Error fetching URL: HTTP 404"}
```

### Examples

```bash
# Basic search
webpawm search "golang generics tutorial"

# Search with specific engine
webpawm search -e google "climate change"

# Deep search with academic papers
webpawm search -d deep --academic "transformer architecture"

# Multi-engine search
webpawm search --engines google,bing "webassembly"

# Fetch a web page as Markdown
webpawm fetch "https://go.dev/doc/tutorial/getting-started"

# Fetch raw HTML
webpawm fetch -r "https://example.com"

# Fetch with pagination
webpawm fetch -n 5000 -s 5000 "https://example.com/large-page"
```

---

## Mode 2: REST API (remote)

When a webpawm server is running (`webpawm web -l localhost:8087`).

### Health & Discovery

```bash
# Server health check
curl -s http://localhost:8087/api/health
# → {"status":"ok","version":"1.0.0"}

# List configured engines
curl -s http://localhost:8087/api/engines
# → {"engines":["arxiv","bingcn","google"],"default":"bingcn"}
```

### Search

```bash
curl -s -X POST http://localhost:8087/api/search \
  -H "Content-Type: application/json" \
  -d '{"query": "golang generics", "engine": "google", "max_results": 5}'
```

### Fetch

```bash
curl -s -X POST http://localhost:8087/api/fetch \
  -H "Content-Type: application/json" \
  -d '{"url": "https://example.com", "max_length": 3000}'
```

### Authentication

If `api_key` is configured, include it:

```bash
curl -s -X POST http://localhost:8087/api/search \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-api-key" \
  -d '{"query": "test"}'
```

### Request Parameters (Search)

| Field | Type | Required | CLI flag | Description |
|-------|------|----------|----------|-------------|
| `query` | string | yes | positional | Search query |
| `engine` | string | no | `-e` | Single engine name |
| `engines` | []string | no | `--engines` | Multiple engine names |
| `max_results` | int | no | `-n` | Max results (default: 10) |
| `language` | string | no | `-l` | Language code |
| `arxiv_category` | string | no | `--arxiv-category` | Arxiv category |
| `search_depth` | string | no | `-d` | `quick`/`normal`/`deep` |
| `include_academic` | bool | no | `--academic` | Include academic papers |
| `auto_query_expand` | bool | no | `--no-expand` (inverted) | Auto expand queries (default: true) |
| `auto_deduplicate` | bool | no | `--no-dedup` (inverted) | Auto deduplicate results (default: true) |

### Request Parameters (Fetch)

| Field | Type | Required | CLI flag | Description |
|-------|------|----------|----------|-------------|
| `url` | string | yes | positional | Website URL (http/https only) |
| `max_length` | int | no | `-n` | Max chars (default: 5000) |
| `start_index` | int | no | `-s` | Start char index (default: 0) |
| `raw` | bool | no | `-r` | Return raw HTML (default: false) |

### Success Response Schema (Search)

```json
{
  "total_results": 3,
  "summary": {
    "total_raw_results": 3,
    "total_unique_results": 3,
    "original_query": "golang generics",
    "search_queries": ["golang generics", "golang generics latest news"],
    "engines_used": ["bingcn"],
    "search_depth": "normal"
  },
  "results": [
    {
      "index": 1,
      "title": "Result Title",
      "link": "https://example.com/result",
      "snippet": "Result snippet text..."
    }
  ],
  "search_time": "2026-05-07T00:00:00+08:00"
}
```

### Success Response Schema (Fetch)

```json
{
  "url": "https://example.com",
  "content": "# Markdown content of the page...",
  "content_type": "markdown",
  "original_length": 1250,
  "truncated": false,
  "next_start": 0,
  "error": ""
}
```

**Pagination:** When `truncated` is `true`, pass `next_start` as `start_index` in the next request to continue reading.

**Fetch errors:** On HTTP errors (404, timeout, etc.), the response has `"error"` populated with details. The HTTP status is still 200. Check the `error` field, not the HTTP status code.

### Error Responses

```json
{"error": "Bad Request", "message": "query is required"}
{"error": "Unauthorized", "message": "API key required..."}
{"error": "Payload Too Large", "message": "request body exceeds 1MB limit"}
{"error": "Unsupported Media Type", "message": "Content-Type must be application/json"}
```

HTTP status codes: 200 (success, including fetch errors), 400 (validation error), 401 (auth required), 413 (body too large), 415 (not JSON), 500 (server error).
