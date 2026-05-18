package repository

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/music-streaming/user-service/internal/domain"
)

type UserModel struct {
	ID        string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Email     string    `gorm:"uniqueIndex;not null"`
	Password  string    `gorm:"not null"`
	Username  string    `gorm:"not null"`
	Role      string    `gorm:"default:user"`
	Verified  bool      `gorm:"default:false"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

func (m *UserModel) ToDomain() *domain.User {
	return &domain.User{
		ID:        m.ID,
		Email:     m.Email,
		Password:  m.Password,
		Username:  m.Username,
		Role:      m.Role,
		Verified:  m.Verified,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}

func UserFromDomain(u *domain.User) *UserModel {
	return &UserModel{
		ID:        u.ID,
		Email:     u.Email,
		Password:  u.Password,
		Username:  u.Username,
		Role:      u.Role,
		Verified:  u.Verified,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
}

type userRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) domain.UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) Create(ctx context.Context, user *domain.User) error {
	model := UserFromDomain(user)
	return r.db.WithContext(ctx).Create(model).Error
}

func (r *userRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	var model UserModel
	err := r.db.WithContext(ctx).First(&model, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, domain.ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	return model.ToDomain(), nil
}

func (r *userRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	var model UserModel
	err := r.db.WithContext(ctx).First(&model, "email = ?", email).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, domain.ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	return model.ToDomain(), nil
}

func (r *userRepository) Update(ctx context.Context, user *domain.User) error {
	model := UserFromDomain(user)
	return r.db.WithContext(ctx).Model(&UserModel{}).
		Where("id = ?", model.ID).
		Updates(map[string]interface{}{
			"username":   model.Username,
			"email":      model.Email,
			"updated_at": time.Now(),
		}).Error
}

func (r *userRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&UserModel{}, "id = ?", id).Error
}

func (r *userRepository) UpdatePassword(ctx context.Context, id, hashedPassword string) error {
	return r.db.WithContext(ctx).Model(&UserModel{}).
		Where("id = ?", id).
		Update("password", hashedPassword).Error
}

func (r *userRepository) VerifyEmail(ctx context.Context, userID string) error {
	return r.db.WithContext(ctx).Model(&UserModel{}).
		Where("id = ?", userID).
		Update("verified", true).Error
}