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
    def screenshot(self, sid, full_page=True): 
        return self._req("POST", f"/sessions/{sid}/screenshot", {"full_page": full_page})
    def snapshot(self, sid): return self._req("POST", f"/sessions/{sid}/snapshot", {"depth": "compact"})

def main():
    axon = AxonClient()
    sid = "kuralit_full_v3"
    
    print(f"Starting session {sid}...")
    axon.start_session(sid)
    
    print("Setting viewport to 1280x800...")
    axon.resize(sid, 1280, 800)
    
    print("Navigating to https://kuralit.com...")
    axon.navigate(sid, "https://kuralit.com", wait_until="networkidle")
    
    # Wait for initial animations
    time.sleep(2)
    
    print("Scrolling to trigger lazy loads...")
    # Scroll down 3 times using the body as ref or just selector-less
    # Since ref "" is targetting window.scrollBy
    for i in range(3):
        print(f"  Scroll {i+1}...")
        axon.act(sid, "", "scroll", value="800")
        time.sleep(1)
    
    # Scroll back to top to ensure sticky headers are in place (optional)
    # axon.act(sid, "", "scroll", value="-5000")
    # time.sleep(1)
    
    print("Taking FULL PAGE screenshot...")
    scr_res = axon.screenshot(sid, full_page=True)
    print(f"Full Page Screenshot Result: {scr_res}")
    
    if "error" not in scr_res:
        path = scr_res.get('path')
        print(f"Captured: {path}")

if __name__ == "__main__":
    main()
