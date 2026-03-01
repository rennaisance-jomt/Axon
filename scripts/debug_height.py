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
    def status(self, sid): return self._req("GET", f"/sessions/{sid}/status")
    def screenshot(self, sid, full_page=False): 
        return self._req("POST", f"/sessions/{sid}/screenshot", {"full_page": full_page})

def main():
    axon = AxonClient()
    sid = "debug_height"
    
    print(f"Creating session {sid}...")
    axon.start_session(sid)
    
    print("Navigating...")
    res = axon.navigate(sid, "https://kuralit.com")
    print(f"Nav Result: {res}")
    
    time.sleep(2)
    
    print("Checking Status...")
    stat = axon.status(sid)
    print(f"Full Status: {json.dumps(stat, indent=2)}")

if __name__ == "__main__":
    main()
