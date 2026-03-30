#!/usr/bin/env bun

/**
 * Webpawm MCP Test Script
 *
 * Tests the MCP server via HTTP JSON-RPC.
 *
 * Usage:
 *   bun run scripts/test-mcp.ts
 *
 * Environment:
 *   WEBPAWM_MCP   MCP server URL (default: http://localhost:8087/mcp)
 */

const MCP_URL = process.env.WEBPAWM_MCP || "http://localhost:8087/mcp";

// Colors
const GREEN = "\x1b[32m";
const RED = "\x1b[31m";
const YELLOW = "\x1b[33m";
const NC = "\x1b[0m";

let passed = 0;
let failed = 0;
let nextId = 1;
let mcpSessionId = "";

function jsonRequest(method: string, params?: any): any {
  return { jsonrpc: "2.0", id: nextId++, method, params };
}

async function mcpRequest(method: string, params?: any): Promise<any> {
  const req = jsonRequest(method, params);

  const reqHeaders: Record<string, string> = {
    "Content-Type": "application/json",
    "Accept": "application/json",
  };

  if (mcpSessionId) {
    reqHeaders["Mcp-Session-Id"] = mcpSessionId;
  }

  const res = await fetch(MCP_URL, {
    method: "POST",
    headers: reqHeaders,
    body: JSON.stringify(req),
  });

  // Save MCP session ID from response header
  const newSessionId = res.headers.get("Mcp-Session-Id");
  if (newSessionId) {
    mcpSessionId = newSessionId;
  }

  if (!res.ok) {
    throw new Error(`HTTP ${res.status}: ${await res.text()}`);
  }

  // Parse JSON response directly (not SSE)
  return res.json();
}

async function test(name: string, fn: () => Promise<void>): Promise<void> {
  process.stdout.write(`${YELLOW}[TEST]${NC} ${name}... `);
  try {
    await fn();
    console.log(`${GREEN}OK${NC}`);
    passed++;
  } catch (err: any) {
    console.log(`${RED}FAIL${NC}: ${err.message}`);
    failed++;
  }
}

async function main() {
  console.warn("=== Webpawm MCP Test ===");
  console.warn(`URL: ${MCP_URL}`);
  console.warn("");

  // Test 1: Initialize
  await test("initialize", async () => {
    const resp = await mcpRequest("initialize", {
      protocolVersion: "2024-11-05",
      capabilities: {},
      clientInfo: { name: "test-client", version: "1.0.0" },
    });
    if (!resp.result) {
      throw new Error("No result");
    }
  });

  // Test 2: List tools
  await test("tools/list", async () => {
    const resp = await mcpRequest("tools/list");
    console.error(`  Full response: ${JSON.stringify(resp, null, 2)}`);
    if (!resp.result?.tools?.length) {
      throw new Error("No tools returned");
    }
    console.error(`  Found ${resp.result.tools.length} tools: ${resp.result.tools.map((t: any) => t.name).join(", ")}`);
  });

  // Test 3: web_search - simple query
  await test("web_search (simple query)", async () => {
    const resp = await mcpRequest("tools/call", {
      name: "web_search",
      arguments: { query: "hello world" },
    });
    console.error(`  Full response: ${JSON.stringify(resp, null, 2)}`);
    if (!resp.result?.content?.[0]?.text) {
      throw new Error("No output");
    }
    console.error(`  Response length: ${resp.result.content[0].text.length} chars`);
  });

  // Test 4: web_search - with specific engine
  await test("web_search (single engine)", async () => {
    const resp = await mcpRequest("tools/call", {
      name: "web_search",
      arguments: { query: "golang", engine: "google" },
    });
    console.error(`  Full response: ${JSON.stringify(resp, null, 2)}`);
    if (!resp.result?.content?.[0]?.text) {
      throw new Error("No output");
    }
  });

  // Test 5: web_search - deep search with academic
  await test("web_search (deep with academic)", async () => {
    const resp = await mcpRequest("tools/call", {
      name: "web_search",
      arguments: {
        query: "machine learning",
        search_depth: "deep",
        include_academic: true,
      },
    });
    console.error(`  Full response: ${JSON.stringify(resp, null, 2)}`);
    if (!resp.result?.content?.[0]?.text) {
      throw new Error("No output");
    }
  });

  // Test 6: web_fetch - simple URL
  await test("web_fetch (simple URL)", async () => {
    const resp = await mcpRequest("tools/call", {
      name: "web_fetch",
      arguments: { url: "https://example.com" },
    });
    console.error(`  Full response: ${JSON.stringify(resp, null, 2)}`);
    if (!resp.result?.content?.[0]?.text) {
      throw new Error("No output");
    }
    console.error(`  Response length: ${resp.result.content[0].text.length} chars`);
  });

  // Test 7: web_fetch - with max_length
  await test("web_fetch (with max_length)", async () => {
    const resp = await mcpRequest("tools/call", {
      name: "web_fetch",
      arguments: { url: "https://example.com", max_length: 100 },
    });
    console.error(`  Full response: ${JSON.stringify(resp, null, 2)}`);
    if (!resp.result?.content?.[0]?.text) {
      throw new Error("No output");
    }
  });

  // Test 8: web_fetch - raw HTML
  await test("web_fetch (raw HTML)", async () => {
    const resp = await mcpRequest("tools/call", {
      name: "web_fetch",
      arguments: { url: "https://example.com", raw: true },
    });
    console.error(`  Full response: ${JSON.stringify(resp, null, 2)}`);
    if (!resp.result?.content?.[0]?.text) {
      throw new Error("No output");
    }
  });

  // Summary
  console.warn("");
  console.warn(`=== Results: ${passed} passed, ${failed} failed ===`);

  if (failed > 0) {
    process.exit(1);
  }
}

main().catch(console.error);
