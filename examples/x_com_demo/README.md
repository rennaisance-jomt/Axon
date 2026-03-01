# Axon x.com Demo

> Comprehensive demonstration of all Axon AI-Native Browser capabilities on x.com (Twitter)

## Overview

This demo showcases the complete feature set of the Axon AI-Native Browser through a Python client that interacts with x.com. It demonstrates how AI agents can browse the web without CSS selectors, massive HTML dumps, or human-in-the-loop.

## Prerequisites

1. **Axon Server Running**
   ```bash
   # From the Axon project root
   go run cmd/axon-cli/main.go
   ```
   The server will start on `http://localhost:8020`

2. **Python Dependencies**
   ```bash
   pip install requests
   ```

## Quick Start

```bash
# Run the comprehensive demo
python examples/x_com_demo/x_com_demo.py
```

## What This Demo Shows

### Core Capabilities

| Section | Capability | Description |
|---------|------------|-------------|
| 1 | Server Connection | Health check and server verification |
| 2 | Session Management | Create, list, close named sessions |
| 3 | Navigation | Navigate with wait conditions |
| 4 | Semantic Snapshots | Compact/standard/full page representations |
| 5 | Page State Detection | Auto-detect logged_in, captcha, etc. |
| 6 | Screenshot Capture | Viewport and full-page screenshots |
| 7 | Cookie Management | Get, set, export, clear cookies |
| 8 | Network Traffic | Inspect API requests and responses |
| 9 | Tab Management | Create, list, switch, close tabs |
| 10 | Element Interaction | Click, fill, press actions |
| 11 | Intent-Based Interaction | Find by function, not ref |
| 12 | Error Handling | Structured error responses |
| 13 | Security Features | SSRF protection testing |
| 14 | Audit Log | Tamper-evident action logging |
| 15 | Wait Conditions | Wait for elements/text/network |

## Example Usage

### Create a Session and Navigate

```python
from x_com_demo import AxonClient

axon = AxonClient()

# Create session
axon.create_session("my_session")

# Navigate to x.com
axon.navigate("https://x.com")

# Check auth state
status = axon.get_session_status()
print(status.get("auth_state"))  # "logged_in" or "logged_out"
```

### Get a Semantic Snapshot

```python
# Get compact snapshot (50-500 tokens)
snapshot = axon.get_snapshot(depth="compact")
print(snapshot["content"])
# Output:
# PAGE: x.com/home
# TITLE: Home / X
# STATE: logged_in
# 
# NAV:
#   [n1] Home  [n2] Explore  [n3] Notifications
# 
# COMPOSE:
#   [e1] Post text (textbox) — social.publish-input
```

### Interact with Elements

```python
# Using refs from snapshot
axon.act(ref="e1", action="fill", value="Hello from Axon!")
axon.act(ref="a1", action="click", confirm=True)  # Irreversible action
```

### Intent-Based Interaction

```python
# Find by what it does, not by ref
axon.find_and_act(intent="search box", action="fill", value="OpenAI")
axon.find_and_act(intent="login button", action="click")
```

## Project Structure

```
examples/x_com_demo/
├── x_com_demo.py    # Main demo script with all capabilities
└── README.md        # This file
```

## Key Concepts

### Semantic Snapshots

Instead of HTML dumps (thousands of tokens), Axon provides semantic snapshots:

- **Compact**: 50-500 tokens, high-level view
- **Standard**: Medium detail
- **Full**: Complete data

### Intent Classification

Axon automatically classifies elements:

| Element | Intent |
|---------|--------|
| Login button | `auth.login` |
| Search box | `search.query` |
| Post button | `social.publish` |
| Delete button | `content.delete` (IRREVERSIBLE) |

### Action Reversibility

Axon enforces safety by classifying actions:

| Action Type | Requires Confirm |
|-------------|-----------------|
| Read-only (snapshot, navigate) | No |
| Write-reversible (fill, click) | No |
| Write-irreversible (submit, delete) | Yes |

## Learn More

- [Architecture](../docs/ARCHITECTURE.md)
- [Features](../docs/FEATURES.md)
- [API Specification](../docs/API_SPEC.md)
- [Security Model](../docs/SECURITY.md)
