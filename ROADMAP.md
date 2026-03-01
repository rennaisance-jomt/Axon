# Axon — Development Roadmap

**Version:** 0.1 | **Date:** February 2026

---

## Phased Plan

```
Phase 1: Foundation     ████████░░░░░░░░░░░░   v0.1 → v1.0
Phase 2: Intelligence   ░░░░░░░░████████░░░░   v1.0 → v1.5
Phase 3: Ecosystem      ░░░░░░░░░░░░░░░░████   v1.5 → v2.0
```

---

## Phase 1: Foundation (v0.1 → v1.0)
**Goal:** Consolidate into a zero-overhead pure Go single binary using context pooling and native CDP for maximum efficiency. Make Axon the fastest semantic engine available.

| Feature | Status | Notes |
|---|---|---|
| Zero-Overhead Context Pooling | 🔲 Planned | Single daemon + incognito contexts |
| Native CDP DOM Extraction | 🔲 Planned | Fast, pierces Shadow DOM instantly |
| Native CDP Full-Page Screenshots | 🔲 Planned | Accurate, non-resizing snapshots |
| Headless-Native Network Blocking | 🔲 Planned | Drop visual assets for speed |
| Event-Driven Auto-Waiting | 🔲 Planned | Replace Sleep with CDP events |
| HTTP Control Server (localhost:8020) | 🔲 Planned | Replace agent-browser daemon |
| Session management | 🔲 Planned | Named sessions, profile loading |
| Navigate action | 🔲 Planned | `load` wait mode, not networkidle |
| Semantic snapshot | 🔲 Planned | ARIA tree → compact text |
| Click, Fill, Press actions | 🔲 Planned | Ref-based |
| Screenshot & PDF | 🔲 Planned | |
| SSRF protection | 🔲 Planned | Port to Axon from ClawSEC |
| Action reversibility classifier | 🔲 Planned | Basic rule-based |
| Audit logging | 🔲 Planned | Append-only file |
| SuperClaw Executor integration | ✅ Done | browser_tools.py exists |
| Session cookie loading | ✅ Done | x_session.json working |
| Windows native support | ✅ Done | TCP not Unix sockets |

---

## Phase 2: Intelligence (v1.0 → v1.5)
**Goal:** Make Axon genuinely understand web pages, not just parse them.

| Feature | Status | Notes |
|---|---|---|
| High-Compression Intent Graphs | 🔲 Planned | Collapse input + button nodes |
| Element intent classification | 🔲 Planned | Rule-based + embedding |
| Page state detection | 🔲 Planned | logged_in/captcha/error/etc. |
| Token-optimized snapshot format | 🔲 Planned | Target 50–500 tokens |
| Prompt injection detection | 🔲 Planned | Pattern + embedding |
| CAPTCHA structured detection | 🔲 Planned | Return type, not crash |
| Cross-session element memory | 🔲 Planned | SQLite or Redis |
| Intent-based element resolution | 🔲 Planned | "find the search box" |
| LangChain ToolKit | 🔲 Planned | |
| Auto-retry with backoff | 🔲 Planned | |
| Structured error objects | 🔲 Planned | No more raw exceptions |

---

## Phase 3: Ecosystem (v1.5 → v2.0)
**Goal:** Make Axon the standard browser tool for open-source AI agent frameworks.

| Feature | Status | Notes |
|---|---|---|
| MCP Server | 🔲 Planned | Any MCP client can use Axon |
| Axon Studio (debug dashboard) | 🔲 Planned | Local web UI |
| CLI (`axon snapshot`, `axon act`) | 🔲 Planned | |
| Python SDK (`pip install axon`) | 🔲 Planned | |
| Node.js SDK (`npm install axon`) | 🔲 Planned | |
| Public documentation site | 🔲 Planned | |
| Firefox support | 🔲 Planned | |
| Docker image | 🔲 Planned | Cloud-ready |
| Action recording & replay | 🔲 Planned | |
| GitHub Actions integration | 🔲 Planned | CI browser testing |

---

## Immediate Next Steps (The Path to Ultimate Performance)

1. **Zero-Overhead Context Pooling** — Refactor `pool.go` to use a single background Chromium daemon and microscopic isolated Incognito contexts (15ms boot, <10MB RAM).
2. **Native CDP DOM Extraction** — Rip out JS `TreeWalker` in `snapshot.go` and replace with Chromium's native C++ `Accessibility` protocol domain for instant, shadow-DOM piercing extraction.
3. **High-Compression Intent Graphs** — Upgrade snapshot logic to collapse related elements (e.g., input + search button) into single semantic nodes, slashing token costs.
4. **Headless-Native Network Blocking** — Add a strict network interceptor in `go-rod` to drop visual assets (fonts, images, trackers) for blazing fast load times.
5. **Event-Driven Auto-Waiting** — Implement native CDP event listeners (`DOMNodeInserted`, `AnimationCanceled`) to remove flaky `time.Sleep` and wait exactly the right amount of time.
---

*Axon Roadmap v0.1 | February 2026*
