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
    def act(self, sid, ref, action, value=None):
        return self._req("POST", f"/sessions/{sid}/act", {"ref": ref, "action": action, "value": str(value)})
    def status(self, sid): return self._req("GET", f"/sessions/{sid}/status")
    def screenshot(self, sid, full_page=False): 
        return self._req("POST", f"/sessions/{sid}/screenshot", {"full_page": full_page})

def main():
    axon = AxonClient()
    sid = "kuralit_full_smart_v2"
    
    print(f"Creating session {sid}...")
    axon.start_session(sid)
    
    print("Initial resize to 1280x800...")
    axon.resize(sid, 1280, 800)
    
    print("Navigating to Kuralit...")
    axon.navigate(sid, "https://kuralit.com", wait_until="networkidle")
    time.sleep(3)
    
    print("Waking up lazy loaders...")
    axon.act(sid, "", "scroll", value="1000")
    time.sleep(1)
    
    print("Checking page height...")
    stat = axon.status(sid)
    h = stat.get("scroll_height", 0)
    print(f"Detected scroll height: {h}px")
    
    if h > 800:
        target_h = min(h + 200, 8000)
        print(f"Resizing viewport to 1280x{target_h}...")
        axon.resize(sid, 1280, target_h)
        time.sleep(3)
        
        print("Taking final screenshot...")
        scr_res = axon.screenshot(sid, full_page=False)
        print(f"Path: {scr_res.get('path')}")
    else:
        print("Page height too small or not detected correctly.")

if __name__ == "__main__":
    main()
