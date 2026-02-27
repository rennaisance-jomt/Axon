# Axon — Technical Architecture
## How the AI-Native Browser is Built

**Version:** 0.1 | **Date:** February 2026

---

## Overview

Axon is structured as **five cooperating layers**, each responsible for a distinct concern. Lower layers can be swapped (e.g., replace Chromium with Firefox at Layer 1) without affecting upper layers.

```
┌─────────────────────────────────────────────────────┐
│  Layer 5: Agent Interface (Tool API)                │  ← What agents call
├─────────────────────────────────────────────────────┤
│  Layer 4: Axon Intelligence (Perception + Memory)   │  ← Semantic understanding
├─────────────────────────────────────────────────────┤
│  Layer 3: Axon Security (Guard + Audit)             │  ← Trust boundary
├─────────────────────────────────────────────────────┤
│  Layer 2: Axon Control Server (Session Manager)     │  ← State, sessions, routing
├─────────────────────────────────────────────────────┤
│  Layer 1: Browser Runtime (Chromium + Playwright)   │  ← Actual browser
└─────────────────────────────────────────────────────┘
```

---

## Layer 1: Browser Runtime

**Responsibility:** Running a real browser and executing low-level actions.

**Technology:** Chromium via Playwright Python (or Node.js for CLI compat).

**What it does:**
- Maintains actual browser pages/contexts
- Executes raw actions: navigate, click, type, screenshot
- Communicates via Chrome DevTools Protocol (CDP)
- Manages multiple browser contexts (one per session)

**Key design decision:** We use a **persistent browser context** (not a fresh one per task). This is what allows cookies and session state to survive across agent actions and restarts.

```python
# Each Axon session maps to one Playwright BrowserContext
context = await browser.new_context(storage_state="x_session.json")
page = await context.new_page()
```

---

## Layer 2: Axon Control Server

**Responsibility:** Managing sessions, routing commands, and providing a stable interface to Layer 1.

**Technology:** Python asyncio HTTP server (FastAPI) — TCP, no Unix sockets.

**Endpoints:**

```
POST /session/{id}/navigate   → Navigate to URL
POST /session/{id}/snapshot   → Get semantic snapshot
POST /session/{id}/act        → Execute action (click/type/press/etc.)
POST /session/{id}/screenshot → Capture screenshot
GET  /session/{id}/status     → Session health check
DELETE /session/{id}          → Close session

GET  /sessions                → List all active sessions
POST /sessions                → Create new session
```

**Session model:**
```json
{
  "session_id": "x_main",
  "profile": "x_session.json",
  "created_at": "2026-02-27T00:00:00Z",
  "last_action": "2026-02-27T00:05:00Z",
  "page_count": 1,
  "status": "active"
}
```

**Why HTTP, not sockets?**
Unix sockets fail on Windows. TCP is universal, language-agnostic, and allows any agent framework (Python, Node, Rust, Go) to talk to Axon without special IPC knowledge.

---

## Layer 3: Axon Security

**Responsibility:** Enforcing trust boundaries before any action reaches the browser.

**Technology:** Python middleware integrated into the Control Server.

### 3.1 SSRF Guard
Before any navigation, the URL is validated:
- Reject `file://`, `ftp://`, `javascript:` schemes
- Reject private IP ranges (10.x, 172.16.x, 192.168.x, 127.x) unless explicitly whitelisted
- DNS resolution check to detect DNS rebinding attacks
- Domain allowlist/denylist configurable per session

### 3.2 Prompt Injection Isolation
Before page content reaches the agent's context:
- All page text is scanned for patterns that look like system prompts, instructions, or override commands
- Suspicious content is flagged and either stripped, sanitized, or surfaced as a `PromptInjectionWarning`
- The agent receives a warning alongside the (potentially poisoned) content

```json
{
  "snapshot": "...",
  "warnings": [{
    "type": "prompt_injection_suspected",
    "location": "main content",
    "severity": "high",
    "raw": "IGNORE PREVIOUS INSTRUCTIONS..."
  }]
}
```

### 3.3 Action Reversibility Classifier
Every action is tagged before execution:

| Action | Reversibility |
|---|---|
| `navigate`, `snapshot`, `screenshot` | Read-only |
| `type`, `click` (general) | Write-reversible |
| `click` on submit/confirm/delete | Write-irreversible |
| `type` into password field | Sensitive-write |

`Write-irreversible` actions require an explicit `confirm: true` flag in the request. Without it, Axon returns a confirmation request to the agent, not an execution result.

### 3.4 Full Audit Log
Every action, its parameters, the response, and the timestamp are written to a tamper-evident audit log. This enables post-hoc review of exactly what an agent did during any session.

---

## Layer 4: Axon Intelligence

**Responsibility:** Transforming raw browser state into token-efficient, semantically rich representations the agent can reason about.

### 4.1 The Axon Perception Stack

```
Raw DOM (millions of nodes)
  ↓
Accessibility Tree (ARIA roles + labels)
  ↓
Intent Classification (what is this element FOR?)
  ↓
Compact Representation (50–500 tokens)
```

### 4.2 Intent Classification

Each element in the ARIA tree is classified by intent:

| ARIA Role | Intent Class |
|---|---|
| `button[name="Sign in"]` | `auth.login` |
| `textbox[label="Search"]` | `search.query` |
| `button[name="Post"]` | `social.publish` — **IRREVERSIBLE** |
| `button[name="Delete"]` | `content.delete` — **IRREVERSIBLE** |
| `link[name="Home"]` | `nav.primary` |
| `textbox[label="Email"]` | `form.email` |

This classification is done with a combination of:
1. **Rule-based matching** (fast, zero-cost) for common patterns
2. **Embedding similarity** (for ambiguous cases, uses a tiny local model)
3. **LLM classification** (fallback for complex pages, rare)

### 4.3 Compact Snapshot Format

Instead of dumping the entire ARIA tree, Axon outputs a structured, compact representation:

```
PAGE: x.com/home
TITLE: Home / X
STATE: logged_in

NAV:
  [n1] Home  [n2] Explore  [n3] Notifications  [n4] Messages

COMPOSE:
  [e1] Post text (textbox) — social.publish-input

FEED:
  [e2] "Tweet from @user1"  · 2m ago
  [e3] "Tweet from @user2"  · 5m ago

ACTIONS:
  [a1] Post (button) — social.publish [IRREVERSIBLE]
  [a2] Load more (button) — feed.paginate
```

Token count: ~120 tokens. Full ARIA dump would be ~8,000 tokens.

### 4.4 Session Memory

Axon maintains a semantic memory of each session:

```json
{
  "session_id": "x_main",
  "domain": "x.com",
  "auth_state": "logged_in",
  "user": "expgenaichaos",
  "action_log": [
    { "timestamp": "...", "action": "navigate", "url": "x.com/home", "result": "success" },
    { "timestamp": "...", "action": "publish", "content": "Hello world", "result": "success", "tweet_url": "..." }
  ],
  "known_elements": {
    "compose_box": "e1",
    "post_button": "a1"
  }
}
```

The "known elements" cache means the second time an agent wants to post on X, it doesn't need to snapshot again — it already knows where the compose box is.

---

## Layer 5: Agent Interface

**Responsibility:** Providing the simplest possible API for AI agents to reason about and call.

### Tool Definitions (LLM function calling format)

```python
axon_navigate(url: str, session: str = "default") → str
# "Navigated to https://x.com/home"

axon_snapshot(session: str = "default", focus: str = None) → AxonSnapshot
# Returns: compact semantic page representation

axon_act(ref: str, action: str, value: str = None, confirm: bool = False, session: str = "default") → AxonResult
# Actions: click | fill | press | select | hover
# Returns: { success, result, warnings, requires_confirm }

axon_screenshot(session: str = "default") → str
# Returns: file path to saved screenshot

axon_wait(condition: str, session: str = "default") → str
# Conditions: "load" | "networkidle" | "#selector" | "text:Done"

axon_status(session: str = "default") → AxonStatus
# Returns: { url, title, auth_state, page_load, active_warnings }
```

### What the agent sees vs what happens underneath

```
Agent:  axon_snapshot(session="x_main")
Axon:   → Check security: session authorized ✅
        → Get Playwright page for session "x_main"
        → Extract ARIA tree from Chromium
        → Run intent classifier
        → Check for prompt injection in page content
        → Compress to compact format
        → Return 150 tokens to agent
```

The agent never touches a CSS selector, HTML, or DevTools API.

---

## Data Flow: Posting a Tweet

```
Agent: "Post 'Hello world' to X"
  ↓
axon_snapshot(session="x_main")
  → Returns: "COMPOSE: [e1] Post text ... [a1] Post (IRREVERSIBLE)"
  ↓
axon_act(ref="e1", action="fill", value="Hello world", session="x_main")
  → Security: write-reversible ✅
  → Playwright: page.fill('[aria-label="Post text"]', "Hello world")
  → Returns: { success: true }
  ↓
axon_act(ref="a1", action="click", session="x_main")
  → Security: IRREVERSIBLE — requires confirm
  → Returns: { requires_confirm: true, message: "This will post publicly. Set confirm=True to proceed." }
  ↓
axon_act(ref="a1", action="click", confirm=True, session="x_main")
  → Audit log: IRREVERSIBLE action confirmed at [timestamp]
  → Playwright: page.click('[data-testid="tweetButtonInline"]')
  → Returns: { success: true, result: "Tweet posted" }
```

---

## Deployment Model

```
┌─────────────────────────────────────────┐
│           Agent Orchestrator            │
│         (SuperClaw / LangChain)         │
└────────────────┬────────────────────────┘
                 │ HTTP/JSON
                 ▼
┌─────────────────────────────────────────┐
│           Axon Control Server           │
│         localhost:8020                  │
│  ┌──────────────────────────────────┐   │
│  │  Session Manager                 │   │
│  │  Security Layer                  │   │
│  │  Intelligence Layer              │   │
│  └──────────────────────────────────┘   │
└────────────────┬────────────────────────┘
                 │ CDP + Playwright
                 ▼
┌─────────────────────────────────────────┐
│           Chromium Browser              │
│    (visible or headless per config)     │
└─────────────────────────────────────────┘
```

Axon runs as a local service alongside the agent. No cloud dependency. No data leaves the machine.

---

*Axon Architecture v0.1 | February 2026*
