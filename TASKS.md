# Axon — Task Tracker
## Current Focus: Phase 1 Re-architecture (Ultimate Performance)

**Last Updated:** February 2026  
**Status:** ✅ COMPLETELY VERIFIED & SHIPPED

---

## 🏃‍♂️ Sprint 1: Zero-Overhead Context Pooling
**Goal:** Drastically reduce memory footprint and session boot time.
- [x] **T1.1** Refactor `pool.go` to launch and maintain exactly ONE background Chromium daemon.
- [x] **T1.2** Rewrite `session.go` to generate isolated `Incognito` contexts instead of full browsers.
- [x] **T1.3** Ensure robust cleanup of contexts when a session is closed.
- [x] 🧪 **VERIFICATION:** Run test scripts and assert session creation time < 50ms and RAM footprint < 20MB per active context.

---

## 🏃‍♂️ Sprint 2: Native CDP DOM Extraction
**Goal:** Extract perfect ARIA accessibility trees without fragile JS injection.
- [x] **T2.1** Remove the JavaScript `TreeWalker` blob from `snapshot.go`.
- [x] **T2.2** Connect directly to Chromium's native C++ `Accessibility` protocol domain using CDP.
- [x] **T2.3** Refactor screenshot logic to use native `Page.captureScreenshot` (no resizing).
- [x] 🧪 **VERIFICATION:** Extract a snapshot from a complex page utilizing Shadow DOMs; verify instant extraction without JS evaluation.

---

## 🏃‍♂️ Sprint 3: High-Compression Intent Graphs
**Goal:** Radically compress LLM token usage.
- [x] **T3.1** Upgrade `snapshot.go` to detect spatial/functional relationships (e.g., grouping a text input with its adjacent search button).
- [x] **T3.2** Collapse grouped elements into single semantic nodes in the API payload.
- [x] 🧪 **VERIFICATION:** Compare token counts of standard HTML, old snapshot logic, and new Intent Graphs. Ensure a >50% token reduction.

---

## 🏃‍♂️ Sprint 4: Headless-Native Network Blocking
**Goal:** Eliminate visual noise and slash page load latency.
- [x] **T4.1** Implement strict network request interception in `go-rod`.
- [x] **T4.2** Create a blocklist dropping `.woff2`, images, media, analytics endpoints, and heavy CSS.
- [x] 🧪 **VERIFICATION:** Load a heavy website (e.g., a major news portal) and verify load time is reduced by at least 70% with zero visual assets loaded.

---

## 🏃‍♂️ Sprint 5: Event-Driven Auto-Waiting
**Goal:** Banish flakiness and `time.Sleep` commands forever.
- [x] **T5.1** Rip out hardcoded timeouts and implicit `networkidle` waits in `actions.go`.
- [x] **T5.2** Wire Axon to listen to raw CDP `DOMNodeInserted` and `AnimationCanceled` events.
- [x] **T5.3** Ensure clicks only fire when the C++ layer confirms the element is visible and still.
- [x] 🧪 **VERIFICATION:** Run aggressive deterministic tests on a dynamic SPA (React/Vue). Assert zero race conditions or missed clicks.
- [x] 🏁 **PHASE 1 INTEGRATION VERIFICATION:** Run an end-to-end multi-agent session using all Sprint features concurrently to ensure total system stability.

---

## Legacy Phase 1 Foundation (Completed)

### Infrastructure

- [x] **T1.1** Initialize Go module: `go mod init github.com/rennaisance-jomt/axon`
- [x] **T1.2** Create project directory structure (cmd, internal, pkg, api, configs, test)
- [x] **T1.3** Set up Makefile with build, test, lint targets
- [x] **T1.4** Configure golangci-lint and gofmt
- [x] **T1.5** Create .gitignore (binaries, data, env files)

---

## Core Server (Layer 2)

### HTTP Server

- [x] **T2.1** Implement Fiber server on port 8020
- [x] **T2.2** Add health check endpoint GET /health
- [x] **T2.3** Add request logging middleware
- [x] **T2.4** Add recovery middleware (panic handler)
- [x] **T2.5** Configure graceful shutdown

### Configuration

- [x] **T2.6** Implement Viper config loading
- [x] **T2.7** Support config file (YAML), env vars, flags
- [x] **T2.8** Add config validation

---

## Browser Runtime (Layer 1)

### Rod Integration

- [x] **T3.1** Implement browser pool manager
- [x] **T3.2** Add automatic Chromium download/install (rod lib)
- [x] **T3.3** Implement browser context creation
- [x] **T3.4** Add browser cleanup on shutdown

### Navigation

- [x] **T3.5** Implement navigate action (POST /navigate)
- [x] **T3.6** Add wait conditions (load, domcontentloaded, networkidle)
- [x] **T3.7** Implement back/forward/reload actions

### Actions

- [x] **T3.8** Implement click action
- [x] **T3.9** Implement fill action (type text)
- [x] **T3.10** Implement press action (key combinations)
- [x] **T3.11** Implement select action (dropdowns)
- [x] **T3.12** Implement hover action
- [x] **T3.13** Implement scroll action

### Screenshots & PDF

- [x] **T3.14** Implement screenshot (full page and element)
- [x] **T3.15** Implement PDF export

---

## Session Management

### Session API

- [x] **T4.1** Implement session creation POST /sessions
- [x] **T4.2** Implement session listing GET /sessions
- [x] **T4.3** Implement session retrieval GET /sessions/{id}
- [x] **T4.4** Implement session deletion DELETE /sessions/{id}
- [x] **T4.5** Add session status tracking (created, active, idle, closed)

### Profile System

- [x] **T4.6** Implement profile loading (Playwright JSON format)
- [x] **T4.7** Implement cookie management
- [x] **T4.8** Implement cookie export

---

## Security Layer (Layer 3)

### SSRF Protection

- [x] **T5.1** Implement URL validation
- [x] **T5.2** Block private IP ranges (10.x, 172.16.x, 192.168.x, 127.x)
- [x] **T5.3** Block dangerous schemes (file://, javascript:, data:)
- [x] **T5.4** Add DNS rebinding protection
- [x] **T5.5** Implement domain allowlist/denylist

### Action Classification

- [x] **T5.6** Implement reversibility classifier (read, write_reversible, write_irreversible)
- [x] **T5.7** Add confirm flag requirement for irreversible actions
- [x] **T5.8** Mark sensitive fields (password, credit card)

### Audit Logging

- [x] **T5.9** Implement audit log storage (BadgerDB)
- [x] **T5.10** Add chain hashing (prev_hash)
- [x] **T5.11** Implement audit log retrieval API
- [x] **T5.12** Add agent ID tracking

---

## Intelligence Layer (Layer 4)

### Snapshot System

- [x] **T6.1** Implement ARIA tree extraction
- [x] **T6.2** Implement compact snapshot format
- [x] **T6.3** Add depth levels (compact, standard, full)
- [x] **T6.4** Implement scoped snapshots (focus selector)

### State Detection

- [x] **T6.5** Implement logged_in/logged_out detection
- [x] **T6.6** Implement loading/ready/error detection
- [x] **T6.7** Implement rate limit detection
- [x] **T6.8** Add page state to snapshot response

### Element Reference

- [x] **T6.9** Implement element ref generation (e1, a1, n1)
- [x] **T6.10** Add element type classification
- [x] **T6.11** Add visible/enabled attributes

---

## Agent Interface (Layer 5)

### Tool API

- [x] **T7.1** Implement axon_navigate tool
- [x] **T7.2** Implement axon_snapshot tool
- [x] **T7.3** Implement axon_act tool
- [x] **T7.4** Implement axon_status tool
- [x] **T7.5** Implement axon_screenshot tool

### Error Handling

- [x] **T7.6** Implement structured error responses
- [x] **T7.7** Add error codes (element_not_found, navigation_failed, etc.)
- [x] **T7.8** Add suggestion field for recovery

### Wait Conditions

- [x] **T7.9** Implement wait for load
- [x] **T7.10** Implement wait for network idle
- [x] **T7.11** Implement wait for selector
- [x] **T7.12** Implement wait for text

---

## Testing

### Unit Tests

- [x] **T8.1** Write tests for SSRF protection
- [x] **T8.2** Write tests for action classification
- [x] **T8.3** Write tests for config loading
- [x] **T8.4** Write tests for session management

### Integration Tests

- [x] **T8.5** Write test for navigate → snapshot → act workflow
- [x] **T8.6** Write test for session persistence
- [x] **T8.7** Write test for profile loading

---

## Documentation

- [x] **T9.1** Update README with build instructions
- [x] **T9.2** Create examples/ directory with sample code
- [x] **T9.3** Add API usage examples to API_SPEC.md

---

## Phase 1 Complete Checklist

```
[x] All 51 tasks completed
[x] HTTP server running on port 8020
[x] Browser automation working (navigate, click, fill, screenshot)
[x] Session management functional
[x] Security layer (SSRF, reversibility, audit)
[x] Semantic snapshots working
[x] All tests passing
[x] v1.0 release ready
```

---

## Phase 2: Intelligence (v1.0 → v1.5)

*To be planned after Phase 1 completion*

---

## Phase 3: Ecosystem (v1.5 → v2.0)

*To be planned after Phase 1 completion*

---

*Axon Task Tracker v1.0 | February 2026*
