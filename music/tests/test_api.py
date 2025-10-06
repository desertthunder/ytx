"""Tests for YouTube Music proxy API."""

from unittest.mock import Mock, patch

import pytest
from fastapi.testclient import TestClient

from src.api.app import app

client = TestClient(app)


@pytest.fixture
def mock_ytmusic():
    """Mock YTMusic instance for testing."""
    with patch("src.api.app.YTMusic") as mock:
        yield mock


@pytest.fixture
def mock_ytmusic_instance(mock_ytmusic):
    """Mock YTMusic instance with common return values."""
    instance = Mock()
    mock_ytmusic.return_value = instance
    return instance


class TestHealthCheck:
    """Tests for health check endpoint."""

    def test_health_check(self, mock_ytmusic):
        """Health check should return healthy status."""
        mock_ytmusic.return_value = Mock()

        res = client.get("/health")
        assert res.status_code == 200
        assert res.json() == {"status": "healthy"}

    def test_health_check_failure(self, mock_ytmusic):
        """Health check should return 503 if ytmusicapi fails."""
        mock_ytmusic.side_effect = Exception("YTMusic initialization failed")

        res = client.get("/health")
        assert res.status_code == 503
        assert "unhealthy" in res.json()["detail"].lower()


class TestAuthentication:
    """Tests for authentication dependency."""

    def test_unauthenticated_request(self, mock_ytmusic):
        """Request without auth header should create unauthenticated client."""
        mock_ytmusic.return_value.get_library_playlists.return_value = []

        res = client.get("/api/library/playlists")
        assert res.status_code == 200
        mock_ytmusic.assert_called_once_with()

    def test_authenticated_request(self, mock_ytmusic, tmp_path):
        """Request with auth header should create authenticated client."""
        auth_file = tmp_path / "browser.json"
        auth_file.write_text('{"cookie": "test"}')

        mock_ytmusic.return_value.get_library_playlists.return_value = []

        res = client.get(
            "/api/library/playlists", headers={"X-Auth-File": str(auth_file)}
        )

        assert res.status_code == 200
        mock_ytmusic.assert_called_once_with(str(auth_file))

    def test_invalid_auth_file(self, mock_ytmusic):
        """Request with invalid auth file should return 400."""
        res = client.get(
            "/api/library/playlists", headers={"X-Auth-File": "/nonexistent/file.json"}
        )
        assert res.status_code == 400
        assert "not found" in res.json()["detail"].lower()


class TestSetupEndpoints:
    """Tests for setup endpoints."""

    def test_setup_success(self, tmp_path):
        """Setup endpoint should create browser.json from headers."""
        with patch("src.api.app.ytmusic_setup") as mock_setup:
            headers_raw = "cookie: test\nuser-agent: test"
            filepath = str(tmp_path / "browser.json")

            res = client.post(
                "/api/setup",
                json={"headers_raw": headers_raw, "filepath": filepath},
            )

            assert res.status_code == 200
            assert res.json()["success"] is True
            assert res.json()["filepath"] == filepath
            mock_setup.assert_called_once_with(
                filepath=filepath, headers_raw=headers_raw
            )

    def test_setup_failure(self):
        """Setup endpoint should return 400 on failure."""
        with patch("src.api.app.ytmusic_setup") as mock_setup:
            mock_setup.side_effect = Exception("Invalid headers")

            res = client.post(
                "/api/setup",
                json={"headers_raw": "invalid", "filepath": "browser.json"},
            )

            assert res.status_code == 400
            assert "failed" in res.json()["detail"].lower()

    def test_setup_oauth(self):
        """OAuth setup endpoint should return instructions."""
        res = client.post("/api/setup/oauth")
        assert res.status_code == 200
        assert "ytmusicapi oauth" in res.json()["message"]


class TestPlaylistsEndpoints:
    """Tests for playlists endpoints."""

    def test_get_playlist(self, mock_ytmusic_instance):
        """Get playlist should return playlist details."""
        expected = {"id": "PL123", "title": "Test Playlist"}
        mock_ytmusic_instance.get_playlist.return_value = expected

        res = client.get("/api/playlists/PL123")
        assert res.status_code == 200
        assert res.json() == expected
        mock_ytmusic_instance.get_playlist.assert_called_once_with("PL123")

    def test_create_playlist(self, mock_ytmusic_instance):
        """Create playlist should return playlist ID."""
        mock_ytmusic_instance.create_playlist.return_value = "PL123"

        res = client.post(
            "/api/playlists",
            json={
                "title": "Test Playlist",
                "description": "Test Description",
                "privacy_status": "PRIVATE",
            },
        )

        assert res.status_code == 200
        assert res.json()["playlist_id"] == "PL123"
        mock_ytmusic_instance.create_playlist.assert_called_once_with(
            "Test Playlist", "Test Description", privacy_status="PRIVATE"
        )

    def test_create_playlist_default_privacy(self, mock_ytmusic_instance):
        """Create playlist should use default privacy if not specified."""
        mock_ytmusic_instance.create_playlist.return_value = "PL123"

        res = client.post(
            "/api/playlists",
            json={"title": "Test Playlist", "description": "Test Description"},
        )

        assert res.status_code == 200
        mock_ytmusic_instance.create_playlist.assert_called_once_with(
            "Test Playlist", "Test Description", privacy_status="PRIVATE"
        )

    def test_edit_playlist(self, mock_ytmusic_instance):
        """Edit playlist should update metadata."""
        mock_ytmusic_instance.edit_playlist.return_value = "OK"

        res = client.put(
            "/api/playlists/PL123",
            json={"title": "Updated Title", "description": "Updated Description"},
        )

        assert res.status_code == 200
        assert res.json()["status"] == "success"
        mock_ytmusic_instance.edit_playlist.assert_called_once_with(
            "PL123", title="Updated Title", description="Updated Description"
        )

    def test_delete_playlist(self, mock_ytmusic_instance):
        """Delete playlist should return success."""
        mock_ytmusic_instance.delete_playlist.return_value = "OK"

        res = client.delete("/api/playlists/PL123")
        assert res.status_code == 200
        assert res.json()["status"] == "success"
        mock_ytmusic_instance.delete_playlist.assert_called_once_with("PL123")

    def test_add_playlist_items(self, mock_ytmusic_instance):
        """Add playlist items should add tracks."""
        mock_ytmusic_instance.add_playlist_items.return_value = {"status": "OK"}

        res = client.post(
            "/api/playlists/PL123/items", json={"video_ids": ["vid1", "vid2"]}
        )

        assert res.status_code == 200
        assert res.json()["status"] == "success"
        mock_ytmusic_instance.add_playlist_items.assert_called_once_with(
            "PL123", ["vid1", "vid2"]
        )

    def test_remove_playlist_items(self, mock_ytmusic_instance):
        """Remove playlist items should remove tracks."""
        mock_ytmusic_instance.remove_playlist_items.return_value = "OK"

        videos = [{"videoId": "vid1", "setVideoId": "set1"}]
        res = client.request(
            "DELETE", "/api/playlists/PL123/items", json={"videos": videos}
        )

        assert res.status_code == 200
        assert res.json()["status"] == "success"
        mock_ytmusic_instance.remove_playlist_items.assert_called_once_with(
            "PL123",
            videos,
        )


class TestLibraryEndpoints:
    """Tests for library endpoints."""

    def test_get_library_playlists(self, mock_ytmusic_instance):
        """Get library playlists should return playlists."""
        expected = [{"id": "PL123", "title": "Playlist 1"}]
        mock_ytmusic_instance.get_library_playlists.return_value = expected

        res = client.get("/api/library/playlists")
        assert res.status_code == 200
        assert res.json() == expected

    def test_get_library_songs(self, mock_ytmusic_instance):
        """Get library songs should return songs."""
        expected = [{"videoId": "vid123", "title": "Song 1"}]
        mock_ytmusic_instance.get_library_songs.return_value = expected

        res = client.get("/api/library/songs")
        assert res.status_code == 200
        assert res.json() == expected

    def test_get_library_albums(self, mock_ytmusic_instance):
        """Get library albums should return albums."""
        expected = [{"browseId": "alb123", "title": "Album 1"}]
        mock_ytmusic_instance.get_library_albums.return_value = expected

        res = client.get("/api/library/albums")
        assert res.status_code == 200
        assert res.json() == expected

    def test_get_library_artists(self, mock_ytmusic_instance):
        """Get library artists should return artists."""
        expected = [{"browseId": "art123", "title": "Artist 1"}]
        mock_ytmusic_instance.get_library_artists.return_value = expected

        res = client.get("/api/library/artists")
        assert res.status_code == 200
        assert res.json() == expected

    def test_get_liked_songs(self, mock_ytmusic_instance):
        """Get liked songs should return liked songs playlist."""
        expected = {"tracks": [{"videoId": "vid123"}]}
        mock_ytmusic_instance.get_liked_songs.return_value = expected

        res = client.get("/api/library/liked-songs")
        assert res.status_code == 200
        assert res.json() == expected

    def test_get_history(self, mock_ytmusic_instance):
        """Get history should return listening history."""
        expected = [{"videoId": "vid123", "title": "Song 1"}]
        mock_ytmusic_instance.get_history.return_value = expected

        res = client.get("/api/library/history")
        assert res.status_code == 200
        assert res.json() == expected

    def test_rate_song(self, mock_ytmusic_instance):
        """Rate song should rate the song."""
        mock_ytmusic_instance.rate_song.return_value = "OK"

        res = client.post("/api/library/songs/vid123/rate", json={"rating": "LIKE"})
        assert res.status_code == 200
        assert res.json()["status"] == "success"
        mock_ytmusic_instance.rate_song.assert_called_once_with("vid123", "LIKE")

    def test_subscribe_artists(self, mock_ytmusic_instance):
        """Subscribe artists should subscribe to artists."""
        mock_ytmusic_instance.subscribe_artists.return_value = "OK"

        res = client.post(
            "/api/library/artists/subscribe", json={"channel_ids": ["ch1", "ch2"]}
        )

        assert res.status_code == 200
        assert res.json()["status"] == "success"
        mock_ytmusic_instance.subscribe_artists.assert_called_once_with(["ch1", "ch2"])


class TestUploadsEndpoints:
    """Tests for uploads endpoints."""

    def test_get_upload_songs(self, mock_ytmusic_instance):
        """Get upload songs should return uploaded songs."""
        expected = [{"videoId": "vid123", "title": "Uploaded Song"}]
        mock_ytmusic_instance.get_library_upload_songs.return_value = expected

        res = client.get("/api/uploads/songs")
        assert res.status_code == 200
        assert res.json() == expected

    def test_get_upload_albums(self, mock_ytmusic_instance):
        """Get upload albums should return uploaded albums."""
        expected = [{"browseId": "alb123", "title": "Uploaded Album"}]
        mock_ytmusic_instance.get_library_upload_albums.return_value = expected

        res = client.get("/api/uploads/albums")
        assert res.status_code == 200
        assert res.json() == expected

    def test_upload_song(self, mock_ytmusic_instance, tmp_path):
        """Upload song should upload file."""
        mock_ytmusic_instance.upload_song.return_value = "vid123"

        test_file = tmp_path / "test.mp3"
        test_file.write_bytes(b"fake audio data")

        with open(test_file, "rb") as f:
            res = client.post(
                "/api/uploads/songs", files={"file": ("test.mp3", f, "audio/mpeg")}
            )

        assert res.status_code == 200
        assert res.json()["status"] == "success"
        assert mock_ytmusic_instance.upload_song.called

    def test_delete_upload(self, mock_ytmusic_instance):
        """Delete upload should delete entity."""
        mock_ytmusic_instance.delete_upload_entity.return_value = "OK"

        res = client.delete("/api/uploads/ent123")
        assert res.status_code == 200
        assert res.json()["status"] == "success"
        mock_ytmusic_instance.delete_upload_entity.assert_called_once_with("ent123")


class TestStubEndpoints:
    """Tests for stub endpoints."""

    def test_podcasts_stub(self):
        """Podcasts endpoint should return 501."""
        res = client.get("/api/podcasts/test")
        assert res.status_code == 501
        assert "not implemented" in res.json()["detail"].lower()

    def test_explore_stub(self):
        """Explore endpoint should return 501."""
        res = client.get("/api/explore/test")
        assert res.status_code == 501
        assert "not implemented" in res.json()["detail"].lower()

    def test_search_stub(self):
        """Search endpoint should return 501."""
        res = client.post("/api/search/test")
        assert res.status_code == 501
        assert "not implemented" in res.json()["detail"].lower()

    def test_browsing_stub(self):
        """Browsing endpoint should return 501."""
        res = client.put("/api/browsing/test")
        assert res.status_code == 501
        assert "not implemented" in res.json()["detail"].lower()


class TestErrorHandling:
    """Tests for error handling."""

    def test_authentication_error(self, mock_ytmusic_instance):
        """Authentication error should return 401."""
        exc = Exception("Please provide authentication before using this function")
        mock_ytmusic_instance.get_library_playlists.side_effect = exc

        res = client.get("/api/library/playlists")
        assert res.status_code == 401
        assert "authentication" in res.json()["detail"].lower()

    def test_not_found_error(self, mock_ytmusic_instance):
        """Not found error should return 404."""
        mock_ytmusic_instance.get_playlist.side_effect = Exception("Playlist not found")
        res = client.get("/api/playlists/invalid")
        assert res.status_code == 404
        assert "not found" in res.json()["detail"].lower()

    def test_invalid_parameter_error(self, mock_ytmusic_instance):
        """Invalid parameter error should return 400."""
        exc = Exception("Invalid privacy status")
        mock_ytmusic_instance.create_playlist.side_effect = exc

        res = client.post(
            "/api/playlists",
            json={"title": "Test", "description": "Test", "privacy_status": "INVALID"},
        )

        assert res.status_code == 400
        assert "invalid" in res.json()["detail"].lower()

    def test_generic_error(self, mock_ytmusic_instance):
        """Generic error should return 500."""
        exc = Exception("Something went wrong")
        mock_ytmusic_instance.get_library_playlists.side_effect = exc

        res = client.get("/api/library/playlists")
        assert res.status_code == 500
        assert "internal error" in res.json()["detail"].lower()
