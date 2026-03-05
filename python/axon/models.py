"""Axon Python SDK - Pydantic models for type safety."""

from typing import Optional, Any
from pydantic import BaseModel, Field


class SessionInfo(BaseModel):
    """Session information."""
    session_id: str = Field(alias="session_id")
    status: str
    profile: Optional[str] = None
    created_at: Optional[str] = None
    last_action: Optional[str] = None
    url: Optional[str] = None
    title: Optional[str] = None
    auth_state: Optional[str] = None
    page_state: Optional[str] = None

    class Config:
        populate_by_name = True


class CreateSessionRequest(BaseModel):
    """Request to create a session."""
    id: str
    profile: Optional[str] = None


class CreateSessionResponse(BaseModel):
    """Response from creating a session."""
    session_id: str = Field(alias="session_id")
    status: str
    profile: Optional[str] = None


class SnapshotElement(BaseModel):
    """An element in the accessibility tree with spatial metadata."""
    ref: str
    type: str
    label: str
    role: Optional[str] = None
    value: Optional[str] = None
    x: float = 0.0
    y: float = 0.0
    width: float = 0.0
    height: float = 0.0
    visible: bool = True
    enabled: bool = True
    intent: Optional[str] = None
    reversible: Optional[str] = None
    related_ref: Optional[str] = Field(None, alias="related_ref")
    vault_suggestion: Optional[str] = None
    
    class Config:
        populate_by_name = True


class SnapshotResponse(BaseModel):
    """Response from taking a snapshot."""
    session_id: str = Field(alias="session_id")
    url: str
    title: str
    elements: list[SnapshotElement] = Field(default_factory=list)
    page_state: Optional[str] = None
    captcha_detected: bool = False
    timestamp: Optional[str] = None
    token_count: int = 0
    content: Optional[str] = None
    
    class Config:
        populate_by_name = True


class ActionRequest(BaseModel):
    """Request to perform an action."""
    action: str
    ref: str
    value: Optional[str] = None
    confirm: bool = False


class ActionResponse(BaseModel):
    """Response from performing an action."""
    success: bool
    session_id: Optional[str] = Field(default=None, alias="session_id")
    action: Optional[str] = None
    message: Optional[str] = None
    error: Optional[str] = None
    result: Optional[str] = None
    requires_confirm: bool = Field(False, alias="requires_confirm")
    
    class Config:
        populate_by_name = True


class NavigateRequest(BaseModel):
    """Request to navigate to a URL."""
    url: str
    wait_until: str = "load"


class NavigateResponse(BaseModel):
    """Response from navigating."""
    session_id: Optional[str] = Field(default=None, alias="session_id")
    url: str
    success: bool
    title: Optional[str] = None
    state: Optional[str] = None
    
    class Config:
        populate_by_name = True


class ReplayFrame(BaseModel):
    """A single frame in a session replay."""
    timestamp: str
    data: str  # Base64 encoded image
    url: str
    metadata: dict[str, Any] = Field(default_factory=dict)


class ReplayResponse(BaseModel):
    """Response from getting a session replay."""
    session_id: str = Field(alias="session_id")
    frames: list[ReplayFrame] = Field(default_factory=list)
    
    class Config:
        populate_by_name = True


class APIError(BaseModel):
    """API error response."""
    error: bool
    error_type: str = Field(alias="error_type")
    message: str
    recoverable: bool


class SessionList(BaseModel):
    """List of sessions."""
    sessions: list[SessionInfo]
