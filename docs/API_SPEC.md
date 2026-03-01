# Axon — API Specification
## OpenAPI 3.0 Definition

**Version:** 1.0  
**Date:** February 2026

---

## Overview

| Base URL | Protocol |
|----------|----------|
| `http://localhost:8020` | HTTP |
| `ws://localhost:8020` | WebSocket (for events) |

---

## Authentication

Currently no authentication. For production, use:
- API Key header: `X-API-Key: your-key`
- Or configure in `config.yaml`

---

## Endpoints Summary

| Method | Path | Description |
|--------|------|-------------|
| GET | `/health` | Health check |
| GET | `/api/v1/sessions` | List all sessions |
| POST | `/api/v1/sessions` | Create session |
| GET | `/api/v1/sessions/{id}` | Get session |
| DELETE | `/api/v1/sessions/{id}` | Delete session |
| POST | `/api/v1/sessions/{id}/navigate` | Navigate to URL |
| POST | `/api/v1/sessions/{id}/snapshot` | Get page snapshot |
| POST | `/api/v1/sessions/{id}/act` | Perform action |
| GET | `/api/v1/sessions/{id}/status` | Get session status |
| POST | `/api/v1/sessions/{id}/screenshot` | Take screenshot |
| POST | `/api/v1/sessions/{id}/wait` | Wait for condition |
| GET | `/api/v1/sessions/{id}/cookies` | Get cookies |
| POST | `/api/v1/sessions/{id}/cookies` | Set cookies |
| GET | `/api/v1/audit` | Get audit logs |

---

## OpenAPI Specification

```yaml
openapi: 3.0.3
info:
  title: Axon API
  description: AI-Native Browser Control Server
  version: 1.0.0
  contact:
    name: Axon Project Team

servers:
  - url: http://localhost:8020
    description: Development server

paths:
  /health:
    get:
      summary: Health check
      operationId: healthCheck
      tags:
        - System
      responses:
        '200':
          description: Server is healthy
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/HealthResponse'

  /api/v1/sessions:
    get:
      summary: List all sessions
      operationId: listSessions
      tags:
        - Sessions
      responses:
        '200':
          description: List of sessions
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/SessionList'

    post:
      summary: Create a new session
      operationId: createSession
      tags:
        - Sessions
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/CreateSessionRequest'
      responses:
        '201':
          description: Session created
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Session'

  /api/v1/sessions/{id}:
    get:
      summary: Get session by ID
      operationId: getSession
      tags:
        - Sessions
      parameters:
        - name: id
          in: path
          required: true
          schema:
            type: string
      responses:
        '200':
          description: Session details
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Session'
        '404':
          description: Session not found

    delete:
      summary: Delete session
      operationId: deleteSession
      tags:
        - Sessions
      parameters:
        - name: id
          in: path
          required: true
          schema:
            type: string
      responses:
        '204':
          description: Session deleted

  /api/v1/sessions/{id}/navigate:
    post:
      summary: Navigate to URL
      operationId: navigate
      tags:
        - Browser
      parameters:
        - name: id
          in: path
          required: true
          schema:
            type: string
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/NavigateRequest'
      responses:
        '200':
          description: Navigation successful
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/NavigateResponse'

  /api/v1/sessions/{id}/snapshot:
    post:
      summary: Get page snapshot
      operationId: snapshot
      tags:
        - Browser
      parameters:
        - name: id
          in: path
          required: true
          schema:
            type: string
      requestBody:
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/SnapshotRequest'
      responses:
        '200':
          description: Snapshot retrieved
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/SnapshotResponse'

  /api/v1/sessions/{id}/act:
    post:
      summary: Perform action on element
      operationId: act
      tags:
        - Browser
      parameters:
        - name: id
          in: path
          required: true
          schema:
            type: string
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/ActRequest'
      responses:
        '200':
          description: Action performed
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/ActResponse'

  /api/v1/sessions/{id}/status:
    get:
      summary: Get session status
      operationId: status
      tags:
        - Browser
      parameters:
        - name: id
          in: path
          required: true
          schema:
            type: string
      responses:
        '200':
          description: Session status
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/StatusResponse'

  /api/v1/sessions/{id}/screenshot:
    post:
      summary: Take screenshot
      operationId: screenshot
      tags:
        - Browser
      parameters:
        - name: id
          in: path
          required: true
          schema:
            type: string
      requestBody:
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/ScreenshotRequest'
      responses:
        '200':
          description: Screenshot taken
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/ScreenshotResponse'

  /api/v1/sessions/{id}/wait:
    post:
      summary: Wait for condition
      operationId: wait
      tags:
        - Browser
      parameters:
        - name: id
          in: path
          required: true
          schema:
            type: string
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/WaitRequest'
      responses:
        '200':
          description: Condition met
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/WaitResponse'

  /api/v1/audit:
    get:
      summary: Get audit logs
      operationId: getAuditLogs
      tags:
        - Audit
      parameters:
        - name: session_id
          in: query
          schema:
            type: string
        - name: limit
          in: query
          schema:
            type: integer
            default: 100
        - name: offset
          in: query
          schema:
            type: integer
            default: 0
      responses:
        '200':
          description: Audit logs
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/AuditLogList'

components:
  schemas:
    HealthResponse:
      type: object
      properties:
        status:
          type: string
          example: ok
        version:
          type: string
          example: 1.0.0
        uptime:
          type: string
          example: 1h23m45s

    SessionList:
      type: object
      properties:
        sessions:
          type: array
          items:
            $ref: '#/components/schemas/Session'

    Session:
      type: object
      properties:
        session_id:
          type: string
        status:
          type: string
          enum: [created, active, idle, closed]
        created_at:
          type: string
          format: date-time
        last_action:
          type: string
          format: date-time
        url:
          type: string
        auth_state:
          type: string

    CreateSessionRequest:
      type: object
      required:
        - id
      properties:
        id:
          type: string
          description: Session identifier
        profile:
          type: string
          description: Path to session profile (cookies)
        headless:
          type: boolean
          default: true

    NavigateRequest:
      type: object
      required:
        - url
      properties:
        url:
          type: string
          format: uri
        wait_until:
          type: string
          enum: [load, domcontentloaded, networkidle]
          default: load

    NavigateResponse:
      type: object
      properties:
        success:
          type: boolean
        url:
          type: string
        title:
          type: string
        state:
          type: string

    SnapshotRequest:
      type: object
      properties:
        focus:
          type: string
          description: CSS selector to focus on
        depth:
          type: string
          enum: [compact, standard, full]
          default: compact

    SnapshotResponse:
      type: object
      properties:
        page:
          type: string
        title:
          type: string
        state:
          type: string
        content:
          type: string
        warnings:
          type: array
          items:
            $ref: '#/components/schemas/Warning'

    ActRequest:
      type: object
      required:
        - ref
        - action
      properties:
        ref:
          type: string
          description: Element reference from snapshot
        action:
          type: string
          enum: [click, fill, press, select, hover, scroll]
        value:
          type: string
          description: Value for fill/press/select
        confirm:
          type: boolean
          default: false

    ActResponse:
      type: object
      properties:
        success:
          type: boolean
        result:
          type: string
        requires_confirm:
          type: boolean
        message:
          type: string
        error_type:
          type: string
        recoverable:
          type: boolean

    StatusResponse:
      type: object
      properties:
        url:
          type: string
        title:
          type: string
        auth_state:
          type: string
        page_state:
          type: string
        warnings:
          type: array
          items:
            $ref: '#/components/schemas/Warning'

    ScreenshotRequest:
      type: object
      properties:
        full_page:
          type: boolean
          default: false
        ref:
          type: string
          description: Element to screenshot

    ScreenshotResponse:
      type: object
      properties:
        path:
          type: string

    WaitRequest:
      type: object
      required:
        - condition
      properties:
        condition:
          type: string
          enum: [load, networkidle, domcontentloaded]
        selector:
          type: string
        text:
          type: string
        timeout:
          type: integer
          default: 30000

    WaitResponse:
      type: object
      properties:
        success:
          type: boolean
        matched:
          type: boolean

    Warning:
      type: object
      properties:
        type:
          type: string
        severity:
          type: string
        message:
          type: string

    AuditLogList:
      type: object
      properties:
        logs:
          type: array
          items:
            $ref: '#/components/schemas/AuditLogEntry'
        total:
          type: integer

    AuditLogEntry:
      type: object
      properties:
        id:
          type: string
        timestamp:
          type: string
          format: date-time
        session_id:
          type: string
        action:
          type: string
        result:
          type: string
        prev_hash:
          type: string
        this_hash:
          type: string
```

---

## Error Responses

| Status Code | Description | Example |
|-------------|-------------|---------|
| 400 | Bad Request | Invalid parameters |
| 404 | Not Found | Session not found |
| 429 | Rate Limited | Too many requests |
| 500 | Internal Error | Server error |

### Error Response Format

```json
{
  "error": true,
  "error_type": "element_not_found",
  "message": "Element [e1] not found",
  "suggestion": "Run snapshot to get fresh refs"
}
```

---

## WebSocket Events (Optional)

For real-time updates:

```javascript
const ws = new WebSocket('ws://localhost:8020/ws/sessions/my_session');

ws.onmessage = (event) => {
  const data = JSON.parse(event.data);
  console.log(data.type, data.payload);
};
```

Event types:
- `navigation` — Page navigated
- `action` — Action completed
- `dom_change` — DOM changed
- `error` — Error occurred

---

## Rate Limits

| Endpoint | Limit |
|----------|-------|
| General | 1000 req/min |
| Snapshot | 100 req/min |
| Act | 200 req/min |

---

<div align="center">

*Axon Project | 2026*  
*An AI-native browser built with ❤️ for AI agents.*

</div>
