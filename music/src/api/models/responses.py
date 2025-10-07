"""Response Models."""

from typing import Any

from pydantic import BaseModel


class HealthCheck(BaseModel):
    """Response model for health check endpoint."""

    status: str


class Message(BaseModel):
    """Response model for generic message."""

    message: str


class CreatePlaylist(BaseModel):
    """Response model for playlist creation."""

    playlist_id: str


class Success(BaseModel):
    """Response model for operations that return success status."""

    status: str
    result: Any


class Setup(BaseModel):
    """Response model for setup operations."""

    success: bool
    filepath: str
    message: str


class SetupWithContent(BaseModel):
    """Response model for setup operations that returns auth content."""

    success: bool
    filepath: str
    message: str
    auth_content: dict[str, Any]
