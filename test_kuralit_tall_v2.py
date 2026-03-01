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
                r = requests.get(url, params=data, timeout=120)
            else:
                r = requests.post(url, json=data, timeout=120)
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
    sid = "kuralit_full_tall"
    
    print(f"Creating session {sid}...")
    axon.start_session(sid)
    
    # 1. Use a standard desktop viewport (NO MORE STRETCHING)
    print("Setting standard viewport (1920x1080)...")
    axon.resize(sid, 1920, 1080)
    
    # 2. Navigate
    print("Navigating to Kuralit...")
    axon.navigate(sid, "https://kuralit.com", wait_until="networkidle")
    
    # 3. Take a ROBUST full-page screenshot
    # Axon will now automatically perform a discovery-scroll internally
    print("Taking robust full-page screenshot (with internal discovery scroll)...")
    scr_res = axon.screenshot(sid, full_page=True)
    print(f"Result: {scr_res}")
    
    if "path" in scr_res:
        print(f"Captured: {scr_res.get('path')}")

if __name__ == "__main__":
    main()
