# Axon Internal: Server Module

The `internal/server` package provides the high-performance HTTP interface for Axon, bridging the gap between browser automation and AI agent tools.

## 🚀 Key Features

### 1. **High-Performance API** (`server.go`)
Powered by the **Fiber** framework for low-latency request handling.
*   **Routing**: Clean, versioned REST API (`/api/v1/...`).
*   **Middleware**: Built-in recover, Logger, and CORS support.
*   **Graceful Shutdown**: Native context-aware shutdown that ensures the browser pool is closed and storage is flushed.

### 2. **Session Handlers** (`handlers.go`)
Implements the core business logic for session operations:
*   **Create/Delete**: Interface for `browser.SessionManager`.
*   **Snapshot**: High-compression semantic snapshots with prompt injection detection.
*   **Act**: Interactive commands with reversibility classification and audit logging.
*   **Audit**: REST interface for retrieving and verifying the cryptographic audit trail.

### 3. **Validation & Mapping**
*   **Request Parsers**: Strict Go types for all incoming JSON payloads.
*   **Error Handling**: Structured API errors with `Recoverable` and `Suggestion` fields for agent-friendly error recovery.

## 📊 Endpoints Overview

| Endpoint | Method | Action |
|:---|:---|:---|
| `/api/v1/sessions` | POST | Create a new session |
| `/api/v1/sessions/:id/navigate` | POST | Navigate to a URL |
| `/api/v1/sessions/:id/snapshot` | POST | Get a semantic snapshot |
| `/api/v1/sessions/:id/act` | POST | Perform actions (click, fill, etc.) |
| `/api/v1/sessions/:id/status` | GET | Current session state |
| `/api/v1/audit` | GET | Retrieve cryptographic logs |

---
*Axon Server v1.0 | February 2026*
