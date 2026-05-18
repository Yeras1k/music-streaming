package integration

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/music-streaming/user-service/internal/repository"
	"github.com/music-streaming/user-service/internal/service"
	"github.com/music-streaming/user-service/pkg/cache"
)

func TestUserIntegration_CreateAndGetUser(t *testing.T) {
	// Skip if no database available
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup test database
	dsn := "host=localhost user=postgres password=postgres dbname=music_test port=5432 sslmode=disable"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Skip("Database not available, skipping integration test")
	}

	// Migrate
	db.AutoMigrate(&repository.UserModel{}, &repository.SessionModel{})

	// Cleanup after test
	defer func() {
		db.Exec("TRUNCATE users, sessions CASCADE")
	}()

	// Setup mocks for external dependencies
	// (In a real integration test, you'd use test containers for Redis and NATS)
	// For now, we'll test the repository layer directly

	userRepo := repository.NewUserRepository(db)

	// Test user creation
	user := &domain.User{
		ID:       "test-id-1",
		Email:    "test@example.com",
		Password: "hashedpassword",
		Username: "testuser",
		Role:     "user",
		Verified: false,
	}

	err = userRepo.Create(context.Background(), user)
	require.NoError(t, err)

	// Test user retrieval
	retrieved, err := userRepo.GetByEmail(context.Background(), "test@example.com")
	require.NoError(t, err)
	assert.Equal(t, user.Email, retrieved.Email)
	assert.Equal(t, user.Username, retrieved.Username)
}