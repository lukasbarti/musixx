package library

import "time"

// Track represents a playable audio item in the musixx library.
type Track struct {
	ID              int64
	Title           string
	Album           *string
	Artist          *string
	DurationSeconds *int
	FilePath        string
	SourceURL       *string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// CreateTrackParams contains the fields required to create a new track entry.
type CreateTrackParams struct {
	Title           string
	Album           *string
	Artist          *string
	DurationSeconds *int
	FilePath        string
	SourceURL       *string
}

// UpdateTrackParams contains the fields that can be updated on an existing track.
type UpdateTrackParams struct {
	ID              int64
	Title           *string
	Album           *string
	Artist          *string
	DurationSeconds *int
	SourceURL       *string
}

// Playlist represents a curated list of tracks stored in the library.
type Playlist struct {
	ID          int64
	Name        string
	Description *string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// CreatePlaylistParams contains the fields required to persist a playlist.
type CreatePlaylistParams struct {
	Name        string
	Description *string
}

// PlaylistTrack represents an association between a playlist and a track.
type PlaylistTrack struct {
	ID         int64
	PlaylistID int64
	TrackID    int64
	Position   int
	CreatedAt  time.Time
}

// PlaylistTrackEntry represents an ordered track within a playlist including track metadata.
type PlaylistTrackEntry struct {
	Track    Track     `json:"track"`
	Position int       `json:"position"`
	AddedAt  time.Time `json:"added_at"`
}
