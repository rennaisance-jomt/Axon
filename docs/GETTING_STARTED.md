# Axon — Getting Started
## Quickstart Guide for Developers

**Version:** 1.0  
**Date:** February 2026

---

## Prerequisites

| Requirement | Version | Notes |
|-------------|---------|-------|
| Go | 1.22+ | [Install](https://go.dev/dl/) |
| Git | 2.0+ | |
| Chrome/Chromium | Latest | Auto-downloaded by Rod |
| 2GB RAM | - | + ~500MB per browser |

---

## 5-Minute Quickstart

### Step 1: Clone the Repository

```bash
git clone https://github.com/rennaisance-jomt/axon.git
cd axon
```

### Step 2: Install Dependencies

```bash
go mod download
```

### Step 3: Build the Binary

```bash
# Production build
go build -o axon ./cmd/axon

# Or use Makefile
make build
```

### Step 4: Start Axon

```bash
# Run with default settings
./axon

# Or with custom config
./axon --config ./configs/config.yaml
```

### Step 5: Verify It's Running

```bash
# Check health
curl http://localhost:8020/health

# List sessions (empty initially)
curl http://localhost:8020/api/v1/sessions
```

Expected output:
```json
{
  "status": "ok",
  "version": "1.0.0",
  "uptime": "2.341s"
}
```

---

## Your First Axon Session

### 1. Create a Session

```bash
curl -X POST http://localhost:8020/api/v1/sessions \
  -H "Content-Type: application/json" \
  -d '{
    "id": "my_session",
    "profile": null
  }'
```

Response:
```json
{
  "session_id": "my_session",
  "status": "created",
  "created_at": "2026-02-27T10:00:00Z"
}
```

### 2. Navigate to a URL

```bash
curl -X POST http://localhost:8020/api/v1/sessions/my_session/navigate \
  -H "Content-Type: application/json" \
  -d '{
    "url": "https://example.com"
  }'
```

Response:
```json
{
  "success": true,
  "url": "https://example.com",
  "title": "Example Domain",
  "state": "ready"
}
```

### 3. Get a Snapshot

```bash
curl -X POST http://localhost:8020/api/v1/sessions/my_session/snapshot
```

Response:
```
PAGE: example.com | Title: Example Domain | State: ready

CONTENT:
  [e1] Example Domain (heading)
  [e2] This domain is for use in illustrative examples in documents. (paragraph)

LINKS:
  [l1] More information... (link) → example.com
```

### 4. Take an Action

```bash
curl -X POST http://localhost:8020/api/v1/sessions/my_session/act \
  -H "Content-Type: application/json" \
  -d '{
    "ref": "l1",
    "action": "click"
  }'
```

### 5. Close the Session

```bash
curl -X DELETE http://localhost:8020/api/vessions/my_session
```

---

## Using the Python SDK

### Installation

```bash
pip install axon-sdk
```

### Example: Post to X.com

```python
import axon

# Connect to Axon server
client = axon.Client("http://localhost:8020")

# Create session with X.com profile
session = client.session_create("x_main", profile="x_session.json")

# Navigate to X
client.navigate("https://x.com/home", session="x_main")

# Get snapshot
snapshot = client.snapshot(session="x_main")
print(snapshot)

# Find the compose box and post
client.act(ref="e1", action="fill", value="Hello from Axon!", session="x_main")
client.act(ref="a1", action="click", session="x_main")
```

---

## Using with LangChain

```python
from langchain.agents import initialize_agent
from langchain.tools import Tool
from axon.langchain import AxonBrowserToolkit

# Get Axon tools
toolkit = AxonBrowserToolkit(session="my_session")
tools = toolkit.get_tools()

# Initialize agent
agent = initialize_agent(
    tools=tools,
    llm=llm,
    agent="zero-shot-react-description"
)

# Use agent
result = agent.run("Go to x.com and post 'Hello world'")
```

---

## Using with AI Agents

```python
# In your agent configuration
BROWSER_BACKEND = "axon"
AXON_URL = "http://localhost:8020"
```

---

## Common Issues

### "Chrome not found"

Rod will automatically download Chromium. If you need to use a specific installation:

```go
// In config
browser:
  binary_path: "/path/to/chrome"
```

### "Port 8020 already in use"

Change the port in config or via flag:

```bash
./axon --port 8021
```

### "Session not found"

Ensure you created the session first:
```bash
curl -X POST http://localhost:8020/api/v1/sessions -d '{"id": "test"}'
```

### "SSRF blocked"

The URL was blocked by security. See SECURITY.md for allowlist configuration.

---

## Next Steps

| Task | Link |
|------|------|
| Configure security | [SECURITY.md](./SECURITY.md) |
| Understand API | [API_SPEC.md](./API_SPEC.md) |
| Set up production | [DEPLOYMENT.md](./DEPLOYMENT.md) |
| Contribute | [CONTRIBUTING.md](./CONTRIBUTING.md) |

---

## Example Scripts

See `examples/` directory for more:

- `examples/basic/main.go` — Basic session example
- `examples/x_post/main.go` — Post to X.com
- `examples/scraper/main.go` — Multi-page scraper

---

<div align="center">

*Axon Project | 2026*  
*An AI-native browser built with ❤️ for AI agents.*

</div>
