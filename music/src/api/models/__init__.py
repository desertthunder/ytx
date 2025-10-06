"""Request & Response Models."""

from typing import Any

from pydantic import BaseModel


class CreatePlaylistReq(BaseModel):
    """Model for creating a playlist."""

    title: str
    description: str
    privacy_status: str = "PRIVATE"


class EditPlaylistReq(BaseModel):
    """Model for editing a playlist."""

    title: str | None = None
    description: str | None = None


class AddPlaylistItemsReq(BaseModel):
    """Model for adding items to a playlist."""

    video_ids: list[str]


class RmPlaylistItemsReq(BaseModel):
    """Model for removing items from a playlist."""

    videos: list[dict[str, str]]


class RateSongReq(BaseModel):
    """Model for rating a song."""

    rating: str  # LIKE, DISLIKE, INDIFFERENT


class SubscribeArtistsReq(BaseModel):
    """Model for subscribing to artists."""

    channel_ids: list[str]


class BrowserSetupReq(BaseModel):
    """Model for browser authentication setup."""

    headers_raw: str
    filepath: str = "browser.json"


class HealthCheckResp(BaseModel):
    """Response model for health check endpoint."""

    status: str


class MessageResp(BaseModel):
    """Generic message response model."""

    message: str


class CreatePlaylistResp(BaseModel):
    """Response model for playlist creation."""

    playlist_id: str


class SuccessResp(BaseModel):
    """Response model for operations that return success status."""

    status: str
    result: Any


class SetupResp(BaseModel):
    """Response model for setup operations."""

    success: bool
    filepath: str
    message: str
