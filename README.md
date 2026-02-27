# Axon
### The AI-Native Browser

> *"Not a browser for humans that AI can use. A browser built for AI that humans can watch."*

---

## What Is Axon?

Axon is a ground-up rethinking of what a browser means when the **primary user is an AI agent**, not a human.

Every browser in existence today — Chrome, Firefox, Safari, Edge — was designed around one assumption: a human is sitting in front of a screen, reading, clicking, scrolling. Even "headless" browsers and automation tools like Playwright and Puppeteer are just regular browsers with the visual layer removed. They still think in terms of pixels, DOM trees, and CSS selectors — concepts that are natural for humans but deeply unnatural for language models.

Axon flips this. The primary interface is semantic, not visual. The primary consumer is a reasoning engine, not a retina.

---

## The Name

**Axon** — the long projection of a nerve cell that transmits signals from the cell body to other neurons or muscles.

In the context of AI agents, Axon is the nerve fiber connecting the agent's brain to the world of the web. It carries signals in both directions: perception inward (what's on the page?), and action outward (click, fill, navigate).

---

## The Problem Axon Solves

### Current state of browser automation for AI

When an AI agent needs to interact with a website today, it faces a stack of abstractions that were never designed for it:

```
AI Agent
   ↓
Tool Call: "click('#submit-btn')"
   ↓
Playwright / Puppeteer
   ↓
Chrome DevTools Protocol (CDP)
   ↓
Chromium rendering engine
   ↓
DOM → Visual pixels → back to DOM
```

Every layer in this stack was designed for humans. The AI agent is a second-class citizen bolted on at the top.

### The symptoms

- Agents need CSS selectors hard-coded into their prompts
- Page snapshots are massive (full HTML dumps = huge token costs)
- No native understanding of *intent* — "click the login button" requires the agent to find a selector
- No memory of past sessions without external state management
- No native error recovery — if a CAPTCHA appears, the agent crashes
- No concept of trust — agent can be hijacked by malicious page content (prompt injection via web)

---

## Documents in This Repository

| Document | Description |
|---|---|
| [RESEARCH.md](./RESEARCH.md) | Deep analysis of existing browser automation tools and their limitations |
| [COMPETITIVE_ANALYSIS.md](./COMPETITIVE_ANALYSIS.md) | How Axon compares to Playwright, Puppeteer, agent-browser, Browser-Use, etc. |
| [ARCHITECTURE.md](./ARCHITECTURE.md) | Full technical architecture of Axon |
| [FEATURES.md](./FEATURES.md) | Complete feature specification |
| [SECURITY.md](./SECURITY.md) | Security model, threat vectors, and mitigations |
| [ROADMAP.md](./ROADMAP.md) | Phased development roadmap |
| [PROPOSAL.md](./PROPOSAL.md) | Formal project proposal |
| [AGENT_INTERFACE.md](./AGENT_INTERFACE.md) | The agent-facing API design |

---

## Core Design Principles

1. **Semantic over visual** — The agent sees meaning, not markup
2. **Intent over selector** — "Click the login button" not "click `div.auth > button:nth-child(2)`"
3. **Memory-first** — Sessions, credentials, and page state persist intelligently
4. **Security by default** — Every action is audited; prompt injection is a first-class threat
5. **Minimal tokens, maximum context** — Every output is optimized for LLM consumption
6. **Failure is data** — CAPTCHAs, popups, and errors are surfaced as structured information, not crashes

---

## Quick Vision

```
Agent: "Post a tweet saying 'Hello world' on my X account"

Axon:
  → Loads x_session (cookies auto-restored)
  → Navigates to x.com/home
  → Identifies: { type: "compose_box", placeholder: "Post text", ref: "e23" }
  → Types "Hello world"
  → Identifies: { type: "submit_button", label: "Post", ref: "e31" }
  → Clicks
  → Returns: { success: true, tweet_url: "x.com/status/..." }
```

No hardcoded selectors. No HTML dumps. No human in the loop.

---

*Axon — Observation Document v0.1 | February 2026*
