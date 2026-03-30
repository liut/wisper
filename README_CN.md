# Webpawm

Webpawm 是一个 MCP（Model Context Protocol）服务器，提供网页搜索功能，支持多种搜索引擎。

## 功能特性

- **多搜索引擎支持**：SearXNG、Google、Bing、Brave、Bing CN（必应中国）、Arxiv
- **多种传输模式**：
  - HTTP/SSE 模式（使用 `webpawm web` 命令）
  - Stdio 模式（使用 `webpawm` 或 `webpawm std` 命令）
- **统一搜索**：单一的 `web_search` 工具，智能默认（多引擎、查询扩展、去重）
- **灵活配置**：支持配置文件（~/.webpawm/config.json）和环境变量
- **访问日志**：内置基于 slog 的 HTTP 访问日志

## 安装

```bash
go install github.com/liut/webpawm@latest
```

或从源码构建：

```bash
git clone https://github.com/liut/webpawm.git
cd webpawm
go build -o webpawm .
```

## 使用方法

### Stdio 模式（默认）

作为本地 MCP 工具运行：

```bash
webpawm
```

### HTTP/SSE 模式

启动 Web 服务器：

```bash
webpawm web --listen localhost:8087
```

服务器提供两个端点：
- HTTP: `http://localhost:8087/mcp`
- SSE: `http://localhost:8087/mcp/sse`

使用 URI 前缀：
```bash
webpawm web --listen localhost:8087 --uri-prefix /api
```
端点变为：
- HTTP: `http://localhost:8087/api/mcp`
- SSE: `http://localhost:8087/api/mcp/sse`

## MCP 工具

Webpawm 提供两个 MCP 工具：

### web_search

统一搜索工具，智能默认。支持单引擎、多引擎并行搜索，以及自动查询扩展和去重。

| 参数 | 类型 | 说明 |
|------|------|------|
| query | string | 搜索查询（必需） |
| engine | string | 使用的单个搜索引擎（可选，与 engines 互斥） |
| engines | array | 使用的搜索引擎列表（可选，与 engine 互斥） |
| max_results | integer | 最大结果数（默认：10） |
| language | string | 语言代码（如 'en', 'zh'） |
| arxiv_category | string | Arxiv 学术论文类别（如 'cs.AI', 'math.CO'） |
| search_depth | string | 搜索深度：'quick'（1次查询）、'normal'（2次查询）、'deep'（3次查询）。默认：'normal' |
| include_academic | boolean | 是否包含 Arxiv 学术论文（默认：false） |
| auto_query_expand | boolean | 是否自动扩展查询（默认：true） |
| auto_deduplicate | boolean | 是否自动按 URL 去重（默认：true） |

**注意**：`engine` 和 `engines` 参数支持枚举值 - 可用引擎列在工具 schema 中。

### web_fetch

获取网页内容并转换为 Markdown 格式返回。

| 参数 | 类型 | 说明 |
|------|------|------|
| url | string | 要获取的网页 URL（必需） |
| max_length | integer | 最大字符数（默认：5000） |
| start_index | integer | 开始字符索引（默认：0） |
| raw | boolean | 返回包含 script 和 style 标签的原始 HTML（默认：false） |

## 配置

### 配置文件

创建 `~/.webpawm/config.json`：

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

### 环境变量

| 变量名 | 说明 |
|--------|------|
| WEBPAWM_SEARCHXNG_URL | SearXNG 基础 URL |
| WEBPAWM_GOOGLE_API_KEY | Google 自定义搜索 API 密钥 |
| WEBPAWM_GOOGLE_CX | Google 搜索引擎 ID |
| WEBPAWM_BING_API_KEY | Bing 搜索 API 密钥 |
| WEBPAWM_BRAVE_API_KEY | Brave 搜索 API 密钥 |
| WEBPAWM_MAX_RESULTS | 默认最大结果数 |
| WEBPAWM_DEFAULT_ENGINE | 默认搜索引擎 |
| WEBPAWM_LISTEN_ADDR | HTTP 监听地址 |
| WEBPAWM_URI_PREFIX | 端点 URI 前缀 |
| WEBPAWM_LOG_LEVEL | 日志级别：debug, info, warn, error |

### 优先级

配置优先级（从高到低）：
1. CLI 标志
2. 环境变量
3. 配置文件
4. 默认值

## 许可证

MIT
