---
title: refactor: Replace map[string]any with typed structs in formatResponse
type: refactor
status: completed
date: 2026-04-06
---

# refactor: Replace map[string]any with typed structs in formatResponse

## Problem Statement

`formatResponse` at `server/handlers.go:150` returns `map[string]any` which lacks type safety, IDE support, and self-documenting structure. Additionally, several related structs (`webSearchOutput`, `SearchResultResponse`) were unused or only served as string wrappers.

## Resulting Call Chain (After Refactor)

```
handleWebSearchHandler (mcp.go)
  └── returns *WebSearchResponse ✅
      └── calls handleWebSearch
              └── returns *WebSearchResponse ✅
                      └── calls formatResponse()
                              └── returns *WebSearchResponse ✅
```

## Changes Summary

| File | Change |
|------|--------|
| `server/response.go` | Added `WebSearchResponse`, `SearchSummary`, `SearchResult`; removed unused `SearchResultResponse` |
| `server/handlers.go` | `formatResponse` and `handleWebSearch` return `*WebSearchResponse` |
| `server/mcp.go` | `handleWebSearchHandler` returns `*WebSearchResponse` |
| `server/params.go` | Removed `webSearchOutput` |

### New Structs (in `server/response.go`)

```go
type WebSearchResponse struct {
    Summary      SearchSummary  `json:"summary"`
    TotalResults int            `json:"total_results"`
    Results      []SearchResult `json:"results"`
    SearchTime   string         `json:"search_time"`
}

type SearchSummary struct {
    OriginalQuery      string   `json:"original_query"`
    SearchQueries      []string `json:"search_queries"`
    EnginesUsed        []string `json:"engines_used"`
    SearchDepth        string   `json:"search_depth"`
    TotalRawResults    int      `json:"total_raw_results"`
    TotalUniqueResults int      `json:"total_unique_results"`
}

type SearchResult struct {
    Index   int    `json:"index"`
    Title   string `json:"title"`
    Link    string `json:"link"`
    Snippet string `json:"snippet"`
}
```

## Acceptance Criteria

- [x] `handleWebSearch` returns `*WebSearchResponse`
- [x] `handleWebSearchHandler` returns `*WebSearchResponse`
- [x] `webSearchOutput` removed from `params.go`
- [x] Code compiles
- [x] MCP server functions correctly (build verified)
- [x] Existing tests pass

## Benefits

- **Type safety** - Compile-time checking throughout the call chain
- **IDE support** - Full autocompletion and refactoring support
- **Self-documenting** - Struct field names document the data
- **Removed dead code** - `SearchResultResponse` and `webSearchOutput` removed

## Files

- `server/response.go` - New typed structs
- `server/handlers.go` - Updated return types
- `server/mcp.go` - Updated return type
- `server/params.go` - Removed `webSearchOutput`
