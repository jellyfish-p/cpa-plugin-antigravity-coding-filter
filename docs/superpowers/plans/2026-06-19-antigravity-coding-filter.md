# Antigravity Coding Filter Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a tested CLIProxyAPI dynamic plugin that blocks non-Antigravity coding software traffic on the Antigravity route.

**Architecture:** Put classifier and RPC envelope handling in pure Go files so unit tests do not depend on dynamic library loading. Keep the c-shared ABI entrypoint isolated in `main.go`, with all behavior delegated to testable functions.

**Tech Stack:** Go 1.26, standard library JSON handling, CLIProxyAPI v7 plugin RPC method names.

---

### Task 1: Classifier

**Files:**
- Create: `filter.go`
- Test: `filter_test.go`

- [ ] **Step 1: Write failing classifier tests**

Create tests for system keyword matching, non-system keyword pass-through, `prompt_cache_key`, `metadata.user_id`, and clean requests.

- [ ] **Step 2: Run classifier tests to verify failure**

Run: `go test ./...`
Expected: FAIL because classifier functions do not exist yet.

- [ ] **Step 3: Implement classifier**

Implement `classifyRequest(body []byte) filterDecision`, recursive JSON traversal, `system` text extraction, and case-insensitive keyword matching.

- [ ] **Step 4: Run classifier tests to verify pass**

Run: `go test ./...`
Expected: PASS for classifier tests.

### Task 2: Plugin RPC Handler

**Files:**
- Create: `plugin.go`
- Test: `plugin_test.go`

- [ ] **Step 1: Write failing RPC tests**

Test `plugin.register`, `model.route` block/pass decisions, `executor.execute`, `executor.execute_stream`, and unknown methods.

- [ ] **Step 2: Run RPC tests to verify failure**

Run: `go test ./...`
Expected: FAIL because RPC handler does not exist yet.

- [ ] **Step 3: Implement RPC handler**

Implement JSON envelopes, registration response, route response, and block responses.

- [ ] **Step 4: Run RPC tests to verify pass**

Run: `go test ./...`
Expected: PASS.

### Task 3: ABI Entrypoint And Docs

**Files:**
- Modify: `main.go`
- Create: `README.md`

- [ ] **Step 1: Add thin c-shared entrypoint**

Expose `cliproxy_plugin_init`, `call`, `free_buffer`, and `shutdown` in `main.go`. Delegate method dispatch to `handlePluginCall`.

- [ ] **Step 2: Add README**

Document build command, plugin config, and tested detection rules.

- [ ] **Step 3: Verify**

Run: `go test ./...`
Expected: PASS.

Run: `go test -run TestClassifyRequest ./...`
Expected: PASS.
