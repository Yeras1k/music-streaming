package handler

import (
	"context"
	"io"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/music-streaming/proto/music"
	"github.com/music-streaming/music-service/internal/domain"
	"github.com/music-streaming/music-service/internal/service"
)

type MusicHandler struct {
	pb.UnimplementedMusicServiceServer
	service *service.MusicService
}

func NewMusicHandler(svc *service.MusicService) *MusicHandler {
	return &MusicHandler{service: svc}
}

func (h *MusicHandler) UploadTrack(ctx context.Context, req *pb.UploadTrackRequest) (*pb.UploadTrackResponse, error) {
	track, err := h.service.UploadTrack(
		ctx,
		req.UserId,
		req.Title,
		req.Artist,
		req.Album,
		req.Genre,
		req.Duration,
		req.AudioData,
	)
	if err != nil {
		if err == domain.ErrRateLimitExceeded {
			return nil, status.Error(codes.ResourceExhausted, "upload limit exceeded")
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &pb.UploadTrackResponse{
		TrackId: track.ID,
		Message: "Track uploaded successfully",
	}, nil
}

func (h *MusicHandler) GetTrack(ctx context.Context, req *pb.GetTrackRequest) (*pb.Track, error) {
	track, err := h.service.GetTrack(ctx, req.TrackId)
	if err != nil {
		if err == domain.ErrTrackNotFound {
			return nil, status.Error(codes.NotFound, "track not found")
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &pb.Track{
		Id:        track.ID,
		UserId:    track.UserID,
		Title:     track.Title,
		Artist:    track.Artist,
		Album:     track.Album,
		Duration:  track.Duration,
		Genre:     track.Genre,
		Url:       track.URL,
		Plays:     int32(track.Plays),
		Likes:     int32(track.Likes),
		CreatedAt: track.CreatedAt.Unix(),
	}, nil
}

func (h *MusicHandler) ListTracks(ctx context.Context, req *pb.ListTracksRequest) (*pb.ListTracksResponse, error) {
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize < 1 || req.PageSize > 100 {
		req.PageSize = 20
	}

	tracks, total, err := h.service.ListTracks(ctx, req.Page, req.PageSize)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	pbTracks := make([]*pb.Track, len(tracks))
	for i, t := range tracks {
		pbTracks[i] = &pb.Track{
			Id:       t.ID,
			Title:    t.Title,
			Artist:   t.Artist,
			Album:    t.Album,
			Duration: t.Duration,
			Url:      t.URL,
			Plays:    int32(t.Plays),
			Likes:    int32(t.Likes),
		}
	}
	return &pb.ListTracksResponse{Tracks: pbTracks, Total: int32(total)}, nil
}

func (h *MusicHandler) SearchTracks(ctx context.Context, req *pb.SearchTracksRequest) (*pb.SearchTracksResponse, error) {
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize < 1 || req.PageSize > 100 {
		req.PageSize = 20
	}

	tracks, total, err := h.service.SearchTracks(ctx, req.Query, req.Page, req.PageSize)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	pbTracks := make([]*pb.Track, len(tracks))
	for i, t := range tracks {
		pbTracks[i] = &pb.Track{
			Id:     t.ID,
			Title:  t.Title,
			Artist: t.Artist,
			Album:  t.Album,
		}
	}
	return &pb.SearchTracksResponse{Tracks: pbTracks, Total: int32(total)}, nil
}

func (h *MusicHandler) CreatePlaylist(ctx context.Context, req *pb.CreatePlaylistRequest) (*pb.Playlist, error) {
	playlist, err := h.service.CreatePlaylist(ctx, req.UserId, req.Name, req.Description, req.IsPublic)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &pb.Playlist{
		Id:          playlist.ID,
		UserId:      playlist.UserID,
		Name:        playlist.Name,
		Description: playlist.Description,
		IsPublic:    playlist.IsPublic,
		CreatedAt:   playlist.CreatedAt.Unix(),
	}, nil
}

func (h *MusicHandler) AddToPlaylist(ctx context.Context, req *pb.AddToPlaylistRequest) (*pb.Playlist, error) {
	playlist, err := h.service.AddToPlaylist(ctx, req.PlaylistId, req.TrackId, req.UserId)
	if err != nil {
		if err == domain.ErrUnauthorized {
			return nil, status.Error(codes.PermissionDenied, "not authorized")
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &pb.Playlist{Id: playlist.ID, Name: playlist.Name}, nil
}

func (h *MusicHandler) RemoveFromPlaylist(ctx context.Context, req *pb.RemoveFromPlaylistRequest) (*pb.Playlist, error) {
	playlist, err := h.service.RemoveFromPlaylist(ctx, req.PlaylistId, req.TrackId, req.UserId)
	if err != nil {
		if err == domain.ErrUnauthorized {
			return nil, status.Error(codes.PermissionDenied, "not authorized")
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &pb.Playlist{Id: playlist.ID}, nil
}

func (h *MusicHandler) GetPlaylist(ctx context.Context, req *pb.GetPlaylistRequest) (*pb.Playlist, error) {
	playlist, tracks, err := h.service.GetPlaylist(ctx, req.PlaylistId, req.UserId)
	if err != nil {
		if err == domain.ErrUnauthorized {
			return nil, status.Error(codes.PermissionDenied, "not authorized")
		}
		if err == domain.ErrPlaylistNotFound {
			return nil, status.Error(codes.NotFound, "playlist not found")
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	pbTracks := make([]*pb.Track, len(tracks))
	for i, t := range tracks {
		pbTracks[i] = &pb.Track{
			Id:     t.ID,
			Title:  t.Title,
			Artist: t.Artist,
		}
	}

	return &pb.Playlist{
		Id:          playlist.ID,
		Name:        playlist.Name,
		Description: playlist.Description,
		Tracks:      pbTracks,
		IsPublic:    playlist.IsPublic,
		TrackCount:  int32(len(tracks)),
	}, nil
}

func (h *MusicHandler) GetUserPlaylists(ctx context.Context, req *pb.GetUserPlaylistsRequest) (*pb.GetUserPlaylistsResponse, error) {
	playlists, err := h.service.GetUserPlaylists(ctx, req.UserId)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	pbPlaylists := make([]*pb.Playlist, len(playlists))
	for i, p := range playlists {
		pbPlaylists[i] = &pb.Playlist{
			Id:         p.ID,
			Name:       p.Name,
			IsPublic:   p.IsPublic,
			TrackCount: 0, // Would need to count tracks
		}
	}
	return &pb.GetUserPlaylistsResponse{Playlists: pbPlaylists}, nil
}

func (h *MusicHandler) StreamTrack(req *pb.StreamTrackRequest, stream pb.MusicService_StreamTrackServer) error {
	// This would stream audio data
	// For now, return not implemented
	return status.Error(codes.Unimplemented, "streaming not implemented yet")
}

func (h *MusicHandler) GetRecommendations(ctx context.Context, req *pb.GetRecommendationsRequest) (*pb.RecommendationsResponse, error) {
	tracks, err := h.service.GetRecommendations(ctx, req.UserId, req.Limit)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	pbTracks := make([]*pb.Track, len(tracks))
	for i, t := range tracks {
		pbTracks[i] = &pb.Track{
			Id:     t.ID,
			Title:  t.Title,
			Artist: t.Artist,
		}
	}
	return &pb.RecommendationsResponse{Tracks: pbTracks}, nil
}

func (h *MusicHandler) LikeTrack(ctx context.Context, req *pb.LikeTrackRequest) (*pb.LikeTrackResponse, error) {
	liked, totalLikes, err := h.service.LikeTrack(ctx, req.UserId, req.TrackId)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &pb.LikeTrackResponse{
		Liked:      liked,
		TotalLikes: int32(totalLikes),
	}, nil
}

func (h *MusicHandler) GetLikedTracks(ctx context.Context, req *pb.GetLikedTracksRequest) (*pb.GetLikedTracksResponse, error) {
	// Implementation would fetch liked tracks
	return &pb.GetLikedTracksResponse{}, nil
}

func (h *MusicHandler) AddToQueue(ctx context.Context, req *pb.AddToQueueRequest) (*pb.AddToQueueResponse, error) {
	position, err := h.service.AddToQueue(ctx, req.UserId, req.TrackId)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &pb.AddToQueueResponse{Position: position}, nil
}

func (h *MusicHandler) GetQueue(ctx context.Context, req *pb.GetQueueRequest) (*pb.GetQueueResponse, error) {
	tracks, err := h.service.GetQueue(ctx, req.UserId)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	pbTracks := make([]*pb.Track, len(tracks))
	for i, t := range tracks {
		pbTracks[i] = &pb.Track{
			Id:     t.ID,
			Title:  t.Title,
			Artist: t.Artist,
		}
	}
	return &pb.GetQueueResponse{Tracks: pbTracks}, nil
}