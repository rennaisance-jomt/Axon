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

def main():
    axon = AxonClient()
    sid = "kuralit_long_viewport"
    
    print(f"Starting session {sid}...")
    axon.start_session(sid)
    
    # Use a Very Tall Viewport to capture more content without the "Full Page" bug
    print("Setting ULTRA TALL viewport (1280x4000)...")
    axon.resize(sid, 1280, 4000)
    
    print("Navigating...")
    axon.navigate(sid, "https://kuralit.com", wait_until="networkidle")
    
    time.sleep(3)
    
    print("Taking screenshot...")
    scr_res = axon.screenshot(sid, full_page=False)
    print(f"Screenshot Result: {scr_res}")

if __name__ == "__main__":
    main()
