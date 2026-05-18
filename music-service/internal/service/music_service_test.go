package service

import (
	"context"
	"testing"

	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/music-streaming/music-service/internal/domain"
)

type MockTrackRepository struct {
	mock.Mock
}

func (m *MockTrackRepository) Create(ctx context.Context, track *domain.Track) error {
	args := m.Called(ctx, track)
	return args.Error(0)
}

func (m *MockTrackRepository) GetByID(ctx context.Context, id string) (*domain.Track, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Track), args.Error(1)
}

func (m *MockTrackRepository) Update(ctx context.Context, track *domain.Track) error {
	args := m.Called(ctx, track)
	return args.Error(0)
}

func (m *MockTrackRepository) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockTrackRepository) List(ctx context.Context, page, pageSize int32) ([]domain.Track, int64, error) {
	args := m.Called(ctx, page, pageSize)
	return args.Get(0).([]domain.Track), args.Get(1).(int64), args.Error(2)
}

func (m *MockTrackRepository) Search(ctx context.Context, query string, page, pageSize int32) ([]domain.Track, int64, error) {
	args := m.Called(ctx, query, page, pageSize)
	return args.Get(0).([]domain.Track), args.Get(1).(int64), args.Error(2)
}

func (m *MockTrackRepository) IncrementPlays(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockTrackRepository) IncrementLikes(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockTrackRepository) DecrementLikes(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockTrackRepository) GetByUserID(ctx context.Context, userID string) ([]domain.Track, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]domain.Track), args.Error(1)
}

type MockPlaylistRepository struct {
	mock.Mock
}

func (m *MockPlaylistRepository) Create(ctx context.Context, playlist *domain.Playlist) error {
	args := m.Called(ctx, playlist)
	return args.Error(0)
}

func (m *MockPlaylistRepository) GetByID(ctx context.Context, id string) (*domain.Playlist, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Playlist), args.Error(1)
}

func (m *MockPlaylistRepository) Update(ctx context.Context, playlist *domain.Playlist) error {
	args := m.Called(ctx, playlist)
	return args.Error(0)
}

func (m *MockPlaylistRepository) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockPlaylistRepository) GetByUserID(ctx context.Context, userID string) ([]domain.Playlist, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]domain.Playlist), args.Error(1)
}

func (m *MockPlaylistRepository) AddTrack(ctx context.Context, playlistID, trackID string) error {
	args := m.Called(ctx, playlistID, trackID)
	return args.Error(0)
}

func (m *MockPlaylistRepository) RemoveTrack(ctx context.Context, playlistID, trackID string) error {
	args := m.Called(ctx, playlistID, trackID)
	return args.Error(0)
}

func (m *MockPlaylistRepository) GetTracks(ctx context.Context, playlistID string) ([]domain.Track, error) {
	args := m.Called(ctx, playlistID)
	return args.Get(0).([]domain.Track), args.Error(1)
}

type MockLikeRepository struct {
	mock.Mock
}

func (m *MockLikeRepository) Create(ctx context.Context, userID, trackID string) error {
	args := m.Called(ctx, userID, trackID)
	return args.Error(0)
}

func (m *MockLikeRepository) Delete(ctx context.Context, userID, trackID string) error {
	args := m.Called(ctx, userID, trackID)
	return args.Error(0)
}

func (m *MockLikeRepository) IsLiked(ctx context.Context, userID, trackID string) (bool, error) {
	args := m.Called(ctx, userID, trackID)
	return args.Bool(0), args.Error(1)
}

func (m *MockLikeRepository) GetUserLikes(ctx context.Context, userID string, page, pageSize int32) ([]domain.Track, int64, error) {
	args := m.Called(ctx, userID, page, pageSize)
	return args.Get(0).([]domain.Track), args.Get(1).(int64), args.Error(2)
}

type MockTrackCache struct {
	mock.Mock
}

func (m *MockTrackCache) SetTrack(ctx context.Context, track *domain.Track) error {
	args := m.Called(ctx, track)
	return args.Error(0)
}

func (m *MockTrackCache) GetTrack(ctx context.Context, trackID string) (*domain.Track, error) {
	args := m.Called(ctx, trackID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Track), args.Error(1)
}

func (m *MockTrackCache) InvalidateTrack(ctx context.Context, trackID string) error {
	args := m.Called(ctx, trackID)
	return args.Error(0)
}

func (m *MockTrackCache) SetPlaylist(ctx context.Context, playlistID string, tracks []domain.Track) error {
	args := m.Called(ctx, playlistID, tracks)
	return args.Error(0)
}

func (m *MockTrackCache) GetPlaylist(ctx context.Context, playlistID string) ([]domain.Track, error) {
	args := m.Called(ctx, playlistID)
	return args.Get(0).([]domain.Track), args.Error(1)
}

func (m *MockTrackCache) InvalidatePlaylist(ctx context.Context, playlistID string) error {
	args := m.Called(ctx, playlistID)
	return args.Error(0)
}

func TestMusicService_GetTrack_CacheHit(t *testing.T) {
	mockTrackRepo := new(MockTrackRepository)
	mockPlaylistRepo := new(MockPlaylistRepository)
	mockLikeRepo := new(MockLikeRepository)
	mockCache := new(MockTrackCache)
	mockEvents := new(MockEventPublisher)
	mockWorker := new(MockWorkerPool)
	mockRedis := new(MockRedisClient)

	expectedTrack := &domain.Track{ID: "123", Title: "Test Song", Artist: "Test Artist"}
	mockCache.On("GetTrack", mock.Anything, "123").Return(expectedTrack, nil)

	service := NewMusicService(mockTrackRepo, mockPlaylistRepo, mockLikeRepo, mockCache, mockEvents, mockWorker, "/uploads", mockRedis)

	track, err := service.GetTrack(context.Background(), "123")

	assert.NoError(t, err)
	assert.Equal(t, expectedTrack, track)
}

func TestMusicService_LikeTrack_NewLike(t *testing.T) {
	mockTrackRepo := new(MockTrackRepository)
	mockPlaylistRepo := new(MockPlaylistRepository)
	mockLikeRepo := new(MockLikeRepository)
	mockCache := new(MockTrackCache)
	mockEvents := new(MockEventPublisher)
	mockWorker := new(MockWorkerPool)
	mockRedis := new(MockRedisClient)

	mockLikeRepo.On("IsLiked", mock.Anything, "user123", "track123").Return(false, nil)
	mockLikeRepo.On("Create", mock.Anything, "user123", "track123").Return(nil)
	mockTrackRepo.On("IncrementLikes", mock.Anything, "track123").Return(nil)
	mockTrackRepo.On("GetByID", mock.Anything, "track123").Return(&domain.Track{ID: "track123", Likes: 5}, nil)

	service := NewMusicService(mockTrackRepo, mockPlaylistRepo, mockLikeRepo, mockCache, mockEvents, mockWorker, "/uploads", mockRedis)

	liked, likes, err := service.LikeTrack(context.Background(), "user123", "track123")

	assert.NoError(t, err)
	assert.True(t, liked)
	assert.Equal(t, int64(5), likes)
}

func TestMusicService_LikeTrack_Unlike(t *testing.T) {
	mockTrackRepo := new(MockTrackRepository)
	mockPlaylistRepo := new(MockPlaylistRepository)
	mockLikeRepo := new(MockLikeRepository)
	mockCache := new(MockTrackCache)
	mockEvents := new(MockEventPublisher)
	mockWorker := new(MockWorkerPool)
	mockRedis := new(MockRedisClient)

	mockLikeRepo.On("IsLiked", mock.Anything, "user123", "track123").Return(true, nil)
	mockLikeRepo.On("Delete", mock.Anything, "user123", "track123").Return(nil)
	mockTrackRepo.On("DecrementLikes", mock.Anything, "track123").Return(nil)

	service := NewMusicService(mockTrackRepo, mockPlaylistRepo, mockLikeRepo, mockCache, mockEvents, mockWorker, "/uploads", mockRedis)

	liked, _, err := service.LikeTrack(context.Background(), "user123", "track123")

	assert.NoError(t, err)
	assert.False(t, liked)
}