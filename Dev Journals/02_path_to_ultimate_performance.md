# Path to Ultimate AI-Native Performance

**Date:** February 28, 2026
**Topic:** How to Make Axon the Fastest, Lightest, Most Powerful Semantic Engine

To achieve the ultimate goal of making Axon wildly powerful, pure AI-agent native, super lightweight, and possessing insane performance, we must double down on core engine optimizations and strip away anything that resembles 'human browser mimicking'. 

Here is the master plan for reaching peak performance.

## 1. Zero-Overhead Context Pooling (The Performance Fix)
Instead of pooling entire Chrome instances (which cost hundreds of megabytes of RAM and seconds to boot), we run exactly **one** invisible Chromium daemon in the background.
*   **Actionable Step:** Refactor `internal/browser/pool.go`. When an agent creates a session, Axon generates an isolated `Browser Context` (an incognito-like sandbox) linked to the daemon. 
*   **The Result:** Spinning up a new agent session goes from taking ~2,000ms to **15ms**. Memory overhead drops to <10MB per session.

## 2. Native CDP DOM Extraction (The AI-Native Fix)
Injecting JavaScript strings into pages to try and figure out what a button does is fragile and slow.
*   **Actionable Step:** Refactor `internal/browser/snapshot.go`. Remove the custom JS `TreeWalker`. Connect directly to Chromium's native C++ `Accessibility` protocol domain using CDP.
*   **The Result:** Perfect extraction of the accessibility tree in microseconds. It intrinsically understands Shadow DOMs (which break JS extractors) and surfaces native ARIA roles instantly.

## 3. High-Compression Intent Graphs (The Super-Lightweight Fix)
Currently, we extract elements. We need to go further and extract relationships.
*   **Actionable Step:** Enhance `snapshot.go` to build an 'Intent Graph' rather than a flat list of elements. If a `<input>` has a neighboring `<button>` that says "Search", Axon collapses them into a single `{"intent": "search", "actionable_id": "e1"}` object payload.
*   **The Result:** Token usage per page drops another 50%, saving the LLM massive context window space while actually increasing reasoning accuracy.

## 4. Headless-Native Caching & Blockers
Agent browsers don't need CSS animations, custom fonts, or tracking scripts to function (unless specifically asked).
*   **Actionable Step:** Implement a strict Network Interceptor in `go-rod` that drops requests for `.woff2`, `.gif`, analytics domains, and massive CSS payloads during agent-mode.
*   **The Result:** Page load times drop by 70%. CPU usage crashes. The agent gets to the DOM instantly.

## 5. Event-Driven Auto-Waiting
Stop writing `time.Sleep(1 * time.Second)` or blindly waiting for generic `networkidle` states. 
*   **Actionable Step:** Wire Axon to listen to CDP `DOMNodeInserted` and `AnimationCanceled` native events. When the agent says "click e5", Axon queries the C++ layer: *Is e5 visible? Is it still?* If yes, click instantly.
*   **The Result:** Flakiness evaporates. Speed increases exponentially because we never wait longer than the exact millisecond required.

## Summary Architecture
By compiling pure Go with `go-rod` managing a single heavily-optimized Chromium daemon via raw CDP protocols, Axon will be a self-contained `< 30MB` binary executable. It will sit locally on an agent's server, intercept prompt strings, instantly map them to semantic DOM trees, execute precise clicks, and return highly-compressed token snapshots. 

This is what will make Axon untouchable.
