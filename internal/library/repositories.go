package library

import "context"

// TrackRepository defines persistence operations for tracks.
type TrackRepository interface {
	Create(ctx context.Context, params CreateTrackParams) (Track, error)
	GetByID(ctx context.Context, id int64) (Track, error)
	List(ctx context.Context) ([]Track, error)
	Update(ctx context.Context, params UpdateTrackParams) (Track, error)
	Delete(ctx context.Context, id int64) error
}

// PlaylistRepository defines persistence operations for playlists.
type PlaylistRepository interface {
	Create(ctx context.Context, params CreatePlaylistParams) (Playlist, error)
	GetByID(ctx context.Context, id int64) (Playlist, error)
	List(ctx context.Context) ([]Playlist, error)
	AddTrack(ctx context.Context, playlistID, trackID int64) error
	Delete(ctx context.Context, id int64) error
}
