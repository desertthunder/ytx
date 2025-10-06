-- Rollback initial schema

DROP INDEX IF EXISTS idx_migrations_status;
DROP INDEX IF EXISTS idx_migrations_user_id;
DROP TABLE IF EXISTS migrations_sequence;
DROP TABLE IF EXISTS migrations;

DROP INDEX IF EXISTS idx_playlist_tracks_position;
DROP INDEX IF EXISTS idx_playlist_tracks_playlist;
DROP TABLE IF EXISTS playlist_tracks_sequence;
DROP TABLE IF EXISTS playlist_tracks;

DROP INDEX IF EXISTS idx_tracks_isrc;
DROP INDEX IF EXISTS idx_tracks_service;
DROP TABLE IF EXISTS tracks_sequence;
DROP TABLE IF EXISTS tracks;

DROP INDEX IF EXISTS idx_playlists_service;
DROP INDEX IF EXISTS idx_playlists_user_id;
DROP TABLE IF EXISTS playlists_sequence;
DROP TABLE IF EXISTS playlists;

DROP TABLE IF EXISTS users_sequence;
DROP TABLE IF EXISTS users;
