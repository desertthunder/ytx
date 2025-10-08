# Song Migrator (YTX/`ytx`)

A web service to transfer playlists between Spotify & YouTube music.

## Music Package (Python)

A FastAPI proxy around [ytmusicapi](https://github.com/sigma67/ytmusicapi) that runs on port 8080.

From music, run `python -m cli`

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

### v0.3

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

### v0.5

| Command                                                         | Description                                             | Example                                 |
| --------------------------------------------------------------- | ------------------------------------------------------- | --------------------------------------- |
| `ytx m[usic]b[rainz] artist "Daft Punk"`                        | Search artists by name                                  | Returns JSON list of matches            |
| `ytx m[usic]b[rainz] release "Discovery"`                       | Search releases (albums)                                | Returns album metadata                  |
| `ytx m[usic]b[rainz] recording "Harder Better Faster Stronger"` | Search tracks/recordings                                | Prints ISRC, duration, MBID             |
| `ytx m[usic]b[rainz] enrich --input playlist.json`              | Enrich your Spotify playlist JSON with MusicBrainz data | Adds canonical IDs, release dates, etc. |
| `ytx m[usic]b[rainz] browse --tag electronic`                   | Browse tagged recordings or artists                     | Category browsing                       |

### v0.6

| Command                                 | Description                    | Example                                   |
| --------------------------------------- | ------------------------------ | ----------------------------------------- |
| `ytx arch[ive] search "Daft Punk live"` | Search all Archive collections | Returns titles, identifiers               |
| `ytx arch[ive] audio "Nirvana"`         | Restrict to audio recordings   | Only returns items in audio collections   |
| `ytx arch[ive] metadata <identifier>`   | Get full metadata for an item  | `ytx archive metadata daftpunk_live_2001` |
| `ytx arch[ive] fetch <identifier>`      | Download item or metadata      | Downloads files or JSON                   |
| `ytx arch[ive] import <identifier>`     | Save metadata locally in cache | `ytx archive import daftpunk_live_2001`   |
