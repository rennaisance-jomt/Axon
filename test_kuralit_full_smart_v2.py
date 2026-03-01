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
            print(f"DEBUG: {method} {path} status: {r.status_code}")
            r.raise_for_status()
            return r.json()
        except requests.exceptions.RequestException as e:
            msg = e.response.text if e.response else str(e)
            return {"error": True, "message": msg}

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
    sid = "kuralit_full_smart_final"
    
    print(f"Creating session {sid}...")
    axon.start_session(sid)
    
    print("Step 1: Initial Desktop Viewport (1280x800)")
    axon.resize(sid, 1280, 800)
    
    print("Step 2: Navigating to Kuralit...")
    axon.navigate(sid, "https://kuralit.com", wait_until="networkidle")
    time.sleep(3)
    
    print("Step 3: Triggering Lazy Loading Content...")
    # Scroll down in chunks to ensure lazy images/components load
    for i in range(3):
        print(f"  Scrolling layer {i+1}...")
        axon.act(sid, "", "scroll", value="1000")
        time.sleep(1.5)
    
    print("Step 4: Detecting total Rendered Page Height...")
    stat = axon.status(sid)
    h = stat.get("scroll_height", 0)
    print(f"  Detected Height: {h}px")
    
    if h > 800:
        # Step 5: Expand Viewport to match Page Height
        # We add 200px buffer for safety
        final_h = min(h + 200, 10000) 
        print(f"Step 5: Expanding Viewport to 1280x{final_h} for 'Full Page' Snapshot...")
        axon.resize(sid, 1280, final_h)
        time.sleep(3) # Wait for layout to settle
        
        print("Step 6: Capturing High-Res Snapshot...")
        scr_res = axon.screenshot(sid, full_page=False)
        path = scr_res.get('path')
        print(f"  SUCCESS! Screenshot saved to: {path}")
        
        # Verify the file exists and is accessible
        if path:
            print(f"DONE: View the result at {path}")
    else:
        print("ERROR: Could not detect page height correctly. Height was 0 or too small.")

if __name__ == "__main__":
    main()
