# Axon Engineering: Areas for Future Optimization & Improvement

**Date:** March 1, 2026
**Subject:** Technical Audit and Strategic Roadmap for Axon AI-Native Browser

Following a deep-dive review of the Axon codebase and a competitive analysis of the AI browser automation industry, this document outlines the critical trajectory for Axon to surpass current industry leaders (Stagehand, Browserbase, Vercel). To win, Axon must shift from being a "controllable browser" to a "high-fidelity sensory system for AI."

---

## 1. Architectural Resilience: Scaling from Daemon to Managed Fleet
**Current State:** `internal/browser/pool.go` manages a single persistent Chromium daemon via `rod`.
**The Vulnerability:** 
- **Single Point of Failure**: If the parent Chromium process hangs, all active sessions crash.
- **Resource Leakage**: Chromium accumulates memory and zombie processes over long-running sessions.
**Proposed Improvement:** 
- **Managed Worker Pool**: Transition to a multi-browser pool with automated process rotation.
- **Lifecycle Management**: Implement `MaxSessionLife` and `MaxMemoryThreshold`. After these limits, the pool gracefully drains the process and rotates in a fresh instance without interrupting other active sessions.

## 2. Intelligence: Zero-Token Visual Perception (Spatial Maps)
**Current State:** `internal/browser/snapshot.go` relies on the Chromium Accessibility Tree (AXTree). 
**The Opportunity:** Traditional vision models (GPT-4V) are slow and expensive.
**Proposed Improvement:** 
- **Vectorized Spatial Snapshots**: Instead of sending actual screenshots, Axon will generate a lightweight **Spatial Map** JSON. This encodes the absolute coordinates, Z-index, size, and visually dominant colors of elements.
- **Vision-AX Alignment**: By matching spatial data with the AXTree, an agent can "see" that a "Submit" button is large and red without ever needing to process a single pixel, reducing token cost by 100x compared to visual screenshots.

## 3. Reliability: The "Time Machine" (Delta Rollback)
**Current State:** Agents frequently run into "Dead Ends" or perform irreversible actions by mistake.
**Proposed Improvement:** 
- **Session-Level Checkpointing**: Implement a native mechanism to save the complete browser state (DOM + JS Memory + Cookies + Storage) before any action classified as "Write Irreversible."
- **Autonomous Recovery**: If an agent detects a failure (e.g., "Access Denied" or "Invalid Input") after an action, Axon allows a **Delta Rollback** to the exact millisecond before the interaction, enabling the agent to try a different path without starting the whole script over.

## 4. Stability: Self-Healing Semantic Locators
**Current State:** Selectors break when developers change CSS classes or obfuscated React IDs change.
**Proposed Improvement:** 
- **Multi-Anchor Identification**: Implement a scoring-based element resolver that tracks:
    1.  **Semantic Signature** (AXTree path and role).
    2.  **Visual DNA** (Color, icon SVG path, and size).
    3.  **Spatial Context** (Proximity to stable elements like menus or logos).
- **Goal**: If the CSS classes change, Axon "Self-Heals" by identifying the element that still matches the Semantic and Spatial anchors.

## 5. Security & Safety: Proactive Defense & Sandboxing
**Current State:** Static regex scanning for prompt injection.
**Proposed Improvement:** 
- **Local Model Guardrails**: Integrate a lightweight, local "Safe-Guard" model (e.g., Llama-Guard) to scan the AXTree stream for manipulation attempts ("ignore all previous instructions").
- **SSRF Hardening**: Add automatic request interception to prevent agents from being tricked into accessing internal cloud metadata endpoints (e.g., `169.254.169.254`) by malicious sites.

## 6. Performance: Semantic Proxy Filtering
**Current State:** Standard browsers download ads, trackers, and background videos that agents do not need.
**Proposed Improvement:** 
- **Intent-Driven Blocking**: At the protocol level (CDP), Axon will strip out any network traffic that doesn't contribute to the "Semantic State" of the page.
- **Result**: 4x faster page load times and significantly reduced CPU/Memory overhead per session, while still spoofing headers to remain invisible to anti-bot systems like Cloudflare.

## 7. Observability & DevEx: The "Agent Vision" Debugger
**Current State:** Developers must look at raw terminal logs or standard screenshots.
**Proposed Improvement:** 
- **OpenTelemetry Standard**: Export every action and snapshot as a span in a trace.
- **Vision Overlay API**: A real-time web interface that overlays Axon's internal thoughts—semantic refs (`b1`, `t5`), intent classifications, and proposed action paths—on top of the live browser view.

---

### The Benchmark to Crush
The industry focuses on "Success Rate." Axon will focus on **Goal Completion Efficiency (GCE)**:
`GCE = (Goals Accomplished) / (Total Token Spend + Total Execution Latency)`.

By moving these specialized "Agent Senses" into the browser core, Axon will achieve a GCE **10x higher** than any high-level Node.js wrapper like Stagehand or Vercel.

---
*End of Strategy Document - March 1, 2026*
