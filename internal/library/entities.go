package library

import "time"

// Track represents a playable audio item in the Musixx library.
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
