-- Add soft delete support to all tables

-- Add deleted_at column to users
ALTER TABLE users ADD COLUMN deleted_at TIMESTAMP DEFAULT NULL;

-- Add deleted_at column to playlists
ALTER TABLE playlists ADD COLUMN deleted_at TIMESTAMP DEFAULT NULL;

-- Add deleted_at column to tracks
ALTER TABLE tracks ADD COLUMN deleted_at TIMESTAMP DEFAULT NULL;

-- Add deleted_at column to playlist_tracks
ALTER TABLE playlist_tracks ADD COLUMN deleted_at TIMESTAMP DEFAULT NULL;

-- Add deleted_at column to migrations
ALTER TABLE migrations ADD COLUMN deleted_at TIMESTAMP DEFAULT NULL;

-- Create indexes for soft delete queries
CREATE INDEX IF NOT EXISTS idx_users_deleted_at ON users(deleted_at);
CREATE INDEX IF NOT EXISTS idx_playlists_deleted_at ON playlists(deleted_at);
CREATE INDEX IF NOT EXISTS idx_tracks_deleted_at ON tracks(deleted_at);
CREATE INDEX IF NOT EXISTS idx_playlist_tracks_deleted_at ON playlist_tracks(deleted_at);
CREATE INDEX IF NOT EXISTS idx_migrations_deleted_at ON migrations(deleted_at);
