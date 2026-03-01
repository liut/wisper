# Wisper

Wisper is an MCP (Model Context Protocol) server that provides web search capabilities with multiple search engine support.

## Features

- **Multiple Search Engines**: Supports SearXNG, Google, Bing, Brave, Bing CN (China), and Arxiv
- **Multiple Transport Modes**:
  - HTTP/SSE mode (via `wisper web` command)
  - Stdio mode (via `wisper` or `wisper std` command)
- **Smart Search**: Intelligent query optimization with result aggregation
- **Multi-Engine Search**: Search across multiple engines simultaneously
- **Flexible Configuration**: Support for config file (~/.wisper/config.json) and environment variables
- **Access Logging**: Built-in slog-based HTTP access logging

## Installation

```bash
go install github.com/liut/wisper@latest
```

Or build from source:

```bash
git clone https://github.com/liut/wisper.git
cd wisper
go build -o wisper .
```

## Usage

### Stdio Mode (Default)

Run as a local MCP tool:

```bash
wisper
```

### HTTP/SSE Mode

Start the web server:

```bash
wisper web --listen localhost:8087
```

The server provides two endpoints:
- HTTP: `http://localhost:8087/mcp`
- SSE: `http://localhost:8087/mcp/sse`

With URI prefix:
```bash
wisper web --listen localhost:8087 --uri-prefix /api
```
Endpoints become:
- HTTP: `http://localhost:8087/api/mcp`
- SSE: `http://localhost:8087/api/mcp/sse`

## MCP Tools

Wisper provides three MCP tools:

### web_search

Search the web using a single search engine.

| Parameter | Type | Description |
|-----------|------|-------------|
| query | string | The search query (required) |
| engine | string | Search engine to use (optional) |
| max_results | integer | Maximum results (1-50, default: 10) |
| language | string | Language code (e.g., 'en', 'zh') |
| arxiv_category | string | Arxiv category for academic papers |

### multi_search

Search across multiple search engines simultaneously.

| Parameter | Type | Description |
|-----------|------|-------------|
| query | string | The search query (required) |
| engines | array | List of search engines (optional) |
| max_results_per_engine | integer | Max results per engine (1-20, default: 5) |

### smart_search

Intelligently search with query optimization and result aggregation.

| Parameter | Type | Description |
|-----------|------|-------------|
| question | string | User's question or search intent (required) |
| search_depth | string | 'quick', 'normal', or 'deep' |
| include_academic | boolean | Include academic papers from Arxiv |

## Configuration

### Config File

Create `~/.wisper/config.json`:

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
| WISPER_SEARCHXNG_URL | SearXNG base URL |
| WISPER_GOOGLE_API_KEY | Google Custom Search API key |
| WISPER_GOOGLE_CX | Google Search Engine ID |
| WISPER_BING_API_KEY | Bing Search API key |
| WISPER_BRAVE_API_KEY | Brave Search API key |
| WISPER_MAX_RESULTS | Default max results |
| WISPER_DEFAULT_ENGINE | Default search engine |
| WISPER_LISTEN_ADDR | HTTP listen address |
| WISPER_URI_PREFIX | URI prefix for endpoints |
| WISPER_LOG_LEVEL | Log level: debug, info, warn, error |

### Priority

Configuration priority (highest to lowest):
1. CLI flags
2. Environment variables
3. Config file
4. Default values

## License

MIT
