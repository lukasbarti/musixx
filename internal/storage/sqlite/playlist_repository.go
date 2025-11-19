package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/lukasbarti/musixx/internal/library"
)

type playlistRepository struct {
	db *sql.DB
}

// PlaylistRepository provides access to playlist persistence backed by SQLite.
func (s *Store) PlaylistRepository() library.PlaylistRepository {
	return &playlistRepository{db: s.db}
}

func (r *playlistRepository) Create(ctx context.Context, params library.CreatePlaylistParams) (library.Playlist, error) {
	const query = `INSERT INTO playlists (name, description) VALUES (?, ?)`

	result, err := r.db.ExecContext(ctx, query, params.Name, valueOrNilPtr(params.Description))
	if err != nil {
		return library.Playlist{}, fmt.Errorf("insert playlist: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return library.Playlist{}, fmt.Errorf("fetch inserted playlist id: %w", err)
	}

	playlist, err := r.GetByID(ctx, id)
	if err != nil {
		return library.Playlist{}, fmt.Errorf("load inserted playlist: %w", err)
	}

	return playlist, nil
}

func (r *playlistRepository) GetByID(ctx context.Context, id int64) (library.Playlist, error) {
	const query = `SELECT id, name, description, created_at, updated_at FROM playlists WHERE id = ?`

	row := r.db.QueryRowContext(ctx, query, id)
	playlist, err := scanPlaylist(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return library.Playlist{}, library.ErrNotFound
		}
		return library.Playlist{}, fmt.Errorf("query playlist %d: %w", id, err)
	}

	return playlist, nil
}

func (r *playlistRepository) List(ctx context.Context) ([]library.Playlist, error) {
	const query = `SELECT id, name, description, created_at, updated_at FROM playlists ORDER BY created_at DESC`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list playlists: %w", err)
	}
	defer rows.Close()

	var playlists []library.Playlist
	for rows.Next() {
		playlist, err := scanPlaylist(rows)
		if err != nil {
			return nil, fmt.Errorf("scan playlist row: %w", err)
		}
		playlists = append(playlists, playlist)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate playlists: %w", err)
	}

	return playlists, nil
}

func (r *playlistRepository) ListTracks(ctx context.Context, playlistID int64) ([]library.PlaylistTrackEntry, error) {
	const query = `SELECT
        pt.position,
        pt.created_at,
        t.id,
        t.title,
        t.album,
        t.artist,
        t.duration_seconds,
        t.file_path,
        t.source_url,
        t.created_at,
        t.updated_at
    FROM playlist_tracks pt
    INNER JOIN tracks t ON t.id = pt.track_id
    WHERE pt.playlist_id = ?
    ORDER BY pt.position ASC, pt.id ASC`

	rows, err := r.db.QueryContext(ctx, query, playlistID)
	if err != nil {
		return nil, fmt.Errorf("list tracks for playlist %d: %w", playlistID, err)
	}
	defer rows.Close()

	var entries []library.PlaylistTrackEntry
	for rows.Next() {
		entry, err := scanPlaylistTrackEntry(rows)
		if err != nil {
			return nil, fmt.Errorf("scan playlist track row: %w", err)
		}
		entries = append(entries, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate playlist tracks: %w", err)
	}

	return entries, nil
}

func (r *playlistRepository) AddTrack(ctx context.Context, playlistID, trackID int64) error {
	const query = `
    INSERT INTO playlist_tracks (playlist_id, track_id, position)
    VALUES (?, ?, COALESCE((SELECT MAX(position) + 1 FROM playlist_tracks WHERE playlist_id = ?), 0))
    `

	if _, err := r.db.ExecContext(ctx, query, playlistID, trackID, playlistID); err != nil {
		return fmt.Errorf("insert playlist track: %w", err)
	}

	return nil
}

func (r *playlistRepository) Delete(ctx context.Context, id int64) error {
	const query = `DELETE FROM playlists WHERE id = ?`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete playlist %d: %w", id, err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected deleting playlist %d: %w", id, err)
	}
	if affected == 0 {
		return library.ErrNotFound
	}

	return nil
}

func scanPlaylist(scanner sqlScanner) (library.Playlist, error) {
	var (
		playlist  library.Playlist
		desc      sql.NullString
		createdAt time.Time
		updatedAt time.Time
	)

	if err := scanner.Scan(
		&playlist.ID,
		&playlist.Name,
		&desc,
		&createdAt,
		&updatedAt,
	); err != nil {
		return library.Playlist{}, err
	}

	if desc.Valid {
		playlist.Description = &desc.String
	}
	playlist.CreatedAt = createdAt
	playlist.UpdatedAt = updatedAt

	return playlist, nil
}

func scanPlaylistTrackEntry(scanner sqlScanner) (library.PlaylistTrackEntry, error) {
	var (
		entry          library.PlaylistTrackEntry
		album          sql.NullString
		artist         sql.NullString
		duration       sql.NullInt64
		sourceURL      sql.NullString
		trackCreatedAt time.Time
		trackUpdatedAt time.Time
	)

	if err := scanner.Scan(
		&entry.Position,
		&entry.AddedAt,
		&entry.Track.ID,
		&entry.Track.Title,
		&album,
		&artist,
		&duration,
		&entry.Track.FilePath,
		&sourceURL,
		&trackCreatedAt,
		&trackUpdatedAt,
	); err != nil {
		return library.PlaylistTrackEntry{}, err
	}

	if album.Valid {
		entry.Track.Album = &album.String
	}
	if artist.Valid {
		entry.Track.Artist = &artist.String
	}
	if duration.Valid {
		v := int(duration.Int64)
		entry.Track.DurationSeconds = &v
	}
	if sourceURL.Valid {
		entry.Track.SourceURL = &sourceURL.String
	}
	entry.Track.CreatedAt = trackCreatedAt
	entry.Track.UpdatedAt = trackUpdatedAt

	return entry, nil
}
