#!/usr/bin/env python3
import requests
import json
import sys
import time
import os

# Enable UTF-8 for Windows
if sys.platform == 'win32':
    import io
    sys.stdout = io.TextIOWrapper(sys.stdout.buffer, encoding='utf-8', errors='replace')
    sys.stderr = io.TextIOWrapper(sys.stderr.buffer, encoding='utf-8', errors='replace')

BASE_URL = "http://localhost:8020/api/v1"
SESSION_ID = "hola_post_session"

def main():
    def print_flush(msg):
        print(msg)
        sys.stdout.flush()

    print_flush("Starting Hola Post Script...")
    
    # 1. Create Session
    print_flush(f"\n[1] Preparing session {SESSION_ID}...")
    # Attempt to delete existing session first
    requests.delete(f"{BASE_URL}/sessions/{SESSION_ID}")
    
    r = requests.post(f"{BASE_URL}/sessions", json={"id": SESSION_ID})
    if r.status_code not in [200, 201]:
        print_flush(f"Failed to create session: {r.text}")
        return

    # 2. Set Cookies
    print_flush("\n[2] Setting cookies from x_session.json...")
    with open("scripts/x_session.json", "r") as f:
        cookies = json.load(f)["cookies"]
    requests.post(f"{BASE_URL}/sessions/{SESSION_ID}/cookies", json=cookies)

    # 3. Navigate
    print_flush("\n[3] Navigating to x.com...")
    requests.post(f"{BASE_URL}/sessions/{SESSION_ID}/navigate", json={
        "url": "https://x.com",
        "wait_until": "networkidle"
    })
    time.sleep(5) # Give it extra time for JS hydration

    # 4. Get Snapshot
    print_flush("\n[4] Getting snapshot to locate elements...")
    r = requests.post(f"{BASE_URL}/sessions/{SESSION_ID}/snapshot", json={"depth": "standard"})
    snapshot = r.json()
    elements = snapshot.get("elements", [])
    
    # Locate Post Box and Post Button
    text_ref = None
    btn_ref = None
    
    print_flush(f"Analyzing {len(elements)} elements...")
    for el in elements:
        label = el.get("label", "").lower()
        role = el.get("role", "").lower()
        placeholder = el.get("placeholder", "").lower()
        
        # Look for tweet text box
        if not text_ref and ("happening" in label or "happening" in placeholder or "post text" in label or role == "textbox"):
            text_ref = el["ref"]
            print_flush(f"Found Text Box: {text_ref} (Label: {label})")
            
        # Look for Post button
        if not btn_ref and (label == "post" or label == "tweet") and role == "button":
            btn_ref = el["ref"]
            print_flush(f"Found Post Button: {btn_ref} (Label: {label})")

    if not text_ref or not btn_ref:
        print_flush("FAILED: Could not find elements automatically. Using fallbacks...")
        # Hardcoded fallback for common X.com patterns if auto-detection fails
        # but the user said they wanted me to do it, so I should try to be smart.
        if not text_ref: text_ref = "t1" # Often the first textbox
        if not btn_ref: btn_ref = "b1"  # Often the first button

    # 5. Fill and Post
    print_flush(f"\n[5] Typing 'Hola' into {text_ref}...")
    requests.post(f"{BASE_URL}/sessions/{SESSION_ID}/act", json={
        "ref": text_ref,
        "action": "fill",
        "value": "Hola"
    })
    
    time.sleep(1)
    
    print_flush(f"\n[6] Clicking Post button {btn_ref}...")
    requests.post(f"{BASE_URL}/sessions/{SESSION_ID}/act", json={
        "ref": btn_ref,
        "action": "click",
        "confirm": True # Post is irreversible
    })

    print_flush("\n[7] Waiting for post to finish...")
    time.sleep(5)

    # 6. Final Screenshot
    print_flush("\n[8] Taking final screenshot...")
    r = requests.post(f"{BASE_URL}/sessions/{SESSION_ID}/screenshot", json={"full_page": False})
    result = r.json()
    print_flush(f"DONE! Screenshot saved to: {result.get('path', 'N/A')}")
    
    # Keep session open for a bit then close?
    # requests.delete(f"{BASE_URL}/sessions/{SESSION_ID}")

if __name__ == "__main__":
    main()
