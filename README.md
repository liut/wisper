# Webpawm

Webpawm is an MCP (Model Context Protocol) server that provides web search capabilities with multiple search engine support.

## Features

- **Multiple Search Engines**: Supports SearXNG, Google, Bing, Brave, Bing CN (China), and Arxiv
- **Multiple Transport Modes**:
  - HTTP/SSE mode (via `webpawm web` command)
  - Stdio mode (via `webpawm` or `webpawm std` command)
- **Unified Search**: Single `web_search` tool with smart defaults (multi-engine, query expansion, deduplication)
- **Flexible Configuration**: Support for config file (~/.webpawm/config.json) and environment variables
- **Access Logging**: Built-in slog-based HTTP access logging

## Installation

```bash
go install github.com/liut/webpawm@latest
```

Or build from source:

```bash
git clone https://github.com/liut/webpawm.git
cd webpawm
go build -o webpawm .
```

## Usage

### Stdio Mode (Default)

Run as a local MCP tool:

```bash
webpawm
```

### HTTP/SSE Mode

Start the web server:

```bash
webpawm web --listen localhost:8087
```

The server provides two endpoints:
- HTTP: `http://localhost:8087/mcp`
- SSE: `http://localhost:8087/mcp/sse`

With URI prefix:
```bash
webpawm web --listen localhost:8087 --uri-prefix /api
```
Endpoints become:
- HTTP: `http://localhost:8087/api/mcp`
- SSE: `http://localhost:8087/api/mcp/sse`

## MCP Tools

Webpawm provides two MCP tools:

### web_search

Unified search tool with intelligent defaults. Supports single engine, multi-engine parallel search, and automatic query expansion with deduplication.

| Parameter | Type | Description |
|-----------|------|-------------|
| query | string | The search query (required) |
| engine | string | Single search engine to use (optional, mutually exclusive with engines) |
| engines | array | List of search engines to use (optional, mutually exclusive with engine) |
| max_results | integer | Maximum results to return (default: 10) |
| language | string | Language code for search results (e.g., 'en', 'zh') |
| arxiv_category | string | Arxiv category for academic paper search (e.g., 'cs.AI', 'math.CO') |
| search_depth | string | 'quick' (1 query), 'normal' (2 queries), 'deep' (3 queries). Default: 'normal' |
| include_academic | boolean | Include academic papers from Arxiv (default: false) |
| auto_query_expand | boolean | Automatically expand query with variations (default: true) |
| auto_deduplicate | boolean | Automatically deduplicate results by URL (default: true) |

**Note**: `engine` and `engines` support enum values - available engines are listed in the tool schema.

### web_fetch

Fetch a website and return its content with HTML to Markdown conversion.

| Parameter | Type | Description |
|-----------|------|-------------|
| url | string | URL of the website to fetch (required) |
| max_length | integer | Maximum number of characters to return (default: 5000) |
| start_index | integer | Start content from this character index (default: 0) |
| raw | boolean | Return raw HTML including script and style blocks (default: false) |

## Configuration

### Config File

Create `~/.webpawm/config.json`:

```json
{
  "searchxng_url": "https://searchx.ng",
  "google_api_key": "your-api-key",
  "google_cx": "your-search-engine-id",
  "bing_api_key": "your-bing-api-key",
  "brave_api_key": "your-brave-api-key",
  "max_results": 10,
  "default_engine": "searchxng",
  "listen_addr": "localhost:8087",
  "uri_prefix": "",
  "log_level": "info"
}
```

### Environment Variables

| Variable | Description |
|----------|-------------|
| WEBPAWM_SEARCHXNG_URL | SearXNG base URL |
| WEBPAWM_GOOGLE_API_KEY | Google Custom Search API key |
| WEBPAWM_GOOGLE_CX | Google Search Engine ID |
| WEBPAWM_BING_API_KEY | Bing Search API key |
| WEBPAWM_BRAVE_API_KEY | Brave Search API key |
| WEBPAWM_MAX_RESULTS | Default max results |
| WEBPAWM_DEFAULT_ENGINE | Default search engine |
| WEBPAWM_LISTEN_ADDR | HTTP listen address |
| WEBPAWM_URI_PREFIX | URI prefix for endpoints |
| WEBPAWM_LOG_LEVEL | Log level: debug, info, warn, error |

### Priority

Configuration priority (highest to lowest):
1. CLI flags
2. Environment variables
3. Config file
4. Default values

## License

MIT
