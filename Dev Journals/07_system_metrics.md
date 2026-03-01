# Axon: System Metrics & KPIs

Based on the current architecture of Axon, we can extract several key metrics to measure efficiency, security, and performance. These "numbers" are critical for benchmarking Axon against traditional browser automation tools.

## 1. Goal Completion Efficiency (GCE)
This is the North Star metric for Axon. It measures how effectively the agent completes tasks relative to the cost and time spent.
*   **Formula**: `GCE = (Goals Accomplished) / (Total Token Spend + Execution Latency)`
*   **Token Savings**: Calculated by comparing the raw `AXTree` size vs. the "Collapsed Intent Graph" output.

## 2. Efficiency Metrics (Token Optimization)
These metrics quantify the performance gains from Axon's semantic compression layers.

| Metric | Source | Description |
| :--- | :--- | :--- |
| **Token Count** | `Snapshot.TokenCount` | Estimated tokens per page snapshot (calculated as `chars / 4`). |
| **Intent Compression Ratio** | `CollapseIntentGraph` | Number of raw interactive elements vs. the number of collapsed intent groups. |
| **High-Fidelity Element Density** | `Snapshot.Elements` | Ratio of "Actionable" elements to the total page size. |
| **Snapshot Latency** | `Extractor.Extract` | Time taken to fetch AXTree and classify intents. |

## 3. Reliability & Safety Metrics (Audit)
Pulled from the cryptographically chained audit log (`AuditEntry`) stored in BadgerDB.

| Metric | Description |
| :--- | :--- |
| **Action Throughput** | Total interactions (clicks, fills, navs) across all sessions. |
| **Reversibility Distribution** | % of actions classified as `read`, `write_reversible`, and `write_irreversible`. |
| **Confirmation Rate** | % of high-risk actions that required explicit user approval (`confirm=true`). |
| **Success Rate** | Ratio of `result: "success"` entries in the audit trail. |
| **Security Blocks** | Number of blocked SSRF attempts and detected prompt injections. |

## 4. Operational Performance
Operational data related to browser lifecycle and resource usage.

*   **Server Uptime**: Time since `server.Start()`.
*   **Active Session Load**: Total number of concurrent browser contexts managed by the pool.
*   **Session Lifecycle**: Average session duration from `CreatedAt` to `Delete`.
*   **Idle Drain**: Number of sessions automatically closed due to inactivity.

## 5. Semantic Intelligence KPIs
Metrics on how "smart" the browser's sensory layer is.

*   **Intent Classification Coverage**: % of elements successfully classified vs. marked as `unknown`.
*   **Semantic Drift**: Frequency of "Ref" updates required during a single session (indicates locator stability).
*   **Auth State Hit Rate**: Accuracy of the `StateDetector` in identifying login states across different domains.

---
**Implementation Tip**: These can be exposed via a new `/api/v1/metrics` endpoint using the existing `db` and `sessions` manager.
