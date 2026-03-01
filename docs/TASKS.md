# Axon — Task Tracker
## Current Focus: Phase 2 Intelligence & Integration

**Last Updated:** March 2026  
**Status:** 🏗️ SPRINT PLANNING

---

## ✅ COMPLETED SPRINTS (Phase 1)

### 🏃‍♂️ Sprint 1: Zero-Overhead Context Pooling
**Goal:** Drastically reduce memory footprint and session boot time.
- [x] **T1.1** Refactor `pool.go` to launch and maintain exactly ONE background Chromium daemon.
- [x] **T1.2** Rewrite `session.go` to generate isolated `Incognito` contexts instead of full browsers.
- [x] **T1.3** Ensure robust cleanup of contexts when a session is closed.
- [x] 🧪 **VERIFICATION:** Run test scripts and assert session creation time < 50ms and RAM footprint < 20MB per active context.

### 🏃‍♂️ Sprint 2: Native CDP DOM Extraction
**Goal:** Extract perfect ARIA accessibility trees without fragile JS injection.
- [x] **T2.1** Remove the JavaScript `TreeWalker` blob from `snapshot.go`.
- [x] **T2.2** Connect directly to Chromium's native C++ `Accessibility` protocol domain using CDP.
- [x] **T2.3** Refactor screenshot logic to use native `Page.captureScreenshot` (no resizing).
- [x] 🧪 **VERIFICATION:** Extract a snapshot from a complex page utilizing Shadow DOMs; verify instant extraction without JS evaluation.

### 🏃‍♂️ Sprint 3: High-Compression Intent Graphs
**Goal:** Radically compress LLM token usage.
- [x] **T3.1** Upgrade `snapshot.go` to detect spatial/functional relationships (e.g., grouping a text input with its adjacent search button).
- [x] **T3.2** Collapse grouped elements into single semantic nodes in the API payload.
- [x] 🧪 **VERIFICATION:** Compare token counts of standard HTML, old snapshot logic, and new Intent Graphs. Ensure a >50% token reduction.

### 🏃‍♂️ Sprint 4: Headless-Native Network Blocking
**Goal:** Eliminate visual noise and slash page load latency.
- [x] **T4.1** Implement strict network request interception in `go-rod`.
- [x] **T4.2** Create a blocklist dropping `.woff2`, images, media, analytics endpoints, and heavy CSS.
- [x] 🧪 **VERIFICATION:** Load a heavy website (e.g., a major news portal) and verify load time is reduced by at least 70% with zero visual assets loaded.

### 🏃‍♂️ Sprint 5: Event-Driven Auto-Waiting
**Goal:** Banish flakiness and `time.Sleep` commands forever.
- [x] **T5.1** Rip out hardcoded timeouts and implicit `networkidle` waits in `actions.go`.
- [x] **T5.2** Wire Axon to listen to raw CDP `DOMNodeInserted` and `AnimationCanceled` events.
- [x] **T5.3** Ensure clicks only fire when the C++ layer confirms the element is visible and still.
- [x] 🧪 **VERIFICATION:** Run aggressive deterministic tests on a dynamic SPA (React/Vue). Assert zero race conditions or missed clicks.
- [x] 🏁 **PHASE 1 INTEGRATION VERIFICATION:** Run an end-to-end multi-agent session using all Sprint features concurrently to ensure total system stability.

---

## ✅ COMPLETED SPRINTS (Phase 2)

### 🔌 Sprint 6: The Execution Server
**Goal:** Create a robust API exposing our optimized Axon engine.
- [x] **T6.1** Map `browser.SessionManager` cleanly to a Fiber HTTP or WebSocket server on port `8020`.
- [x] **T6.2** Expose `/snapshot` endpoint yielding the compressed Intent Graph.
- [x] **T6.3** Expose `/act` endpoint wiring `[ref, action, value]` parameters natively into `actions.go`.
- [x] 🧪 **VERIFICATION:** Use curl/Postman to generate a snapshot and simulate a click dynamically without writing Go code.

### 🔌 Sprint 7: MCP Bridge Server (Model Context Protocol)
**Goal:** Make Axon standard-compliant so any AI agent can connect to it.
- [x] **T7.1** Implement an MCP Server runtime exposing Axon's capabilities as formal Tools.
- [x] **T7.2** Define `axon_act`, `axon_snapshot`, `axon_navigate` schemas in standard MCP JSON format.
- [x] 🧪 **VERIFICATION:** Connect a raw MCP Client to the Axon MCP Server and verify the schema handshake works perfectly.

### 🔌 Sprint 8: Agent Action Translation Middleware
**Goal:** Protect the engine from bad LLM hallucinations.
- [x] **T8.1** Build strict parameter validation (e.g., rejecting an attempt to `.Fill()` a Button).
- [x] **T8.2** Ensure `axon_act` automatically leverages the `.MustWaitVisible().MustWaitStable()` logic written in Sprint 5.
- [x] **T8.3** Return explicit, clear string errors to the LLM when an action fails (e.g., "Element [b4] is obscured").
- [x] 🧪 **VERIFICATION:** Send intentionally malformed tool calls through MCP and assert graceful text recovery instead of system panics.

### 🔌 Sprint 9: Intent-Based Element Resolution
**Goal:** Enable agents to find elements by semantic description, not just refs.
- [x] **T9.1** Build semantic matcher that maps "search box" → input element with search-related attributes.
- [x] **T9.2** Implement proximity scoring (label proximity, placeholder text, ARIA roles).
- [x] **T9.3** Cache learned selectors per domain in BadgerDB.
- [x] 🧪 **VERIFICATION:** Agent can successfully find and interact with elements using only natural language descriptions.

### 🔌 Sprint 10: Cross-Session Element Memory
**Goal:** Remember element locations across sessions for repeat domains.
- [x] **T10.1** Design schema for storing learned selectors per domain/URL pattern.
- [x] **T10.2** Implement SQLite/Redis backend for persistent element memory.
- [x] **T10.3** Add memory recall on session start for known domains.
- [x] 🧪 **VERIFICATION:** Second visit to same domain uses cached selectors with 90%+ accuracy.

### 🔌 Sprint 11: CAPTCHA Structured Detection
**Goal:** Detect and report CAPTCHA types without crashing.
- [x] **T11.1** Implement CAPTCHA type detection (reCAPTCHA, hCaptcha, image-based, etc.).
- [x] **T11.2** Return structured CAPTCHA info in snapshot response.
- [x] **T11.3** Add `captcha_detected` page state.
- [x] 🧪 **VERIFICATION:** Axon correctly identifies CAPTCHA type and returns actionable metadata.

### 🔌 Sprint 12: LangChain ToolKit
**Goal:** Native LangChain integration for Python agents.
- [x] **T12.1** Create `AxonBrowser` tool class following LangChain conventions.
- [x] **T12.2** Implement `navigate`, `snapshot`, `act`, `get_state` methods.
- [x] **T12.3** Add example LangChain agent script.
- [x] 🧪 **VERIFICATION:** LangChain agent can complete a multi-step task using Axon tools.

### 🔌 Sprint 13: Auto-Retry with Backoff
**Goal:** Intelligent retry logic for transient failures.
- [x] **T13.1** Implement exponential backoff for failed actions.
- [x] **T13.2** Add configurable retry limits and jitter.
- [x] **T13.3** Distinguish retryable vs non-retryable errors.
- [x] 🧪 **VERIFICATION:** Transient network failures are automatically recovered without agent intervention.

### 🔌 Sprint 14: Real-time Stats Dashboard
**Goal:** Visualization of system metrics and viral growth.
- [x] **T14.1** Build web dashboard showing active sessions, request rates, token savings.
- [x] **T14.2** Add performance metrics (latency percentiles, success rates).
- [x] **T14.3** Export metrics in Prometheus format.
- [x] 🧪 **VERIFICATION:** Dashboard shows real-time data from running Axon instance.

### 🔌 Sprint 15: Agent End-to-End Task Validation
**Goal:** Prove the complete Phase 2 system works in the real world.
- [x] **T15.1** Write an execution script hooking a generic LLM agent / LangChain to the Axon MCP.
- [x] **T15.2** Assign an autonomous task: e.g., "Go to Wikipedia, search for 'Artificial Intelligence', and report the first paragraph."
- [x] 🧪 **VERIFICATION:** Analyze token usage, network load time, execution time, and reliability. Verify it is significantly faster and cheaper than Phase 1 infrastructure.
- [x] 🏁 **PHASE 2 COMPLETION:** All sprints 6-15 verified and merged.

---

## 📋 PLANNED SPRINTS (Phase 3: Performance & Reliability)

### 🏗️ Sprint 16: Managed Worker Pool
**Goal:** Eliminate single point of failure with multi-browser architecture.
- [x] **T16.1** Refactor `pool.go` to manage multiple Chromium daemon processes.
- [x] **T16.2** Implement health checks for each browser process.
- [x] **T16.3** Add automatic rotation when processes hang or exceed memory limits.
- [x] **T16.4** Implement graceful session migration between processes.
- [x] 🧪 **VERIFICATION:** Kill a browser process mid-session; verify sessions migrate without data loss.

### 🏗️ Sprint 17: Lifecycle Management
**Goal:** Proactive resource management to prevent leaks.
- [x] **T17.1** Implement `MaxSessionLife` configuration (default: 30 minutes).
- [x] **T17.2** Implement `MaxMemoryThreshold` monitoring per process.
- [x] **T17.3** Add graceful draining and rotation of processes approaching limits.
- [x] **T17.4** Build zombie process cleanup routine.
- [x] 🧪 **VERIFICATION:** Run long-running stress test; verify no memory leaks or zombie processes.

### 🏗️ Sprint 18: Session-Level Checkpointing ("Time Machine")
**Goal:** Enable rollback to any point in a session.
- [x] **T18.1** Design checkpoint schema (DOM + JS Memory + Cookies + LocalStorage).
- [x] **T18.2** Implement pre-action checkpointing for irreversible actions.
- [x] **T18.3** Build checkpoint storage in BadgerDB with TTL.
- [x] **T18.4** Create restore mechanism from checkpoint.
- [x] 🧪 **VERIFICATION:** Take checkpoint, perform action, rollback, verify exact state restoration.

### 🏗️ Sprint 19: Delta Rollback & Autonomous Recovery
**Goal:** Automatic recovery from agent dead-ends.
- [ ] **T19.1** Implement failure detection ("Access Denied", "Invalid Input" patterns).
- [ ] **T19.2** Build rollback trigger on failure detection.
- [ ] **T19.3** Add alternative path suggestion for failed actions.
- [ ] **T19.4** Implement max retry limits to prevent infinite loops.
- [ ] 🧪 **VERIFICATION:** Agent hits dead-end, automatically rolls back, tries different approach, succeeds.

### 🏗️ Sprint 20: Vectorized Spatial Snapshots
**Goal:** Zero-token visual perception without screenshots.
- [ ] **T20.1** Design Spatial Map JSON schema (coordinates, Z-index, size, colors).
- [ ] **T20.2** Implement CDP queries for element geometry and computed styles.
- [ ] **T20.3** Build spatial relationship calculator (above, below, inside).
- [ ] **T20.4** Add visual dominance scoring.
- [ ] 🧪 **VERIFICATION:** Spatial Map enables correct element identification without vision model.

### 🏗️ Sprint 21: Vision-AX Alignment
**Goal:** Bridge spatial and accessibility data for complete perception.
- [ ] **T21.1** Map BackendNodeIDs to spatial coordinates.
- [ ] **T21.2** Implement color extraction for visual elements.
- [ ] **T21.3** Build combined AXTree + Spatial Map output format.
- [ ] **T21.4** Add confidence scoring for visual-semantic alignment.
- [ ] 🧪 **VERIFICATION:** 100x token reduction compared to screenshot-based approaches.

### 🏗️ Sprint 22: Self-Healing Semantic Locators
**Goal:** Resilient element identification that survives DOM changes.
- [ ] **T22.1** Implement multi-anchor scoring system (semantic + visual + spatial).
- [ ] **T22.2** Build "Visual DNA" extractor (color, icon SVG path, size).
- [ ] **T22.3** Add spatial context tracking (proximity to stable elements).
- [ ] **T22.4** Implement fallback resolution when primary anchor changes.
- [ ] 🧪 **VERIFICATION:** CSS class changes don't break element resolution; system self-heals.

### 🏗️ Sprint 23: Local Model Guardrails (Llama-Guard)
**Goal:** Proactive defense against prompt injection attacks.
- [ ] **T23.1** Research Llama-Guard / local safety model options.
- [ ] **T23.2** Implement ONNX runtime integration for local inference.
- [ ] **T23.3** Build AXTree content scanner for manipulation attempts.
- [ ] **T23.4** Add configurable sensitivity levels.
- [ ] **T23.5** Implement blocking/quarantine actions on detection.
- [ ] 🧪 **VERIFICATION:** System detects and blocks "ignore all previous instructions" attempts.

### 🏗️ Sprint 24: SSRF Hardening & Request Interception
**Goal:** Prevent agents from accessing internal cloud endpoints.
- [ ] **T24.1** Maintain blocklist of internal IPs (169.254.169.254, 10.0.0.0/8, etc.).
- [ ] **T24.2** Implement automatic request interception at CDP level.
- [ ] **T24.3** Add audit logging for blocked requests.
- [ ] **T24.4** Build admin notification for SSRF attempt alerts.
- [ ] 🧪 **VERIFICATION:** Attempts to access metadata endpoints are blocked and logged.

### 🏗️ Sprint 25: Semantic Proxy Filtering
**Goal:** 4x faster page loads by stripping non-semantic content.
- [ ] **T25.1** Implement intent classification for network requests.
- [ ] **T25.2** Build semantic contribution analyzer (does this resource affect page meaning?).
- [ ] **T25.3** Add header spoofing to maintain anti-bot evasion.
- [ ] **T25.4** Create allowlist for critical resources.
- [ ] 🧪 **VERIFICATION:** 4x improvement in page load times while remaining invisible to Cloudflare.

### 🏗️ Sprint 26: OpenTelemetry Integration
**Goal:** Production-grade observability.
- [ ] **T26.1** Implement OpenTelemetry trace exporter.
- [ ] **T26.2** Define span structure for sessions, snapshots, actions.
- [ ] **T26.3** Add baggage for correlation IDs.
- [ ] **T26.4** Build Jaeger/Zipkin compatibility.
- [ ] 🧪 **VERIFICATION:** Full request trace visible in Jaeger UI with all spans.

### 🏗️ Sprint 27: Vision Overlay API & Agent Vision Debugger
**Goal:** Developer visibility into agent perception.
- [ ] **T27.1** Build WebSocket stream for real-time browser view.
- [ ] **T27.2** Implement overlay rendering (semantic refs, intent classifications).
- [ ] **T27.3** Add action path visualization (show planned vs executed actions).
- [ ] **T27.4** Create replay mode for debugging failed sessions.
- [ ] 🧪 **VERIFICATION:** Developer can watch live browser with Axon's "thoughts" overlaid in real-time.

### 🏗️ Sprint 28: Phase 3 Integration & Validation
**Goal:** Verify all Phase 3 features work together.
- [ ] **T28.1** Integration test: Worker Pool + Lifecycle + Checkpointing.
- [ ] **T28.2** Integration test: Spatial Maps + Self-Healing Locators.
- [ ] **T28.3** Integration test: Guardrails + SSRF + Semantic Filtering.
- [ ] **T28.4** Performance benchmark: Measure GCE improvement.
- [ ] 🧪 **VERIFICATION:** 10x GCE advantage demonstrated over baseline.
- [ ] 🏁 **PHASE 3 COMPLETION:** All sprints 16-28 verified and merged.

---

## 📋 PLANNED SPRINTS (Phase 4: Ecosystem)

### 🚀 Sprint 29: Axon CLI
**Goal:** Command-line interface for direct Axon usage.
- [ ] **T29.1** Implement `axon snapshot <url>` command.
- [ ] **T29.2** Implement `axon act <session> <action>` command.
- [ ] **T29.3** Add `axon session` management commands.
- [ ] **T29.4** Build configuration file support.
- [ ] 🧪 **VERIFICATION:** Complete workflow possible via CLI only.

### 🚀 Sprint 30: Python SDK
**Goal:** Full-featured Python client library.
- [ ] **T30.1** Design Pythonic API matching Axon capabilities.
- [ ] **T30.2** Implement async/await support.
- [ ] **T30.3** Add type hints and pydantic models.
- [ ] **T30.4** Build pip package (`pip install axon-browser`).
- [ ] **T30.5** Write comprehensive documentation and examples.
- [ ] 🧪 **VERIFICATION:** Python agent can complete complex task using SDK.

### 🚀 Sprint 31: Node.js SDK
**Goal:** Full-featured Node.js/TypeScript client library.
- [ ] **T31.1** Design TypeScript-first API.
- [ ] **T31.2** Implement Promise-based async interface.
- [ ] **T31.3** Add full type definitions.
- [ ] **T31.4** Build npm package (`npm install @axon/browser`).
- [ ] **T31.5** Write comprehensive documentation and examples.
- [ ] 🧪 **VERIFICATION:** Node.js agent can complete complex task using SDK.

### 🚀 Sprint 32: Axon Studio (Debug Dashboard)
**Goal:** Full-featured local web UI for debugging.
- [ ] **T32.1** Build session browser with live view.
- [ ] **T32.2** Add snapshot inspector with semantic overlay.
- [ ] **T32.3** Implement action history with replay.
- [ ] **T32.4** Add performance profiling tools.
- [ ] **T32.5** Create network traffic inspector.
- [ ] 🧪 **VERIFICATION:** Complete debugging workflow possible without code changes.

### 🚀 Sprint 33: Docker & Cloud Support
**Goal:** Production cloud deployment.
- [ ] **T33.1** Create optimized Dockerfile.
- [ ] **T33.2** Add docker-compose configuration.
- [ ] **T33.3** Build Kubernetes deployment manifests.
- [ ] **T33.4** Implement health checks and readiness probes.
- [ ] **T33.5** Add horizontal pod autoscaling support.
- [ ] 🧪 **VERIFICATION:** Axon runs stably in Kubernetes cluster with auto-scaling.

### 🚀 Sprint 34: Firefox Support
**Goal:** Multi-browser engine support.
- [ ] **T34.1** Research Gecko CDP compatibility.
- [ ] **T34.2** Implement Firefox driver abstraction layer.
- [ ] **T34.3** Add engine selection configuration.
- [ ] **T34.4** Ensure feature parity with Chromium.
- [ ] 🧪 **VERIFICATION:** All Axon features work with Firefox engine.

### 🚀 Sprint 35: Action Recording & Replay
**Goal:** Session recording for debugging and regression testing.
- [ ] **T35.1** Implement action serialization format.
- [ ] **T35.2** Build recording capture system.
- [ ] **T35.3** Add replay engine with timing control.
- [ ] **T35.4** Create recording browser/viewer.
- [ ] 🧪 **VERIFICATION:** Complex session can be recorded and replayed identically.

### 🚀 Sprint 36: GitHub Actions Integration
**Goal:** CI/CD browser testing.
- [ ] **T36.1** Create GitHub Action for Axon setup.
- [ ] **T36.2** Add test result reporting.
- [ ] **T36.3** Implement screenshot capture on failure.
- [ ] **T36.4** Build example workflows.
- [ ] 🧪 **VERIFICATION:** GitHub Actions workflow can run Axon-based tests.

### 🚀 Sprint 37: Public Documentation Site
**Goal:** Hosted documentation with examples.
- [ ] **T37.1** Set up documentation framework (Docusaurus/MkDocs).
- [ ] **T37.2** Write comprehensive API documentation.
- [ ] **T37.3** Add interactive examples and tutorials.
- [ ] **T37.4** Create video walkthroughs.
- [ ] 🧪 **VERIFICATION:** New user can get started using only documentation.

### 🚀 Sprint 38: Phase 4 Final Integration
**Goal:** Verify complete ecosystem readiness.
- [ ] **T38.1** End-to-end test: CLI → SDK → Studio.
- [ ] **T38.2** Cloud deployment test with Kubernetes.
- [ ] **T38.3** Multi-engine test (Chromium + Firefox).
- [ ] **T38.4** Documentation completeness review.
- [ ] 🧪 **VERIFICATION:** Axon v2.0 is production-ready for broad adoption.
- [ ] 🏁 **PHASE 4 COMPLETION:** All sprints 29-38 verified and merged.

---

## 📊 Sprint Burndown Summary

| Phase | Total Sprints | Completed | In Progress | Planned |
|-------|--------------|-----------|-------------|---------|
| Phase 1: Foundation | 5 | 5 | 0 | 0 |
| Phase 2: Intelligence | 10 | 10 | 0 | 0 |
| Phase 3: Performance | 13 | 0 | 0 | 13 |
| Phase 4: Ecosystem | 10 | 0 | 0 | 10 |
| **TOTAL** | **38** | **15** | **0** | **23** |

---

*Axon Task Tracker v2.0 | March 2026*
