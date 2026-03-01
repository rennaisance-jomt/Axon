# Axon — Agent Interface Design
## The Tool API Built for Language Models

**Version:** 0.1 | **Date:** February 2026

---

## Design Philosophy

The Axon agent interface is designed around two constraints:

1. **Token budget is finite.** Every byte the agent receives costs money and takes up context window. Axon outputs the minimum information needed to take the next correct action.

2. **Agents can't "try again visually."** A human who misclicks can see what happened and correct. An agent has no visual loop — it needs structured, unambiguous feedback that tells it exactly what happened and what to do next.

---

## Tool Definitions (OpenAI Function Calling Format)

### `axon_navigate`
Navigate to a URL in the specified session.

```json
{
  "name": "axon_navigate",
  "description": "Navigate to a URL in the agent's browser session. Returns the final URL after any redirects.",
  "parameters": {
    "type": "object",
    "properties": {
      "url": {
        "type": "string",
        "description": "The URL to navigate to. Must be http:// or https://."
      },
      "session": {
        "type": "string",
        "description": "Browser session name. Default: 'default'",
        "default": "default"
      }
    },
    "required": ["url"]
  }
}
```

**Returns:**
```json
{ "url": "https://x.com/home", "state": "logged_in", "title": "Home / X" }
```

---

### `axon_snapshot`
Get a compact semantic view of the current page.

```json
{
  "name": "axon_snapshot",
  "description": "Get a compact semantic snapshot of the current page, including interactive elements and their ref IDs. Use ref IDs with axon_act to interact with elements.",
  "parameters": {
    "type": "object",
    "properties": {
      "session": { "type": "string", "default": "default" },
      "focus": {
        "type": "string",
        "description": "Optional CSS selector to focus snapshot on a specific section of the page."
      },
      "depth": {
        "type": "string",
        "enum": ["compact", "standard", "full"],
        "description": "How much detail to include. 'compact' is fastest and cheapest.",
        "default": "compact"
      }
    }
  }
}
```

**Returns:**
```
PAGE: x.com/home | Title: Home / X | State: logged_in

COMPOSE:
  [e1] Post text (textbox) — social.publish-input

ACTIONS:
  [a1] Post (button) — social.publish [IRREVERSIBLE, requires confirm=true]

FEED:
  [e2] Tweet by @alice · 2m
  [e3] Tweet by @bob · 5m

NAV:
  [n1] Home  [n2] Explore  [n3] Notifications  [n4] Messages
```

---

### `axon_act`
Perform an action on an element using its ref from the snapshot.

```json
{
  "name": "axon_act",
  "description": "Perform an action on a page element. Use ref IDs from axon_snapshot. Irreversible actions (marked with [IRREVERSIBLE]) require confirm=true.",
  "parameters": {
    "type": "object",
    "properties": {
      "ref": {
        "type": "string",
        "description": "Element ref from snapshot (e.g. 'e1', 'a1', 'n3')"
      },
      "action": {
        "type": "string",
        "enum": ["click", "fill", "press", "select", "hover", "scroll"],
        "description": "Action to perform"
      },
      "value": {
        "type": "string",
        "description": "For 'fill': text to type. For 'press': key name (e.g. 'Enter'). For 'select': option value."
      },
      "confirm": {
        "type": "boolean",
        "description": "Required for IRREVERSIBLE actions. Set to true only when you are certain.",
        "default": false
      },
      "session": { "type": "string", "default": "default" }
    },
    "required": ["ref", "action"]
  }
}
```

**Returns (success):**
```json
{ "success": true, "result": "Clicked: Post button" }
```

**Returns (confirmation required):**
```json
{
  "success": false,
  "requires_confirm": true,
  "message": "This action (social.publish) is irreversible. It will post publicly to X.com. Pass confirm=true to proceed.",
  "intent": "social.publish"
}
```

**Returns (error):**
```json
{
  "success": false,
  "error_type": "element_not_found",
  "message": "Element [e1] not found. The page may have changed.",
  "suggestion": "Run axon_snapshot() to get fresh element refs.",
  "recoverable": true
}
```

---

### `axon_status`
Get the current state of the browser session.

```json
{
  "name": "axon_status",
  "description": "Get the current state of the browser session including URL, auth state, and any active warnings.",
  "parameters": {
    "type": "object",
    "properties": {
      "session": { "type": "string", "default": "default" }
    }
  }
}
```

**Returns:**
```json
{
  "url": "https://x.com/home",
  "title": "Home / X",
  "auth_state": "logged_in",
  "page_state": "ready",
  "warnings": [],
  "active_session": "x_main"
}
```

---

### `axon_screenshot`
Capture a screenshot of the current page.

```json
{
  "name": "axon_screenshot",
  "description": "Take a screenshot of the current browser page and save it to the work directory.",
  "parameters": {
    "type": "object",
    "properties": {
      "session": { "type": "string", "default": "default" },
      "full_page": { "type": "boolean", "default": false }
    }
  }
}
```

**Returns:**
```json
{ "path": "C:\\SC\\screenshots\\x_main_20260227.png" }
```

---

## Typical Agent Interaction Patterns

### Pattern 1: Read a Page
```
agent → axon_navigate("https://news.ycombinator.com")
axon  → { url, state: "ready" }

agent → axon_snapshot()
axon  → compact list of stories with refs

agent → "Top story title is X" (uses snapshot to answer)
```

### Pattern 2: Fill a Form
```
agent → axon_navigate("https://github.com/login")
axon  → { url, state: "logged_out" }

agent → axon_snapshot()
axon  → "[e1] Username (textbox), [e2] Password (textbox), [a1] Sign in (button)"

agent → axon_act(ref="e1", action="fill", value="myuser")
axon  → { success: true }

agent → axon_act(ref="e2", action="fill", value="[VAULT:github_password]")
axon  → { success: true }

agent → axon_act(ref="a1", action="click")
axon  → { success: true, result: "Clicked: Sign in button" }

agent → axon_status()
axon  → { auth_state: "logged_in", url: "github.com/dashboard" }
```

### Pattern 3: Post on Social Media (Irreversible)
```
agent → axon_navigate("https://x.com/home", session="x_main")
agent → axon_snapshot(session="x_main")
axon  → "[e1] Post text (textbox), [a1] Post (button) [IRREVERSIBLE]"

agent → axon_act(ref="e1", action="fill", value="Hello from Axon!", session="x_main")
agent → axon_act(ref="a1", action="click", session="x_main")
axon  → { requires_confirm: true, message: "Irreversible: will post publicly" }

agent → [checks with orchestrator or user]
agent → axon_act(ref="a1", action="click", confirm=True, session="x_main")
axon  → { success: true, result: "Tweet posted" }
```

---

## Error Catalogue

| error_type | Meaning | Recoverable | Suggested Action |
|---|---|---|---|
| `element_not_found` | Ref no longer exists | Yes | Re-run `axon_snapshot` |
| `navigation_failed` | URL unreachable | Maybe | Check URL, try again |
| `ssrf_blocked` | URL is private network | No | Don't navigate there |
| `captcha` | CAPTCHA challenge present | Maybe | Request human help |
| `auth_required` | Session not logged in | Yes | Load credentials |
| `rate_limited` | Too many requests | Yes | Wait and retry |
| `irreversible_unconfirmed` | Needs `confirm=True` | Yes | Add confirm flag |
| `injection_warning` | Suspected prompt injection | Yes | Proceed with caution |
| `timeout` | Action took too long | Yes | Retry or wait |
| `session_not_found` | Session ID doesn't exist | Yes | Create session first |

---

*Axon Agent Interface Design v0.1 | February 2026*
