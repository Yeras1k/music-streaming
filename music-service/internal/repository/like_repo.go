package repository

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/music-streaming/music-service/internal/domain"
)

type LikeModel struct {
	UserID    string    `gorm:"type:uuid;primaryKey"`
	TrackID   string    `gorm:"type:uuid;primaryKey"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
}

type likeRepository struct {
	db *gorm.DB
}

func NewLikeRepository(db *gorm.DB) domain.LikeRepository {
	return &likeRepository{db: db}
}

func (r *likeRepository) Create(ctx context.Context, userID, trackID string) error {
	model := &LikeModel{
		UserID:  userID,
		TrackID: trackID,
	}
	return r.db.WithContext(ctx).Create(model).Error
}

func (r *likeRepository) Delete(ctx context.Context, userID, trackID string) error {
	return r.db.WithContext(ctx).
		Where("user_id = ? AND track_id = ?", userID, trackID).
		Delete(&LikeModel{}).Error
}

func (r *likeRepository) IsLiked(ctx context.Context, userID, trackID string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&LikeModel{}).
		Where("user_id = ? AND track_id = ?", userID, trackID).
		Count(&count).Error
	return count > 0, err
}

func (r *likeRepository) GetUserLikes(ctx context.Context, userID string, page, pageSize int32) ([]domain.Track, int64, error) {
	var models []TrackModel
	offset := (page - 1) * pageSize

	var total int64
	r.db.WithContext(ctx).
		Model(&TrackModel{}).
		Joins("JOIN like_models ON like_models.track_id = track_models.id").
		Where("like_models.user_id = ?", userID).
		Count(&total)

	err := r.db.WithContext(ctx).
		Table("track_models").
		Joins("JOIN like_models ON like_models.track_id = track_models.id").
		Where("like_models.user_id = ?", userID).
		Offset(int(offset)).
		Limit(int(pageSize)).
		Order("like_models.created_at DESC").
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