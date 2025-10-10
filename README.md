# Song Migrator (YTX/`ytx`)

A web service to transfer playlists between Spotify & YouTube music.

## Music Package (Python)

A FastAPI proxy around [ytmusicapi](https://github.com/sigma67/ytmusicapi) that runs on port 8080.

From music, run `uv run proxy`

## CLI

### Usage

#### Setup

Initialize database and create config.toml

```sh
ytx setup database --config config.toml
```

Configure YouTube Music authentication from browser headers

```sh
# From browser DevTools: Copy network request as cURL
ytx setup youtube --curl "curl 'https://music.youtube.com/...' -H '...'"
# Or save cURL to a file and reference it
ytx setup youtube --curl-file auth.sh

# Custom output location
ytx setup youtube --curl-file auth.sh --output ~/my-auth.json
```

#### Auth

```sh
# Authenticate with Spotify (opens browser, saves tokens to config.toml)
# Tokens are automatically loaded on subsequent commands
ytx spotify auth

# Check authentication status
ytx auth status
```

#### Examples

```sh
# List playlists
ytx spotify playlists --limit 10 --json --pretty

# Export playlist to JSON
ytx spotify export --id <playlist-id> --output mylist.json

# Search for tracks
ytx ytmusic search "Daft Punk Derezzed"

# Create playlist
ytx ytmusic create "My Cool Mix"

# Add tracks to playlist
ytx ytmusic add --playlist-id XYZ --track "Song Name"

# Full Spotify → YouTube Music sync
ytx transfer run --source "My Spotify Mix" --dest "My YT Mix"

# Compare playlists
ytx transfer diff --source-id 123 --dest-id 456 --source-service spotify --dest-service youtube

# Requests to proxy
ytx api get /ytmusic/search?q=beatles --json
ytx api post /playlist/create -d '{"name":"My Mix"}'

# Full state dump
ytx api dump
```

#### Flags

- `--json` / `--pretty`: Toggle JSON output formatting
- `--save`: Save API responses locally

## ROADMAP

### v0.1 ✓

| Command           | Description                                                         | Example                                        |
| ----------------- | ------------------------------------------------------------------- | ---------------------------------------------- |
| `ytx auth login`  | Upload a `headers_auth.json` to the FastAPI `/auth/upload` endpoint | `ytx auth login ~/Downloads/headers_auth.json` |
| `ytx auth status` | Check current authentication state (calls `/health`)                | `ytx auth status`                              |

| Command                   | Description                        | Example                                           |
| ------------------------- | ---------------------------------- | ------------------------------------------------- |
| `ytx spot[ify] playlists` | List Spotify playlists             | `ytx spotify playlists --limit 10`                |
| `ytx spot[ify] export`    | Export playlist JSON for debugging | `ytx spotify export --id 4df7... --o mylist.json` |

| Feature                     | Description                                    |
| --------------------------- | ---------------------------------------------- |
| `--json` / `--pretty` flags | Toggle between raw or pretty-printed output    |
| `--save` flag               | Save API responses locally for caching / debug |

| Command        | Description                                       | Example                                                |
| -------------- | ------------------------------------------------- | ------------------------------------------------------ |
| `ytx api get`  | Direct GET to your FastAPI proxy, prints raw JSON | `ytx api get /ytmusic/search?q=beatles --json`         |
| `ytx api post` | Direct POST with JSON body                        | `ytx api post /playlist/create -d '{"name":"My Mix"}'` |

### v0.2 ✓

| Command                  | Description                            | Example                                                               |
| ------------------------ | -------------------------------------- | --------------------------------------------------------------------- |
| `ytx yt[m[usic]] search` | Search YouTube Music proxy for a track | `ytx ytmusic search "Daft Punk Harder Better"`                        |
| `ytx yt[m[usic]] create` | Create playlist on YouTube Music       | `ytx ytmusic create "My Cool Mix"`                                    |
| `ytx yt[m[usic]] add`    | Add tracks to an existing playlist     | `ytx ytmusic add --playlist-id XYZ --track "Daft Punk Harder Better"` |

| Command             | Description                                           | Example                                                                                          |
| ------------------- | ----------------------------------------------------- | ------------------------------------------------------------------------------------------------ |
| `ytx transfer run`  | Run full Spotify → YouTube Music sync                 | `ytx transfer run --source "My Spotify Mix" --dest "My YT Mix"`                                  |
| `ytx transfer diff` | Compare and show missing tracks between two playlists | `ytx transfer diff --source-id 123 --dest-id 456 --source-service spotify --dest-service youtube` |

| Command        | Description                                            |
| -------------- | ------------------------------------------------------ |
| `ytx api dump` | Full proxy state dump (playlists, songs, albums, etc) |

### v0.3 ✓

| Feature   | Description                                    |
| --------- | ---------------------------------------------- |
| `ytx tui` | Launch BubbleTea TUI for interactive transfers |

- Persistence layer
    - See [`models`](/internal/models/models.go) & [`database`](/internal/shared/database.go)

### v0.4

| Feature       | Description                                  |
| ------------- | -------------------------------------------- |
| `ytx doctor`  | Runs health checks against FastAPI endpoints |
| `ytx version` | Shows CLI + proxy versions                   |

- Better, more accurate error handling
    - Application shows "no match" when there should be an error or warning about the proxy server's health & status
- Make help view with key binding implementation more composable

### v0.5

| Feature   | Description                                  |
| --------- | -------------------------------------------- |
| `ytx web` | Launch HTMX-based web server for transfers   |

| Route                      | Method | Description                                  |
| -------------------------- | ------ | -------------------------------------------- |
| `/`                        | GET    | Playlist list view with server-side rendering |
| `/playlists/{id}/tracks`   | GET    | HTMX partial for track preview               |
| `/transfer`                | POST   | Start playlist transfer                      |

- Server-rendered HTML templates with HTMX for dynamic updates
- Playlist table with interactive row selection
- Track preview modal using HTMX partial swaps
- Transfer confirmation workflow
- Integration with existing services and tasks

### v0.6

| Route                   | Method | Description                           |
| ----------------------- | ------ | ------------------------------------- |
| `/transfer/{id}/stream` | GET    | SSE endpoint for real-time progress   |
| `/transfer/{id}/result` | GET    | Final transfer results view           |

- Server-Sent Events for streaming transfer progress
- Live progress bar with phase/step updates
- MigrationJob persistence across requests
- Results page with matched/failed tracks breakdown

### v0.7

| Route                       | Method | Description                    |
| --------------------------- | ------ | ------------------------------ |
| `/auth/spotify`             | GET    | OAuth initiation flow          |
| `/auth/spotify/callback`    | GET    | OAuth callback handler         |

- Cookie-based session management
- OAuth flow for Spotify authentication
- Session middleware for protected routes
- Automatic token refresh on expiration
- Auth status display in navigation

### v0.8

| Route                      | Method | Description                                  |
| -------------------------- | ------ | -------------------------------------------- |
| `/setup/youtube`           | GET    | YouTube Music setup form                     |
| `/setup/youtube/validate`  | POST   | Validate and parse cURL command              |
| `/setup/youtube/save`      | POST   | Generate and save headers_auth.json          |

- Web UI for YouTube Music authentication setup
- Form to paste cURL command from browser DevTools
- cURL command parsing and validation
- Header extraction and format verification
- Real-time validation feedback with error messages
- Generated headers_auth.json download or server-side save
- Integration with existing YouTube setup command logic

### v0.9

Kube

### v1.0+

| Command                                                         | Description                                             | Example                                 |
| --------------------------------------------------------------- | ------------------------------------------------------- | --------------------------------------- |
| `ytx m[usic]b[rainz] artist "Daft Punk"`                        | Search artists by name                                  | Returns JSON list of matches            |
| `ytx m[usic]b[rainz] release "Discovery"`                       | Search releases (albums)                                | Returns album metadata                  |
| `ytx m[usic]b[rainz] recording "Harder Better Faster Stronger"` | Search tracks/recordings                                | Prints ISRC, duration, MBID             |
| `ytx m[usic]b[rainz] enrich --input playlist.json`              | Enrich your Spotify playlist JSON with MusicBrainz data | Adds canonical IDs, release dates, etc. |
| `ytx m[usic]b[rainz] browse --tag electronic`                   | Browse tagged recordings or artists                     | Category browsing                       |

| Command                                 | Description                    | Example                                   |
| --------------------------------------- | ------------------------------ | ----------------------------------------- |
| `ytx arch[ive] search "Daft Punk live"` | Search all Archive collections | Returns titles, identifiers               |
| `ytx arch[ive] audio "Nirvana"`         | Restrict to audio recordings   | Only returns items in audio collections   |
| `ytx arch[ive] metadata <identifier>`   | Get full metadata for an item  | `ytx archive metadata daftpunk_live_2001` |
| `ytx arch[ive] fetch <identifier>`      | Download item or metadata      | Downloads files or JSON                   |
| `ytx arch[ive] import <identifier>`     | Save metadata locally in cache | `ytx archive import daftpunk_live_2001`   |
