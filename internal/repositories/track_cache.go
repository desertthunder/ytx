package repositories

import (
	"fmt"
	"strings"

	"github.com/desertthunder/ytx/internal/models"
)

// TrackCacheAdapter implements tasks.TrackCacher using TrackRepository.
//
// Provides automatic track caching with deduplication via service+service_id constraints.
// Duplicate tracks are silently ignored (UNIQUE constraint violations).
type TrackCacheAdapter struct {
	repo *TrackRepository
}

// NewTrackCacheAdapter creates a new TrackCacheAdapter with the given repository
func NewTrackCacheAdapter(repo *TrackRepository) *TrackCacheAdapter {
	return &TrackCacheAdapter{repo: repo}
}

// CacheTrack caches a track from a service.
// Returns nil if the track already exists (deduplication).
// Only returns errors for actual failures (not constraint violations).
func (a *TrackCacheAdapter) CacheTrack(service, serviceID string, track models.Track) error {
	existing, err := a.repo.GetByServiceID(service, serviceID)
	if err == nil && existing != nil {
		return nil
	}

	persistedTrack := models.NewPersistedTrack(0, service, serviceID, track)

	err = a.repo.Create(persistedTrack)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint") {
			return nil
		}
		return fmt.Errorf("failed to cache track: %w", err)
	}

	return nil
}
