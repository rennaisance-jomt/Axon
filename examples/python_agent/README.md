# Python Agent Connector Example

This directory contains a very simple Python script demonstrating how an AI Agent (an LLM) would interact with the Axon browser using Python's `requests` library. 

### Why Python?
Most AI developers write their agents using Python libraries like **LangChain**, **CrewAI**, or **AutoGen**. Axon runs as a Go backend (for speed and low-level browser automation), but exposes a simple HTTP API so any Python Agent can control it natively.

### Running

To try this locally:

1. In one terminal window, start the Axon Go Server:
   ```bash
   go build -o axon.exe ./cmd/axon
   ./axon.exe
   ```

2. In another terminal window, run the Python Agent Simulation:
   ```bash
   pip install requests
   python examples/python_agent/main.py
   ```

### How it works
The `AxonClient` sends exact commands over HTTP to the running `axon.exe` server to:
1. Initialize an incognito session.
2. Navigate the browser.
3. Grab the clean JSON snapshot.
4. Interact with an element (click).
