package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/lukasbarti/musixx/internal/library"
)

type trackRepository struct {
	db *sql.DB
}

// TrackRepository provides access to track persistence backed by SQLite.
func (s *Store) TrackRepository() library.TrackRepository {
	return &trackRepository{db: s.db}
}

func (r *trackRepository) Create(ctx context.Context, params library.CreateTrackParams) (library.Track, error) {
	const query = `INSERT INTO tracks (title, album, artist, duration_seconds, file_path, source_url)
	VALUES (?, ?, ?, ?, ?, ?)`

	result, err := r.db.ExecContext(ctx, query,
		params.Title,
		valueOrNilPtr(params.Album),
		valueOrNilPtr(params.Artist),
		intOrNil(params.DurationSeconds),
		params.FilePath,
		valueOrNilPtr(params.SourceURL),
	)
	if err != nil {
		return library.Track{}, fmt.Errorf("insert track: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return library.Track{}, fmt.Errorf("fetch inserted track id: %w", err)
	}

	track, err := r.GetByID(ctx, id)
	if err != nil {
		return library.Track{}, fmt.Errorf("load inserted track: %w", err)
	}

	return track, nil
}

func (r *trackRepository) GetByID(ctx context.Context, id int64) (library.Track, error) {
	const query = `SELECT id, title, album, artist, duration_seconds, file_path, source_url, created_at, updated_at
	FROM tracks WHERE id = ?`

	row := r.db.QueryRowContext(ctx, query, id)
	track, err := scanTrack(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return library.Track{}, library.ErrNotFound
		}
		return library.Track{}, fmt.Errorf("query track %d: %w", id, err)
	}

	return track, nil
}

func (r *trackRepository) List(ctx context.Context) ([]library.Track, error) {
	const query = `SELECT id, title, album, artist, duration_seconds, file_path, source_url, created_at, updated_at
	FROM tracks ORDER BY created_at DESC`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list tracks: %w", err)
	}
	defer rows.Close()

	var tracks []library.Track
	for rows.Next() {
		track, err := scanTrack(rows)
		if err != nil {
			return nil, fmt.Errorf("scan track row: %w", err)
		}
		tracks = append(tracks, track)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate tracks: %w", err)
	}

	return tracks, nil
}

func (r *trackRepository) Update(ctx context.Context, params library.UpdateTrackParams) (library.Track, error) {
	setClauses := make([]string, 0, 5)
	args := make([]any, 0, 6)

	if params.Title != nil {
		setClauses = append(setClauses, "title = ?")
		args = append(args, *params.Title)
	}
	if params.Album != nil {
		setClauses = append(setClauses, "album = ?")
		args = append(args, *params.Album)
	}
	if params.Artist != nil {
		setClauses = append(setClauses, "artist = ?")
		args = append(args, *params.Artist)
	}
	if params.DurationSeconds != nil {
		setClauses = append(setClauses, "duration_seconds = ?")
		args = append(args, *params.DurationSeconds)
	}
	if params.SourceURL != nil {
		setClauses = append(setClauses, "source_url = ?")
		args = append(args, *params.SourceURL)
	}

	if len(setClauses) == 0 {
		return r.GetByID(ctx, params.ID)
	}

	setClauses = append(setClauses, "updated_at = CURRENT_TIMESTAMP")

	args = append(args, params.ID)
	query := fmt.Sprintf("UPDATE tracks SET %s WHERE id = ?", strings.Join(setClauses, ", "))

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return library.Track{}, fmt.Errorf("update track %d: %w", params.ID, err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return library.Track{}, fmt.Errorf("fetch updated rows for track %d: %w", params.ID, err)
	}
	if affected == 0 {
		return library.Track{}, library.ErrNotFound
	}

	track, err := r.GetByID(ctx, params.ID)
	if err != nil {
		return library.Track{}, fmt.Errorf("reload track %d: %w", params.ID, err)
	}

	return track, nil
}

func (r *trackRepository) Delete(ctx context.Context, id int64) error {
	const query = `DELETE FROM tracks WHERE id = ?`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete track %d: %w", id, err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected deleting track %d: %w", id, err)
	}
	if affected == 0 {
		return library.ErrNotFound
	}

	return nil
}

type sqlScanner interface {
	Scan(dest ...any) error
}

func scanTrack(scanner sqlScanner) (library.Track, error) {
	var (
		track     library.Track
		album     sql.NullString
		artist    sql.NullString
		duration  sql.NullInt64
		sourceURL sql.NullString
		createdAt time.Time
		updatedAt time.Time
	)

	if err := scanner.Scan(
		&track.ID,
		&track.Title,
		&album,
		&artist,
		&duration,
		&track.FilePath,
		&sourceURL,
		&createdAt,
		&updatedAt,
	); err != nil {
		return library.Track{}, err
	}

	if album.Valid {
		track.Album = &album.String
	}
	if artist.Valid {
		track.Artist = &artist.String
	}
	if duration.Valid {
		val := int(duration.Int64)
		track.DurationSeconds = &val
	}
	if sourceURL.Valid {
		track.SourceURL = &sourceURL.String
	}
	track.CreatedAt = createdAt
	track.UpdatedAt = updatedAt

	return track, nil
}

func valueOrNilPtr(ptr *string) any {
	if ptr == nil {
		return nil
	}
	return *ptr
}

func intOrNil(ptr *int) any {
	if ptr == nil {
		return nil
	}
	return *ptr
}
