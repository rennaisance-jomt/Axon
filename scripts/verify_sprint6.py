import requests
import json
import time
import sys

# Define base URL
base_url = "http://127.0.0.1:8020/api/v1"

def print_step(msg):
    print(f"\n\033[96m=== {msg} ===\033[0m")

def check_health():
    resp = requests.get("http://127.0.0.1:8020/health")
    resp.raise_for_status()
    print("Health check OK:", resp.json())

def create_session(session_id="test_session"):
    print_step(f"Creating Session: {session_id}")
    resp = requests.post(f"{base_url}/sessions", json={"id": session_id})
    if resp.status_code == 201:
        print("Session created:", resp.json())
    elif resp.status_code == 409:
        print("Session already exists.")
    else:
        resp.raise_for_status()

def navigate(session_id, url):
    print_step(f"Navigating to {url}")
    resp = requests.post(f"{base_url}/sessions/{session_id}/navigate", json={"url": url})
    resp.raise_for_status()
    print("Navigation successful:", resp.json())
    time.sleep(2) # brief pause to let page stabilize

def get_snapshot(session_id):
    print_step("Getting Semantic Snapshot")
    resp = requests.post(f"{base_url}/sessions/{session_id}/snapshot", json={"depth": "compact"})
    resp.raise_for_status()
    snapshot = resp.json()
    print("--- SNAPSHOT CONTENT ---")
    print(snapshot.get("content", "No content returned!"))
    print("------------------------")
    print("RAW JSON:", json.dumps(snapshot, indent=2))
    return snapshot

def click_element(session_id, ref):
    print_step(f"Clicking Element REF: {ref}")
    resp = requests.post(f"{base_url}/sessions/{session_id}/act", json={
        "action": "click",
        "ref": ref
    })
    
    if resp.status_code == 200:
        print("Action response:", resp.json())
    else:
        print("Action failed:", resp.json())

if __name__ == "__main__":
    sid = "test_sprint6"
    try:
        check_health()
        create_session(sid)
        navigate(sid, "https://en.wikipedia.org/wiki/Main_Page")
        snap = get_snapshot(sid)
        
        # Try to find the "Contents" link
        ref_to_click = "None"
        for elem in snap.get("elements", []):
            if elem.get("label") == "Contents":
                ref_to_click = elem.get("ref")
                break
                
        if ref_to_click != "None":
            click_element(sid, ref_to_click)
            time.sleep(1) # wait for nav
            print_step("Taking Snapshot After Click")
            post_click_snap = get_snapshot(sid)
        else:
            print("Could not find the 'Contents' link ref in the snapshot.")
            
        print_step("Sprint 6 Verification Complete!")
    except Exception as e:
        print("Error during test:", e)
        sys.exit(1)
