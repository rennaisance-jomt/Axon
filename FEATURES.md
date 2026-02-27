# Axon — Feature Specification
## Complete Feature Set for AI-Native Browser

**Version:** 0.1 | **Date:** February 2026

---

## Feature Groups

1. [Core Perception](#1-core-perception)
2. [Core Action](#2-core-action)
3. [Session Management](#3-session-management)
4. [Security & Trust](#4-security--trust)
5. [Intelligence & Memory](#5-intelligence--memory)
6. [Error Handling](#6-error-handling)
7. [Agent Integrations](#7-agent-integrations)
8. [Developer & Debug Tools](#8-developer--debug-tools)

---

## 1. Core Perception

### 1.1 Compact Semantic Snapshot
**Priority:** P0 (must have)

A text representation of the page designed for LLM consumption. Contains only meaningful, interactive elements labeled with stable refs.

- Default output: 50–500 tokens
- Layered depth: `compact` → `standard` → `full`
- Scoped snapshots: Focus on a specific section (`focus="#main-content"`)
- Frame support: Snapshot contents of iframes
- Automatic intent tagging per element

### 1.2 Live Element Monitoring
**Priority:** P1

Watch for specific elements to appear or disappear without polling snapshots.

```python
axon_wait(condition="text:Tweet posted", timeout=10)
axon_wait(condition="#compose-box", timeout=5)
```

### 1.3 Page State Detection
**Priority:** P0

Automatically detect high-level page states:

| State | Detection |
|---|---|
| `logged_in` | Auth cookies + user element present |
| `logged_out` | Login form visible |
| `captcha` | Known CAPTCHA patterns |
| `rate_limited` | 429 response or "too many requests" text |
| `error_page` | 4xx/5xx detected |
| `loading` | No meaningful content yet |
| `interstitial` | Cookie consent, age gate, etc. |

State is returned with every snapshot automatically.

### 1.4 Network Traffic Access
**Priority:** P2

Agent can inspect outgoing requests and responses:

```python
axon_requests(filter="api", session="default")
axon_response_body(url="**/api/timeline", session="default")
```

### 1.5 Screenshot & PDF Export
**Priority:** P0

```python
axon_screenshot(full_page=True, session="default")
axon_pdf(path="output.pdf", session="default")
axon_screenshot(ref="e3", session="default")  # element-specific
```

---

## 2. Core Action

### 2.1 Navigation
**Priority:** P0

```python
axon_navigate(url, wait_until="load")
axon_back()
axon_forward()
axon_reload()
```

### 2.2 Element Interaction
**Priority:** P0

```python
axon_act(ref, action="click")
axon_act(ref, action="fill", value="text")
axon_act(ref, action="press", value="Enter")
axon_act(ref, action="select", value="OptionA")
axon_act(ref, action="hover")
axon_act(ref, action="scroll")
axon_act(ref, action="drag", target_ref="e5")
```

### 2.3 Intent-Based Interaction
**Priority:** P1

Find and interact with elements by describing what they do, not by ref:

```python
axon_find_and_act(intent="search box", action="fill", value="OpenAI")
axon_find_and_act(intent="login button", action="click")
axon_find_and_act(intent="email input", action="fill", value="me@example.com")
```

### 2.4 Form Filling (Batch)
**Priority:** P1

Fill multiple fields in one call:

```python
axon_fill_form({
  "email": "user@example.com",
  "password": "[VAULT:x_password]",
  "remember_me": True
}, session="default")
```

### 2.5 Tab Management
**Priority:** P1

```python
axon_new_tab(url=None)
axon_list_tabs()
axon_switch_tab(index=1)
axon_close_tab(index=1)
```

### 2.6 Download & Upload
**Priority:** P2

```python
axon_click_download(ref, save_path="~/downloads/report.pdf")
axon_upload(ref, file_path="/tmp/document.pdf")
```

---

## 3. Session Management

### 3.1 Named Sessions
**Priority:** P0

Each session is a named, persistent browser context. Sessions survive process restarts.

```python
axon_session_create("x_main", profile="x_session.json")
axon_session_list()
axon_session_status("x_main")
axon_session_close("x_main")
```

### 3.2 Profile System (Auth Vault)
**Priority:** P0

Sessions can be initialized with saved authentication state:

- `x_session.json` — X.com cookies
- `gmail_session.json` — Gmail cookies  
- `github_session.json` — GitHub cookies

Profile files are encrypted at rest using the system vault key.

### 3.3 Cookie Management
**Priority:** P1

```python
axon_cookies_get(session="x_main", domain=".x.com")
axon_cookies_set(session="x_main", cookies=[...])
axon_cookies_export(session="x_main", path="backup.json")
axon_cookies_clear(session="x_main")
```

### 3.4 Session Isolation
**Priority:** P0

Each session runs in a fully isolated browser context:
- No shared cookies between sessions
- No shared local storage
- Separate network caches
- Optional: separate Chromium process per session (maximum isolation)

### 3.5 Session Sharing Between Agents
**Priority:** P2

Multiple agents can share read access to a session's snapshot:

```python
# Agent A writes
axon_act(session="x_shared", ref="e1", action="fill", value="hello")

# Agent B reads
axon_snapshot(session="x_shared")  # sees Agent A's work
```

Write access is serialized via session lock.

---

## 4. Security & Trust

### 4.1 SSRF Protection
**Priority:** P0

- Block navigation to private IP ranges
- Block `file://`, `javascript:`, `data:` URLs
- DNS rebinding prevention
- Configurable domain allowlist/denylist

### 4.2 Prompt Injection Detection
**Priority:** P0

- Scan page content before it reaches agent context
- Pattern matching + embedding-based detection
- Configurable: `warn`, `strip`, `block`
- Always logs detected attempts to audit trail

### 4.3 Action Reversibility Enforcement
**Priority:** P0

- Every write action classified as reversible or irreversible
- Irreversible actions require explicit `confirm=True`
- Double-confirmation for actions on highly sensitive domains (e.g. banking)
- Full confirmation audit trail

### 4.4 Sensitive Input Masking
**Priority:** P0

- Password fields automatically masked in logs and snapshots
- Credit card numbers, SSNs, API keys detected and redacted from memory
- Vault integration for injecting secrets without exposing in agent context

### 4.5 Audit Trail
**Priority:** P0

Every session action logged with:
- Timestamp
- Agent ID (who called it)
- Action type and parameters (secrets redacted)
- Response/result
- Chain hash (tamper-evident)

### 4.6 Content Security Policy
**Priority:** P1

- JavaScript execution optional (can be disabled per session)
- `evaluate()` always requires explicit authorization
- Page JavaScript cannot read Axon's session state or cookies

---

## 5. Intelligence & Memory

### 5.1 Element Intent Classification
**Priority:** P0

Classify every interactive element by its semantic purpose:

```
auth.login, auth.logout, auth.register
search.query, search.submit
social.publish, social.like, social.share, social.comment
nav.primary, nav.secondary, nav.breadcrumb
form.email, form.password, form.username, form.search
content.delete, content.edit, content.save
commerce.add_to_cart, commerce.checkout, commerce.payment
```

### 5.2 Cross-Session Knowledge
**Priority:** P1

Axon remembers the structure of frequently visited pages:

```json
{
  "domain": "x.com",
  "learned_elements": {
    "compose_box": { "selector": "[data-testid='tweetTextarea_0']", "intent": "social.publish-input" },
    "post_button": { "selector": "[data-testid='tweetButtonInline']", "intent": "social.publish" }
  },
  "visit_count": 47,
  "last_visited": "2026-02-27"
}
```

This eliminates the need for snapshot + element search on familiar pages.

### 5.3 Action Memory & Outcome Tracking
**Priority:** P1

Track what happened as a result of each action:

```json
{
  "action": "click",
  "ref": "a1",
  "intent": "social.publish",
  "outcome": { "success": true, "result_url": "x.com/status/123" },
  "timestamp": "..."
}
```

### 5.4 Page Change Detection
**Priority:** P2

After any action, detect what changed on the page:

```python
axon_diff()
# Returns: { added: [...], removed: [...], changed: [...] }
```

---

## 6. Error Handling

### 6.1 Structured Error Types
**Priority:** P0

Instead of crashes, every failure returns a structured object:

```python
{
  "success": False,
  "error_type": "element_not_found",  # | "navigation_failed" | "timeout" | "captcha" | "rate_limited" | "auth_required"
  "message": "Element [ref=e5] not found. Page may have changed.",
  "suggestion": "Run axon_snapshot() to get fresh element refs.",
  "recoverable": True
}
```

### 6.2 CAPTCHA Detection & Handling
**Priority:** P1

```python
{
  "success": False,
  "error_type": "captcha",
  "captcha_type": "cloudflare_turnstile",  # | "recaptcha_v2" | "funcaptcha" | "hcaptcha"
  "message": "CAPTCHA challenge detected.",
  "options": ["request_human_help", "skip_page", "retry_later"]
}
```

### 6.3 Auto-Retry with Backoff
**Priority:** P1

Configurable retry logic for transient failures (network hiccup, stale ref):

```python
axon_act(ref="e1", action="click", retry=3, retry_delay=1.0)
```

### 6.4 Page Recovery
**Priority:** P2

If a page enters an unknown state (unexpected popup, redirect, error):

```python
axon_recover(target_url="x.com/home", session="x_main")
# Attempts to navigate back to a known good state
```

---

## 7. Agent Integrations

### 7.1 SuperClaw Native Integration
**Priority:** P0

Axon is the browser backend for SuperClaw agents. All browser tool calls from SuperClaw route through Axon automatically.

### 7.2 LangChain Tool Wrapper
**Priority:** P1

```python
from axon.langchain import AxonBrowserToolkit
tools = AxonBrowserToolkit(session="default").get_tools()
agent = initialize_agent(tools, llm, ...)
```

### 7.3 OpenAI Function Calling Schema
**Priority:** P1

Axon auto-generates OpenAI-compatible function schemas for all tools.

### 7.4 MCP (Model Context Protocol) Server
**Priority:** P2

Axon exposes itself as an MCP server so any MCP-compatible agent can use it:

```json
{
  "mcpServers": {
    "axon": {
      "url": "http://localhost:8020/mcp"
    }
  }
}
```

### 7.5 REST API
**Priority:** P0

Full HTTP API for language-agnostic integrations.

---

## 8. Developer & Debug Tools

### 8.1 Axon Studio (Debug Dashboard)
**Priority:** P2

A local web UI showing:
- Active sessions and their current page
- Live snapshot view
- Action history with replay
- Security audit log
- Error log

### 8.2 CLI
**Priority:** P1

```bash
axon status
axon sessions
axon snapshot --session x_main
axon navigate x.com/home --session x_main
axon screenshot --session x_main
```

### 8.3 Playwright Trace Recording
**Priority:** P2

Record full Playwright traces for debugging:

```python
axon_trace_start(session="default")
# ... do things ...
axon_trace_stop(path="trace.zip", session="default")
```

### 8.4 Action Replay
**Priority:** P2

Record and replay sequences of actions for testing:

```python
axon_record_start(session="default")
# ... agent does things ...
sequence = axon_record_stop(session="default")
axon_replay(sequence, session="default")
```

---

## Feature Priority Summary

| Priority | Features | Phase |
|---|---|---|
| **P0** | Snapshot, Act, Navigate, Session, SSRF, Reversibility, Audit, State Detection, Error Types | v1.0 |
| **P1** | Intent-Based Interaction, Cookie Mgmt, Prompt Injection, Tab Mgmt, CAPTCHA Detection, Auto-Retry, LangChain, CLI | v1.5 |
| **P2** | Cross-Session Memory, Network Inspection, Diff, Page Recovery, MCP Server, Studio Dashboard, Replay | v2.0 |

---

*Axon Feature Specification v0.1 | February 2026*
