import requests

base = "http://127.0.0.1:8020/api/v1"
sid = "debug_session"

print(f"Creating session {sid}...")
r = requests.post(f"{base}/sessions", json={"id": sid})
print(f"Status: {r.status_code}")
print(f"Response: {r.text}")

if r.status_code in [201, 200]:
    print(f"Navigating...")
    r = requests.post(f"{base}/sessions/{sid}/navigate", json={"url": "https://example.com"})
    print(f"Status: {r.status_code}")
    print(f"Response: {r.text}")
else:
    print("Failed to create session.")
