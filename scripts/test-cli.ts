#!/usr/bin/env bun

/**
 * Webpawm CLI Test Script
 *
 * Tests the CLI search and fetch subcommands.
 *
 * Usage:
 *   bun run scripts/test-cli.ts
 *
 * Environment:
 *   WEBPAWM_BIN   Path to webpawm binary (default: webpawm in PATH)
 */

const WEBPAWM_BIN = process.env.WEBPAWM_BIN || "webpawm";

// Colors
const GREEN = "\x1b[32m";
const RED = "\x1b[31m";
const YELLOW = "\x1b[33m";
const NC = "\x1b[0m";

let passed = 0;
let failed = 0;

async function run(args: string[]): Promise<{ stdout: string; stderr: string; exitCode: number }> {
  const proc = Bun.spawn([WEBPAWM_BIN, ...args], {
    stdout: "pipe",
    stderr: "pipe",
  });
  const stdout = await new Response(proc.stdout).text();
  const stderr = await new Response(proc.stderr).text();
  const exitCode = await proc.exited;
  return { stdout, stderr, exitCode };
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
  console.warn("=== Webpawm CLI Test ===");
  console.warn(`Binary: ${WEBPAWM_BIN}`);
  console.warn("");

  // Test 1: search - basic query
  await test("search (basic query)", async () => {
    const { stdout, stderr, exitCode } = await run(["search", "hello world"]);
    if (exitCode !== 0) throw new Error(`exit code ${exitCode}: ${stderr}`);
    const data = JSON.parse(stdout);
    if (!data.results) throw new Error("missing 'results' field");
    if (!data.summary) throw new Error("missing 'summary' field");
    console.error(`  Results: ${data.total_results}, Engines: ${data.summary.engines_used.join(",")}`);
  });

  // Test 2: search - with specific engine
  await test("search (single engine)", async () => {
    const { stdout, stderr, exitCode } = await run(["search", "-e", "bingcn", "golang"]);
    if (exitCode !== 0) throw new Error(`exit code ${exitCode}: ${stderr}`);
    const data = JSON.parse(stdout);
    if (!data.summary.engines_used.includes("bingcn")) {
      throw new Error(`expected bingcn, got: ${data.summary.engines_used}`);
    }
    console.error(`  Engine: ${data.summary.engines_used[0]}, Results: ${data.total_results}`);
  });

  // Test 3: search - with max-results
  await test("search (max results)", async () => {
    const { stdout, exitCode } = await run(["search", "-e", "bingcn", "-n", "3", "test"]);
    if (exitCode !== 0) throw new Error(`exit code ${exitCode}`);
    const data = JSON.parse(stdout);
    if (data.total_results > 3) throw new Error(`expected <=3 results, got ${data.total_results}`);
  });

  // Test 4: search - deep with academic
  await test("search (deep with academic)", async () => {
    const { stdout, exitCode } = await run([
      "search", "-e", "bingcn", "-d", "deep", "--academic", "machine learning",
    ]);
    if (exitCode !== 0) throw new Error(`exit code ${exitCode}`);
    const data = JSON.parse(stdout);
    if (data.summary.search_depth !== "deep") {
      throw new Error(`expected deep, got: ${data.summary.search_depth}`);
    }
    console.error(`  Depth: ${data.summary.search_depth}, Queries: ${data.summary.search_queries.length}`);
  });

  // Test 5: search - no query (error case)
  await test("search (missing query → error)", async () => {
    const { exitCode } = await run(["search"]);
    if (exitCode === 0) throw new Error("expected non-zero exit code");
  });

  // Test 6: fetch - basic URL
  await test("fetch (basic URL)", async () => {
    const { stdout, exitCode } = await run(["fetch", "https://httpbin.org/json"]);
    if (exitCode !== 0) throw new Error(`exit code ${exitCode}`);
    const data = JSON.parse(stdout);
    if (!data.content) throw new Error("missing 'content' field");
    console.error(`  Content type: ${data.content_type}, Length: ${data.original_length}`);
  });

  // Test 7: fetch - with max-length
  await test("fetch (with max-length)", async () => {
    const { stdout, exitCode } = await run(["fetch", "-n", "50", "https://httpbin.org/json"]);
    if (exitCode !== 0) throw new Error(`exit code ${exitCode}`);
    const data = JSON.parse(stdout);
    if (data.original_length > 51) throw new Error(`expected <=51 chars, got ${data.original_length}`);
    if (!data.truncated) console.error(`  (not truncated, content was short enough)`);
    console.error(`  Length: ${data.original_length}, Truncated: ${data.truncated}`);
  });

  // Test 8: fetch - raw HTML
  await test("fetch (raw HTML)", async () => {
    const { stdout, exitCode } = await run(["fetch", "-r", "https://example.com"]);
    if (exitCode !== 0) throw new Error(`exit code ${exitCode}`);
    const data = JSON.parse(stdout);
    if (data.content_type !== "raw") throw new Error(`expected raw, got: ${data.content_type}`);
    console.error(`  Content type: ${data.content_type}`);
  });

  // Test 9: fetch - no URL (error case)
  await test("fetch (missing URL → error)", async () => {
    const { exitCode } = await run(["fetch"]);
    if (exitCode === 0) throw new Error("expected non-zero exit code");
  });

  // Test 10: fetch - blocked private IP (SSRF protection)
  await test("fetch (blocked private IP)", async () => {
    const { stdout, exitCode } = await run(["fetch", "http://127.0.0.1:9999/"]);
    // SSRF validation returns an error via stderr
    if (exitCode === 0) throw new Error("expected non-zero exit code for private IP");
    // The JSON output should contain error info OR the process should exit non-zero
    console.error(`  Correctly rejected`);
  });

  // Test 11: search - invalid depth (validation)
  await test("search (invalid depth → error)", async () => {
    const { exitCode } = await run(["search", "-d", "invalid", "test"]);
    if (exitCode === 0) throw new Error("expected non-zero exit code for invalid depth");
  });

  // Summary
  console.warn("");
  console.warn(`=== Results: ${passed} passed, ${failed} failed ===`);
  if (failed > 0) process.exit(1);
}

main().catch(console.error);
