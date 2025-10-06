-- Initial schema for song migration service

-- Users table
CREATE TABLE IF NOT EXISTS users (
    id TEXT PRIMARY KEY,
    sequence INTEGER NOT NULL UNIQUE,
    email TEXT UNIQUE NOT NULL,
    name TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Sequence counter for users
CREATE TABLE IF NOT EXISTS users_sequence (
    id INTEGER PRIMARY KEY,
    value INTEGER NOT NULL DEFAULT 0
);
INSERT INTO users_sequence (id, value) VALUES (1, 0);

-- Playlists table (cached playlist metadata)
CREATE TABLE IF NOT EXISTS playlists (
    id TEXT PRIMARY KEY,
    sequence INTEGER NOT NULL UNIQUE,
    service TEXT NOT NULL, -- 'spotify' or 'youtube'
    service_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    track_count INTEGER DEFAULT 0,
    public BOOLEAN DEFAULT false,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    UNIQUE(service, service_id)
);

-- Sequence counter for playlists
CREATE TABLE IF NOT EXISTS playlists_sequence (
    id INTEGER PRIMARY KEY,
    value INTEGER NOT NULL DEFAULT 0
);
INSERT INTO playlists_sequence (id, value) VALUES (1, 0);
CREATE INDEX IF NOT EXISTS idx_playlists_user_id ON playlists(user_id);
CREATE INDEX IF NOT EXISTS idx_playlists_service ON playlists(service);

-- Tracks table (cached track metadata)
CREATE TABLE IF NOT EXISTS tracks (
    id TEXT PRIMARY KEY,
    sequence INTEGER NOT NULL UNIQUE,
    service TEXT NOT NULL,
    service_id TEXT NOT NULL,
    title TEXT NOT NULL,
    artist TEXT NOT NULL,
    album TEXT,
    duration INTEGER, -- Duration in seconds
    isrc TEXT, -- International Standard Recording Code
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(service, service_id)
);

-- Sequence counter for tracks
CREATE TABLE IF NOT EXISTS tracks_sequence (
    id INTEGER PRIMARY KEY,
    value INTEGER NOT NULL DEFAULT 0
);
INSERT INTO tracks_sequence (id, value) VALUES (1, 0);
CREATE INDEX IF NOT EXISTS idx_tracks_service ON tracks(service);
CREATE INDEX IF NOT EXISTS idx_tracks_isrc ON tracks(isrc);

-- Playlist tracks junction table
CREATE TABLE IF NOT EXISTS playlist_tracks (
    id TEXT PRIMARY KEY,
    sequence INTEGER NOT NULL UNIQUE,
    playlist_id TEXT NOT NULL,
    track_id TEXT NOT NULL,
    position INTEGER NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (playlist_id) REFERENCES playlists(id) ON DELETE CASCADE,
    FOREIGN KEY (track_id) REFERENCES tracks(id) ON DELETE CASCADE,
    UNIQUE(playlist_id, track_id)
);

-- Sequence counter for playlist_tracks
CREATE TABLE IF NOT EXISTS playlist_tracks_sequence (
    id INTEGER PRIMARY KEY,
    value INTEGER NOT NULL DEFAULT 0
);
INSERT INTO playlist_tracks_sequence (id, value) VALUES (1, 0);
CREATE INDEX IF NOT EXISTS idx_playlist_tracks_playlist ON playlist_tracks(playlist_id);
CREATE INDEX IF NOT EXISTS idx_playlist_tracks_position ON playlist_tracks(playlist_id, position);

-- Migrations table (tracks migration jobs between services)
CREATE TABLE IF NOT EXISTS migrations (
    id TEXT PRIMARY KEY,
    sequence INTEGER NOT NULL UNIQUE,
    user_id TEXT NOT NULL,
    source_service TEXT NOT NULL,
    source_playlist_id TEXT NOT NULL,
    target_service TEXT NOT NULL,
    target_playlist_id TEXT,
    status TEXT DEFAULT 'pending', -- pending, in_progress, completed, failed
    tracks_total INTEGER DEFAULT 0,
    tracks_migrated INTEGER DEFAULT 0,
    tracks_failed INTEGER DEFAULT 0,
    error_message TEXT,
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (source_playlist_id) REFERENCES playlists(id),
    FOREIGN KEY (target_playlist_id) REFERENCES playlists(id)
);

-- Sequence counter for migrations
CREATE TABLE IF NOT EXISTS migrations_sequence (
    id INTEGER PRIMARY KEY,
    value INTEGER NOT NULL DEFAULT 0
);
INSERT INTO migrations_sequence (id, value) VALUES (1, 0);
CREATE INDEX IF NOT EXISTS idx_migrations_user_id ON migrations(user_id);
CREATE INDEX IF NOT EXISTS idx_migrations_status ON migrations(status);
