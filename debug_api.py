import requests
import json

r = requests.get("http://localhost:8020/api/v1/sessions")
print(f"Sessions: {r.json()}")

# Try creating a session and see the response
payload = {"id": "test_debug"}
r = requests.post("http://localhost:8020/api/v1/sessions", json=payload)
print(f"Create Response ({r.status_code}): {r.text}")
