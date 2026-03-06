"""Axon Python SDK - Async client for browser automation."""

import os
from typing import Optional
import aiohttp
from urllib.parse import urlsplit

from .models import (
    SessionInfo,
    CreateSessionResponse,
    SnapshotResponse,
    ActionResponse,
    NavigateResponse,
    ReplayResponse,
    SessionList,
    APIError,
)
from .engine import AxonEngine


class Axon:
    """
    Async Axon client for browser automation.
    
    Example usage:
    
    ```python
    import asyncio
    from axon import Axon
    
    async def main():
        async with Axon("http://localhost:8020/api/v1") as axon:
            # Create a session
            session = await axon.create_session("mysession")
            
            # Navigate to a URL
            await axon.navigate("mysession", "https://github.com")
            
            # Get snapshot
            snapshot = await axon.snapshot("mysession")
            print(snapshot.title)
            
            # Perform action
            result = await axon.act("mysession", "click", "e1")
    
    asyncio.run(main())
    ```
    """

    def __init__(
        self,
        api_url: Optional[str] = None,
        timeout: float = 30.0,
        start_engine: bool = False,
        binary_path: Optional[str] = None,
        config_path: Optional[str] = None,
    ):
        """
        Initialize the Axon client.
        
        Args:
            api_url: Base URL for the Axon API. 
            timeout: Request timeout in seconds.
            start_engine: If True, automatically start the Axon engine if not running.
            binary_path: Path to the Axon binary (used if start_engine is True).
            config_path: Path to the Axon config file (used if start_engine is True).
        """
        self.api_url = api_url or os.getenv("AXON_API_URL", "http://localhost:8020/api/v1")
        self.timeout = aiohttp.ClientTimeout(total=timeout)
        self._session: Optional[aiohttp.ClientSession] = None
        self.engine: Optional[AxonEngine] = None
        
        if start_engine:
            # Parse port from api_url if possible
            port = 8020
            if ":" in self.api_url:
                try:
                    port_part = self.api_url.split(":")[-1].split("/")[0]
                    port = int(port_part)
                except: pass
            
            self.engine = AxonEngine(binary_path=binary_path, config_path=config_path, port=port)
            self.engine.start()

    async def __aenter__(self) -> "Axon":
        """Async context manager entry."""
        self._session = aiohttp.ClientSession(timeout=self.timeout)
        return self

    async def __aexit__(self, exc_type, exc_val, exc_tb) -> None:
        """Async context manager exit."""
        if self.engine:
            # Try multiple endpoint paths to ensure we hit one that works
            endpoints = [
                "/internal/shutdown/sync",
                "/api/v1/internal/shutdown/sync",
                "/internal/shutdown",
                "/api/v1/internal/shutdown"
            ]
            
            success = False
            for endpoint in endpoints:
                try:
                    print(f"Sending {endpoint} request to clean up Chromium...")
                    result = await self._request("POST", endpoint)
                    print(f"Shutdown successful via {endpoint}: {result}")
                    success = True
                    break
                except Exception as e:
                    print(f"Endpoint {endpoint} failed: {e}")
            
            # Even if all API endpoints fail, add an extra sleep to allow cleanup
            import asyncio
            if success:
                # Longer wait when successful to ensure proper cleanup
                await asyncio.sleep(2.0)
            else:
                # If all endpoints failed, wait even longer
                print("All shutdown endpoints failed. Waiting for natural process termination...")
                await asyncio.sleep(3.0)
                
            # Additional direct cleanup attempt using system commands
            self._direct_cleanup_chromium()

        if self._session:
            await self._session.close()
            
        if self.engine:
            self.engine.stop()
            
    def _direct_cleanup_chromium(self):
        """Direct system-level cleanup of Chrome processes as a last resort."""
        import platform
        import subprocess
        import os
        
        print("Performing direct system-level Chrome process cleanup...")
        
        try:
            system = platform.system()
            if system == "Windows":
                # Windows cleanup
                subprocess.run(["taskkill", "/F", "/IM", "chrome.exe", "/T"], 
                               stdout=subprocess.PIPE, stderr=subprocess.PIPE, check=False)
                subprocess.run(["taskkill", "/F", "/IM", "chromium.exe", "/T"], 
                               stdout=subprocess.PIPE, stderr=subprocess.PIPE, check=False)
                # Also try wmic for stubborn processes
                subprocess.run(["wmic", "process", "where", "name like '%chrome%'", "delete"], 
                               stdout=subprocess.PIPE, stderr=subprocess.PIPE, check=False)
            elif system == "Darwin":  # macOS
                subprocess.run(["pkill", "-9", "Chrome"], 
                               stdout=subprocess.PIPE, stderr=subprocess.PIPE, check=False)
            elif system == "Linux":
                subprocess.run(["pkill", "-9", "-f", "chrom"], 
                               stdout=subprocess.PIPE, stderr=subprocess.PIPE, check=False)
            
            print("Direct cleanup completed")
        except Exception as e:
            print(f"Error during direct cleanup: {e}")

    async def _request(
        self,
        method: str,
        path: str,
        **kwargs,
    ) -> dict:
        """Make an API request."""
        # Normalize URL building so we don't accidentally produce duplicated
        # paths like /api/v1/api/v1/... when callers provide absolute API paths.
        parsed = urlsplit(self.api_url)
        base_origin = f"{parsed.scheme}://{parsed.netloc}"

        if path.startswith("/internal/") or path.startswith("/api/"):
            # Absolute service path (already rooted at server origin)
            url = f"{base_origin}{path}"
        else:
            # Relative API path under configured api_url
            url = f"{self.api_url}{path}"
            
        async with self._session.request(method, url, **kwargs) as response:
            if response.status >= 400:
                error_text = await response.text()
                raise AxonError(
                    f"API error ({response.status}): {error_text}",
                    status_code=response.status,
                )
            if response.status == 204:
                return {}
            return await response.json()

    # Session Management

    async def create_session(
        self,
        session_id: str,
        profile: Optional[str] = None,
    ) -> CreateSessionResponse:
        """
        Create a new browser session.
        
        Args:
            session_id: Unique identifier for the session.
            profile: Optional browser profile name.
            
        Returns:
            CreateSessionResponse with session details.
        """
        data = {"id": session_id}
        if profile:
            data["profile"] = profile
        
        result = await self._request("POST", "/sessions", json=data)
        return CreateSessionResponse(**result)

    async def get_session(self, session_id: str) -> SessionInfo:
        """
        Get session information.
        
        Args:
            session_id: The session ID.
            
        Returns:
            SessionInfo with session details.
        """
        result = await self._request("GET", f"/sessions/{session_id}")
        return SessionInfo(**result)

    async def list_sessions(self) -> SessionList:
        """
        List all active sessions.
        
        Returns:
            SessionList containing all sessions.
        """
        result = await self._request("GET", "/sessions")
        return SessionList(**result)

    async def delete_session(self, session_id: str) -> None:
        """
        Delete a session.
        
        Args:
            session_id: The session ID to delete.
        """
        await self._request("DELETE", f"/sessions/{session_id}")

    # Navigation

    async def navigate(self, session_id: str, url: str, wait_until: str = "load") -> NavigateResponse:
        """
        Navigate to a URL.
        
        Args:
            session_id: The session ID.
            url: The URL to navigate to.
            wait_until: Condition to wait for (none, load, domcontentloaded, networkidle).
            
        Returns:
            NavigateResponse with navigation result.
        """
        result = await self._request(
            "POST",
            f"/sessions/{session_id}/navigate",
            json={"url": url, "wait_until": wait_until},
        )
        return NavigateResponse(**result)

    # Snapshot

    async def snapshot(
        self,
        session_id: str,
        ref: Optional[str] = None,
    ) -> SnapshotResponse:
        """
        Get a snapshot of the current page.
        
        Args:
            session_id: The session ID.
            ref: Optional element reference to focus on.
            
        Returns:
            SnapshotResponse with page elements.
        """
        data = {}
        if ref:
            data["ref"] = ref
        
        result = await self._request(
            "POST",
            f"/sessions/{session_id}/snapshot",
            json=data,
        )
        return SnapshotResponse(**result)

    # Actions

    async def act(
        self,
        session_id: str,
        action: str,
        ref: str,
        value: Optional[str] = None,
        confirm: bool = False,
    ) -> ActionResponse:
        """
        Perform an action on an element.
        
        Args:
            session_id: The session ID.
            action: Action to perform (click, fill, hover, select, etc.)
            ref: Element reference ID.
            value: Value for fill/select actions.
            confirm: Confirm irreversible action.
            
        Returns:
            ActionResponse with result.
        """
        data = {
            "action": action,
            "ref": ref,
            "confirm": confirm,
        }
        if value is not None:
            data["value"] = value
        
        result = await self._request(
            "POST",
            f"/sessions/{session_id}/act",
            json=data,
        )
        return ActionResponse(**result)

    async def click(self, session_id: str, ref: str) -> ActionResponse:
        """Click an element."""
        return await self.act(session_id, "click", ref)

    async def fill(
        self,
        session_id: str,
        ref: str,
        value: str,
    ) -> ActionResponse:
        """Fill an input field."""
        return await self.act(session_id, "fill", ref, value=value)

    async def vault_fill(
        self,
        session_id: str,
        ref: str,
        secret_name: str,
        field: str = "password",
    ) -> ActionResponse:
        """
        Fill an input field using a secret from the Intelligence Vault.
        
        Args:
            session_id: The session ID.
            ref: Element reference ID.
            secret_name: Name of the secret in the vault.
            field: Field name to inject (username, password, value).
            
        Returns:
            ActionResponse with result.
        """
        vault_ref = f"@vault:{secret_name}:{field}"
        return await self.fill(session_id, ref, vault_ref)

    # Vault Management

    async def add_secret(
        self,
        name: str,
        value: str,
        url: str,
        username: Optional[str] = None,
        password: Optional[str] = None,
        labels: Optional[list] = None,
    ) -> bool:
        """
        Add a secret to the Intelligence Vault.
        
        Args:
            name: Friendly name for the secret.
            value: Secret value (for generic secrets).
            url: Domain/URL the secret is bound to.
            username: Optional username.
            password: Optional password.
            labels: Optional labels for categorization.
            
        Returns:
            True if successful.
        """
        data = {
            "name": name,
            "value": value,
            "url": url,
            "username": username,
            "password": password,
            "labels": labels or [],
        }
        result = await self._request("POST", "/vault/secrets", json=data)
        return result.get("success", False)

    async def list_secrets(self) -> list:
        """
        List all secrets in the Intelligence Vault.
        
        Returns:
            List of secret metadata.
        """
        result = await self._request("GET", "/vault/secrets")
        return result.get("secrets", [])

    async def delete_secret(self, name: str) -> bool:
        """
        Delete a secret from the Intelligence Vault.
        
        Args:
            name: The name of the secret to delete.
            
        Returns:
            True if successful.
        """
        result = await self._request("DELETE", f"/vault/secrets/{name}")
        return result.get("success", False)

    async def hover(self, session_id: str, ref: str) -> ActionResponse:
        """Hover over an element."""
        return await self.act(session_id, "hover", ref)

    async def select(
        self,
        session_id: str,
        ref: str,
        value: str,
    ) -> ActionResponse:
        """Select an option."""
        return await self.act(session_id, "select", ref, value=value)

    async def press(
        self,
        session_id: str,
        ref: str,
        key: str,
    ) -> ActionResponse:
        """Press a key."""
        return await self.act(session_id, "press", ref, value=key)

    # Smart Agent Tools
    
    async def smart_interact(
        self,
        session_id: str,
        intent: str,
        action: str = "click",
        value: Optional[str] = None,
        require_confirm: bool = True
    ) -> ActionResponse:
        """
        High-level interaction tool for agents.
        Automatically resolves intent and handles safety checks.
        """
        return await self.find_and_act(
            session_id=session_id,
            action=action,
            intent=intent,
            value=value
        )

    # Find and Act

    async def find_and_act(
        self,
        session_id: str,
        action: str,
        intent: str,
        value: Optional[str] = None,
    ) -> ActionResponse:
        """
        Find an element by intent/description and perform an action.
        
        Args:
            session_id: The session ID.
            action: Action to perform.
            intent: Semantic intent or description of the element.
            value: Optional value for the action.
            
        Returns:
            ActionResponse with result.
        """
        data = {
            "action": action,
            "intent": intent,
        }
        if value is not None:
            data["value"] = value
        
        result = await self._request(
            "POST",
            f"/sessions/{session_id}/find_and_act",
            json=data,
        )
        return ActionResponse(**result)

    # Replay

    async def replay(self, session_id: str) -> ReplayResponse:
        """
        Get a replay of the session history.
        
        Args:
            session_id: The session ID.
            
        Returns:
            ReplayResponse with historical frames and metadata.
        """
        result = await self._request("GET", f"/sessions/{session_id}/replay")
        return ReplayResponse(**result)

    # Status

    async def status(self, session_id: str) -> dict:
        """
        Get session status.
        
        Args:
            session_id: The session ID.
            
        Returns:
            Status information.
        """
        return await self._request("GET", f"/sessions/{session_id}/status")


class AxonError(Exception):
    """Axon API error."""

    def __init__(self, message: str, status_code: int = 0):
        super().__init__(message)
        self.status_code = status_code