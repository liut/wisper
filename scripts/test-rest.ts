#!/usr/bin/env bun

/**
 * Webpawm REST API Test Script
 *
 * Tests the REST API endpoints (search, fetch, health, engines).
 *
 * Usage:
 *   bun run scripts/test-rest.ts
 *
 * Environment:
 *   WEBPAWM_REST  REST API base URL (default: http://localhost:8087)
 *   WEBPAWM_KEY   API key for authentication (optional)
 */

const BASE = process.env.WEBPAWM_REST || "http://localhost:8087";
const API_KEY = process.env.WEBPAWM_KEY || "";

// Colors
const GREEN = "\x1b[32m";
const RED = "\x1b[31m";
const YELLOW = "\x1b[33m";
const NC = "\x1b[0m";

let passed = 0;
let failed = 0;

function headers(): Record<string, string> {
  const h: Record<string, string> = { "Content-Type": "application/json" };
  if (API_KEY) h["X-API-Key"] = API_KEY;
  return h;
}

async function post(path: string, body: any): Promise<{ status: number; data: any }> {
  const res = await fetch(`${BASE}${path}`, {
    method: "POST",
    headers: headers(),
    body: JSON.stringify(body),
  });
  const data = await res.json();
  return { status: res.status, data };
}

async function get(path: string): Promise<{ status: number; data: any }> {
  const h: Record<string, string> = {};
  if (API_KEY) h["X-API-Key"] = API_KEY;
  const res = await fetch(`${BASE}${path}`, { method: "GET", headers: h });
  const data = await res.json();
  return { status: res.status, data };
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
  console.warn("=== Webpawm REST API Test ===");
  console.warn(`URL: ${BASE}`);
  if (API_KEY) console.warn(`Auth: API key configured`);
  console.warn("");

  // Test 1: Health check
  await test("GET /api/health", async () => {
    const { status, data } = await get("/api/health");
    if (status !== 200) throw new Error(`expected 200, got ${status}`);
    if (data.status !== "ok") throw new Error(`expected ok, got: ${JSON.stringify(data)}`);
    console.error(`  Status: ${data.status}, Version: ${data.version}`);
  });

  // Test 2: Engines list
  await test("GET /api/engines", async () => {
    const { status, data } = await get("/api/engines");
    if (status !== 200) throw new Error(`expected 200, got ${status}`);
    if (!data.engines?.length) throw new Error("no engines returned");
    console.error(`  Engines: ${data.engines.join(", ")}, Default: ${data.default}`);
  });

  // Test 3: search - basic query
  await test("POST /api/search (basic)", async () => {
    const { status, data } = await post("/api/search", { query: "hello world" });
    if (status !== 200) throw new Error(`expected 200, got ${status}`);
    if (!data.results) throw new Error("missing 'results'");
    if (!data.summary) throw new Error("missing 'summary'");
    console.error(`  Results: ${data.total_results}, Engines: ${data.summary.engines_used.join(",")}`);
  });

  // Test 4: search - with engine
  await test("POST /api/search (with engine)", async () => {
    const { status, data } = await post("/api/search", {
      query: "golang",
      engine: "bingcn",
    });
    if (status !== 200) throw new Error(`expected 200, got ${status}`);
    if (!data.summary.engines_used.includes("bingcn")) {
      throw new Error(`expected bingcn in engines_used`);
    }
  });

  // Test 5: search - max_results
  await test("POST /api/search (max_results)", async () => {
    const { status, data } = await post("/api/search", {
      query: "test",
      engine: "bingcn",
      max_results: 3,
    });
    if (status !== 200) throw new Error(`expected 200, got ${status}`);
    if (data.total_results > 3) throw new Error(`expected <=3, got ${data.total_results}`);
  });

  // Test 6: search - validation error (empty query)
  await test("POST /api/search (empty query → 400)", async () => {
    const { status, data } = await post("/api/search", { query: "" });
    if (status !== 400) throw new Error(`expected 400, got ${status}`);
    if (!data.error) throw new Error("missing 'error' field");
    console.error(`  Error: ${data.error} - ${data.message}`);
  });

  // Test 7: search - missing body (400)
  await test("POST /api/search (empty body → 400)", async () => {
    const res = await fetch(`${BASE}/api/search`, {
      method: "POST",
      headers: headers(),
      body: "{}",
    });
    const data = await res.json();
    if (res.status !== 400) throw new Error(`expected 400, got ${res.status}`);
    console.error(`  Error: ${data.error} - ${data.message}`);
  });

  // Test 8: fetch - basic URL
  await test("POST /api/fetch (basic)", async () => {
    const { status, data } = await post("/api/fetch", { url: "https://httpbin.org/json" });
    if (status !== 200) throw new Error(`expected 200, got ${status}`);
    if (!data.content) throw new Error("missing 'content'");
    console.error(`  Type: ${data.content_type}, Length: ${data.original_length}`);
  });

  // Test 9: fetch - with max_length
  await test("POST /api/fetch (max_length)", async () => {
    const { status, data } = await post("/api/fetch", {
      url: "https://httpbin.org/json",
      max_length: 50,
    });
    if (status !== 200) throw new Error(`expected 200, got ${status}`);
    console.error(`  Length: ${data.original_length}, Truncated: ${data.truncated}`);
  });

  // Test 10: fetch - raw HTML
  await test("POST /api/fetch (raw)", async () => {
    const { status, data } = await post("/api/fetch", {
      url: "https://example.com",
      raw: true,
    });
    if (status !== 200) throw new Error(`expected 200, got ${status}`);
    if (data.content_type !== "raw") throw new Error(`expected raw, got: ${data.content_type}`);
  });

  // Test 11: fetch - empty URL (validation error)
  await test("POST /api/fetch (empty URL → 400)", async () => {
    const { status, data } = await post("/api/fetch", { url: "" });
    if (status !== 400) throw new Error(`expected 400, got ${status}`);
    if (!data.error) throw new Error("missing 'error' field");
  });

  // Test 12: no Content-Type → still works (lenient)
  await test("POST /api/search (no Content-Type → accepted)", async () => {
    const res = await fetch(`${BASE}/api/search`, {
      method: "POST",
      body: JSON.stringify({ query: "test", engine: "bingcn" }),
    });
    if (res.status !== 200) throw new Error(`expected 200, got ${res.status}`);
  });

  // Test 13: wrong Content-Type → 415
  await test("POST /api/search (wrong Content-Type → 415)", async () => {
    const res = await fetch(`${BASE}/api/search`, {
      method: "POST",
      headers: { "Content-Type": "text/plain" },
      body: JSON.stringify({ query: "test" }),
    });
    if (res.status !== 415) throw new Error(`expected 415, got ${res.status}`);
  });

  // Test 14: unknown field → rejected (DisallowUnknownFields)
  await test("POST /api/search (unknown field → 400)", async () => {
    const { status } = await post("/api/search", { query: "test", bogus_field: 123 });
    if (status !== 400) throw new Error(`expected 400, got ${status}`);
  });

  // Summary
  console.warn("");
  console.warn(`=== Results: ${passed} passed, ${failed} failed ===`);
  if (failed > 0) process.exit(1);
}

main().catch(console.error);
