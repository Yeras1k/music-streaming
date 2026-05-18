package repository

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/music-streaming/music-service/internal/domain"
)

type PlaylistModel struct {
	ID          string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID      string    `gorm:"type:uuid;not null;index"`
	Name        string    `gorm:"not null"`
	Description string    `gorm:"type:text"`
	IsPublic    bool      `gorm:"default:false"`
	CreatedAt   time.Time `gorm:"autoCreateTime"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime"`
}

func (m *PlaylistModel) ToDomain() *domain.Playlist {
	return &domain.Playlist{
		ID:          m.ID,
		UserID:      m.UserID,
		Name:        m.Name,
		Description: m.Description,
		IsPublic:    m.IsPublic,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
	}
}

type PlaylistTrackModel struct {
	PlaylistID string    `gorm:"type:uuid;primaryKey"`
	TrackID    string    `gorm:"type:uuid;primaryKey"`
	AddedAt    time.Time `gorm:"autoCreateTime"`
}

type playlistRepository struct {
	db *gorm.DB
}

func NewPlaylistRepository(db *gorm.DB) domain.PlaylistRepository {
	return &playlistRepository{db: db}
}

func (r *playlistRepository) Create(ctx context.Context, playlist *domain.Playlist) error {
	model := &PlaylistModel{
		ID:          playlist.ID,
		UserID:      playlist.UserID,
		Name:        playlist.Name,
		Description: playlist.Description,
		IsPublic:    playlist.IsPublic,
	}
	return r.db.WithContext(ctx).Create(model).Error
}

func (r *playlistRepository) GetByID(ctx context.Context, id string) (*domain.Playlist, error) {
	var model PlaylistModel
	err := r.db.WithContext(ctx).First(&model, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, domain.ErrPlaylistNotFound
	}
	if err != nil {
		return nil, err
	}
	return model.ToDomain(), nil
}

func (r *playlistRepository) Update(ctx context.Context, playlist *domain.Playlist) error {
	return r.db.WithContext(ctx).Model(&PlaylistModel{}).
		Where("id = ?", playlist.ID).
		Updates(map[string]interface{}{
			"name":        playlist.Name,
			"description": playlist.Description,
			"is_public":   playlist.IsPublic,
			"updated_at":  time.Now(),
		}).Error
}

func (r *playlistRepository) Delete(ctx context.Context, id string) error {
	// Delete playlist tracks first (cascade)
	if err := r.db.WithContext(ctx).Delete(&PlaylistTrackModel{}, "playlist_id = ?", id).Error; err != nil {
		return err
	}
	return r.db.WithContext(ctx).Delete(&PlaylistModel{}, "id = ?", id).Error
}

func (r *playlistRepository) GetByUserID(ctx context.Context, userID string) ([]domain.Playlist, error) {
	var models []PlaylistModel
	err := r.db.WithContext(ctx).
		Where("user_id = ? OR is_public = true", userID).
		Order("created_at DESC").
		Find(&models).Error
	if err != nil {
		return nil, err
	}
	playlists := make([]domain.Playlist, len(models))
	for i, m := range models {
		playlists[i] = *m.ToDomain()
	}
	return playlists, nil
}

func (r *playlistRepository) AddTrack(ctx context.Context, playlistID, trackID string) error {
	model := &PlaylistTrackModel{
		PlaylistID: playlistID,
		TrackID:    trackID,
	}
	return r.db.WithContext(ctx).Create(model).Error
}

func (r *playlistRepository) RemoveTrack(ctx context.Context, playlistID, trackID string) error {
	return r.db.WithContext(ctx).
		Where("playlist_id = ? AND track_id = ?", playlistID, trackID).
		Delete(&PlaylistTrackModel{}).Error
}

func (r *playlistRepository) GetTracks(ctx context.Context, playlistID string) ([]domain.Track, error) {
	var models []TrackModel
	err := r.db.WithContext(ctx).
		Table("track_models").
		Joins("JOIN playlist_track_models ON playlist_track_models.track_id = track_models.id").
		Where("playlist_track_models.playlist_id = ?", playlistID).
		Find(&models).Error
	if err != nil {
		return nil, err
	}
	tracks := make([]domain.Track, len(models))
	for i, m := range models {
		tracks[i] = *m.ToDomain()
	}
	return tracks, nil
}