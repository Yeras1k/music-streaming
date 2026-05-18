package repository

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/music-streaming/music-service/internal/domain"
)

type TrackModel struct {
	ID        string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID    string    `gorm:"type:uuid;not null;index"`
	Title     string    `gorm:"not null;index"`
	Artist    string    `gorm:"not null;index"`
	Album     string    `gorm:"index"`
	Duration  int32     `gorm:"not null"`
	Genre     string    `gorm:"index"`
	URL       string    `gorm:"not null"`
	Plays     int64     `gorm:"default:0"`
	Likes     int64     `gorm:"default:0"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

func (m *TrackModel) ToDomain() *domain.Track {
	return &domain.Track{
		ID:        m.ID,
		UserID:    m.UserID,
		Title:     m.Title,
		Artist:    m.Artist,
		Album:     m.Album,
		Duration:  m.Duration,
		Genre:     m.Genre,
		URL:       m.URL,
		Plays:     m.Plays,
		Likes:     m.Likes,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}

type trackRepository struct {
	db *gorm.DB
}

func NewTrackRepository(db *gorm.DB) domain.TrackRepository {
	return &trackRepository{db: db}
}

func (r *trackRepository) Create(ctx context.Context, track *domain.Track) error {
	model := &TrackModel{
		ID:       track.ID,
		UserID:   track.UserID,
		Title:    track.Title,
		Artist:   track.Artist,
		Album:    track.Album,
		Duration: track.Duration,
		Genre:    track.Genre,
		URL:      track.URL,
	}
	return r.db.WithContext(ctx).Create(model).Error
}

func (r *trackRepository) GetByID(ctx context.Context, id string) (*domain.Track, error) {
	var model TrackModel
	err := r.db.WithContext(ctx).First(&model, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, domain.ErrTrackNotFound
	}
	if err != nil {
		return nil, err
	}
	return model.ToDomain(), nil
}

func (r *trackRepository) Update(ctx context.Context, track *domain.Track) error {
	return r.db.WithContext(ctx).Model(&TrackModel{}).
		Where("id = ?", track.ID).
		Updates(map[string]interface{}{
			"title":      track.Title,
			"artist":     track.Artist,
			"album":      track.Album,
			"genre":      track.Genre,
			"updated_at": time.Now(),
		}).Error
}

func (r *trackRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&TrackModel{}, "id = ?", id).Error
}

func (r *trackRepository) List(ctx context.Context, page, pageSize int32) ([]domain.Track, int64, error) {
	var models []TrackModel
	offset := (page - 1) * pageSize

	var total int64
	r.db.WithContext(ctx).Model(&TrackModel{}).Count(&total)

	err := r.db.WithContext(ctx).
		Offset(int(offset)).
		Limit(int(pageSize)).
		Order("created_at DESC").
		Find(&models).Error

	if err != nil {
		return nil, 0, err
	}

	tracks := make([]domain.Track, len(models))
	for i, m := range models {
		tracks[i] = *m.ToDomain()
	}
	return tracks, total, nil
}

func (r *trackRepository) Search(ctx context.Context, query string, page, pageSize int32) ([]domain.Track, int64, error) {
	var models []TrackModel
	offset := (page - 1) * pageSize

	searchQuery := "%" + query + "%"
	var total int64

	r.db.WithContext(ctx).Model(&TrackModel{}).
		Where("title ILIKE ? OR artist ILIKE ? OR album ILIKE ?", searchQuery, searchQuery, searchQuery).
		Count(&total)

	err := r.db.WithContext(ctx).
		Where("title ILIKE ? OR artist ILIKE ? OR album ILIKE ?", searchQuery, searchQuery, searchQuery).
		Offset(int(offset)).
		Limit(int(pageSize)).
		Order("plays DESC").
		Find(&models).Error

	if err != nil {
		return nil, 0, err
	}

	tracks := make([]domain.Track, len(models))
	for i, m := range models {
		tracks[i] = *m.ToDomain()
	}
	return tracks, total, nil
}

func (r *trackRepository) IncrementPlays(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Model(&TrackModel{}).
		Where("id = ?", id).
		UpdateColumn("plays", gorm.Expr("plays + ?", 1)).Error
}

func (r *trackRepository) IncrementLikes(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Model(&TrackModel{}).
		Where("id = ?", id).
		UpdateColumn("likes", gorm.Expr("likes + ?", 1)).Error
}

func (r *trackRepository) DecrementLikes(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Model(&TrackModel{}).
		Where("id = ?", id).
		UpdateColumn("likes", gorm.Expr("likes - ?", 1)).Error
}

func (r *trackRepository) GetByUserID(ctx context.Context, userID string) ([]domain.Track, error) {
	var models []TrackModel
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
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