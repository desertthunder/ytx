-- Rollback soft delete support

-- Drop indexes
DROP INDEX IF EXISTS idx_migrations_deleted_at;
DROP INDEX IF EXISTS idx_playlist_tracks_deleted_at;
DROP INDEX IF EXISTS idx_tracks_deleted_at;
DROP INDEX IF EXISTS idx_playlists_deleted_at;
DROP INDEX IF EXISTS idx_users_deleted_at;

-- Remove deleted_at columns (SQLite requires creating new tables without the column)
-- Users
CREATE TABLE users_new (
    id TEXT PRIMARY KEY,
    sequence INTEGER NOT NULL UNIQUE,
    email TEXT UNIQUE NOT NULL,
    name TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
INSERT INTO users_new SELECT id, sequence, email, name, created_at, updated_at FROM users;
DROP TABLE users;
ALTER TABLE users_new RENAME TO users;

-- Playlists
CREATE TABLE playlists_new (
    id TEXT PRIMARY KEY,
    sequence INTEGER NOT NULL UNIQUE,
    service TEXT NOT NULL,
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
INSERT INTO playlists_new SELECT id, sequence, service, service_id, user_id, name, description, track_count, public, created_at, updated_at FROM playlists;
DROP TABLE playlists;
ALTER TABLE playlists_new RENAME TO playlists;
CREATE INDEX IF NOT EXISTS idx_playlists_user_id ON playlists(user_id);
CREATE INDEX IF NOT EXISTS idx_playlists_service ON playlists(service);

-- Tracks
CREATE TABLE tracks_new (
    id TEXT PRIMARY KEY,
    sequence INTEGER NOT NULL UNIQUE,
    service TEXT NOT NULL,
    service_id TEXT NOT NULL,
    title TEXT NOT NULL,
    artist TEXT NOT NULL,
    album TEXT,
    duration INTEGER,
    isrc TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(service, service_id)
);
INSERT INTO tracks_new SELECT id, sequence, service, service_id, title, artist, album, duration, isrc, created_at, updated_at FROM tracks;
DROP TABLE tracks;
ALTER TABLE tracks_new RENAME TO tracks;
CREATE INDEX IF NOT EXISTS idx_tracks_service ON tracks(service);
CREATE INDEX IF NOT EXISTS idx_tracks_isrc ON tracks(isrc);

-- Playlist tracks
CREATE TABLE playlist_tracks_new (
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
INSERT INTO playlist_tracks_new SELECT id, sequence, playlist_id, track_id, position, created_at FROM playlist_tracks;
DROP TABLE playlist_tracks;
ALTER TABLE playlist_tracks_new RENAME TO playlist_tracks;
CREATE INDEX IF NOT EXISTS idx_playlist_tracks_playlist ON playlist_tracks(playlist_id);
CREATE INDEX IF NOT EXISTS idx_playlist_tracks_position ON playlist_tracks(playlist_id, position);

-- Migrations
CREATE TABLE migrations_new (
    id TEXT PRIMARY KEY,
    sequence INTEGER NOT NULL UNIQUE,
    user_id TEXT NOT NULL,
    source_service TEXT NOT NULL,
    source_playlist_id TEXT NOT NULL,
    target_service TEXT NOT NULL,
    target_playlist_id TEXT,
    status TEXT DEFAULT 'pending',
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
INSERT INTO migrations_new SELECT id, sequence, user_id, source_service, source_playlist_id, target_service, target_playlist_id, status, tracks_total, tracks_migrated, tracks_failed, error_message, started_at, completed_at, created_at, updated_at FROM migrations;
DROP TABLE migrations;
ALTER TABLE migrations_new RENAME TO migrations;
CREATE INDEX IF NOT EXISTS idx_migrations_user_id ON migrations(user_id);
CREATE INDEX IF NOT EXISTS idx_migrations_status ON migrations(status);
