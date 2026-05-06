---
title: feat: Add CLI subcommands and REST API endpoints for search/fetch
type: feat
status: completed
date: 2026-05-07
---

# feat: Add CLI subcommands and REST API endpoints for search/fetch

## Overview

当前项目仅通过 MCP 协议暴露 `web_search` 和 `web_fetch` 两个工具。不支持 MCP 但支持 Skill 的 AI Agent 无法使用。需要新增两种调用方式：

1. **CLI 子命令**：`webpawm search "query"` / `webpawm fetch "url"`，JSON 输出到 stdout
2. **REST API 端点**：`POST /api/search` / `POST /api/fetch`，返回纯 JSON（非 MCP 协议）

这两种方式复用现有的 handler 逻辑（`handleWebSearch` / `handleWebFetch`），它们已经是协议无关的纯业务逻辑。

## Problem Statement / Motivation

- 部分 AI Agent 平台不支持 MCP 协议，但支持通过 Shell 执行命令或调用 HTTP API
- 用户希望编写 Skill 文件来让这些 Agent 使用 webpawm 的能力
- 当前无 CLI 直接调用方式，也无 REST API

## Proposed Solution

### 架构：薄封装层 + 复用现有 Handler

```
                    ┌──────────────────────────┐
                    │  handleWebSearch (导出)    │
                    │  handleWebFetch  (导出)    │
                    │  server/handle_*.go       │
                    └────┬──────┬──────┬────────┘
                         │      │      │
                    ┌────┘      │      └────────────┐
                    ▼           ▼                   ▼
              server/mcp.go  main.go           server/rest.go
              (MCP 工具)     (CLI 子命令)       (REST Handler)
```

### 变更范围

| 文件 | 变更 | 说明 |
|------|------|------|
| `server/handle_search.go` | 导出 `handleWebSearch` → `HandleWebSearch` | 供 main 和 rest 包调用 |
| `server/handle_fetch.go` | 导出 `handleWebFetch` → `HandleWebFetch` | 同上 |
| `server/mcp.go` | 更新内部调用，清理无用 wrapper | MCP 注册改用导出名 |
| `server/rest.go` | **新建** | REST handler：解析 JSON body → 调用 handler → 写 JSON 响应 |
| `main.go` | 新增 `searchCmd`、`fetchCmd`；注册 REST 路由 | CLI 和 HTTP 路由注册 |

### 设计决策

#### 1. 错误响应格式（REST）

遵循 `server/http.go` 中已有的 auth middleware 模式：

```json
{"error": "Bad Request", "message": "query is required"}
```

- 400：参数校验失败
- 413：请求体过大
- 415：Content-Type 不是 application/json
- 500：内部错误
- `WebFetchResponse.Error` 非空时仍返回 200（fetch 错误是数据，不是协议错误）

新增 `writeJSONError(w, statusCode, error, message)` 辅助函数。

#### 2. CLI 标志

所有 `WebSearchParams` / `WebFetchParams` 字段均暴露为可选 flag：

**search 子命令：**
| Flag | 类型 | 默认值 | 对应字段 |
|------|------|--------|----------|
| `-e, --engine` | string | 配置的默认引擎 | Engine |
| `--engines` | stringSlice | nil | Engines |
| `-n, --max-results` | int | 配置的 max_results | MaxResults |
| `-l, --language` | string | "" | Language |
| `--arxiv-category` | string | "" | ArxivCategory |
| `-d, --depth` | string | "normal" | SearchDepth |
| `--academic` | bool | false | IncludeAcademic |
| `--no-expand` | bool | false | AutoQueryExpand (取反) |
| `--no-dedup` | bool | false | AutoDeduplicate (取反) |

**fetch 子命令：**
| Flag | 类型 | 默认值 | 对应字段 |
|------|------|--------|----------|
| `-n, --max-length` | int | 5000 | MaxLength |
| `-s, --start-index` | int | 0 | StartIndex |
| `-r, --raw` | bool | false | Raw |

#### 3. REST 路由前缀

使用固定 `/api/` 前缀，**不受 `uri_prefix` 影响**。REST API 不是 MCP 协议端点，应与 MCP 路由分离。

#### 4. CLI 输出格式

使用 `json.Encoder` + `SetIndent("", "  ")` 输出 pretty-printed JSON，便于人工阅读和 `jq` 管道处理。

#### 5. CLI 退出码

- 0：成功生成 JSON 响应（即使响应中包含 error 字段，如 fetch HTTP 404）
- 1：基础设施错误（配置加载失败、参数校验失败）

#### 6. 请求体大小限制

REST handler 使用 `http.MaxBytesReader` 限制请求体为 1MB，超出返回 413。

## Technical Considerations

- **导出 handler 影响面小**：`handleWebSearch` 只在 `mcp.go` 中被调用 1 次，`handleWebFetch` 也是 1 次
- **WebServer 并发安全**：字段均为构造后只读，无需额外同步
- **中间件自动复用**：REST 路由注册在同一个 `http.ServeMux`，自动获得 API Key 认证、日志、安全头
- **Content-Type 校验**：REST handler 要求 `application/json`，否则返回 415
- **无新配置字段**：完全复用现有 Config

## System-Wide Impact

### Interaction Graph

```
CLI 调用链:
  cobra.Run → runSearchCommand/runFetchCommand
    → getConfig() → viper.Unmarshal
    → server.NewWebServer(config) → 初始化所有引擎
    → srv.HandleWebSearch(ctx, params) / srv.HandleWebFetch(ctx, params)
    → json.Encoder(os.Stdout).Encode(resp)

REST 调用链:
  HTTP POST /api/search
    → APIKeyAuthMiddleware (可选)
    → LoggingMiddleware
    → SecurityHeadersMiddleware
    → RESTHandler.HandleSearch
      → json.Decode(r.Body) → WebSearchParams
      → srv.HandleWebSearch(ctx, params)
      → json.NewEncoder(w).Encode(resp)
```

### Error Propagation

- **CLI**: handler 返回 error → `fmt.Fprintf(os.Stderr, ...)` + `os.Exit(1)`
- **REST 校验错误**: 400 + JSON error body
- **REST handler 错误**: 500 + JSON error body
- **Fetch 业务错误**: 200 + `WebFetchResponse{Error: "..."}`（与 MCP 行为一致）

### API Surface Parity

| 功能 | MCP | CLI (新增) | REST (新增) |
|------|-----|-----------|-------------|
| web_search | ✅ `mcp__webpawm__web_search` | ✅ `webpawm search` | ✅ `POST /api/search` |
| web_fetch | ✅ `mcp__webpawm__web_fetch` | ✅ `webpawm fetch` | ✅ `POST /api/fetch` |

### Integration Test Scenarios

1. **CLI search 基本流程**：`webpawm search "test"` → stdout 输出有效 JSON → exit 0
2. **CLI fetch 基本流程**：`webpawm fetch "https://httpbin.org/json"` → stdout 输出含 Markdown 内容的 JSON
3. **REST search 带 API Key**：`curl -H "X-API-Key: xxx" -H "Content-Type: application/json" -d '{"query":"test"}' http://localhost:8087/api/search` → 200 JSON
4. **REST fetch 无 API Key 拒绝**：配置了 API Key 时，不带 Key 请求 → 401
5. **REST 参数校验**：`POST /api/search` 空 body → 400 `{"error":"Bad Request","message":"query is required"}`

## Acceptance Criteria

### Functional Requirements

- [x] `webpawm search "query"` 输出 JSON 搜索结果到 stdout，exit 0
- [x] `webpawm search` 无参数时输出错误到 stderr，exit 1
- [x] `webpawm fetch "url"` 输出 JSON 抓取结果到 stdout，exit 0
- [x] `webpawm fetch` 无参数时输出错误到 stderr，exit 1
- [x] 所有 search/fetch 参数可通过 CLI flag 指定
- [x] `POST /api/search` 接受 JSON body，返回 JSON 搜索结果
- [x] `POST /api/fetch` 接受 JSON body，返回 JSON 抓取结果
- [x] REST 端点与 MCP 端点共享同一中间件（日志、安全头、API Key 认证）
- [x] REST 端点不在 `uri_prefix` 下，固定使用 `/api/` 前缀
- [x] REST 请求体超过 1MB 返回 413
- [x] REST 非 JSON Content-Type 返回 415
- [x] REST 参数校验失败返回 400 + JSON error body

### Non-Functional Requirements

- [x] 不引入新的外部依赖
- [x] 现有 MCP 功能不受影响（回归测试通过）
- [x] CLI 和 REST 复用相同的 handler 逻辑，无代码重复

## Dependencies & Risks

- 无外部依赖变更
- 风险低：变更仅限于新增代码路径，不修改现有 MCP 逻辑
- handler 导出是破坏性变更（仅影响包内调用，mcp.go 同步更新即可）

## Implementation Plan

### Step 1: 导出 handler

- `server/handle_search.go`: `handleWebSearch` → `HandleWebSearch`
- `server/handle_fetch.go`: `handleWebFetch` → `HandleWebFetch`
- `server/mcp.go`: 更新调用方，移除无用的 `handleWebSearchHandler` wrapper

### Step 2: 新增 REST handler

- 新建 `server/rest.go`
- 实现 `RESTHandler` 结构体，持有 `*WebServer`
- 实现 `HandleSearch(w, r)` 和 `HandleFetch(w, r)` 方法
- 实现 `writeJSONError(w, statusCode, error, message)` 辅助函数
- 请求体限制 1MB，Content-Type 校验

### Step 3: 新增 CLI 子命令

- `main.go`: 新增 `searchCmd` 和 `fetchCmd` cobra 命令
- 实现 `runSearchCommand` / `runFetchCommand`
- 所有参数暴露为 flag

### Step 4: 注册 REST 路由

- `main.go` `startHTTPServer()`: 在 mux 上注册 `POST /api/search` 和 `POST /api/fetch`

### Step 5: 编写 Skill 文件

- 新建 `skills/webpawm.md`，描述如何通过 CLI 和 REST 使用这两个工具

### Step 6: 测试验证

- 运行现有测试确保无回归
- 手动验证 CLI 和 REST 端点

## Sources & References

### Internal References

- Handler 逻辑: `server/handle_search.go:14`, `server/handle_fetch.go:23`
- MCP 注册: `server/mcp.go:28-39`
- 中间件: `server/http.go:24-75`
- 参数结构: `server/params.go`
- 响应结构: `server/response.go`
- HTTP 服务器启动: `main.go:236-319`
- Cobra 命令模式: `main.go:61-89`

### Design Patterns

- 统一 handler 模式: `docs/solutions/design-patterns/unified-mcp-search-tool.md`
- 架构分层决策: `docs/plans/2026-04-06-001-refactor-analyze-main-server-migration-plan.md`
- 类型化响应结构: `docs/plans/2026-04-06-002-refactor-format-response-structs-plan.md`
