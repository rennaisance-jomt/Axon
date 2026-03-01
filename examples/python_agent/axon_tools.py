"""
Axon Browser Tools for LangChain

This module provides LangChain-compatible tools for browser automation using Axon.
"""

import json
import requests
from typing import Optional, Dict, Any, List
from pydantic import BaseModel, Field
from langchain.tools import BaseTool
from langchain.callbacks.manager import CallbackManagerForToolRun


class AxonBaseTool(BaseTool):
    """Base class for Axon tools."""
    
    base_url: str = Field(default="http://localhost:8020", description="Axon server URL")
    session_id: str = Field(default="default", description="Session ID to use")
    
    def _make_request(self, method: str, endpoint: str, data: Optional[Dict] = None) -> Dict[str, Any]:
        """Make a request to the Axon server."""
        url = f"{self.base_url}{endpoint}"
        try:
            if method == "GET":
                response = requests.get(url, timeout=30)
            elif method == "POST":
                response = requests.post(url, json=data, timeout=30)
            elif method == "DELETE":
                response = requests.delete(url, timeout=30)
            else:
                raise ValueError(f"Unsupported method: {method}")
            
            response.raise_for_status()
            return response.json() if response.content else {}
        except requests.exceptions.ConnectionError:
            return {"error": "Cannot connect to Axon server. Is it running?"}
        except requests.exceptions.Timeout:
            return {"error": "Request timed out"}
        except requests.exceptions.HTTPError as e:
            try:
                return e.response.json()
            except:
                return {"error": f"HTTP error: {e.response.status_code}"}


class AxonNavigateTool(AxonBaseTool):
    """Tool for navigating to a URL."""
    
    name: str = "axon_navigate"
    description: str = """Navigate to a URL in the browser. 
    Use this to load a webpage before interacting with it.
    Input should be a URL string (e.g., "https://example.com")."""
    
    def _run(self, url: str, run_manager: Optional[CallbackManagerForToolRun] = None) -> str:
        """Navigate to a URL."""
        result = self._make_request(
            "POST",
            f"/api/v1/sessions/{self.session_id}/navigate",
            {"url": url, "wait_until": "load"}
        )
        
        if "error" in result:
            return f"Navigation failed: {result['error']}"
        
        return f"Successfully navigated to {result.get('url', url)}. Page title: {result.get('title', 'Unknown')}. State: {result.get('state', 'unknown')}"


class AxonSnapshotTool(AxonBaseTool):
    """Tool for getting a semantic snapshot of the current page."""
    
    name: str = "axon_snapshot"
    description: str = """Get a semantic snapshot of the current page.
    This returns a compact representation of interactive elements with refs (like 'b1', 't2').
    Use this to understand what's on the page before taking actions.
    Returns element refs you can use with axon_act."""
    
    def _run(self, run_manager: Optional[CallbackManagerForToolRun] = None) -> str:
        """Get a snapshot of the current page."""
        result = self._make_request(
            "GET",
            f"/api/v1/sessions/{self.session_id}/snapshot"
        )
        
        if "error" in result:
            return f"Failed to get snapshot: {result['error']}"
        
        content = result.get('content', 'No content available')
        warnings = result.get('warnings', [])
        
        response = f"Page Snapshot:\n{content}"
        
        if warnings:
            response += f"\n\nWarnings: {json.dumps(warnings, indent=2)}"
        
        return response


class AxonActTool(AxonBaseTool):
    """Tool for performing actions on elements."""
    
    name: str = "axon_act"
    description: str = """Perform an action on an element.
    Use this to click buttons, fill forms, select options, etc.
    Input should be a JSON string with:
    - ref: Element reference from snapshot (e.g., "b4", "t12")
    - action: One of "click", "fill", "press", "select", "hover", "scroll"
    - value: Value for fill/select actions (optional)
    Example: {"ref": "b4", "action": "click"}"""
    
    def _run(self, action_json: str, run_manager: Optional[CallbackManagerForToolRun] = None) -> str:
        """Perform an action."""
        try:
            params = json.loads(action_json)
        except json.JSONDecodeError:
            return "Invalid JSON. Example: {\"ref\": \"b4\", \"action\": \"click\"}"
        
        result = self._make_request(
            "POST",
            f"/api/v1/sessions/{self.session_id}/act",
            params
        )
        
        if "error" in result:
            return f"Action failed: {result['error']}. {result.get('suggestion', '')}"
        
        if result.get('requires_confirm'):
            return f"This action requires confirmation: {result.get('message', '')}"
        
        return f"Action {result.get('action', 'completed')} successful on element {result.get('ref', 'unknown')}"


class AxonFindAndActTool(AxonBaseTool):
    """Tool for finding elements by intent and acting on them."""
    
    name: str = "axon_find_and_act"
    description: str = """Find an element by description and perform an action.
    Use this when you know what you want to do but don't have a specific ref.
    Input should be a JSON string with:
    - intent: Description of the element (e.g., "search box", "login button")
    - action: One of "click", "fill", "press", "select", "hover"
    - value: Value for fill actions (optional)
    Example: {"intent": "search box", "action": "fill", "value": "OpenAI"}"""
    
    def _run(self, params_json: str, run_manager: Optional[CallbackManagerForToolRun] = None) -> str:
        """Find and act on an element."""
        try:
            params = json.loads(params_json)
        except json.JSONDecodeError:
            return "Invalid JSON. Example: {\"intent\": \"search box\", \"action\": \"click\"}"
        
        # First get snapshot to find the element
        snapshot = self._make_request(
            "GET",
            f"/api/v1/sessions/{self.session_id}/snapshot"
        )
        
        if "error" in snapshot:
            return f"Failed to get snapshot: {snapshot['error']}"
        
        # Use the MCP find_and_act endpoint
        result = self._make_request(
            "POST",
            f"/api/v1/sessions/{self.session_id}/find_and_act",
            params
        )
        
        if "error" in result:
            return f"Find and act failed: {result['error']}"
        
        return f"Successfully {result.get('action', 'acted')} on element matching '{params.get('intent')}'"


class AxonGetStatusTool(AxonBaseTool):
    """Tool for getting page status."""
    
    name: str = "axon_get_status"
    description: str = """Get the current page status including URL, title, auth state, and page state.
    Use this to check if you're logged in, if there's a CAPTCHA, or the page is ready."""
    
    def _run(self, run_manager: Optional[CallbackManagerForToolRun] = None) -> str:
        """Get page status."""
        result = self._make_request(
            "GET",
            f"/api/v1/sessions/{self.session_id}/status"
        )
        
        if "error" in result:
            return f"Failed to get status: {result['error']}"
        
        return json.dumps(result, indent=2)


class AxonWaitTool(AxonBaseTool):
    """Tool for waiting for conditions."""
    
    name: str = "axon_wait"
    description: str = """Wait for a specific condition on the page.
    Input should be a JSON string with:
    - condition: "selector", "text", or "navigation"
    - selector: CSS selector to wait for (if condition is "selector")
    - text: Text to wait for (if condition is "text")
    - timeout: Timeout in seconds (default: 10)
    Example: {"condition": "text", "text": "Loading complete"}"""
    
    def _run(self, params_json: str, run_manager: Optional[CallbackManagerForToolRun] = None) -> str:
        """Wait for a condition."""
        try:
            params = json.loads(params_json)
        except json.JSONDecodeError:
            return "Invalid JSON"
        
        result = self._make_request(
            "POST",
            f"/api/v1/sessions/{self.session_id}/wait",
            params
        )
        
        if "error" in result:
            return f"Wait failed: {result['error']}"
        
        if result.get('matched'):
            return "Condition matched successfully"
        
        return "Condition not matched within timeout"


class AxonScreenshotTool(AxonBaseTool):
    """Tool for taking screenshots."""
    
    name: str = "axon_screenshot"
    description: str = """Take a screenshot of the current page.
    Use this to capture the visual state of the page.
    Input can be empty for full page, or JSON with:
    - full_page: true/false (default: true)
    - ref: Element reference for element-specific screenshot (optional)
    Example: {"full_page": true}"""
    
    def _run(self, params_json: str = "", run_manager: Optional[CallbackManagerForToolRun] = None) -> str:
        """Take a screenshot."""
        params = {}
        if params_json:
            try:
                params = json.loads(params_json)
            except:
                pass
        
        result = self._make_request(
            "POST",
            f"/api/v1/sessions/{self.session_id}/screenshot",
            params
        )
        
        if "error" in result:
            return f"Screenshot failed: {result['error']}"
        
        return f"Screenshot saved to: {result.get('path', 'unknown')}"


def get_axon_tools(base_url: str = "http://localhost:8020", session_id: str = "default") -> List[BaseTool]:
    """
    Get all Axon tools for LangChain.
    
    Args:
        base_url: The URL of the Axon server
        session_id: The session ID to use
        
    Returns:
        List of Axon tools
    """
    tools = [
        AxonNavigateTool(base_url=base_url, session_id=session_id),
        AxonSnapshotTool(base_url=base_url, session_id=session_id),
        AxonActTool(base_url=base_url, session_id=session_id),
        AxonFindAndActTool(base_url=base_url, session_id=session_id),
        AxonGetStatusTool(base_url=base_url, session_id=session_id),
        AxonWaitTool(base_url=base_url, session_id=session_id),
        AxonScreenshotTool(base_url=base_url, session_id=session_id),
    ]
    return tools


# Example usage
if __name__ == "__main__":
    # Test the tools
    navigate = AxonNavigateTool()
    print(navigate._run("https://example.com"))
    
    snapshot = AxonSnapshotTool()
    print(snapshot._run())
