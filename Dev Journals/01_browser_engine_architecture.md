# Architecture Decision: Go-Rod vs Playwright-Go

**Date:** February 28, 2026
**Topic:** Core Browser Engine Selection and Optimization

## The Underlying Problem
Axon's primary goal is to be a lightweight, insanely efficient, AI-native browser interface. However, early prototyping revealed significant friction and performance issues when capturing high-quality full-page screenshots and extracting reliable semantic data from the DOM. 

These issues trace back to the abstraction level of our current browser automation library: `go-rod`.

## Analysis: `go-rod` vs. `playwright-go`

### `go-rod` (Current Engine)
`go-rod` is a pure Go driver that communicates directly with Chromium via the Chrome DevTools Protocol (CDP) WebSockets.

**Pros for Axon:**
*   **True Single Binary:** Compiles directly into the `axon.exe` binary without requiring Node.js, Python, or external background services. Extremely lightweight distribution.
*   **Minimal Idle Overhead:** The driver itself consumes almost zero CPU/RAM.
*   **Direct Protocol Access:** Easy to bypass anti-bot systems by spoofing fingerprints at the wire level.
*   **Dependency Free:** No massive `node_modules` or chained driver installations required for end-users.

**Cons for Axon:**
*   **Extremely Low-Level:** Every complex action (e.g., full-page screenshots) requires manual window resizing, height calculation, and managing repaint delays. This causes flakiness and performance hits.
*   **Brittle DOM Extraction:** Extracting semantic data (like the accessibility tree) currently relies on injecting custom JavaScript chunks (like `TreeWalker`). This breaks easily on modern web apps using Shadow DOMs or cross-origin Iframes.
*   **Manual State Management:** Implicit waiting for elements or network idle states requires custom polling/timeout logic, leading to race conditions.

### `playwright-go` (Alternative Engine)
A Go wrapper around Microsoft's Playwright engine.

**Pros for Axon:**
*   **Flawless High-Level APIs:** `page.Screenshot(FullPage: true)` works perfectly out-of-the-box.
*   **Native Auto-Waiting:** Built-in C++ level checks for element visibility, actionability, and stillness before interactions, eliminating 90% of race conditions.
*   **Native Accessibility Trees:** Direct, instant access to the browser's accessibility layer, penetrating Shadow DOMs naturally (crucial for LLM understanding).
*   **Hyper-Efficient Contexts:** Built from the ground up to spin up isolated "Browser Contexts" (incognito tabs) in milliseconds instead of launching new browser processes.

**Cons for Axon:**
*   **Heavyweight Dependencies:** Requires a background Node.js process to act as the translation bridge. Users must run an installation command to download specific driver versions and patched browsers.
*   **Breaks Single Binary Promise:** Axon would no longer be a simple drop-in `.exe`.

## The Decision

To maintain Axon's core identity as a **lightweight, highly efficient** tool that acts as a single compiled binary, **we are sticking with `go-rod`.**

However, to achieve the performance, speed, and stability of Playwright, we must drastically alter *how* we use `go-rod` by mimicking Playwright's internal architecture. 

## Architectural Refactors Required for `go-rod`

To fix the current efficiency and stability issues, we need to implement three major refactors:

### 1. Shift to "Context Pooling" instead of "Browser Pooling"
**Current state (`internal/browser/pool.go`):** Pre-launching multiple intensive Chromium browser processes and keeping them in a channel. This causes massive memory overhead and risks state-bleeding between sessions if caches/cookies aren't perfectly cleared.

**The Fix:**
Launch exactly **one** headless browser instance when the Axon server starts.
When a request arrives, generate an isolated `Browser.Incognito()` context.
*   *Why?* Contexts spin up in ~15ms, consume single-digit megabytes of RAM, and guarantee 100% clean state isolation. When a session ends, destroying the context instantly reclaims all memory without killing the root browser process.

### 2. Native CDP DOM Extraction
**Current state (`internal/browser/snapshot.go`):** Injecting heavy JavaScript strings to walk the DOM.
**The Fix:** Stop using JS evaluation. Use `go-rod`'s raw CDP bindings to query the `Accessibility` or `DOMSnapshot` domains directly. This is significantly faster, uses less CPU on the target page, and natively pierces the Shadow DOM.

### 3. Native CDP Full-Page Screenshots
**Current state:** Resizing viewports to match page height, leading to visual glitches on tracking/infinite-scroll pages.
**The Fix:** Use the raw `Page.captureScreenshot` protocol command with `clip` metrics. This captures the entire rendering surface instantly without manipulating the active window size.
