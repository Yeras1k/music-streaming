package domain

import "time"

type UserRegisteredEvent struct {
	Event     string `json:"event"`
	UserID    string `json:"user_id"`
	Email     string `json:"email"`
	Username  string `json:"username"`
	Timestamp int64  `json:"timestamp"`
}

func NewUserRegisteredEvent(userID, email, username string) *UserRegisteredEvent {
	return &UserRegisteredEvent{
		Event:     "user_registered",
		UserID:    userID,
		Email:     email,
		Username:  username,
		Timestamp: time.Now().Unix(),
	}
}

type UserDeletedEvent struct {
	Event     string `json:"event"`
	UserID    string `json:"user_id"`
	Timestamp int64  `json:"timestamp"`
}

func NewUserDeletedEvent(userID string) *UserDeletedEvent {
	return &UserDeletedEvent{
		Event:     "user_deleted",
		UserID:    userID,
		Timestamp: time.Now().Unix(),
	}
}