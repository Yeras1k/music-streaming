package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"

	"github.com/music-streaming/music-service/internal/domain"
)

type TrackCache struct {
	client *redis.Client
}

func NewTrackCache(client *redis.Client) *TrackCache {
	return &TrackCache{client: client}
}

func (c *TrackCache) SetTrack(ctx context.Context, track *domain.Track) error {
	data, err := json.Marshal(track)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, fmt.Sprintf("track:%s", track.ID), data, 1*time.Hour).Err()
}

func (c *TrackCache) GetTrack(ctx context.Context, trackID string) (*domain.Track, error) {
	data, err := c.client.Get(ctx, fmt.Sprintf("track:%s", trackID)).Bytes()
	if err != nil {
		return nil, err
	}
	var track domain.Track
	if err := json.Unmarshal(data, &track); err != nil {
		return nil, err
	}
	return &track, nil
}

func (c *TrackCache) InvalidateTrack(ctx context.Context, trackID string) error {
	return c.client.Del(ctx, fmt.Sprintf("track:%s", trackID)).Err()
}

func (c *TrackCache) SetPlaylist(ctx context.Context, playlistID string, tracks []domain.Track) error {
	data, err := json.Marshal(tracks)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, fmt.Sprintf("playlist:%s", playlistID), data, 10*time.Minute).Err()
}

func (c *TrackCache) GetPlaylist(ctx context.Context, playlistID string) ([]domain.Track, error) {
	data, err := c.client.Get(ctx, fmt.Sprintf("playlist:%s", playlistID)).Bytes()
	if err != nil {
		return nil, err
	}
	var tracks []domain.Track
	if err := json.Unmarshal(data, &tracks); err != nil {
		return nil, err
	}
	return tracks, nil
}

func (c *TrackCache) InvalidatePlaylist(ctx context.Context, playlistID string) error {
	return c.client.Del(ctx, fmt.Sprintf("playlist:%s", playlistID)).Err()
}