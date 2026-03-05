import os
import subprocess
import time
import socket
import logging
from pathlib import Path
from typing import Optional

logger = logging.getLogger("axon.engine")

class AxonEngine:
    """
    Manages the Axon browser engine process.
    Handles starting, stopping, and health checking of the Go binary.
    """
    
    def __init__(
        self,
        binary_path: Optional[str] = None,
        config_path: Optional[str] = None,
        port: int = 8020,
        host: str = "127.0.0.1"
    ):
        self.port = port
        self.host = host
        self.process: Optional[subprocess.Popen] = None
        
        # Determine binary path
        if binary_path:
            self.binary_path = Path(binary_path)
        else:
            # Try to find axon.exe in the package or current directory
            base_dir = Path(__file__).parent.parent
            potential_paths = [
                base_dir / "bin" / "axon.exe",
                base_dir / "axon.exe",
                Path("axon.exe"),
                Path("./bin/axon.exe")
            ]
            self.binary_path = None
            for p in potential_paths:
                if p.exists():
                    self.binary_path = p
                    break
                    
        self.config_path = config_path
        
    def is_running(self) -> bool:
        """Check if the port is open."""
        with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as s:
            return s.connect_ex((self.host, self.port)) == 0

    def start(self, timeout: int = 15):
        """Start the Axon engine process."""
        if self.is_running():
            logger.info(f"Axon engine already running on {self.host}:{self.port}")
            return

        if not self.binary_path or not self.binary_path.exists():
            raise FileNotFoundError(f"Axon binary not found at {self.binary_path}. Please provide a valid path.")

        cmd = [str(self.binary_path)]
        if self.config_path:
            cmd.extend(["--config", self.config_path])
        
        logger.info(f"Starting Axon engine: {' '.join(cmd)}")
        
        # Start the process
        # NOTE: Showing stdout/stderr for debugging browser cleanup issues
        self.process = subprocess.Popen(
            cmd,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            creationflags=subprocess.CREATE_NO_WINDOW if os.name == 'nt' else 0
        )
        
        # Read output in background thread for logging
        def read_output(pipe, prefix):
            for line in pipe:
                print(f"{prefix}: {line.decode().strip()}")
        
        import threading
        threading.Thread(target=read_output, args=(self.process.stdout, "STDOUT"), daemon=True).start()
        threading.Thread(target=read_output, args=(self.process.stderr, "STDERR"), daemon=True).start()
        
        # Wait for engine to be ready
        start_time = time.time()
        while time.time() - start_time < timeout:
            if self.is_running():
                logger.info("Axon engine started successfully.")
                return
            time.sleep(0.5)
            
        self.stop()
        raise TimeoutError("Timed out waiting for Axon engine to start.")

    def stop(self):
        """Stop the Axon engine process."""
        if self.process:
            logger.info("Stopping Axon engine process...")
            self.process.terminate()
            try:
                self.process.wait(timeout=5)
            except subprocess.TimeoutExpired:
                self.process.kill()
            self.process = None
            logger.info("Axon engine stopped.")
        elif self.is_running():
            logger.warning("Axon engine is running but was not started by this instance. Cannot stop it.")
