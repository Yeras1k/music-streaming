package repository

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/music-streaming/user-service/internal/domain"
)

type SessionModel struct {
	ID           string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID       string    `gorm:"type:uuid;not null;index"`
	Token        string    `gorm:"uniqueIndex;not null"`
	RefreshToken string    `gorm:"uniqueIndex"`
	ExpiresAt    time.Time `gorm:"not null"`
	CreatedAt    time.Time `gorm:"autoCreateTime"`
}

func (m *SessionModel) ToDomain() *domain.Session {
	return &domain.Session{
		ID:           m.ID,
		UserID:       m.UserID,
		Token:        m.Token,
		RefreshToken: m.RefreshToken,
		ExpiresAt:    m.ExpiresAt,
		CreatedAt:    m.CreatedAt,
	}
}

type sessionRepository struct {
	db *gorm.DB
}

func NewSessionRepository(db *gorm.DB) domain.SessionRepository {
	return &sessionRepository{db: db}
}

func (r *sessionRepository) Create(ctx context.Context, session *domain.Session) error {
	model := &SessionModel{
		ID:           session.ID,
		UserID:       session.UserID,
		Token:        session.Token,
		RefreshToken: session.RefreshToken,
		ExpiresAt:    session.ExpiresAt,
	}
	return r.db.WithContext(ctx).Create(model).Error
}

func (r *sessionRepository) GetByToken(ctx context.Context, token string) (*domain.Session, error) {
	var model SessionModel
	err := r.db.WithContext(ctx).
		Where("token = ? AND expires_at > ?", token, time.Now()).
		First(&model).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, domain.ErrSessionNotFound
	}
	if err != nil {
		return nil, err
	}
	return model.ToDomain(), nil
}

func (r *sessionRepository) GetByRefreshToken(ctx context.Context, refreshToken string) (*domain.Session, error) {
	var model SessionModel
	err := r.db.WithContext(ctx).Where("refresh_token = ?", refreshToken).First(&model).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, domain.ErrSessionNotFound
	}
	if err != nil {
		return nil, err
	}
	return model.ToDomain(), nil
}

func (r *sessionRepository) Delete(ctx context.Context, token string) error {
	return r.db.WithContext(ctx).Delete(&SessionModel{}, "token = ?", token).Error
}

func (r *sessionRepository) DeleteByUserID(ctx context.Context, userID string) error {
	return r.db.WithContext(ctx).Delete(&SessionModel{}, "user_id = ?", userID).Error
}