import requests
import json
import time

class AxonClient:
    """Python SDK roughly simulating an LLM interacting with Axon."""
    def __init__(self, base_url="http://127.0.0.1:8020/api/v1"):
        self.base_url = base_url

    def _req(self, method, path, data=None):
        url = f"{self.base_url}{path}"
        try:
            if method == "GET":
                r = requests.get(url, params=data)
            else:
                r = requests.post(url, json=data)
            r.raise_for_status()
            return r.json()
        except requests.exceptions.RequestException as e:
            msg = e.response.json() if e.response else str(e)
            return {"error": True, "message": msg}

    def start_session(self, sid): return self._req("POST", "/sessions", {"id": sid})
    def navigate(self, sid, url): return self._req("POST", f"/sessions/{sid}/navigate", {"url": url})
    def snapshot(self, sid): return self._req("POST", f"/sessions/{sid}/snapshot", {"depth": "compact"})
    def act(self, sid, ref, action): return self._req("POST", f"/sessions/{sid}/act", {"ref": ref, "action": action, "confirm": True})


    def wait_for_server(self, timeout=30):
        print(f"Waiting for Axon server to be ready on {self.base_url}...")
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
    print("=== Axon AI Agent Example (Python) ===")
    axon = AxonClient()
    if not axon.wait_for_server():
        return
    
    sid = "agent_demo_1"
    
    print("\n[Orchestrator] Starting session...")
    axon.start_session(sid)
    
    print("\n[Orchestrator] Instructing Agent to Navigate to example.com")
    res = axon.navigate(sid, "https://example.com")
    print(f" -> Result: {res}")
    
    print("\n[Orchestrator] Agent asks for a semantic snapshot of the page...")
    snap = axon.snapshot(sid)
    elements = snap.get("elements", [])
    for el in elements:
        print(f"    - ID: {el.get('ref')} | Label: '{el.get('label')}' | Type: {el.get('type')}")
        
    if elements:
        target = elements[0].get('ref')
        print(f"\n[Orchestrator] Agent decides to click on element: {target}")
        act_res = axon.act(sid, target, "click")
        print(f" -> Result: {act_res}")
        
    print("\n=== Demo Complete ===")

if __name__ == "__main__":
    main()
