import requests
import json
import time
import os

class AxonClient:
    def __init__(self, base_url="http://127.0.0.1:8020/api/v1"):
        self.base_url = base_url

    def _req(self, method, path, data=None):
        url = f"{self.base_url}{path}"
        try:
            if method == "GET":
                r = requests.get(url, params=data, timeout=60)
            else:
                r = requests.post(url, json=data, timeout=60)
            r.raise_for_status()
            return r.json()
        except requests.exceptions.RequestException as e:
            try:
                msg = e.response.json() if e.response else str(e)
            except:
                msg = str(e)
            return {"error": True, "message": msg}

    def start_session(self, sid): return self._req("POST", "/sessions", {"id": sid})
    def navigate(self, sid, url): return self._req("POST", f"/sessions/{sid}/navigate", {"url": url})
    def wait(self, sid, condition="networkidle", timeout=30000): 
        return self._req("POST", f"/sessions/{sid}/wait", {"condition": condition, "timeout": timeout})
    def snapshot(self, sid): return self._req("POST", f"/sessions/{sid}/snapshot", {"depth": "compact"})
    def screenshot(self, sid, full_page=False): 
        return self._req("POST", f"/sessions/{sid}/screenshot", {"full_page": full_page})
    def delete_session(self, sid): return requests.delete(f"{self.base_url}/sessions/{sid}")

def main():
    axon = AxonClient()
    
    sid = "kuralit_v2"
    print(f"Starting session {sid}...")
    axon.start_session(sid)
    
    print(f"Navigating to https://kuralit.com ...")
    # Kuralit.com usually loads fast, but SPAs need networkidle
    axon.navigate(sid, "https://kuralit.com")
    
    print(f"Waiting for networkidle...")
    wait_res = axon.wait(sid, condition="networkidle", timeout=15000)
    print(f"Wait Result: {wait_res}")
    
    # Extra buffer for animations
    time.sleep(3)
    
    print(f"Taking viewport screenshot (better for sticky headers)...")
    scr_res = axon.screenshot(sid, full_page=False)
    print(f"Screenshot Result: {scr_res}")
    
    print(f"Getting snapshot...")
    snap = axon.snapshot(sid)
    print(f"Snapshot Title: '{snap.get('title')}'")
    
    # Save results
    with open("kuralit_snapshot_v2.json", "w") as f:
        json.dump(snap, f, indent=2)
        
    print(f"Results saved. Screenshot at {scr_res.get('path')}")

if __name__ == "__main__":
    main()
