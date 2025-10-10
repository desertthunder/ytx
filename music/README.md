# YouTube Music Proxy API

A FastAPI-based proxy server for YouTube Music API operations. Enables playlist migration and library management through a REST interface wrapping ytmusicapi.

## Installation

```bash
uv sync
# or with pip
pip install -e .
```

## Authentication

YouTube Music requires authentication for most operations.

### Browser Authentication (Recommended)

__Using ytx CLI__ (easiest):

1. Log into YouTube Music in your browser
2. Open Developer Tools (F12) → Network tab
3. Find a POST request to `music.youtube.com/youtubei/v1/browse`
4. Right-click → Copy → Copy as cURL
5. Run setup command:

```bash
ytx setup youtube --curl "curl 'https://music.youtube.com/...' -H '...'"
# or save to file: ytx setup youtube --curl-file auth.sh
```

__Using the API endpoint directly__:

```bash
curl -X POST http://localhost:8080/api/setup \
  -H "Content-Type: application/json" \
  -d '{
    "headers_raw": "cookie: PASTE_YOUR_COOKIE_HERE\nuser-agent: YOUR_USER_AGENT",
    "filepath": "browser.json"
  }'
```

The endpoint parses the headers and returns `auth_content` JSON that can be saved as `browser.json`.

__Using ytmusicapi CLI__:

```bash
ytmusicapi browser --file browser.json
```

### OAuth Authentication

```bash
ytmusicapi oauth --file oauth.json
```

### Using Credentials

Pass the credentials file path via the `X-Auth-File` header in API requests:

```bash
curl -H "X-Auth-File: /path/to/browser.json" http://localhost:8080/api/library/playlists
```

## Running the Server

__Using the installed command__:

```bash
uv run proxy serve
```

__With options__:

```bash
uv run proxy serve --port 9000 --reload
```

__Development mode__:

```bash
uv run proxy serve --reload
```

Server runs on `http://localhost:8080` by default.

## API Endpoints

### Health Check

`GET /health` - Server health status

### Setup Domain

`POST /api/setup` - Create browser.json from request headers (requires: headers_raw, optional: filepath)
`POST /api/setup/oauth` - Get OAuth setup instructions

### Playlists Domain

`GET /api/playlists/{playlist_id}` - Retrieve playlist details
`POST /api/playlists` - Create new playlist (requires: title, description, privacy_status)
`PUT /api/playlists/{playlist_id}` - Edit playlist metadata
`DELETE /api/playlists/{playlist_id}` - Delete playlist
`POST /api/playlists/{playlist_id}/items` - Add tracks (requires: video_ids array)
`DELETE /api/playlists/{playlist_id}/items` - Remove tracks (requires: videos array)

### Library Domain

`GET /api/library/playlists` - List user playlists
`GET /api/library/songs` - Get saved songs
`GET /api/library/albums` - Get saved albums
`GET /api/library/artists` - Get saved artists
`GET /api/library/liked-songs` - Fetch liked tracks
`GET /api/library/history` - View listening history
`POST /api/library/songs/{video_id}/rate` - Rate song (LIKE/DISLIKE/INDIFFERENT)
`POST /api/library/artists/subscribe` - Follow artists (requires: channel_ids array)

### Uploads Domain

`GET /api/uploads/songs` - List uploaded tracks
`GET /api/uploads/albums` - List uploaded albums
`POST /api/uploads/songs` - Upload music file (multipart/form-data)
`DELETE /api/uploads/{entity_id}` - Remove uploaded content

### Search Domain

`GET /api/search` - Search YouTube Music for tracks, albums, artists, etc. (query params: q=query string, filter=songs|videos|albums|artists|playlists)

### Stub Domains (Not Implemented)

`/api/podcasts/*` - Returns 501
`/api/explore/*` - Returns 501
`/api/browsing/*` - Returns 501

## Usage Examples

__Unauthenticated request__ (limited functionality):

```bash
curl http://localhost:8080/api/library/playlists
```

__Authenticated request__:

```bash
curl -H "X-Auth-File: /path/to/browser.json" \
  http://localhost:8080/api/library/playlists
```

__Create a playlist__:

```bash
curl -X POST \
  -H "X-Auth-File: /path/to/browser.json" \
  -H "Content-Type: application/json" \
  -d '{"title":"My Playlist","description":"Test playlist","privacy_status":"PRIVATE"}' \
  http://localhost:8080/api/playlists
```

__Add tracks to playlist__:

```bash
curl -X POST \
  -H "X-Auth-File: /path/to/browser.json" \
  -H "Content-Type: application/json" \
  -d '{"video_ids":["vid1","vid2","vid3"]}' \
  http://localhost:8080/api/playlists/PLAYLIST_ID/items
```

__Upload a song__:

```bash
curl -X POST \
  -H "X-Auth-File: /path/to/browser.json" \
  -F "file=@/path/to/song.mp3" \
  http://localhost:8080/api/uploads/songs
```

__Search for tracks__:

```bash
curl "http://localhost:8080/api/search?q=Daft%20Punk&filter=songs"
```

## Error Handling

The API returns standard HTTP status codes and error responses include a `detail` field with the error message.

## Testing

See the [justfile](../justfile) in the root of the project

## Development

__Installation__:

```bash
uv sync
```

__Linting & Formatting__:

```bash
ruff check .
ruff format .
```
