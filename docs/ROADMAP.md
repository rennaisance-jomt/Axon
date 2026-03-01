# Axon — Development Roadmap

**Version:** 0.2 | **Date:** March 2026

---

## Phased Plan

```
Phase 1: Foundation     ████████████████████   v0.1 → v1.0 (COMPLETED)
Phase 2: Intelligence   ████████████████████   v1.0 → v1.5 (COMPLETED)
Phase 3: Performance    ░░░░░░░░░░░░░░░░░░░░   v1.5 → v1.8 (PLANNED)
Phase 4: Ecosystem      ░░░░░░░░░░░░░░░░░░░░   v1.8 → v2.0 (PLANNED)
```

---

## Phase 1: Foundation (v0.1 → v1.0)
**Goal:** Consolidate into a zero-overhead pure Go single binary using context pooling and native CDP for maximum efficiency. Make Axon the fastest semantic engine available.

| Feature | Status | Notes |
|---|---|---|
| Zero-Overhead Context Pooling | ✅ Done | Single daemon + incognito contexts |
| Native CDP DOM Extraction | ✅ Done | Fast, pierces Shadow DOM instantly |
| Native CDP Full-Page Screenshots | ✅ Done | Accurate, native capture beyond viewport |
| Headless-Native Network Blocking | ✅ Done | Drop visual assets for speed (70%+ latency reduction) |
| Event-Driven Auto-Waiting | ✅ Done | Replace Sleep with CDP layout/animation events |
| High-Compression Intent Graphs | ✅ Done | Collapse input + button nodes (98% token reduction) |
| HTTP Control Server (8020) | ✅ Done | Performance-first Fiber API layer |
| Session management | ✅ Done | Named sessions, profile loading, browser pooling |
| Navigate, Click, Fill, Press | ✅ Done | Wrapped native CDP actions for external API |
| SSRF protection | ✅ Done | Multi-tier defense layer (internal/security) |
| Action Reversibility Classifier | ✅ Done | Rule-based classification (internal/security) |
| Cryptographic Audit Logging | ✅ Done | SHA-256 chained logs with BadgerDB storage |
| SuperClaw Executor integration | ✅ Done | browser_tools.py exists |
| Windows native support | ✅ Done | TCP not Unix sockets |

---

## Phase 2: Intelligence & Integration (v1.0 → v1.5) ✅ COMPLETED
**Goal:** Connect the high-performance Axon engine seamlessly to agent frameworks via standard protocols and intelligent interpreters.

| Feature | Status | Notes |
|---|---|---|
| Model Context Protocol (MCP) Bridge | ✅ Done | Standardizing Axon for Claude / Agents |
| Intent-to-Action Translator | ✅ Done | Mapping LLM intent strings to Axon execution |
| Page state detection | ✅ Done | logged_in / captcha / error / etc. |
| Prompt injection detection | ✅ Done | Pattern + embedding (internal/security) |
| CAPTCHA structured detection | ✅ Done | Return type, not crash |
| Cross-session element memory | ✅ Done | SQLite or Redis for learned selectors |
| Intent-based element resolution | ✅ Done | "find the search box" via semantic matching |
| LangChain ToolKit | ✅ Done | Native LangChain integration |
| Auto-retry with backoff | ✅ Done | Intelligent retry logic |
| Structured error objects | ✅ Done | No more raw exceptions (pkg/types) |
| Real-time Stats Dashboard | ✅ Done | Visualization of viral metrics |
| BackendNodeID Resolution | ✅ Done | Zero-flakiness element resolution |
| Semantic Element Resolution | ✅ Done | X.com demo validated |

---

## Phase 3: Performance & Reliability (v1.5 → v1.8)
**Goal:** Transform Axon from a "controllable browser" to a "high-fidelity sensory system for AI" with advanced resilience and perception capabilities.

### 3.1 Architectural Resilience
| Feature | Status | Notes |
|---|---|---|
| Managed Worker Pool | 🔲 Planned | Multi-browser pool with automated rotation |
| Lifecycle Management | 🔲 Planned | MaxSessionLife + MaxMemoryThreshold |
| Process Health Monitoring | 🔲 Planned | Automatic process rotation on hang/crash |
| Resource Leak Prevention | 🔲 Planned | Zombie process cleanup |

### 3.2 Advanced Intelligence (Zero-Token Perception)
| Feature | Status | Notes |
|---|---|---|
| Vectorized Spatial Snapshots | 🔲 Planned | Spatial Map JSON encoding coordinates, Z-index, colors |
| Vision-AX Alignment | 🔲 Planned | Match spatial data with AXTree for 100x token reduction |
| Self-Healing Semantic Locators | 🔲 Planned | Multi-anchor identification (semantic + visual + spatial) |
| Visual DNA Tracking | 🔲 Planned | Color, icon SVG path, size anchors |
| Intent Graph Enhancement | 🔲 Planned | Collapse semantic relationships further |

### 3.3 Reliability & Recovery
| Feature | Status | Notes |
|---|---|---|
| Session-Level Checkpointing | 🔲 Planned | Save DOM + JS Memory + Cookies + Storage |
| Delta Rollback ("Time Machine") | 🔲 Planned | Rollback to millisecond before irreversible actions |
| Autonomous Recovery | 🔲 Planned | Auto-retry different paths on failure detection |
| Action Dead-End Detection | 🔲 Planned | Detect and recover from agent dead-ends |

### 3.4 Security & Safety Hardening
| Feature | Status | Notes |
|---|---|---|
| Local Model Guardrails | 🔲 Planned | Integrate Llama-Guard for AXTree manipulation scanning |
| SSRF Hardening | 🔲 Planned | Auto-block internal cloud metadata endpoints |
| Proactive Defense | 🔲 Planned | Real-time prompt injection detection in page content |
| Request Interception | 🔲 Planned | Block malicious internal network requests |

### 3.5 Performance Optimization
| Feature | Status | Notes |
|---|---|---|
| Semantic Proxy Filtering | 🔲 Planned | Intent-driven blocking at CDP protocol level |
| Header Spoofing for Anti-Bot | 🔲 Planned | Remain invisible to Cloudflare while filtering |
| 4x Page Load Improvement | 🔲 Target | Strip non-semantic traffic |

### 3.6 Observability & DevEx
| Feature | Status | Notes |
|---|---|---|
| OpenTelemetry Standard Export | 🔲 Planned | Every action/snapshot as trace span |
| Vision Overlay API | 🔲 Planned | Real-time web UI with semantic overlays |
| Agent Vision Debugger | 🔲 Planned | Show semantic refs, intent classifications, action paths |
| Live Browser View | 🔲 Planned | Overlay Axon's internal thoughts on browser |

---

## Phase 4: Ecosystem (v1.8 → v2.0)
**Goal:** Make Axon the standard browser tool for open-source AI agent frameworks with full SDK support and cloud readiness.

| Feature | Status | Notes |
|---|---|---|
| MCP Server | 🔲 Planned | Any MCP client can use Axon |
| Axon Studio (debug dashboard) | 🔲 Planned | Local web UI with full debugging |
| CLI (`axon snapshot`, `axon act`) | 🔲 Planned | Command-line interface |
| Python SDK (`pip install axon`) | 🔲 Planned | Full Python client library |
| Node.js SDK (`npm install axon`) | 🔲 Planned | Full Node.js client library |
| Public documentation site | 🔲 Planned | Hosted docs with examples |
| Firefox support | 🔲 Planned | Gecko engine integration |
| Docker image | 🔲 Planned | Cloud-ready container |
| Action recording & replay | 🔲 Planned | Record and replay agent sessions |
| GitHub Actions integration | 🔲 Planned | CI browser testing support |
| Kubernetes Operator | 🔲 Planned | Scale Axon in cloud environments |
| Multi-tenancy Support | 🔲 Planned | Isolate sessions per organization |

---

## The North Star: Goal Completion Efficiency (GCE)

**Target Metric:**
```
GCE = (Goals Accomplished) / (Total Token Spend + Total Execution Latency)
```

**Target:** 10x GCE advantage over competitors (Stagehand, Vercel Agent Browser, Browserbase)

| Metric | Current | Target (v2.0) |
|---|---|---|
| Tokens per page view | 500-5,000 | 50-500 |
| Latency per action | 200-500ms | 80-200ms |
| Session startup | ~2,000ms | ~15ms |
| Memory per session | ~100MB | <10MB |
| Vision model dependency | Required | Optional (Spatial Maps) |
| Success rate | 85% | 95%+ with auto-recovery |

---

## Immediate Next Steps (Phase 2 Integration)

1. **The Execution Server** — Build the robust local Fiber/WebSocket server that exposes Axon's internal `session.go` and `actions.go` capabilities securely via API.
2. **The MCP Bridge** — Implement the Model Context Protocol (MCP) server layer so Axon can be natively connected to any LLM Agent or Claude Desktop.
3. **Agent Action Translators** — Build the middleware that converts LLM intentions (e.g., `{"action": "click", "ref": "b4"}`) directly into Axon's new rock-solid `element.MustWaitVisible().Click()` invocations.
4. **End-to-End Agent Test** — Run an autonomous session from prompt to completed task.

---

## Phase 3 Preparation Steps

1. **Architectural Audit** — Review `internal/browser/pool.go` for multi-browser pool architecture
2. **Spatial Map Design** — Design JSON schema for Vectorized Spatial Snapshots
3. **Checkpoint System Design** — Plan DOM + JS Memory + Storage serialization
4. **Llama-Guard Integration** — Research ONNX/local model integration options
5. **OpenTelemetry Setup** — Define trace/span structure for Axon actions

---

*Axon Roadmap v0.2 | March 2026*
