package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"

	"github.com/music-streaming/user-service/internal/domain"
)

type UserCache struct {
	client *redis.Client
}

func NewUserCache(client *redis.Client) *UserCache {
	return &UserCache{client: client}
}

func (c *UserCache) SetUser(ctx context.Context, user *domain.User) error {
	data, err := json.Marshal(user)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, fmt.Sprintf("user:%s", user.ID), data, 30*time.Minute).Err()
}

func (c *UserCache) GetUser(ctx context.Context, userID string) (*domain.User, error) {
	data, err := c.client.Get(ctx, fmt.Sprintf("user:%s", userID)).Bytes()
	if err != nil {
		return nil, err
	}
	var user domain.User
	if err := json.Unmarshal(data, &user); err != nil {
		return nil, err
	}
	return &user, nil
}

func (c *UserCache) InvalidateUser(ctx context.Context, userID string) error {
	return c.client.Del(ctx, fmt.Sprintf("user:%s", userID)).Err()
}

func (c *UserCache) SetVerificationToken(ctx context.Context, userID, token string) error {
	return c.client.Set(ctx, fmt.Sprintf("verify:%s", userID), token, 24*time.Hour).Err()
}

func (c *UserCache) GetVerificationToken(ctx context.Context, userID string) (string, error) {
	return c.client.Get(ctx, fmt.Sprintf("verify:%s", userID)).Result()
}

func (c *UserCache) DeleteVerificationToken(ctx context.Context, userID string) error {
	return c.client.Del(ctx, fmt.Sprintf("verify:%s", userID)).Err()
}

func (c *UserCache) SetResetToken(ctx context.Context, userID, token string) error {
	return c.client.Set(ctx, fmt.Sprintf("reset:%s", userID), token, 1*time.Hour).Err()
}

func (c *UserCache) GetResetTokenUser(ctx context.Context, token string) (string, error) {
	keys, err := c.client.Keys(ctx, "reset:*").Result()
	if err != nil {
		return "", err
	}
	for _, key := range keys {
		val, err := c.client.Get(ctx, key).Result()
		if err == nil && val == token {
			return key[6:], nil
		}
	}
	return "", redis.Nil
}

func (c *UserCache) DeleteResetToken(ctx context.Context, token string) error {
	userID, err := c.GetResetTokenUser(ctx, token)
	if err != nil {
		return err
	}
	return c.client.Del(ctx, fmt.Sprintf("reset:%s", userID)).Err()
}

func (c *UserCache) SetSession(ctx context.Context, token, userID string) error {
	data := fmt.Sprintf(`{"user_id":"%s"}`, userID)
	return c.client.Set(ctx, fmt.Sprintf("session:%s", token), data, 24*time.Hour).Err()
}

func (c *UserCache) GetSession(ctx context.Context, token string) (string, error) {
	data, err := c.client.Get(ctx, fmt.Sprintf("session:%s", token)).Result()
	if err != nil {
		return "", err
	}
	var result struct {
		UserID string `json:"user_id"`
	}
	if err := json.Unmarshal([]byte(data), &result); err != nil {
		return "", err
	}
	return result.UserID, nil
}

func (c *UserCache) DeleteSession(ctx context.Context, token string) error {
	return c.client.Del(ctx, fmt.Sprintf("session:%s", token)).Err()
}