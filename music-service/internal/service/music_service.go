package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"

	"github.com/music-streaming/music-service/internal/domain"
	"github.com/music-streaming/music-service/internal/worker"
	"github.com/music-streaming/music-service/pkg/cache"
	"github.com/music-streaming/music-service/pkg/events"
)

type MusicService struct {
	trackRepo    domain.TrackRepository
	playlistRepo domain.PlaylistRepository
	likeRepo     domain.LikeRepository
	cache        *cache.TrackCache
	events       *events.EventPublisher
	workerPool   *worker.WorkerPool
	uploadPath   string
	redis        *redis.Client
}

func NewMusicService(
	trackRepo domain.TrackRepository,
	playlistRepo domain.PlaylistRepository,
	likeRepo domain.LikeRepository,
	cache *cache.TrackCache,
	events *events.EventPublisher,
	workerPool *worker.WorkerPool,
	uploadPath string,
	redis *redis.Client,
) *MusicService {
	return &MusicService{
		trackRepo:    trackRepo,
		playlistRepo: playlistRepo,
		likeRepo:     likeRepo,
		cache:        cache,
		events:       events,
		workerPool:   workerPool,
		uploadPath:   uploadPath,
		redis:        redis,
	}
}

func (s *MusicService) UploadTrack(ctx context.Context, userID, title, artist, album, genre string, duration int32, audioData []byte) (*domain.Track, error) {
	// Rate limiting
	if err := s.checkRateLimit(ctx, userID); err != nil {
		return nil, err
	}

	// Generate track ID
	trackID := uuid.New().String()

	// Save audio file
	filename := fmt.Sprintf("%s.mp3", trackID)
	filePath := filepath.Join(s.uploadPath, filename)

	if err := os.WriteFile(filePath, audioData, 0644); err != nil {
		return nil, fmt.Errorf("failed to save audio file: %w", err)
	}

	// Create track record
	track := &domain.Track{
		ID:        trackID,
		UserID:    userID,
		Title:     title,
		Artist:    artist,
		Album:     album,
		Genre:     genre,
		Duration:  duration,
		URL:       fmt.Sprintf("/uploads/%s", filename),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.trackRepo.Create(ctx, track); err != nil {
		os.Remove(filePath)
		return nil, fmt.Errorf("failed to create track record: %w", err)
	}

	// Queue transcoding job
	s.workerPool.Enqueue(worker.Job{
		ID:   trackID,
		Type: "transcode",
		Payload: map[string]interface{}{
			"input_path":  filePath,
			"output_path": filepath.Join(s.uploadPath, fmt.Sprintf("%s_transcoded.mp3", trackID)),
		},
	})

	// Publish event
	s.events.PublishTrackUploaded(ctx, domain.NewTrackUploadedEvent(trackID, userID, title, artist))

	// Cache track
	s.cache.SetTrack(ctx, track)

	return track, nil
}

func (s *MusicService) GetTrack(ctx context.Context, trackID string) (*domain.Track, error) {
	// Check cache first
	if track, err := s.cache.GetTrack(ctx, trackID); err == nil {
		// Increment plays asynchronously
		go s.trackRepo.IncrementPlays(context.Background(), trackID)
		return track, nil
	}

	// Get from database
	track, err := s.trackRepo.GetByID(ctx, trackID)
	if err != nil {
		return nil, err
	}

	// Cache for future requests
	s.cache.SetTrack(ctx, track)

	// Increment plays asynchronously
	go s.trackRepo.IncrementPlays(context.Background(), trackID)

	return track, nil
}

func (s *MusicService) ListTracks(ctx context.Context, page, pageSize int32) ([]domain.Track, int64, error) {
	return s.trackRepo.List(ctx, page, pageSize)
}

func (s *MusicService) SearchTracks(ctx context.Context, query string, page, pageSize int32) ([]domain.Track, int64, error) {
	return s.trackRepo.Search(ctx, query, page, pageSize)
}

func (s *MusicService) CreatePlaylist(ctx context.Context, userID, name, description string, isPublic bool) (*domain.Playlist, error) {
	playlist := &domain.Playlist{
		ID:          uuid.New().String(),
		UserID:      userID,
		Name:        name,
		Description: description,
		IsPublic:    isPublic,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := s.playlistRepo.Create(ctx, playlist); err != nil {
		return nil, err
	}

	return playlist, nil
}

func (s *MusicService) AddToPlaylist(ctx context.Context, playlistID, trackID, userID string) (*domain.Playlist, error) {
	// Verify ownership
	playlist, err := s.playlistRepo.GetByID(ctx, playlistID)
	if err != nil {
		return nil, err
	}
	if playlist.UserID != userID {
		return nil, domain.ErrUnauthorized
	}

	// Add track
	if err := s.playlistRepo.AddTrack(ctx, playlistID, trackID); err != nil {
		return nil, err
	}

	// Invalidate cache
	s.cache.InvalidatePlaylist(ctx, playlistID)

	return playlist, nil
}

func (s *MusicService) RemoveFromPlaylist(ctx context.Context, playlistID, trackID, userID string) (*domain.Playlist, error) {
	// Verify ownership
	playlist, err := s.playlistRepo.GetByID(ctx, playlistID)
	if err != nil {
		return nil, err
	}
	if playlist.UserID != userID {
		return nil, domain.ErrUnauthorized
	}

	// Remove track
	if err := s.playlistRepo.RemoveTrack(ctx, playlistID, trackID); err != nil {
		return nil, err
	}

	// Invalidate cache
	s.cache.InvalidatePlaylist(ctx, playlistID)

	return playlist, nil
}

func (s *MusicService) GetPlaylist(ctx context.Context, playlistID, userID string) (*domain.Playlist, []domain.Track, error) {
	playlist, err := s.playlistRepo.GetByID(ctx, playlistID)
	if err != nil {
		return nil, nil, err
	}

	// Check if user can view this playlist
	if !playlist.IsPublic && playlist.UserID != userID {
		return nil, nil, domain.ErrUnauthorized
	}

	// Try cache first
	if tracks, err := s.cache.GetPlaylist(ctx, playlistID); err == nil {
		return playlist, tracks, nil
	}

	// Get tracks from database
	tracks, err := s.playlistRepo.GetTracks(ctx, playlistID)
	if err != nil {
		return nil, nil, err
	}

	// Cache for future
	s.cache.SetPlaylist(ctx, playlistID, tracks)

	return playlist, tracks, nil
}

func (s *MusicService) GetUserPlaylists(ctx context.Context, userID string) ([]domain.Playlist, error) {
	return s.playlistRepo.GetByUserID(ctx, userID)
}

func (s *MusicService) LikeTrack(ctx context.Context, userID, trackID string) (bool, int64, error) {
	// Check if already liked
	isLiked, err := s.likeRepo.IsLiked(ctx, userID, trackID)
	if err != nil {
		return false, 0, err
	}

	if isLiked {
		// Unlike
		if err := s.likeRepo.Delete(ctx, userID, trackID); err != nil {
			return false, 0, err
		}
		if err := s.trackRepo.DecrementLikes(ctx, trackID); err != nil {
			return false, 0, err
		}
		return false, 0, nil
	}

	// Like
	if err := s.likeRepo.Create(ctx, userID, trackID); err != nil {
		return false, 0, err
	}
	if err := s.trackRepo.IncrementLikes(ctx, trackID); err != nil {
		return false, 0, err
	}

	// Get updated like count
	track, err := s.trackRepo.GetByID(ctx, trackID)
	if err != nil {
		return true, 0, nil
	}

	return true, track.Likes, nil
}

func (s *MusicService) GetRecommendations(ctx context.Context, userID string, limit int32) ([]domain.Track, error) {
	// Simple recommendation: Get most played tracks
	tracks, _, err := s.trackRepo.List(ctx, 1, limit)
	if err != nil {
		return nil, err
	}
	return tracks, nil
}

func (s *MusicService) AddToQueue(ctx context.Context, userID, trackID string) (int32, error) {
	key := fmt.Sprintf("queue:%s", userID)

	// Add to Redis list
	size, err := s.redis.RPush(ctx, key, trackID).Result()
	if err != nil {
		return 0, err
	}

	// Set expiration
	s.redis.Expire(ctx, key, 24*time.Hour)

	return int32(size), nil
}

func (s *MusicService) GetQueue(ctx context.Context, userID string) ([]domain.Track, error) {
	key := fmt.Sprintf("queue:%s", userID)

	// Get track IDs from Redis
	trackIDs, err := s.redis.LRange(ctx, key, 0, -1).Result()
	if err != nil {
		return nil, err
	}

	// Get tracks from database
	var tracks []domain.Track
	for _, id := range trackIDs {
		track, err := s.trackRepo.GetByID(ctx, id)
		if err == nil {
			tracks = append(tracks, *track)
		}
	}

	return tracks, nil
}

func (s *MusicService) checkRateLimit(ctx context.Context, userID string) error {
	key := fmt.Sprintf("rate:upload:%s", userID)

	count, err := s.redis.Incr(ctx, key).Result()
	if err != nil {
		return nil // Don't block on rate limit error
	}

	if count > 10 {
		return domain.ErrRateLimitExceeded
	}

	if count == 1 {
		s.redis.Expire(ctx, key, time.Hour)
	}

	return nil
}