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
            msg = e.response.text if e.response else str(e)
            return {"error": True, "message": msg}

    def start_session(self, sid): return self._req("POST", "/sessions", {"id": sid})
    def navigate(self, sid, url, wait_until="networkidle"): 
        return self._req("POST", f"/sessions/{sid}/navigate", {"url": url, "wait_until": wait_until})
    def resize(self, sid, width, height):
        return self._req("POST", f"/sessions/{sid}/resize", {"width": width, "height": height})
    def act(self, sid, ref, action, value=None):
        return self._req("POST", f"/sessions/{sid}/act", {"ref": ref, "action": action, "value": str(value)})
    def screenshot(self, sid, full_page=True): 
        return self._req("POST", f"/sessions/{sid}/screenshot", {"full_page": full_page})

def main():
    axon = AxonClient()
    sid = "kuralit_full_final_v4"
    
    print(f"Creating session {sid}...")
    res = axon.start_session(sid)
    if "error" in res:
        print(f"Error: {res}")
        return

    print("Setting HD resolution...")
    axon.resize(sid, 1920, 1080)
    
    print("Navigating to Kuralit...")
    axon.navigate(sid, "https://kuralit.com", wait_until="networkidle")
    
    print("Scrolling to bottom to trigger all animations/lazy loads...")
    for i in range(5):
        axon.act(sid, "", "scroll", value="800")
        time.sleep(1)
        
    print("Wait 3s for stabilization...")
    time.sleep(3)
    
    print("Taking TRUE FULL PAGE screenshot...")
    scr_res = axon.screenshot(sid, full_page=True)
    print(f"Screenshot Result: {scr_res}")
    
    if "path" in scr_res:
        print(f"Success! Captured: {scr_res.get('path')}")

if __name__ == "__main__":
    main()
