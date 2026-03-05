#!/usr/bin/env python3
"""
Axon Resource Cleanup Verification
Proof that Axon's core engine properly terminates Chromium processes on context exit.
"""

import asyncio
import os
import sys
import time
import subprocess
import platform
import logging

from axon import Axon

# Configure logging
logging.basicConfig(level=logging.INFO, format="%(message)s")
logger = logging.getLogger("cleanup_verify")

def get_chromium_process_count():
    """Get the count of running Chromium/Chrome processes across platforms."""
    try:
        if platform.system() == "Windows":
            # Using tasklist to find chrome processes
            cmd = ["tasklist", "/fi", "imagename eq chrome*", "/fo", "csv", "/nh"]
            output = subprocess.check_output(cmd, stderr=subprocess.STDOUT).decode("utf-8").strip()
            processes = output.split("\n")
            return len([p for p in processes if p and "chrome" in p.lower()])
        else:  # Linux and macOS
            cmd = ["ps", "-ef"]
            output = subprocess.check_output(cmd).decode("utf-8")
            return len([line for line in output.split("\n") if "chrome" in line or "chromium" in line])
    except Exception as e:
        logger.debug(f"Error checking process count: {e}")
        return 0

async def verify_axon_cleanup():
    """Execute a session and confirm all associated browser processes are reaped."""
    logger.info("=== Axon Resource Cleanup Verification ===")
    
    # 1. Baseline check
    initial_count = get_chromium_process_count()
    logger.info(f"[*] Baseline: {initial_count} Chromium processes detected.")
    
    # 2. Run Axon session
    logger.info("[*] Initializing Axon engine and session...")
    async with Axon("http://localhost:8020/api/v1", start_engine=True) as axon:
        session_id = "cleanup-verification-id"
        await axon.create_session(session_id)
        
        # Perform some basic interactions to ensure browser is fully initialized
        await axon.navigate(session_id, "https://example.com")
        await axon.snapshot(session_id)
        
        active_count = get_chromium_process_count()
        logger.info(f"[+] Active: Session started. Current process count: {active_count}")
        
        # Explicitly delete session within the block (optional test)
        await axon.delete_session(session_id)
        logger.info("[+] Session explicitly deleted.")

    # 3. Post-exit verification
    logger.info("[*] Axon context manager exited. Waiting for process reaping...")
    
    # Give the OS a few seconds to clean up the process tree
    for i in range(3):
        time.sleep(1)
        final_count = get_chromium_process_count()
        if final_count <= initial_count:
            break
            
    logger.info(f"[*] Final Check: {final_count} Chromium processes remaining.")
    
    if final_count > initial_count:
        logger.error(f"\n❌ CLEANUP FAILED: {final_count - initial_count} zombie processes detected.")
        return False
    else:
        logger.info("\n✅ CLEANUP SUCCESSFUL: All resources were released properly.\n")
        return True

if __name__ == "__main__":
    try:
        success = asyncio.run(verify_axon_cleanup())
        sys.exit(0 if success else 1)
    except KeyboardInterrupt:
        logger.info("\nTest interrupted by user.")
        sys.exit(1)
    except Exception as e:
        logger.error(f"\nUnexpected error during verification: {e}")
        sys.exit(1)
