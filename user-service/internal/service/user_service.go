package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/music-streaming/user-service/internal/domain"
	"github.com/music-streaming/user-service/pkg/cache"
	"github.com/music-streaming/user-service/pkg/email"
	"github.com/music-streaming/user-service/pkg/events"
)

type UserService struct {
	userRepo    domain.UserRepository
	sessionRepo domain.SessionRepository
	cache       *cache.UserCache
	email       *email.EmailSender
	events      *events.EventPublisher
	jwtSecret   []byte
}

func NewUserService(
	userRepo domain.UserRepository,
	sessionRepo domain.SessionRepository,
	cache *cache.UserCache,
	email *email.EmailSender,
	events *events.EventPublisher,
	jwtSecret string,
) *UserService {
	return &UserService{
		userRepo:    userRepo,
		sessionRepo: sessionRepo,
		cache:       cache,
		email:       email,
		events:      events,
		jwtSecret:   []byte(jwtSecret),
	}
}

func (s *UserService) Register(ctx context.Context, email, password, username string) (string, error) {
	// Check if user already exists
	if _, err := s.userRepo.GetByEmail(ctx, email); err == nil {
		return "", domain.ErrUserExists
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	// Create user
	userID := uuid.New().String()
	user := &domain.User{
		ID:       userID,
		Email:    email,
		Password: string(hashedPassword),
		Username: username,
		Role:     "user",
		Verified: false,
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return "", err
	}

	// Generate verification token
	verifyToken := generateRandomToken(32)
	if err := s.cache.SetVerificationToken(ctx, userID, verifyToken); err != nil {
		return "", err
	}

	// Send verification email (async)
	go s.email.SendVerificationEmail(email, userID, verifyToken)

	// Publish event (async)
	go s.events.PublishUserRegistered(ctx, domain.NewUserRegisteredEvent(userID, email, username))

	return userID, nil
}

func (s *UserService) Login(ctx context.Context, email, password string) (string, string, error) {
	// Get user
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return "", "", domain.ErrInvalidCredentials
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return "", "", domain.ErrInvalidCredentials
	}

	// Check if email is verified
	if !user.Verified {
		return "", "", domain.ErrEmailNotVerified
	}

	// Generate tokens
	accessToken, refreshToken, err := s.generateTokens(user.ID)
	if err != nil {
		return "", "", err
	}

	// Create session
	session := &domain.Session{
		ID:           uuid.New().String(),
		UserID:       user.ID,
		Token:        accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    time.Now().Add(24 * time.Hour),
	}

	if err := s.sessionRepo.Create(ctx, session); err != nil {
		return "", "", err
	}

	// Cache session
	if err := s.cache.SetSession(ctx, accessToken, user.ID); err != nil {
		return "", "", err
	}

	// Cache user
	if err := s.cache.SetUser(ctx, user); err != nil {
		return "", "", err
	}

	return accessToken, refreshToken, nil
}

func (s *UserService) GetUser(ctx context.Context, userID string) (*domain.User, error) {
	// Try cache first
	if user, err := s.cache.GetUser(ctx, userID); err == nil {
		return user, nil
	}

	// Get from database
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Cache for future
	_ = s.cache.SetUser(ctx, user)

	return user, nil
}

func (s *UserService) UpdateUser(ctx context.Context, userID, username, email string) (*domain.User, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if username != "" {
		user.Username = username
	}
	if email != "" {
		user.Email = email
	}
	user.UpdatedAt = time.Now()

	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, err
	}

	// Invalidate cache
	_ = s.cache.InvalidateUser(ctx, userID)

	return user, nil
}

func (s *UserService) DeleteUser(ctx context.Context, userID string) error {
	if err := s.userRepo.Delete(ctx, userID); err != nil {
		return err
	}

	// Delete all sessions
	_ = s.sessionRepo.DeleteByUserID(ctx, userID)

	// Invalidate cache
	_ = s.cache.InvalidateUser(ctx, userID)

	// Publish event (async)
	go s.events.PublishUserDeleted(ctx, domain.NewUserDeletedEvent(userID))

	return nil
}

func (s *UserService) ValidateToken(ctx context.Context, token string) (string, bool) {
	// Try cache first
	if userID, err := s.cache.GetSession(ctx, token); err == nil {
		return userID, true
	}

	// Check database
	session, err := s.sessionRepo.GetByToken(ctx, token)
	if err != nil {
		return "", false
	}

	// Parse JWT
	parsedToken, err := jwt.Parse(token, func(t *jwt.Token) (interface{}, error) {
		return s.jwtSecret, nil
	})

	if err != nil || !parsedToken.Valid {
		return "", false
	}

	// Cache for future
	_ = s.cache.SetSession(ctx, token, session.UserID)

	return session.UserID, true
}

func (s *UserService) Logout(ctx context.Context, token string) error {
	// Delete from database
	if err := s.sessionRepo.Delete(ctx, token); err != nil {
		return err
	}

	// Delete from cache
	_ = s.cache.DeleteSession(ctx, token)

	return nil
}

func (s *UserService) ChangePassword(ctx context.Context, userID, oldPassword, newPassword string) error {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}

	// Verify old password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(oldPassword)); err != nil {
		return domain.ErrInvalidPassword
	}

	// Hash new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	// Update password
	if err := s.userRepo.UpdatePassword(ctx, userID, string(hashedPassword)); err != nil {
		return err
	}

	// Delete all sessions (force re-login)
	_ = s.sessionRepo.DeleteByUserID(ctx, userID)

	return nil
}

func (s *UserService) VerifyEmail(ctx context.Context, userID, token string) error {
	storedToken, err := s.cache.GetVerificationToken(ctx, userID)
	if err != nil || storedToken != token {
		return domain.ErrInvalidToken
	}

	if err := s.userRepo.VerifyEmail(ctx, userID); err != nil {
		return err
	}

	return s.cache.DeleteVerificationToken(ctx, userID)
}

func (s *UserService) ForgotPassword(ctx context.Context, email string) error {
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		// Don't reveal if user exists
		return nil
	}

	resetToken := generateRandomToken(32)
	if err := s.cache.SetResetToken(ctx, user.ID, resetToken); err != nil {
		return err
	}

	go s.email.SendPasswordResetEmail(email, resetToken)

	return nil
}

func (s *UserService) ResetPassword(ctx context.Context, token, newPassword string) error {
	userID, err := s.cache.GetResetTokenUser(ctx, token)
	if err != nil {
		return domain.ErrInvalidToken
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	if err := s.userRepo.UpdatePassword(ctx, userID, string(hashedPassword)); err != nil {
		return err
	}

	// Delete reset token
	_ = s.cache.DeleteResetToken(ctx, token)

	// Delete all sessions
	_ = s.sessionRepo.DeleteByUserID(ctx, userID)

	return nil
}

func (s *UserService) generateTokens(userID string) (string, string, error) {
	// Access token (15 minutes)
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": userID,
		"type":    "access",
		"exp":     time.Now().Add(15 * time.Minute).Unix(),
	})

	accessTokenString, err := accessToken.SignedString(s.jwtSecret)
	if err != nil {
		return "", "", err
	}

	// Refresh token (7 days)
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": userID,
		"type":    "refresh",
		"exp":     time.Now().Add(7 * 24 * time.Hour).Unix(),
	})

	refreshTokenString, err := refreshToken.SignedString(s.jwtSecret)
	if err != nil {
		return "", "", err
	}

	return accessTokenString, refreshTokenString, nil
}

func generateRandomToken(length int) string {
	bytes := make([]byte, length)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}