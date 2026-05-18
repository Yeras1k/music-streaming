package domain

import "time"

type TrackUploadedEvent struct {
	Event     string `json:"event"`
	TrackID   string `json:"track_id"`
	UserID    string `json:"user_id"`
	Title     string `json:"title"`
	Artist    string `json:"artist"`
	Timestamp int64  `json:"timestamp"`
}

func NewTrackUploadedEvent(trackID, userID, title, artist string) *TrackUploadedEvent {
	return &TrackUploadedEvent{
		Event:     "track_uploaded",
		TrackID:   trackID,
		UserID:    userID,
		Title:     title,
		Artist:    artist,
		Timestamp: time.Now().Unix(),
	}
}