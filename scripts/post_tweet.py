#!/usr/bin/env python3
"""
Post a tweet on x.com using Axon with session cookies.
"""

import requests
import json
import sys
import time

# Enable UTF-8 for Windows
if sys.platform == 'win32':
    import io
    sys.stdout = io.TextIOWrapper(sys.stdout.buffer, encoding='utf-8', errors='replace')

BASE_URL = "http://localhost:8020/api/v1"
SESSION_ID = "x_post_session"

def check_server():
    """Check if Axon server is running."""
    try:
        r = requests.get("http://localhost:8020/health", timeout=5)
        return r.status_code == 200
    except:
        return False

def create_session():
    """Create a new session."""
    r = requests.post(f"{BASE_URL}/sessions", json={"id": SESSION_ID})
    print(f"Create session: {r.status_code}")
    return r.json()

def set_cookies_all():
    """Load and set all cookies at once from x_session.json."""
    with open("scripts/x_session.json", "r") as f:
        session_data = json.load(f)
    
    # Send all cookies as an array
    cookies = session_data["cookies"]
    r = requests.post(f"{BASE_URL}/sessions/{SESSION_ID}/cookies", json=cookies)
    print(f"Set cookies: {r.status_code} - {r.text}")
    return r.json()

def navigate():
    """Navigate to x.com."""
    r = requests.post(f"{BASE_URL}/sessions/{SESSION_ID}/navigate", json={
        "url": "https://x.com",
        "wait_until": "networkidle"
    })
    print(f"Navigate: {r.status_code}")
    result = r.json()
    print(f"  URL: {result.get('url', 'N/A')}")
    return result

def get_snapshot():
    """Get a semantic snapshot of the page."""
    r = requests.post(f"{BASE_URL}/sessions/{SESSION_ID}/snapshot", json={"depth": "standard"})
    print(f"Snapshot: {r.status_code}")
    result = r.json()
    print(f"  Token count: {result.get('token_count', 'N/A')}")
    print(f"  Content:\n{result.get('content', 'N/A')}")
    return result

def get_snapshot_full():
    """Get a full semantic snapshot to find element refs."""
    r = requests.post(f"{BASE_URL}/sessions/{SESSION_ID}/snapshot", json={"depth": "full"})
    print(f"Snapshot (full): {r.status_code}")
    result = r.json()
    return result

def screenshot():
    """Take a screenshot."""
    r = requests.post(f"{BASE_URL}/sessions/{SESSION_ID}/screenshot", json={"full_page": True})
    print(f"Screenshot: {r.status_code}")
    result = r.json()
    print(f"  Path: {result.get('path', 'N/A')}")
    return result

def act(ref, action, value=None, confirm=False):
    """Perform an action."""
    payload = {"ref": ref, "action": action, "confirm": confirm}
    if value:
        payload["value"] = value
    r = requests.post(f"{BASE_URL}/sessions/{SESSION_ID}/act", json=payload)
    print(f"Act {action} on {ref}: {r.status_code}")
    result = r.json()
    print(f"  Result: {json.dumps(result, indent=2)}")
    return result

def get_status():
    """Get session status."""
    r = requests.get(f"{BASE_URL}/sessions/{SESSION_ID}/status")
    result = r.json()
    print(f"  URL: {result.get('url', 'N/A')}")
    print(f"  Auth State: {result.get('auth_state', 'N/A')}")
    return result

def main():
    print("=" * 60)
    print("  Posting 'Hola' to x.com")
    print("=" * 60)
    
    # Check server
    print("\n[1] Checking server...")
    if not check_server():
        print("ERROR: Axon server not running!")
        sys.exit(1)
    print("OK: Server is running")
    
    # Create session
    print("\n[2] Creating session...")
    create_session()
    
    # Set cookies
    print("\n[3] Loading cookies...")
    set_cookies_all()
    
    # Navigate
    print("\n[4] Navigating to x.com...")
    navigate()
    time.sleep(3)  # Wait for page to fully load
    
    # Get status
    print("\n[5] Checking session status...")
    get_status()
    
    # Get snapshot to find elements
    print("\n[6] Getting page snapshot...")
    snapshot = get_snapshot()
    
    # Get full snapshot to find element refs
    print("\n[7] Getting full snapshot for element refs...")
    full_snapshot = get_snapshot_full()
    
    # Print full content to find refs
    print("\n[8] Analyzing page structure...")
    content = full_snapshot.get('content', '')
    print(f"Full content:\n{content}")
    
    # Take initial screenshot
    print("\n[9] Taking initial screenshot...")
    screenshot()
    
    # Try to find compose elements - common x.com refs
    # Based on the snapshot, we'll try to identify the compose box
    print("\n[10] Looking for compose elements...")
    
    # Check if we're logged in
    if "Sign up with Apple" in content or "Sign up" in content:
        print("NOT LOGGED IN - Cookies may have expired or invalid")
    else:
        print("Appears to be logged in - looking for compose box...")
    
    # Try to post (this is example - refs depend on actual page)
    print("\n[11] Taking final screenshot...")
    screenshot()
    
    print("\n" + "=" * 60)
    print("  DONE")
    print("=" * 60)
    print("\nNote: Cookies may have expired. For fresh cookies:")
    print("1. Log into x.com in a browser")
    print("2. Export cookies using browser DevTools")
    print("3. Update scripts/x_session.json with new cookies")

if __name__ == "__main__":
    main()
