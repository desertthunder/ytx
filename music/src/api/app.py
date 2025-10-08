"""FastAPI proxy server for YouTube Music API.

TODO: Create exceptions module
"""

import json
import os
import shutil
import tempfile
from pathlib import Path
from typing import Annotated, Any

from fastapi import Depends, FastAPI, Header, HTTPException, UploadFile, status
from fastapi.responses import JSONResponse
from loguru import logger
from ytmusicapi import YTMusic
from ytmusicapi import setup as ytmusic_setup

from .middleware import LoggingMiddleware, configure_logging
from .models import req, resp

configure_logging()

app = FastAPI(title="YouTube Music Proxy API", version="0.1.0")
app.add_middleware(LoggingMiddleware)

THeaders = Annotated[str | None, Header()]


def get_ytmusic(x_auth_file: THeaders = None, x_auth_data: THeaders = None) -> YTMusic:
    """Create YTMusic client instance based on authentication.

    Accepts either:
    - x_auth_file: Path to local auth file on the proxy server
    - x_auth_data: JSON string containing auth headers (preferred for remote clients)

    Args:
        x_auth_file: Optional path to authentication file on proxy server
        x_auth_data: Optional JSON string with auth headers

    Returns:
        Authenticated or unauthenticated YTMusic instance

    Raises:
        HTTPException: If auth is invalid
    """
    if x_auth_data:
        try:
            auth_dict = json.loads(x_auth_data)
            logger.debug(
                "Loading auth from header data",
                extra={"keys": list(auth_dict.keys())},
            )
            return YTMusic(auth=auth_dict)
        except json.JSONDecodeError as e:
            raise HTTPException(
                status_code=status.HTTP_400_BAD_REQUEST,
                detail=f"Invalid JSON in X-Auth-Data header: {e}",
            ) from e
        except Exception as e:
            raise HTTPException(
                status_code=status.HTTP_400_BAD_REQUEST, detail=f"Authentication failed: {e}"
            ) from e

    if x_auth_file:
        auth_path = Path(x_auth_file)
        if not auth_path.exists():
            raise HTTPException(
                status_code=status.HTTP_400_BAD_REQUEST,
                detail=f"Authentication file not found: {x_auth_file}",
            )

        logger.debug(
            f"Loading auth from file: {x_auth_file}", extra={"file_size": auth_path.stat().st_size}
        )
        return YTMusic(str(auth_path))

    return YTMusic()


TYtMusic = Annotated[YTMusic, Depends(get_ytmusic)]


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
async def health_check() -> resp.HealthCheck:
    """Health check endpoint.

    Verifies the API is running and ytmusicapi is available.
    """
    try:
        YTMusic()
        return resp.HealthCheck(status="healthy")
    except Exception as exn:
        raise HTTPException(
            status_code=status.HTTP_503_SERVICE_UNAVAILABLE,
            detail=f"Service unhealthy: {str(exn)}",
        ) from exn


@app.post("/api/setup")
async def setup_browser(data: req.BrowserSetup) -> resp.SetupWithContent:
    """Set up browser authentication for YouTube Music.

    Accepts raw request headers from browser DevTools and generates browser.json.
    Returns the generated authentication data for the client to save locally.

    Args:
        data: Browser setup request with headers_raw and optional filepath

    Returns:
        Setup result with generated auth content
    """
    try:
        with tempfile.NamedTemporaryFile(mode="w", suffix=".json", delete=False) as tmp:
            tmp_path = tmp.name

        try:
            ytmusic_setup(filepath=tmp_path, headers_raw=data.headers_raw)

            with open(tmp_path) as f:
                auth_content = json.load(f)

            logger.debug("Generated browser.json", extra={"keys": list(auth_content.keys())})

            if data.filepath:
                shutil.copy(tmp_path, data.filepath)
                logger.info(f"Saved browser.json to {data.filepath}")

            return resp.SetupWithContent(
                success=True,
                filepath=data.filepath or tmp_path,
                message="Successfully generated browser authentication",
                auth_content=auth_content,
            )
        finally:
            if os.path.exists(tmp_path):
                os.unlink(tmp_path)

    except Exception as exn:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=f"Setup failed: {str(exn)}",
        ) from exn


@app.post("/api/setup/oauth")
async def setup_oauth() -> resp.Message:
    """Get OAuth setup instructions for YouTube Music authentication.

    OAuth setup requires interactive terminal access and cannot be performed via API.

    Returns:
        Instructions for OAuth setup process
    """
    return resp.Message(
        message="OAuth setup requires interactive terminal. Use ytmusicapi CLI: ytmusicapi oauth"
    )


@app.get("/api/playlists/{playlist_id}")
async def get_playlist(playlist_id: str, ytmusic: TYtMusic) -> dict[str, Any]:
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


class PlaylistResultError(ValueError):
    """Response parsing error."""

    def __init__(self, *args: object) -> None:
        """Adds Playlist ID message to ValueError."""
        super().__init__(*args, "Could not extract playlist_id from response")


def _get_playlist_id(result: str | dict[str, Any]) -> str:
    if isinstance(result, dict):
        id = result.get("playlistId") or result.get("id")
    else:
        id = result
    if not id:
        raise PlaylistResultError

    return id


@app.post("/api/playlists")
async def create_playlist(data: req.CreatePlaylist, ytmusic: TYtMusic) -> resp.CreatePlaylist:
    """Create a new playlist.

    Args:
        data: Playlist creation data
        ytmusic: YTMusic client instance

    Returns:
        Playlist ID of created playlist
    """
    try:
        result = ytmusic.create_playlist(
            data.title, data.description, privacy_status=data.privacy_status
        )
        id = _get_playlist_id(result)
    except Exception as e:
        raise handle_ytmusic_error(e) from e
    else:
        return resp.CreatePlaylist(playlist_id=id)


@app.put("/api/playlists/{playlist_id}")
async def edit_playlist(
    playlist_id: str, data: req.EditPlaylist, ytmusic: TYtMusic
) -> resp.Success:
    """Edit playlist metadata.

    Args:
        playlist_id: YouTube Music playlist ID
        data: Playlist edit data
        ytmusic: YTMusic client instance

    Returns:
        Success status
    """
    result = ytmusic.edit_playlist(playlist_id, title=data.title, description=data.description)
    return resp.Success(status="success", result=result)


@app.delete("/api/playlists/{playlist_id}")
async def delete_playlist(playlist_id: str, ytmusic: TYtMusic) -> resp.Success:
    """Delete a playlist.

    Args:
        playlist_id: YouTube Music playlist ID
        ytmusic: YTMusic client instance

    Returns:
        Success status
    """
    result = ytmusic.delete_playlist(playlist_id)
    return resp.Success(status="success", result=result)


@app.post("/api/playlists/{playlist_id}/items")
async def add_playlist_items(
    playlist_id: str, data: req.AddPlaylistItems, ytmusic: TYtMusic
) -> resp.Success:
    """Add tracks to a playlist.

    Args:
        playlist_id: YouTube Music playlist ID
        data: Video IDs to add
        ytmusic: YTMusic client instance

    Returns:
        Success status with added items info
    """
    result = ytmusic.add_playlist_items(playlist_id, data.video_ids)
    return resp.Success(status="success", result=result)


@app.delete("/api/playlists/{playlist_id}/items")
async def remove_playlist_items(
    playlist_id: str, data: req.RmPlaylistItems, ytmusic: TYtMusic
) -> resp.Success:
    """Remove tracks from a playlist.

    Args:
        playlist_id: YouTube Music playlist ID
        data: Videos to remove
        ytmusic: YTMusic client instance

    Returns:
        Success status
    """
    result = ytmusic.remove_playlist_items(playlist_id, data.videos)
    return resp.Success(status="success", result=result)


@app.get("/api/library/playlists")
async def get_library_playlists(ytmusic: TYtMusic) -> list[dict[str, Any]]:
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
async def get_library_songs(ytmusic: TYtMusic) -> list[dict[str, Any]]:
    """Get user's saved songs.

    Args:
        ytmusic: YTMusic client instance

    Returns:
        List of saved songs
    """
    return ytmusic.get_library_songs()


@app.get("/api/library/albums")
async def get_library_albums(ytmusic: TYtMusic) -> list[dict[str, Any]]:
    """Get user's saved albums.

    Args:
        ytmusic: YTMusic client instance

    Returns:
        List of saved albums
    """
    return ytmusic.get_library_albums()


@app.get("/api/library/artists")
async def get_library_artists(ytmusic: TYtMusic) -> list[dict[str, Any]]:
    """Get user's saved artists.

    Args:
        ytmusic: YTMusic client instance

    Returns:
        List of saved artists
    """
    return ytmusic.get_library_artists()


@app.get("/api/library/liked-songs")
async def get_liked_songs(ytmusic: TYtMusic) -> dict[str, Any]:
    """Get user's liked songs.

    Args:
        ytmusic: YTMusic client instance

    Returns:
        Liked songs playlist
    """
    return ytmusic.get_liked_songs()


@app.get("/api/library/history")
async def get_history(ytmusic: TYtMusic) -> list[dict[str, Any]]:
    """Get user's listening history.

    Args:
        ytmusic: YTMusic client instance

    Returns:
        List of recently played tracks
    """
    return ytmusic.get_history()


@app.post("/api/library/songs/{video_id}/rate")
async def rate_song(video_id: str, data: req.RateSong, ytmusic: TYtMusic) -> resp.Success:
    """Rate a song (like/dislike).

    Args:
        video_id: YouTube Music video ID
        data: Rating value
        ytmusic: YTMusic client instance

    Returns:
        Success status
    """
    result = ytmusic.rate_song(video_id, data.rating)
    return resp.Success(status="success", result=result)


@app.post("/api/library/artists/subscribe")
async def subscribe_artists(data: req.SubscribeArtists, ytmusic: TYtMusic) -> resp.Success:
    """Subscribe to artists.

    Args:
        data: Artist channel IDs to subscribe to
        ytmusic: YTMusic client instance

    Returns:
        Success status
    """
    result = ytmusic.subscribe_artists(data.channel_ids)
    return resp.Success(status="success", result=result)


@app.get("/api/uploads/songs")
async def get_upload_songs(ytmusic: TYtMusic) -> list[dict[str, Any]]:
    """List user's uploaded songs.

    Args:
        ytmusic: YTMusic client instance

    Returns:
        List of uploaded songs
    """
    return ytmusic.get_library_upload_songs()


@app.get("/api/uploads/albums")
async def get_upload_albums(ytmusic: TYtMusic) -> list[dict[str, Any]]:
    """List user's uploaded albums.

    Args:
        ytmusic: YTMusic client instance

    Returns:
        List of uploaded albums
    """
    return ytmusic.get_library_upload_albums()


@app.post("/api/uploads/songs")
async def upload_song(file: UploadFile, ytmusic: TYtMusic) -> resp.Success:
    """Upload a music file to YouTube Music.

    Args:
        file: Music file to upload
        ytmusic: YTMusic client instance

    Returns:
        Upload status and entity ID
    """
    temp_path = Path(f"/tmp/{file.filename}")
    try:
        contents = await file.read()
        temp_path.write_bytes(contents)

        result = ytmusic.upload_song(str(temp_path))
        return resp.Success(status="success", result=result)
    finally:
        if temp_path.exists():
            temp_path.unlink()


@app.delete("/api/uploads/{entity_id}")
async def delete_upload(entity_id: str, ytmusic: TYtMusic) -> resp.Success:
    """Delete an uploaded entity.

    Args:
        entity_id: Entity ID to delete
        ytmusic: YTMusic client instance

    Returns:
        Success status
    """
    result = ytmusic.delete_upload_entity(entity_id)
    return resp.Success(status="success", result=result)


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


@app.get("/api/search")
async def search(q: str, ytmusic: TYtMusic, filter: str | None = None) -> list[dict[str, Any]]:
    """Search YouTube Music for tracks, albums, artists, etc.

    Args:
        q: Search query string
        ytmusic: YTMusic client instance
        filter: Optional filter (songs, videos, albums, artists, playlists)

    Returns:
        List of search results
    """
    try:
        results = ytmusic.search(query=q, filter=filter)
    except Exception as exc:
        raise handle_ytmusic_error(exc) from exc
    else:
        return results


@app.api_route("/api/browsing/{path:path}", methods=["GET", "POST", "PUT", "DELETE"])
async def browsing_stub(path: str) -> JSONResponse:
    """Stub endpoint for browsing - not implemented."""
    return JSONResponse(
        status_code=status.HTTP_501_NOT_IMPLEMENTED,
        content={"detail": "Browsing domain not implemented"},
    )
