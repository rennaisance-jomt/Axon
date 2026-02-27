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
**Goal:** Replace the broken Rust CLI approach with a solid Python HTTP server. Make Axon functional for SuperClaw agents today.

| Feature | Status | Notes |
|---|---|---|
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

## Immediate Next Steps (This Week)

1. **Build Axon Control Server** — FastAPI server on port 8020 with Session Manager
2. **Port snapshot logic** — from agent-browser TCP protocol to Playwright native Python
3. **Implement intent classifier** — start with rule-based, 50 most common patterns
4. **Write Axon client** — replace `browser_engine.py` with a proper SDK class
5. **Integration test** — SuperClaw agent posting to X.com end-to-end using Axon

---

*Axon Roadmap v0.1 | February 2026*
