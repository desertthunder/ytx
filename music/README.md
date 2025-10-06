# YouTube Music Proxy API

A FastAPI-based proxy server for YouTube Music API operations. Enables playlist migration and library management through a REST interface wrapping ytmusicapi.

## Installation

```bash
poetry install
# or with pip
pip install -e .
```

## Authentication

YouTube Music requires authentication for most operations.

### Browser Authentication (Recommended)

__Using the API endpoint__:

1. Log into YouTube Music in your browser
2. Open Developer Tools (F12) → Network tab
3. Find a POST request to `music.youtube.com/youtubei/v1/browse`
4. Right-click → Copy → Copy as cURL (or copy request headers)
5. Extract just the headers and send to the setup endpoint:

```bash
curl -X POST http://localhost:8080/api/setup \
  -H "Content-Type: application/json" \
  -d '{
    "headers_raw": "cookie: PASTE_YOUR_COOKIE_HERE\nuser-agent: YOUR_USER_AGENT",
    "filepath": "browser.json"
  }'
```

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
ytmusic-proxy serve
```

__With options__:

```bash
ytmusic-proxy serve --port 9000 --reload
```

__Development mode__:

```bash
poetry run ytmusic-proxy serve --reload
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

### Stub Domains (Not Implemented)

`/api/podcasts/*` - Returns 501
`/api/explore/*` - Returns 501
`/api/search/*` - Returns 501
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

## Error Handling

The API returns standard HTTP status codes:

- __200__ - Success
- __400__ - Bad request (invalid parameters)
- __401__ - Unauthorized (authentication required)
- __404__ - Not found
- __500__ - Internal server error
- __501__ - Not implemented (stub endpoints)

Error responses include a `detail` field with the error message.

## Testing

Run the test suite:

```bash
pytest tests/
```

With coverage:

```bash
pytest tests/ --cov=src
```

## Development

__Install dev dependencies__:

```bash
poetry install
```

__Run linter__:

```bash
ruff check .
```

__Format code__:

```bash
ruff format .
```

## Project Structure

```sh
music/
├── src/
│   ├── api/
│   │   ├── app.py          # FastAPI application
│   │   └── models.py       # Pydantic models
│   └── cli/
│       └── __main__.py     # CLI entry point
├── tests/
│   └── test_api.py         # API tests
├── pyproject.toml          # Project dependencies
└── README.md
```

## License

MIT
