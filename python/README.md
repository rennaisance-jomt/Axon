# Axon Python SDK

A Python client library for Axon browser automation.

## Installation

```bash
pip install axon-browser
```

## Quick Start (On-Demand Engine)

Axon can now automatically manage the browser engine for you. No need to start the `.exe` manually.

```python
import asyncio
from axon import Axon

async def main():
    # start_engine=True tells the SDK to launch axon.exe automatically
    async with Axon(start_engine=True) as axon:
        # Create a session
        session = await axon.create_session("mysession")
        
        # Navigate
        await axon.navigate("mysession", "https://github.com")
        
        # use the SMART tool (Intent-based)
        result = await axon.smart_interact("mysession", "the search input", "fill", "Axon")
        print(f"Action result: {result.success}")

asyncio.run(main())
```

## Agent Framework Integration (Toolkit)

Axon provides a pre-packaged "Sensory Kit" for agents (like Vamora or LangChain).

```python
from axon import Axon, AxonToolkit

async with Axon(start_engine=True) as axon:
    # 1. Initialize the toolkit
    kit = AxonToolkit(axon, session_id="agent_01")
    
    # 2. Get LLM-ready tools
    tools = await kit.get_tools()
    
    # 3. Run a tool from agent's choice
    output = await kit.run_tool("snapshot", {})
    print(output)
```

## Configuration

The Axon client can be configured via:

1. **Constructor parameter:**
   ```python
   axon = Axon("http://localhost:8020/api/v1")
   ```

2. **Environment variable:**
   ```bash
   export AXON_API_URL=http://localhost:8020/api/v1
   ```
   
   ```python
   axon = Axon()  # Uses AXON_API_URL env var
   ```

## API Reference

### Session Management

```python
# Create a session
session = await axon.create_session("mysession")

# Get session info
info = await axon.get_session("mysession")

# List all sessions
sessions = await axon.list_sessions()

# Delete a session
await axon.delete_session("mysession")
```

### Navigation

```python
# Navigate to a URL
await axon.navigate("mysession", "https://github.com")
```

### Snapshots

```python
# Get page snapshot
snapshot = await axon.snapshot("mysession")

# Print page elements
for element in snapshot.elements:
    print(f"{element.role}: {element.name}")
```

### Actions

```python
# Click
await axon.click("mysession", "e1")

# Fill input
await axon.fill("mysession", "e2", "Hello World")

# Hover
await axon.hover("mysession", "e3")

# Select option
await axon.select("mysession", "e4", "option1")

# Generic action
await axon.act("mysession", "click", "e1")
```

### Find and Act

```python
# Find element by semantic description and perform action
result = await axon.find_and_act(
    "mysession", 
    "click", 
    "search button"
)
```

## Development

```bash
# Clone the repository
git clone https://github.com/rennissance-jomt/axon
cd axon/python

# Install in development mode
pip install -e .

# Install dev dependencies
pip install -e ".[dev]"

# Run tests
pytest

# Lint
ruff check .
```
