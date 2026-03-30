---
title: feat: Add enum customization support to WithInputSchema
type: feat
status: completed
date: 2026-03-30
---

# WithInputSchema Enum 定制化支持

## Overview

扩展 `WithInputSchema` 函数，直接操作 `*jsonschema.Schema` 对象，为特定字段注入 enum 约束。同时解决当前 `web_search` 使用 map 构建 schema 导致的字段顺序不确定问题。

**前提**：已升级 `github.com/mark3labs/mcp-go` 从 `v0.45.0` 到 `v0.46.0`，底层库从 `invopop/jsonschema` 切换到 `google/jsonschema-go`。

## Problem Statement

当前两种方案各有问题：

| 工具 | 方法 | 优点 | 缺点 |
|------|------|------|------|
| `web_search` | `NewToolWithRawSchema` + map | 支持 enum | **字段顺序不确定**（Go map 无序） |
| `web_fetch` | `WithInputSchema[WebFetchParams]()` | **字段顺序正确**（struct 定义顺序） | 不支持 enum |

用户希望结合两者优点：**字段顺序正确 + 支持 enum 定制**。

## Google jsonschema-go 关键发现

`google/jsonschema-go` 的 Schema 结构：

```go
type Schema struct {
    Properties   map[string]*Schema  // 普通 map，无序
    PropertyOrder []string            // 控制 JSON 输出时的属性顺序
    Enum         []any               // enum 约束
    Items        *Schema              // 数组元素的 schema
    // ...
}
```

**重要特性**：
1. `jsonschema.For[T](nil)` 自动设置 `PropertyOrder` 为 struct 字段定义顺序
2. **可直接修改** `schema.Properties[fieldName].Enum`
3. 对于数组字段，修改 `schema.Properties[fieldName].Items.Enum`

## Proposed Solution

创建本地辅助函数 `withInputSchemaWithEnums`，直接在 `*jsonschema.Schema` 对象上操作：

```go
// withInputSchemaWithEnums creates input schema with enum constraints for specific fields
func withInputSchemaWithEnums[T any](enumOverrides map[string][]string) server.ToolOption {
    return func(t *server.Tool) {
        // 1. 生成基础 schema
        schema, err := jsonschema.For[T](&jsonschema.ForOptions{IgnoreInvalidTypes: true})
        if err != nil {
            return
        }

        // 2. 直接修改 Properties 中的 Enum
        for fieldName, enumValues := range enumOverrides {
            if prop, ok := schema.Properties[fieldName]; ok {
                anyValues := make([]any, len(enumValues))
                for i, v := range enumValues {
                    anyValues[i] = v
                }
                prop.Enum = anyValues
            }
        }

        // 3. Marshal 并设置
        mcpSchema, err := json.Marshal(schema)
        if err != nil {
            return
        }

        t.InputSchema.Type = ""
        t.RawInputSchema = json.RawMessage(mcpSchema)
    }
}
```

## Technical Approach

### 修改文件

| 文件 | 修改内容 |
|------|----------|
| `server/mcp.go` | 新增 `withInputSchemaWithEnums` 本地辅助函数 |
| `server/mcp.go` | 将 `web_search` 从 `NewToolWithRawSchema` 改为 `NewTool` + `withInputSchemaWithEnums[WebSearchParams]` |

### 实现步骤

#### Step 1: 创建 withInputSchemaWithEnums 辅助函数

```go
// withInputSchemaWithEnums creates input schema with enum constraints for specific fields
// 直接操作 *jsonschema.Schema 对象，不经过 map 转换
func withInputSchemaWithEnums[T any](enumOverrides map[string][]string) server.ToolOption {
    return func(t *server.Tool) {
        schema, err := jsonschema.For[T](&jsonschema.ForOptions{IgnoreInvalidTypes: true})
        if err != nil {
            return
        }

        for fieldName, enumValues := range enumOverrides {
            if prop, ok := schema.Properties[fieldName]; ok {
                // 转换为 []any
                anyValues := make([]any, len(enumValues))
                for i, v := range enumValues {
                    anyValues[i] = v
                }
                prop.Enum = anyValues
            }
        }

        mcpSchema, err := json.Marshal(schema)
        if err != nil {
            return
        }

        t.InputSchema.Type = ""
        t.RawInputSchema = json.RawMessage(mcpSchema)
    }
}
```

#### Step 2: 重构 web_search 工具注册

将：
```go
engines := s.getAvailableEngines()
webSearchSchema := buildWebSearchSchema(engines)
webSearchTool := mcp.NewToolWithRawSchema("web_search", "...", webSearchSchema)
```

改为：
```go
engines := s.getAvailableEngines()
sortedEngines := slices.Sorted(engines)  // 确保顺序一致
enumOverrides := map[string][]string{
    "engine":  sortedEngines,
    "engines": sortedEngines,
}
webSearchTool := mcp.NewTool("web_search",
    mcp.WithDescription("Search the web using various search engines..."),
    withInputSchemaWithEnums[WebSearchParams](enumOverrides),
)
```

#### Step 3: 移除 buildWebSearchSchema 函数

重构完成后，`buildWebSearchSchema` 不再需要，可移除。

### 关键设计决策

1. **直接操作 Schema 对象**：不 unmarshal 成 map，直接在 `*jsonschema.Schema` 上修改
2. **利用 PropertyOrder**：`google/jsonschema-go` 的 `PropertyOrder` 自动记录 struct 字段顺序
3. **enum 顺序确定性**：使用 `slices.Sorted()` 确保 enum 值顺序一致
4. **向后兼容**：`WebSearchParams` 结构不变，只改变 schema 生成方式

## System-Wide Impact

- **web_search 工具**：InputSchema 生成方式改变，但输出格式不变
- **web_fetch 工具**：不受影响
- **其他工具**：不受影响

## Acceptance Criteria

- [ ] `web_search` 工具的 `engine` 和 `engines` 参数支持 enum
- [ ] enum 值为排序后的引擎名称列表（顺序确定）
- [ ] schema 中字段顺序与 `WebSearchParams` 结构体定义顺序一致
- [ ] `web_search` 工具行为保持不变（仅 schema 生成方式改变）
- [ ] 移除 `buildWebSearchSchema` 函数

## Dependencies

- `github.com/mark3labs/mcp-go@v0.46.0`
- `github.com/google/jsonschema-go@v0.4.2`

## Related Work

- 重构计划：`docs/plans/2026-03-30-001-refactor-unified-search-plan.md`
