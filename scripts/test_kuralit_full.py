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
                r = requests.get(url, params=data, timeout=30)
            else:
                r = requests.post(url, json=data, timeout=30)
            r.raise_for_status()
            return r.json()
        except requests.exceptions.RequestException as e:
            msg = e.response.json() if e.response else str(e)
            return {"error": True, "message": msg}

    def start_session(self, sid): return self._req("POST", "/sessions", {"id": sid})
    def navigate(self, sid, url): return self._req("POST", f"/sessions/{sid}/navigate", {"url": url})
    def snapshot(self, sid): return self._req("POST", f"/sessions/{sid}/snapshot", {"depth": "compact"})
    def screenshot(self, sid): return self._req("POST", f"/sessions/{sid}/screenshot", {"full_page": True})
    def delete_session(self, sid): return requests.delete(f"{self.base_url}/sessions/{sid}")

    def wait_for_server(self, timeout=30):
        print(f"Waiting for Axon server to be ready...")
        start_time = time.time()
        while time.time() - start_time < timeout:
            try:
                r = requests.get(f"http://127.0.0.1:8020/health")
                if r.status_code == 200:
                    print("Server is ready!")
                    return True
            except requests.exceptions.ConnectionError:
                pass
            time.sleep(1)
        print("Timeout waiting for server.")
        return False

def main():
    axon = AxonClient()
    if not axon.wait_for_server():
        return
    
    sid = "kuralit_test"
    print(f"Starting session {sid}...")
    axon.start_session(sid)
    
    print(f"Navigating to https://kuralit.com ...")
    nav_res = axon.navigate(sid, "https://kuralit.com")
    print(f"Navigate Response: {nav_res}")
    
    # Wait a bit for JS to render if needed
    time.sleep(2)
    
    print(f"Getting snapshot...")
    snap = axon.snapshot(sid)
    print(f"Snapshot Title: {snap.get('title')}")
    print(f"Snapshot Content Preview: {snap.get('content')[:500]}...")
    
    print(f"Taking screenshot...")
    scr_res = axon.screenshot(sid)
    print(f"Screenshot Result: {scr_res}")
    
    # Save the snapshot to a file for analysis
    with open("kuralit_snapshot.json", "w") as f:
        json.dump(snap, f, indent=2)
        
    print(f"Results saved to kuralit_snapshot.json and screenshot at {scr_res.get('path')}")

if __name__ == "__main__":
    main()
