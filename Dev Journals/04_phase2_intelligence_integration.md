# Dev Journal: 04 - Phase 2 Intelligence & Integration Architecture

**Date:** February 28, 2026
**Focus:** Connecting the high-performance Axon Engine to Agent Frameworks

## The Objective

Phase 1 yielded an incredibly powerful, lightweight, and deterministic browser engine. However, a browser engine sitting alone on a server doesn't execute autonomous tasks. It needs to be driven by an intelligence layer. 

Phase 2 is entirely focused on **Integration & Intelligence**. Our goal is to expose Axon to Claude, LangChain, or any generic LLM agent in the safest, most standardized way possible.

---

## 1. The Execution Server (Sprint 6)

Currently, the `SessionManager` and `browser.Pool` logic exists purely as a Go library. We need to wrap this in a robust, multi-tenant server.

**Technical Plan:**
- Use **Go Fiber** or WebSockets to expose Axon's capabilities on `localhost:8020`.
- Endpoints must be strictly typed, mapped directly to `actions.go`:
  - `POST /api/v1/session/start` → Boots an incognito context.
  - `GET /api/v1/session/{id}/snapshot` → Triggers the High-Compression Intent Graph parser and returns the token-optimized representation.
  - `POST /api/v1/session/{id}/act` → Accepts a JSON payload like `{"ref": "b4", "action": "click"}` and executes it.

Why HTTP/WebSockets instead of Unix Sockets? Windows compatibility and language-agnostic bridging out of the box.

---

## 2. The MCP Bridge (Sprint 7)

Axon shouldn't require custom API clients written in Python or Node.js. It needs to natively speak the **Model Context Protocol (MCP)**. By becoming an MCP Server, any MCP-compliant agent (like Claude Desktop) immediately learns how to use Axon as a tool.

**Technical Plan:**
- Run an MCP JSON-RPC over STDIO or SSE server alongside the Fiber API.
- Define exactly three high-level Tools for the LLM:
  1. `axon_navigate(url)`
  2. `axon_snapshot()`
  3. `axon_act(ref, action, value)`
- The MCP server acts as a proxy, receiving LLM tool requests, firing them at the internal Fiber API, and returning the structured results back to the LLM.

---

## 3. Agent Action Translators (Sprint 8)

LLMs hallucinate. They will attempt to `.Fill()` a Button, or click a text node that technically exists in the DOM but cannot be interacted with.

**Technical Plan:**
- Build a middleware layer that sits between the MCP input and the raw `actions.go` executor.
- **Strict Validation:** If the LLM sends `{"action": "fill", "ref": "b3"}` where `b3` is a button, the Translator intercepts it and returns: `"Error: Cannot fill a button. Did you mean to click it?"`
- **Auto-Recovery:** Leverage the Sprint 5 `.MustWaitVisible().MustWaitStable()` methods. If an element takes longer than 10 seconds to stabilize, return a descriptive error to the LLM indicating *why* the click failed (e.g., "Element is obscured by a modal overflow"), rather than crashing the system.

---

## 4. End-to-End Agent Integration (Sprint 9)

The ultimate test of Phase 2 will be a fully autonomous scripts running alongside a standard LangChain or custom orchestrating agent.

**Technical Plan:**
- Boot the Axon Control Server + MCP Bridge.
- Prompt the Python agent: *"Go to Wikipedia, search for 'Artificial intelligence', and return the first paragraph of the main article."*
- Monitor the execution flow:
  1. Agent calls `axon_navigate()`.
  2. Agent calls `axon_snapshot()` and receives the compressed Intent Graph.
  3. Agent infers the search box ID and calls `axon_act("search_box_ref", "fill", "Artificial intelligence")`.
  4. Agent submits the form and reads the resulting snapshot.

By the end of Phase 2, Axon will be a fully production-ready, highly-integrated AI Browser tool.
