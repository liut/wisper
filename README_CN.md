# Wisper

Wisper 是一个 MCP（Model Context Protocol）服务器，提供网页搜索功能，支持多种搜索引擎。

## 功能特性

- **多搜索引擎支持**：SearXNG、Google、Bing、Bing CN（必应中国）、Arxiv
- **多种传输模式**：
  - HTTP/SSE 模式（使用 `wisper web` 命令）
  - Stdio 模式（使用 `wisper` 或 `wisper std` 命令）
- **智能搜索**：智能查询优化与结果聚合
- **多引擎搜索**：同时跨多个搜索引擎搜索
- **灵活配置**：支持配置文件（~/.wisper/config.json）和环境变量
- **访问日志**：内置基于 slog 的 HTTP 访问日志

## 安装

```bash
go install github.com/liut/wisper@latest
```

或从源码构建：

```bash
git clone https://github.com/liut/wisper.git
cd wisper
go build -o wisper .
```

## 使用方法

### Stdio 模式（默认）

作为本地 MCP 工具运行：

```bash
wisper
```

### HTTP/SSE 模式

启动 Web 服务器：

```bash
wisper web --listen localhost:8080
```

服务器提供两个端点：
- HTTP: `http://localhost:8080/mcp`
- SSE: `http://localhost:8080/mcp/sse`

使用 URI 前缀：
```bash
wisper web --listen localhost:8080 --uri-prefix /api
```
端点变为：
- HTTP: `http://localhost:8080/api/mcp`
- SSE: `http://localhost:8080/api/mcp/sse`

## MCP 工具

Wisper 提供三个 MCP 工具：

### web_search

使用单个搜索引擎搜索网页。

| 参数 | 类型 | 说明 |
|------|------|------|
| query | string | 搜索查询（必需） |
| engine | string | 使用的搜索引擎（可选） |
| max_results | integer | 最大结果数（1-50，默认：10） |
| language | string | 语言代码（如 'en', 'zh'） |
| arxiv_category | string | Arxiv 学术论文类别 |

### multi_search

同时跨多个搜索引擎搜索。

| 参数 | 类型 | 说明 |
|------|------|------|
| query | string | 搜索查询（必需） |
| engines | array | 搜索引擎列表（可选） |
| max_results_per_engine | integer | 每个引擎的最大结果数（1-20，默认：5） |

### smart_search

智能搜索，包含查询优化和结果聚合。

| 参数 | 类型 | 说明 |
|------|------|------|
| question | string | 用户问题或搜索意图（必需） |
| search_depth | string | 搜索深度：'quick'、'normal' 或 'deep' |
| include_academic | boolean | 是否包含 Arxiv 学术论文 |

## 配置

### 配置文件

创建 `~/.wisper/config.json`：

```json
{
  "searchxng_url": "https://searchx.ng",
  "google_api_key": "your-api-key",
  "google_cx": "your-search-engine-id",
  "bing_api_key": "your-bing-api-key",
  "max_results": 10,
  "default_engine": "searchxng",
  "listen_addr": "localhost:8080",
  "uri_prefix": "",
  "log_level": "info"
}
```

### 环境变量

| 变量名 | 说明 |
|--------|------|
| WISPER_SEARCHXNG_URL | SearXNG 基础 URL |
| WISPER_GOOGLE_API_KEY | Google 自定义搜索 API 密钥 |
| WISPER_GOOGLE_CX | Google 搜索引擎 ID |
| WISPER_BING_API_KEY | Bing 搜索 API 密钥 |
| WISPER_MAX_RESULTS | 默认最大结果数 |
| WISPER_DEFAULT_ENGINE | 默认搜索引擎 |
| WISPER_LISTEN_ADDR | HTTP 监听地址 |
| WISPER_URI_PREFIX | 端点 URI 前缀 |
| WISPER_LOG_LEVEL | 日志级别：debug, info, warn, error |

### 优先级

配置优先级（从高到低）：
1. CLI 标志
2. 环境变量
3. 配置文件
4. 默认值

## 许可证

MIT
