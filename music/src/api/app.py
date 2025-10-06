"""FastAPI proxy server for YouTube Music API."""

from pathlib import Path
from typing import Annotated, Any

from fastapi import Depends, FastAPI, Header, HTTPException, UploadFile, status
from fastapi.responses import JSONResponse
from ytmusicapi import YTMusic
from ytmusicapi import setup as ytmusic_setup

from .models import (
    AddPlaylistItemsReq,
    BrowserSetupReq,
    CreatePlaylistReq,
    CreatePlaylistResp,
    EditPlaylistReq,
    HealthCheckResp,
    MessageResp,
    RateSongReq,
    RmPlaylistItemsReq,
    SetupResp,
    SubscribeArtistsReq,
    SuccessResp,
)

app = FastAPI(title="YouTube Music Proxy API", version="0.1.0")


def get_ytmusic(x_auth_file: Annotated[str | None, Header()] = None) -> YTMusic:
    """Create YTMusic client instance based on authentication header.

    Args:
        x_auth_file: Optional path to authentication file (browser.json or oauth.json)

    Returns:
        Authenticated or unauthenticated YTMusic instance

    Raises:
        HTTPException: If auth file path is invalid
    """
    if x_auth_file:
        auth_path = Path(x_auth_file)
        if not auth_path.exists():
            raise HTTPException(
                status_code=status.HTTP_400_BAD_REQUEST,
                detail=f"Authentication file not found: {x_auth_file}",
            )
        return YTMusic(str(auth_path))

    return YTMusic()


T_YTMusic = Annotated[YTMusic, Depends(get_ytmusic)]


def handle_ytmusic_error(exc: Exception) -> HTTPException:
    """Convert ytmusicapi exceptions to appropriate HTTP exceptions.

    Args:
        exc: The exception raised by ytmusicapi

    Returns:
        HTTPException with appropriate status code and message
    """
    error_msg = str(exc)

    if "authentication" in error_msg.lower():
        return HTTPException(status_code=status.HTTP_401_UNAUTHORIZED, detail=error_msg)

    if "not found" in error_msg.lower():
        return HTTPException(status_code=status.HTTP_404_NOT_FOUND, detail=error_msg)

    if "invalid" in error_msg.lower() or "bad" in error_msg.lower():
        return HTTPException(status_code=status.HTTP_400_BAD_REQUEST, detail=error_msg)

    return HTTPException(
        status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
        detail=f"Internal error: {error_msg}",
    )


@app.get("/health")
async def health_check() -> HealthCheckResp:
    """Health check endpoint.

    Verifies the API is running and ytmusicapi is available.
    """
    try:
        YTMusic()
        return HealthCheckResp(status="healthy")
    except Exception as exn:
        raise HTTPException(
            status_code=status.HTTP_503_SERVICE_UNAVAILABLE,
            detail=f"Service unhealthy: {str(exn)}",
        ) from exn


@app.post("/api/setup")
async def setup_browser(data: BrowserSetupReq) -> SetupResp:
    """Set up browser authentication for YouTube Music.

    Accepts raw request headers from browser DevTools and generates browser.json.

    Args:
        data: Browser setup request with headers_raw and filepath

    Returns:
        Setup result with filepath location
    """
    try:
        ytmusic_setup(filepath=data.filepath, headers_raw=data.headers_raw)
        return SetupResp(
            success=True,
            filepath=data.filepath,
            message=f"Successfully created authentication file at {data.filepath}",
        )
    except Exception as exn:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=f"Setup failed: {str(exn)}",
        ) from exn


@app.post("/api/setup/oauth")
async def setup_oauth() -> MessageResp:
    """Get OAuth setup instructions for YouTube Music authentication.

    OAuth setup requires interactive terminal access and cannot be performed via API.

    Returns:
        Instructions for OAuth setup process
    """
    return MessageResp(
        message="OAuth setup requires interactive terminal. Use ytmusicapi CLI: ytmusicapi oauth"
    )


@app.get("/api/playlists/{playlist_id}")
async def get_playlist(playlist_id: str, ytmusic: T_YTMusic) -> dict[str, Any]:
    """Retrieve playlist details by ID.

    Args:
        playlist_id: YouTube Music playlist ID
        ytmusic: YTMusic client instance

    Returns:
        Playlist details
    """
    try:
        return ytmusic.get_playlist(playlist_id)
    except Exception as exc:
        raise handle_ytmusic_error(exc) from exc


@app.post("/api/playlists")
async def create_playlist(
    data: CreatePlaylistReq, ytmusic: T_YTMusic
) -> CreatePlaylistResp:
    """Create a new playlist.

    Args:
        data: Playlist creation data
        ytmusic: YTMusic client instance

    Returns:
        Playlist ID of created playlist
    """
    try:
        playlist_id = ytmusic.create_playlist(
            data.title,
            data.description,
            privacy_status=data.privacy_status,
        )
    except Exception as e:
        raise handle_ytmusic_error(e) from e
    else:
        return CreatePlaylistResp(playlist_id=playlist_id)


@app.put("/api/playlists/{playlist_id}")
async def edit_playlist(
    playlist_id: str, data: EditPlaylistReq, ytmusic: T_YTMusic
) -> SuccessResp:
    """Edit playlist metadata.

    Args:
        playlist_id: YouTube Music playlist ID
        data: Playlist edit data
        ytmusic: YTMusic client instance

    Returns:
        Success status
    """
    result = ytmusic.edit_playlist(
        playlist_id,
        title=data.title,
        description=data.description,
    )
    return SuccessResp(status="success", result=result)


@app.delete("/api/playlists/{playlist_id}")
async def delete_playlist(playlist_id: str, ytmusic: T_YTMusic) -> SuccessResp:
    """Delete a playlist.

    Args:
        playlist_id: YouTube Music playlist ID
        ytmusic: YTMusic client instance

    Returns:
        Success status
    """
    result = ytmusic.delete_playlist(playlist_id)
    return SuccessResp(status="success", result=result)


@app.post("/api/playlists/{playlist_id}/items")
async def add_playlist_items(
    playlist_id: str, data: AddPlaylistItemsReq, ytmusic: T_YTMusic
) -> SuccessResp:
    """Add tracks to a playlist.

    Args:
        playlist_id: YouTube Music playlist ID
        data: Video IDs to add
        ytmusic: YTMusic client instance

    Returns:
        Success status with added items info
    """
    result = ytmusic.add_playlist_items(playlist_id, data.video_ids)
    return SuccessResp(status="success", result=result)


@app.delete("/api/playlists/{playlist_id}/items")
async def remove_playlist_items(
    playlist_id: str, data: RmPlaylistItemsReq, ytmusic: T_YTMusic
) -> SuccessResp:
    """Remove tracks from a playlist.

    Args:
        playlist_id: YouTube Music playlist ID
        data: Videos to remove
        ytmusic: YTMusic client instance

    Returns:
        Success status
    """
    result = ytmusic.remove_playlist_items(playlist_id, data.videos)
    return SuccessResp(status="success", result=result)


# Library Domain
@app.get("/api/library/playlists")
async def get_library_playlists(ytmusic: T_YTMusic) -> list[dict[str, Any]]:
    """List user's library playlists.

    Args:
        ytmusic: YTMusic client instance

    Returns:
        List of user playlists
    """
    try:
        return ytmusic.get_library_playlists()
    except Exception as e:
        raise handle_ytmusic_error(e) from e


@app.get("/api/library/songs")
async def get_library_songs(ytmusic: T_YTMusic) -> list[dict[str, Any]]:
    """Get user's saved songs.

    Args:
        ytmusic: YTMusic client instance

    Returns:
        List of saved songs
    """
    return ytmusic.get_library_songs()


@app.get("/api/library/albums")
async def get_library_albums(ytmusic: T_YTMusic) -> list[dict[str, Any]]:
    """Get user's saved albums.

    Args:
        ytmusic: YTMusic client instance

    Returns:
        List of saved albums
    """
    return ytmusic.get_library_albums()


@app.get("/api/library/artists")
async def get_library_artists(ytmusic: T_YTMusic) -> list[dict[str, Any]]:
    """Get user's saved artists.

    Args:
        ytmusic: YTMusic client instance

    Returns:
        List of saved artists
    """
    return ytmusic.get_library_artists()


@app.get("/api/library/liked-songs")
async def get_liked_songs(ytmusic: T_YTMusic) -> dict[str, Any]:
    """Get user's liked songs.

    Args:
        ytmusic: YTMusic client instance

    Returns:
        Liked songs playlist
    """
    return ytmusic.get_liked_songs()


@app.get("/api/library/history")
async def get_history(ytmusic: T_YTMusic) -> list[dict[str, Any]]:
    """Get user's listening history.

    Args:
        ytmusic: YTMusic client instance

    Returns:
        List of recently played tracks
    """
    return ytmusic.get_history()


@app.post("/api/library/songs/{video_id}/rate")
async def rate_song(
    video_id: str, data: RateSongReq, ytmusic: T_YTMusic
) -> SuccessResp:
    """Rate a song (like/dislike).

    Args:
        video_id: YouTube Music video ID
        data: Rating value
        ytmusic: YTMusic client instance

    Returns:
        Success status
    """
    result = ytmusic.rate_song(video_id, data.rating)
    return SuccessResp(status="success", result=result)


@app.post("/api/library/artists/subscribe")
async def subscribe_artists(
    data: SubscribeArtistsReq, ytmusic: T_YTMusic
) -> SuccessResp:
    """Subscribe to artists.

    Args:
        data: Artist channel IDs to subscribe to
        ytmusic: YTMusic client instance

    Returns:
        Success status
    """
    result = ytmusic.subscribe_artists(data.channel_ids)
    return SuccessResp(status="success", result=result)


# Uploads Domain
@app.get("/api/uploads/songs")
async def get_upload_songs(ytmusic: T_YTMusic) -> list[dict[str, Any]]:
    """List user's uploaded songs.

    Args:
        ytmusic: YTMusic client instance

    Returns:
        List of uploaded songs
    """
    return ytmusic.get_library_upload_songs()


@app.get("/api/uploads/albums")
async def get_upload_albums(ytmusic: T_YTMusic) -> list[dict[str, Any]]:
    """List user's uploaded albums.

    Args:
        ytmusic: YTMusic client instance

    Returns:
        List of uploaded albums
    """
    return ytmusic.get_library_upload_albums()


@app.post("/api/uploads/songs")
async def upload_song(file: UploadFile, ytmusic: T_YTMusic) -> SuccessResp:
    """Upload a music file to YouTube Music.

    Args:
        file: Music file to upload
        ytmusic: YTMusic client instance

    Returns:
        Upload status and entity ID
    """
    # Save uploaded file temporarily
    temp_path = Path(f"/tmp/{file.filename}")
    try:
        contents = await file.read()
        temp_path.write_bytes(contents)

        result = ytmusic.upload_song(str(temp_path))
        return SuccessResp(status="success", result=result)
    finally:
        if temp_path.exists():
            temp_path.unlink()


@app.delete("/api/uploads/{entity_id}")
async def delete_upload(entity_id: str, ytmusic: T_YTMusic) -> SuccessResp:
    """Delete an uploaded entity.

    Args:
        entity_id: Entity ID to delete
        ytmusic: YTMusic client instance

    Returns:
        Success status
    """
    result = ytmusic.delete_upload_entity(entity_id)
    return SuccessResp(status="success", result=result)


# Stub Domains
@app.api_route("/api/podcasts/{path:path}", methods=["GET", "POST", "PUT", "DELETE"])
async def podcasts_stub(path: str) -> JSONResponse:
    """Stub endpoint for podcasts - not implemented."""
    return JSONResponse(
        status_code=status.HTTP_501_NOT_IMPLEMENTED,
        content={"detail": "Podcasts domain not implemented"},
    )


@app.api_route("/api/explore/{path:path}", methods=["GET", "POST", "PUT", "DELETE"])
async def explore_stub(path: str) -> JSONResponse:
    """Stub endpoint for explore - not implemented."""
    return JSONResponse(
        status_code=status.HTTP_501_NOT_IMPLEMENTED,
        content={"detail": "Explore domain not implemented"},
    )


@app.api_route("/api/search/{path:path}", methods=["GET", "POST", "PUT", "DELETE"])
async def search_stub(path: str) -> JSONResponse:
    """Stub endpoint for search - not implemented."""
    return JSONResponse(
        status_code=status.HTTP_501_NOT_IMPLEMENTED,
        content={"detail": "Search domain not implemented"},
    )


@app.api_route("/api/browsing/{path:path}", methods=["GET", "POST", "PUT", "DELETE"])
async def browsing_stub(path: str) -> JSONResponse:
    """Stub endpoint for browsing - not implemented."""
    return JSONResponse(
        status_code=status.HTTP_501_NOT_IMPLEMENTED,
        content={"detail": "Browsing domain not implemented"},
    )
