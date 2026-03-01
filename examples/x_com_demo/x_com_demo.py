#!/usr/bin/env python3
"""
Axon x.com Comprehensive Demo
==============================
This script showcases ALL capabilities of the Axon AI-Native Browser
by demonstrating various interactions with x.com (Twitter).

Prerequisites:
    - Axon server running on localhost:8020
    - requests library installed: pip install requests

Usage:
    python x_com_demo.py
"""

import requests
import json
import time
import sys
from typing import Optional, Dict, Any


# Enable UTF-8 for Windows
if sys.platform == 'win32':
    import io
    sys.stdout = io.TextIOWrapper(sys.stdout.buffer, encoding='utf-8', errors='replace')
    sys.stderr = io.TextIOWrapper(sys.stderr.buffer, encoding='utf-8', errors='replace')


class AxonClient:
    """
    Comprehensive Python client for Axon AI-Native Browser.
    Demonstrates all major capabilities.
    """
    
    def __init__(self, base_url: str = "http://localhost:8020/api/v1"):
        self.base_url = base_url
        self.session_id: Optional[str] = None
        
    # =========================================================================
    # SECTION 1: SERVER CONNECTION
    # =========================================================================
    
    def check_server_health(self) -> bool:
        """Check if Axon server is running."""
        try:
            r = requests.get("http://localhost:8020/health", timeout=5)
            return r.status_code == 200
        except:
            return False
    
    # =========================================================================
    # SECTION 2: SESSION MANAGEMENT
    # =========================================================================
    
    def create_session(self, session_id: str, profile: str = None) -> Dict[str, Any]:
        """
        Create a new Axon session.
        
        Capabilities demonstrated:
        - Named persistent sessions
        - Profile loading (auth cookies)
        - Session isolation
        """
        self.session_id = session_id
        payload = {"id": session_id}
        if profile:
            payload["profile"] = profile
            
        r = requests.post(f"{self.base_url}/sessions", json=payload)
        return self._handle_response(r)
    
    def list_sessions(self) -> Dict[str, Any]:
        """List all active sessions."""
        r = requests.get(f"{self.base_url}/sessions")
        return self._handle_response(r)
    
    def get_session_status(self, session_id: str = None) -> Dict[str, Any]:
        """Get session status including auth state, URL, etc."""
        sid = session_id or self.session_id
        r = requests.get(f"{self.base_url}/sessions/{sid}/status")
        return self._handle_response(r)
    
    def close_session(self, session_id: str = None) -> Dict[str, Any]:
        """Close a session."""
        sid = session_id or self.session_id
        r = requests.delete(f"{self.base_url}/sessions/{sid}")
        return self._handle_response(r)
    
    # =========================================================================
    # SECTION 3: NAVIGATION
    # =========================================================================
    
    def navigate(self, url: str, wait_until: str = "load", session_id: str = None) -> Dict[str, Any]:
        """
        Navigate to a URL.
        
        Capabilities demonstrated:
        - Navigate to any URL
        - Wait conditions (load, networkidle, domcontentloaded)
        - Automatic SSRF protection
        """
        sid = session_id or self.session_id
        payload = {"url": url, "wait_until": wait_until}
        r = requests.post(f"{self.base_url}/sessions/{sid}/navigate", json=payload)
        return self._handle_response(r)
    
    def go_back(self, session_id: str = None) -> Dict[str, Any]:
        """Navigate back."""
        sid = session_id or self.session_id
        r = requests.post(f"{self.base_url}/sessions/{sid}/back")
        return self._handle_response(r)
    
    def go_forward(self, session_id: str = None) -> Dict[str, Any]:
        """Navigate forward."""
        sid = session_id or self.session_id
        r = requests.post(f"{self.base_url}/sessions/{sid}/forward")
        return self._handle_response(r)
    
    def reload(self, session_id: str = None) -> Dict[str, Any]:
        """Reload the current page."""
        sid = session_id or self.session_id
        r = requests.post(f"{self.base_url}/sessions/{sid}/reload")
        return self._handle_response(r)
    
    # =========================================================================
    # SECTION 4: SEMANTIC SNAPSHOTS
    # =========================================================================
    
    def get_snapshot(self, depth: str = "compact", focus: str = None, session_id: str = None) -> Dict[str, Any]:
        """
        Get a semantic snapshot of the page.
        
        Capabilities demonstrated:
        - Multiple depth levels (compact, standard, full)
        - Scoped snapshots (focus on specific element)
        - Automatic intent classification
        - Page state detection (logged_in, captcha, etc.)
        - Token-optimized output
        """
        sid = session_id or self.session_id
        payload = {"depth": depth}
        if focus:
            payload["focus"] = focus
        r = requests.post(f"{self.base_url}/sessions/{sid}/snapshot", json=payload)
        return self._handle_response(r)
    
    # =========================================================================
    # SECTION 5: ELEMENT INTERACTION
    # =========================================================================
    
    def act(self, ref: str, action: str, value: str = None, confirm: bool = False, session_id: str = None) -> Dict[str, Any]:
        """
        Perform an action on an element.
        
        Capabilities demonstrated:
        - click, fill, press, select, hover, scroll, drag
        - Action reversibility enforcement (requires confirm for irreversible actions)
        - Automatic element visibility/stability waiting
        - Structured error responses
        """
        sid = session_id or self.session_id
        payload = {"ref": ref, "action": action, "confirm": confirm}
        if value:
            payload["value"] = value
        r = requests.post(f"{self.base_url}/sessions/{sid}/act", json=payload)
        return self._handle_response(r)
    
    def fill_form(self, fields: Dict[str, Any], session_id: str = None) -> Dict[str, Any]:
        """
        Fill multiple form fields at once.
        
        Capabilities demonstrated:
        - Batch form filling
        - Secret vault integration (e.g., [VAULT:password])
        """
        sid = session_id or self.session_id
        r = requests.post(f"{self.base_url}/sessions/{sid}/fill_form", json=fields)
        return self._handle_response(r)
    
    # =========================================================================
    # SECTION 6: INTENT-BASED INTERACTION
    # =========================================================================
    
    def find_and_act(self, intent: str, action: str, value: str = None, session_id: str = None) -> Dict[str, Any]:
        """
        Find and interact with elements by describing their function.
        
        Capabilities demonstrated:
        - Intent-based element resolution
        - No need for refs - describe what you want
        - Examples: "search box", "login button", "email input"
        """
        sid = session_id or self.session_id
        payload = {"intent": intent, "action": action}
        if value:
            payload["value"] = value
        r = requests.post(f"{self.base_url}/sessions/{sid}/find_and_act", json=payload)
        return self._handle_response(r)
    
    # =========================================================================
    # SECTION 7: SCREENSHOTS & PDF
    # =========================================================================
    
    def screenshot(self, full_page: bool = False, ref: str = None, session_id: str = None) -> Dict[str, Any]:
        """
        Capture a screenshot.
        
        Capabilities demonstrated:
        - Full page or viewport screenshot
        - Element-specific screenshot
        """
        sid = session_id or self.session_id
        payload = {"full_page": full_page}
        if ref:
            payload["ref"] = ref
        r = requests.post(f"{self.base_url}/sessions/{sid}/screenshot", json=payload)
        return self._handle_response(r)
    
    def pdf(self, path: str = None, session_id: str = None) -> Dict[str, Any]:
        """Export page as PDF."""
        sid = session_id or self.session_id
        payload = {"path": path} if path else {}
        r = requests.post(f"{self.base_url}/sessions/{sid}/pdf", json=payload)
        return self._handle_response(r)
    
    # =========================================================================
    # SECTION 8: COOKIE MANAGEMENT
    # =========================================================================
    
    def get_cookies(self, domain: str = None, session_id: str = None) -> Dict[str, Any]:
        """
        Get cookies for the session.
        
        Capabilities demonstrated:
        - Domain-scoped cookie retrieval
        """
        sid = session_id or self.session_id
        params = {"domain": domain} if domain else {}
        r = requests.get(f"{self.base_url}/sessions/{sid}/cookies", params=params)
        return self._handle_response(r)
    
    def set_cookies(self, cookies: list, session_id: str = None) -> Dict[str, Any]:
        """Set cookies for the session."""
        sid = session_id or self.session_id
        r = requests.post(f"{self.base_url}/sessions/{sid}/cookies", json=cookies)
        return self._handle_response(r)
    
    def export_cookies(self, path: str, session_id: str = None) -> Dict[str, Any]:
        """Export cookies to a file."""
        sid = session_id or self.session_id
        r = requests.post(f"{self.base_url}/sessions/{sid}/cookies/export", json={"path": path})
        return self._handle_response(r)
    
    def clear_cookies(self, session_id: str = None) -> Dict[str, Any]:
        """Clear all cookies for the session."""
        sid = session_id or self.session_id
        r = requests.delete(f"{self.base_url}/sessions/{sid}/cookies")
        return self._handle_response(r)
    
    # =========================================================================
    # SECTION 9: TAB MANAGEMENT
    # =========================================================================
    
    def new_tab(self, url: str = None, session_id: str = None) -> Dict[str, Any]:
        """
        Open a new tab.
        
        Capabilities demonstrated:
        - Tab creation
        - Optional URL to open immediately
        """
        sid = session_id or self.session_id
        payload = {"url": url} if url else {}
        r = requests.post(f"{self.base_url}/sessions/{sid}/tabs", json=payload)
        return self._handle_response(r)
    
    def list_tabs(self, session_id: str = None) -> Dict[str, Any]:
        """List all open tabs."""
        sid = session_id or self.session_id
        r = requests.get(f"{self.base_url}/sessions/{sid}/tabs")
        return self._handle_response(r)
    
    def switch_tab(self, index: int, session_id: str = None) -> Dict[str, Any]:
        """Switch to a different tab."""
        sid = session_id or self.session_id
        r = requests.post(f"{self.base_url}/sessions/{sid}/tabs/{index}/switch")
        return self._handle_response(r)
    
    def close_tab(self, index: int, session_id: str = None) -> Dict[str, Any]:
        """Close a tab."""
        sid = session_id or self.session_id
        r = requests.delete(f"{self.base_url}/sessions/{sid}/tabs/{index}")
        return self._handle_response(r)
    
    # =========================================================================
    # SECTION 10: NETWORK TRAFFIC
    # =========================================================================
    
    def get_requests(self, filter: str = None, session_id: str = None) -> Dict[str, Any]:
        """
        Get network requests.
        
        Capabilities demonstrated:
        - Request inspection
        - Filtering by type (api, xhr, document, etc.)
        """
        sid = session_id or self.session_id
        params = {"filter": filter} if filter else {}
        r = requests.get(f"{self.base_url}/sessions/{sid}/requests", params=params)
        return self._handle_response(r)
    
    def get_response_body(self, url: str, session_id: str = None) -> Dict[str, Any]:
        """Get response body for a specific URL."""
        sid = session_id or self.session_id
        r = requests.get(f"{self.base_url}/sessions/{sid}/response", params={"url": url})
        return self._handle_response(r)
    
    # =========================================================================
    # SECTION 11: WAIT CONDITIONS
    # =========================================================================
    
    def wait(self, condition: str, timeout: int = 30, session_id: str = None) -> Dict[str, Any]:
        """
        Wait for a condition.
        
        Capabilities demonstrated:
        - Wait for element to appear
        - Wait for text to be visible
        - Wait for network idle
        - Timeout handling
        """
        sid = session_id or self.session_id
        payload = {"condition": condition, "timeout": timeout}
        r = requests.post(f"{self.base_url}/sessions/{sid}/wait", json=payload)
        return self._handle_response(r)
    
    # =========================================================================
    # SECTION 12: PAGE STATE DETECTION
    # =========================================================================
    
    def get_page_state(self, session_id: str = None) -> Dict[str, Any]:
        """
        Get the current page state.
        
        Capabilities demonstrated:
        - Automatic state detection:
          - logged_in / logged_out
          - captcha
          - rate_limited
          - error_page
          - loading
          - interstitial (cookie consent, age gate, etc.)
        """
        sid = session_id or self.session_id
        r = requests.get(f"{self.base_url}/sessions/{sid}/state")
        return self._handle_response(r)
    
    # =========================================================================
    # SECTION 13: ERROR HANDLING
    # =========================================================================
    
    def handle_error(self, response: Dict[str, Any]) -> None:
        """
        Demonstrate structured error handling.
        
        Capabilities demonstrated:
        - Error types: element_not_found, navigation_failed, timeout, 
          captcha, rate_limited, auth_required
        - Recovery suggestions
        - Recoverable vs non-recoverable errors
        """
        if not response.get("success", True):
            error_type = response.get("error_type", "unknown")
            message = response.get("message", "No message")
            suggestion = response.get("suggestion", "No suggestion")
            recoverable = response.get("recoverable", False)
            
            print(f"\n  [X] Error Type: {error_type}")
            print(f"     Message: {message}")
            print(f"     Suggestion: {suggestion}")
            print(f"     Recoverable: {recoverable}")
    
    # =========================================================================
    # SECTION 14: SECURITY FEATURES
    # =========================================================================
    
    def test_ssrf_protection(self, url: str, session_id: str = None) -> Dict[str, Any]:
        """
        Test SSRF protection.
        
        Capabilities demonstrated:
        - Blocked: private IPs, file://, javascript:, data:
        - DNS rebinding prevention
        - Domain allowlist/denylist
        """
        sid = session_id or self.session_id
        r = requests.post(f"{self.base_url}/sessions/{sid}/navigate", json={"url": url})
        return self._handle_response(r)
    
    # =========================================================================
    # SECTION 15: AUDIT LOG
    # =========================================================================
    
    def get_audit_log(self, session_id: str = None, limit: int = 100) -> Dict[str, Any]:
        """
        Get audit log.
        
        Capabilities demonstrated:
        - Tamper-evident logging
        - Timestamp, agent ID, action, parameters, results
        - Chain hash for integrity verification
        """
        sid = session_id or self.session_id
        r = requests.get(f"{self.base_url}/audit", params={"session_id": sid, "limit": limit})
        return self._handle_response(r)
    
    # =========================================================================
    # UTILITY METHODS
    # =========================================================================
    
    def _handle_response(self, response: requests.Response) -> Dict[str, Any]:
        """Handle HTTP response and return JSON."""
        try:
            return response.json()
        except:
            return {"error": True, "message": response.text}
    
    def pretty_print(self, data: Dict[str, Any], indent: int = 2) -> None:
        """Pretty print JSON data."""
        print(json.dumps(data, indent=indent))


# =============================================================================
# DEMO RUNNER
# =============================================================================

def print_section(title: str) -> None:
    """Print a section header."""
    print(f"\n{'='*60}")
    print(f"  {title}")
    print(f"{'='*60}\n")


def run_xcom_demo():
    """Run the comprehensive x.com demo."""
    
    print("\n" + "="*60)
    print("  AXON x.com COMPREHENSIVE DEMO")
    print("  Showcasing ALL Axon Capabilities")
    print("="*60)
    
    # Initialize client
    axon = AxonClient()
    
    # -------------------------------------------------------------------------
    # 1. Server Connection
    # -------------------------------------------------------------------------
    print_section("1. SERVER CONNECTION")
    
    print("Checking if Axon server is running...")
    if axon.check_server_health():
        print("[OK] Axon server is running on localhost:8020")
    else:
        print("[ERROR] Axon server not running. Start it with: go run cmd/axon/main.go")
        sys.exit(1)
    
    # -------------------------------------------------------------------------
    # 2. Session Management
    # -------------------------------------------------------------------------
    print_section("2. SESSION MANAGEMENT")
    
    # Create session
    session_id = "x_com_demo"
    print(f"Creating session: {session_id}")
    result = axon.create_session(session_id)
    print(f"[OK] Session created: {result}")
    
    # List sessions
    print("\nListing all sessions...")
    result = axon.list_sessions()
    print(f"Active sessions: {result.get('sessions', [])}")
    
    # -------------------------------------------------------------------------
    # 3. Navigation
    # -------------------------------------------------------------------------
    print_section("3. NAVIGATION")
    
    print("Navigating to x.com...")
    result = axon.navigate("https://x.com", wait_until="load")
    print(f"[OK] Navigated to: {result.get('title', 'Unknown')}")
    print(f"   URL: {result.get('url', 'Unknown')}")
    
    # Check session status
    print("\nGetting session status...")
    status = axon.get_session_status()
    print(f"   Auth State: {status.get('auth_state', 'unknown')}")
    print(f"   URL: {status.get('url', 'unknown')}")
    print(f"   Title: {status.get('title', 'unknown')}")
    
    # -------------------------------------------------------------------------
    # 4. Semantic Snapshots
    # -------------------------------------------------------------------------
    print_section("4. SEMANTIC SNAPSHOTS")
    
    # Compact snapshot
    print("Getting COMPACT snapshot (50-500 tokens)...")
    snapshot = axon.get_snapshot(depth="compact")
    print(f"[OK] Token count: {snapshot.get('token_count', 'N/A')}")
    print(f"   Page state: {snapshot.get('state', 'unknown')}")
    print(f"   Content preview:\n{snapshot.get('content', '')[:500]}...")
    
    # Standard snapshot
    print("\nGetting STANDARD snapshot (more detail)...")
    snapshot = axon.get_snapshot(depth="standard")
    print(f"[OK] Token count: {snapshot.get('token_count', 'N/A')}")
    
    # Full snapshot
    print("\nGetting FULL snapshot (complete data)...")
    snapshot = axon.get_snapshot(depth="full")
    print(f"[OK] Token count: {snapshot.get('token_count', 'N/A')}")
    
    # -------------------------------------------------------------------------
    # 5. Page State Detection
    # -------------------------------------------------------------------------
    print_section("5. PAGE STATE DETECTION")
    
    print("Detecting page state...")
    state = axon.get_page_state()
    print(f"   Detected state: {state.get('state', 'unknown')}")
    print(f"   Details: {state}")
    
    # -------------------------------------------------------------------------
    # 6. Screenshot Capture
    # -------------------------------------------------------------------------
    print_section("6. SCREENSHOT CAPTURE")
    
    print("Taking viewport screenshot...")
    result = axon.screenshot(full_page=False)
    print(f"[OK] Screenshot saved: {result.get('path', 'N/A')}")
    
    print("\nTaking full-page screenshot...")
    result = axon.screenshot(full_page=True)
    print(f"[OK] Full-page screenshot saved: {result.get('path', 'N/A')}")
    
    # -------------------------------------------------------------------------
    # 7. Cookie Management
    # -------------------------------------------------------------------------
    print_section("7. COOKIE MANAGEMENT")
    
    print("Getting cookies for x.com...")
    cookies = axon.get_cookies(domain=".x.com")
    cookie_list = cookies.get('cookies', []) or []
    print(f"   Found {len(cookie_list)} cookies")
    
    print("\nExporting cookies...")
    result = axon.export_cookies("x_session_backup.json")
    print(f"[OK] Cookies exported: {result}")
    
    # -------------------------------------------------------------------------
    # 8. Network Traffic Inspection
    # -------------------------------------------------------------------------
    print_section("8. NETWORK TRAFFIC INSPECTION")
    
    print("Getting API requests...")
    requests_data = axon.get_requests(filter="api")
    print(f"   Found {len(requests_data.get('requests', []))} API requests")
    
    # -------------------------------------------------------------------------
    # 9. Tab Management
    # -------------------------------------------------------------------------
    print_section("9. TAB MANAGEMENT")
    
    print("Opening new tab with Google...")
    result = axon.new_tab(url="https://google.com")
    print(f"[OK] New tab opened: {result}")
    
    print("\nListing tabs...")
    tabs = axon.list_tabs()
    print(f"   Open tabs: {tabs.get('tabs', [])}")
    
    print("\nClosing new tab...")
    result = axon.close_tab(index=1)
    print(f"[OK] Tab closed: {result}")
    
    # -------------------------------------------------------------------------
    # 10. Element Interaction (Demo with mock refs)
    # -------------------------------------------------------------------------
    print_section("10. ELEMENT INTERACTION")
    
    print("Note: Element refs would come from snapshot analysis.")
    print("Example - Filling compose box:")
    print("   axon.act(ref='e1', action='fill', value='Hello from Axon!')")
    print("   axon.act(ref='a1', action='click', confirm=True)  # Irreversible")
    
    # -------------------------------------------------------------------------
    # 11. Intent-Based Interaction
    # -------------------------------------------------------------------------
    print_section("11. INTENT-BASED INTERACTION")
    
    print("Note: These find elements by describing their function.")
    print("Examples:")
    print("   axon.find_and_act(intent='search box', action='fill', value='query')")
    print("   axon.find_and_act(intent='login button', action='click')")
    print("   axon.find_and_act(intent='email input', action='fill', value='user@email.com')")
    
    # -------------------------------------------------------------------------
    # 12. Error Handling Demo
    # -------------------------------------------------------------------------
    print_section("12. ERROR HANDLING")
    
    print("Demonstrating structured error handling...")
    print("If an element is not found, Axon returns:")
    print('   {')
    print('     "success": false,')
    print('     "error_type": "element_not_found",')
    print('     "message": "Element [ref=e5] not found. Page may have changed.",')
    print('     "suggestion": "Run axon_snapshot() to get fresh element refs.",')
    print('     "recoverable": true')
    print('   }')
    
    # -------------------------------------------------------------------------
    # 13. Security Features
    # -------------------------------------------------------------------------
    print_section("13. SECURITY FEATURES")
    
    print("SSRF Protection - Testing blocked URLs:")
    print("   Attempting to navigate to file:///etc/passwd...")
    result = axon.test_ssrf_protection("file:///etc/passwd")
    if not result.get("success", True):
        print(f"   [OK] Blocked: {result.get('message', 'SSRF protection active')}")
    
    print("\n   Attempting to navigate to 127.0.0.1...")
    result = axon.test_ssrf_protection("http://127.0.0.1:8080")
    if not result.get("success", True):
        print(f"   [OK] Blocked: {result.get('message', 'SSRF protection active')}")
    
    # -------------------------------------------------------------------------
    # 14. Audit Log
    # -------------------------------------------------------------------------
    print_section("14. AUDIT LOG")
    
    print("Retrieving audit log...")
    audit = axon.get_audit_log(limit=10)
    print(f"   Found {len(audit.get('entries', []))} audit entries")
    print("   (Each entry includes timestamp, agent ID, action, result, chain hash)")
    
    # -------------------------------------------------------------------------
    # 15. Wait Conditions
    # -------------------------------------------------------------------------
    print_section("15. WAIT CONDITIONS")
    
    print("Available wait conditions:")
    print("   axon.wait(condition='text:Tweet posted', timeout=10)")
    print("   axon.wait(condition='#compose-box', timeout=5)")
    print("   axon.wait(condition='networkidle', timeout=30)")
    
    # -------------------------------------------------------------------------
    # Cleanup
    # -------------------------------------------------------------------------
    print_section("CLEANUP")
    
    print("Closing session...")
    result = axon.close_session()
    print(f"[OK] Session closed: {result}")
    
    print("\n" + "="*60)
    print("  DEMO COMPLETE")
    print("="*60)
    print("\nThis demo showcased:")
    print("  [OK] Session Management")
    print("  [OK] Navigation with wait conditions")
    print("  [OK] Semantic Snapshots (compact/standard/full)")
    print("  [OK] Page State Detection")
    print("  [OK] Screenshot & PDF Export")
    print("  [OK] Cookie Management")
    print("  [OK] Network Traffic Inspection")
    print("  [OK] Tab Management")
    print("  [OK] Element Interaction (click, fill, press)")
    print("  [OK] Intent-Based Interaction")
    print("  [OK] Structured Error Handling")
    print("  [OK] Security Features (SSRF Protection)")
    print("  [OK] Audit Logging")
    print("  [OK] Wait Conditions")
    print("\n")


if __name__ == "__main__":
    run_xcom_demo()
