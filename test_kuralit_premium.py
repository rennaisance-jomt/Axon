import requests
import json
import time

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
            return {"error": True, "message": str(e)}

    def start_session(self, sid): return self._req("POST", "/sessions", {"id": sid})
    def navigate(self, sid, url, wait_until="networkidle"): 
        return self._req("POST", f"/sessions/{sid}/navigate", {"url": url, "wait_until": wait_until})
    def resize(self, sid, width, height):
        return self._req("POST", f"/sessions/{sid}/resize", {"width": width, "height": height})
    def screenshot(self, sid, full_page=False): 
        return self._req("POST", f"/sessions/{sid}/screenshot", {"full_page": full_page})
    def snapshot(self, sid): return self._req("POST", f"/sessions/{sid}/snapshot", {"depth": "compact"})

def main():
    axon = AxonClient()
    sid = "kuralit_premium"
    
    print(f"Starting session {sid}...")
    axon.start_session(sid)
    
    # 1. Set HD resolution first
    print("Resizing to 1920x1080 for premium capture...")
    axon.resize(sid, 1920, 1080)
    
    # 2. Navigate with networkidle wait (now supported natively)
    print("Navigating to https://kuralit.com with networkidle...")
    nav_res = axon.navigate(sid, "https://kuralit.com", wait_until="networkidle")
    print(f"Navigate Title: '{nav_res.get('title')}'")
    
    # 3. Buffer for any final layout shifts
    time.sleep(2)
    
    # 4. Take many screenshots to ensure we get a good one
    print("Taking final premium screenshot...")
    scr_res = axon.screenshot(sid, full_page=False)
    print(f"Final Screenshot: {scr_res.get('path')}")
    
    # 5. Get snapshot and verify title
    snap = axon.snapshot(sid)
    print(f"Snapshot Title: '{snap.get('title')}'")
    
    with open("kuralit_snapshot_premium.json", "w") as f:
        json.dump(snap, f, indent=2)

if __name__ == "__main__":
    main()
