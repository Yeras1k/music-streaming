package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/music-streaming/user-service/internal/domain"
)

// Mock implementations
type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) Create(ctx context.Context, user *domain.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *MockUserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *MockUserRepository) Update(ctx context.Context, user *domain.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepository) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockUserRepository) UpdatePassword(ctx context.Context, id, hashedPassword string) error {
	args := m.Called(ctx, id, hashedPassword)
	return args.Error(0)
}

func (m *MockUserRepository) VerifyEmail(ctx context.Context, userID string) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

type MockSessionRepository struct {
	mock.Mock
}

func (m *MockSessionRepository) Create(ctx context.Context, session *domain.Session) error {
	args := m.Called(ctx, session)
	return args.Error(0)
}

func (m *MockSessionRepository) GetByToken(ctx context.Context, token string) (*domain.Session, error) {
	args := m.Called(ctx, token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Session), args.Error(1)
}

func (m *MockSessionRepository) GetByRefreshToken(ctx context.Context, refreshToken string) (*domain.Session, error) {
	args := m.Called(ctx, refreshToken)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Session), args.Error(1)
}

func (m *MockSessionRepository) Delete(ctx context.Context, token string) error {
	args := m.Called(ctx, token)
	return args.Error(0)
}

func (m *MockSessionRepository) DeleteByUserID(ctx context.Context, userID string) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

type MockUserCache struct {
	mock.Mock
}

func (m *MockUserCache) SetUser(ctx context.Context, user *domain.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserCache) GetUser(ctx context.Context, userID string) (*domain.User, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *MockUserCache) InvalidateUser(ctx context.Context, userID string) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *MockUserCache) SetVerificationToken(ctx context.Context, userID, token string) error {
	args := m.Called(ctx, userID, token)
	return args.Error(0)
}

func (m *MockUserCache) GetVerificationToken(ctx context.Context, userID string) (string, error) {
	args := m.Called(ctx, userID)
	return args.String(0), args.Error(1)
}

func (m *MockUserCache) DeleteVerificationToken(ctx context.Context, userID string) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *MockUserCache) SetResetToken(ctx context.Context, userID, token string) error {
	args := m.Called(ctx, userID, token)
	return args.Error(0)
}

func (m *MockUserCache) GetResetTokenUser(ctx context.Context, token string) (string, error) {
	args := m.Called(ctx, token)
	return args.String(0), args.Error(1)
}

func (m *MockUserCache) DeleteResetToken(ctx context.Context, token string) error {
	args := m.Called(ctx, token)
	return args.Error(0)
}

func (m *MockUserCache) SetSession(ctx context.Context, token, userID string) error {
	args := m.Called(ctx, token, userID)
	return args.Error(0)
}

func (m *MockUserCache) GetSession(ctx context.Context, token string) (string, error) {
	args := m.Called(ctx, token)
	return args.String(0), args.Error(1)
}

func (m *MockUserCache) DeleteSession(ctx context.Context, token string) error {
	args := m.Called(ctx, token)
	return args.Error(0)
}

type MockEmailSender struct {
	mock.Mock
}

func (m *MockEmailSender) SendVerificationEmail(to, userID, token string) {
	m.Called(to, userID, token)
}

func (m *MockEmailSender) SendPasswordResetEmail(to, token string) {
	m.Called(to, token)
}

type MockEventPublisher struct {
	mock.Mock
}

func (m *MockEventPublisher) PublishUserRegistered(ctx context.Context, event *domain.UserRegisteredEvent) {
	m.Called(ctx, event)
}

func (m *MockEventPublisher) PublishUserDeleted(ctx context.Context, event *domain.UserDeletedEvent) {
	m.Called(ctx, event)
}

// Tests
func TestUserService_Register_Success(t *testing.T) {
	mockUserRepo := new(MockUserRepository)
	mockSessionRepo := new(MockSessionRepository)
	mockCache := new(MockUserCache)
	mockEmail := new(MockEmailSender)
	mockEvents := new(MockEventPublisher)

	mockUserRepo.On("GetByEmail", mock.Anything, "test@example.com").Return(nil, domain.ErrUserNotFound)
	mockUserRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.User")).Return(nil)
	mockCache.On("SetVerificationToken", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mockEmail.On("SendVerificationEmail", "test@example.com", mock.Anything, mock.Anything).Return()
	mockEvents.On("PublishUserRegistered", mock.Anything, mock.Anything).Return()

	service := NewUserService(mockUserRepo, mockSessionRepo, mockCache, mockEmail, mockEvents, "test-secret")

	userID, err := service.Register(context.Background(), "test@example.com", "password123", "testuser")

	assert.NoError(t, err)
	assert.NotEmpty(t, userID)
	mockUserRepo.AssertExpectations(t)
	mockCache.AssertExpectations(t)
}

func TestUserService_Register_UserExists(t *testing.T) {
	mockUserRepo := new(MockUserRepository)
	mockSessionRepo := new(MockSessionRepository)
	mockCache := new(MockUserCache)
	mockEmail := new(MockEmailSender)
	mockEvents := new(MockEventPublisher)

	existingUser := &domain.User{ID: "123", Email: "test@example.com"}
	mockUserRepo.On("GetByEmail", mock.Anything, "test@example.com").Return(existingUser, nil)

	service := NewUserService(mockUserRepo, mockSessionRepo, mockCache, mockEmail, mockEvents, "test-secret")

	_, err := service.Register(context.Background(), "test@example.com", "password123", "testuser")

	assert.Equal(t, domain.ErrUserExists, err)
}

func TestUserService_GetUser_CacheHit(t *testing.T) {
	mockUserRepo := new(MockUserRepository)
	mockSessionRepo := new(MockSessionRepository)
	mockCache := new(MockUserCache)
	mockEmail := new(MockEmailSender)
	mockEvents := new(MockEventPublisher)

	expectedUser := &domain.User{ID: "123", Email: "test@example.com", Username: "testuser"}
	mockCache.On("GetUser", mock.Anything, "123").Return(expectedUser, nil)

	service := NewUserService(mockUserRepo, mockSessionRepo, mockCache, mockEmail, mockEvents, "test-secret")

	user, err := service.GetUser(context.Background(), "123")

	assert.NoError(t, err)
	assert.Equal(t, expectedUser, user)
	mockUserRepo.AssertNotCalled(t, "GetByID")
}

func TestUserService_GetUser_DatabaseFallback(t *testing.T) {
	mockUserRepo := new(MockUserRepository)
	mockSessionRepo := new(MockSessionRepository)
	mockCache := new(MockUserCache)
	mockEmail := new(MockEmailSender)
	mockEvents := new(MockEventPublisher)

	expectedUser := &domain.User{ID: "123", Email: "test@example.com", Username: "testuser"}
	mockCache.On("GetUser", mock.Anything, "123").Return(nil, assert.AnError)
	mockUserRepo.On("GetByID", mock.Anything, "123").Return(expectedUser, nil)
	mockCache.On("SetUser", mock.Anything, expectedUser).Return(nil)

	service := NewUserService(mockUserRepo, mockSessionRepo, mockCache, mockEmail, mockEvents, "test-secret")

	user, err := service.GetUser(context.Background(), "123")

	assert.NoError(t, err)
	assert.Equal(t, expectedUser, user)
}