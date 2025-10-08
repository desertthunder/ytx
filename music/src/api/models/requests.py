"""Request & Response Models."""

from pydantic import BaseModel
from ytmusicapi import LikeStatus


class CreatePlaylist(BaseModel):
    """Model for creating a playlist."""

    title: str
    description: str
    privacy_status: str = "PRIVATE"


class EditPlaylist(BaseModel):
    """Model for editing a playlist."""

    title: str | None = None
    description: str | None = None


class AddPlaylistItems(BaseModel):
    """Model for adding items to a playlist."""

    video_ids: list[str]


class RmPlaylistItems(BaseModel):
    """Model for removing items from a playlist."""

    videos: list[dict[str, str]]


class RateSong(BaseModel):
    """Model for rating a song."""

    rating: LikeStatus


class SubscribeArtists(BaseModel):
    """Model for subscribing to artists."""

    channel_ids: list[str]


class BrowserSetup(BaseModel):
    """Model for browser authentication setup."""

    headers_raw: str
    filepath: str | None = None
