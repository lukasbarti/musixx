package sqlite

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/lukasbarti/musixx/internal/library"
)

func TestTrackRepositoryCRUD(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "tracks.db")
	store, err := Open(dbPath)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Fatalf("close store: %v", err)
		}
	})

	repo := store.TrackRepository()
	ctx := context.Background()
	title := "First Track"
	filePath := "library/first.opus"
	duration := 180
	source := "https://example.com"

	created, err := repo.Create(ctx, library.CreateTrackParams{
		Title:           title,
		Album:           nil,
		Artist:          nil,
		DurationSeconds: &duration,
		FilePath:        filePath,
		SourceURL:       &source,
	})
	if err != nil {
		t.Fatalf("create track: %v", err)
	}

	if created.ID == 0 {
		t.Fatalf("expected ID to be set")
	}
	if created.Title != title {
		t.Fatalf("unexpected title: %s", created.Title)
	}
	if created.DurationSeconds == nil || *created.DurationSeconds != duration {
		t.Fatalf("unexpected duration: %+v", created.DurationSeconds)
	}
	if created.SourceURL == nil || *created.SourceURL != source {
		t.Fatalf("unexpected source: %+v", created.SourceURL)
	}

	retrieved, err := repo.GetByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("get track: %v", err)
	}
	if retrieved.Title != title {
		t.Fatalf("retrieved title mismatch: %s", retrieved.Title)
	}

	tracks, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("list tracks: %v", err)
	}
	if len(tracks) != 1 {
		t.Fatalf("expected 1 track, got %d", len(tracks))
	}

	updatedTitle := "Updated Track"
	updatedDuration := 240
	updatedSource := "https://example.com/updated"

	updated, err := repo.Update(ctx, library.UpdateTrackParams{
		ID:              created.ID,
		Title:           &updatedTitle,
		DurationSeconds: &updatedDuration,
		SourceURL:       &updatedSource,
	})
	if err != nil {
		t.Fatalf("update track: %v", err)
	}
	if updated.Title != updatedTitle {
		t.Fatalf("expected title to be updated, got %s", updated.Title)
	}
	if updated.DurationSeconds == nil || *updated.DurationSeconds != updatedDuration {
		t.Fatalf("expected duration to be updated, got %+v", updated.DurationSeconds)
	}
	if updated.SourceURL == nil || *updated.SourceURL != updatedSource {
		t.Fatalf("expected source to be updated, got %+v", updated.SourceURL)
	}
	if updated.UpdatedAt.Before(updated.CreatedAt) {
		t.Fatalf("expected updated_at to be >= created_at")
	}

	if err := repo.Delete(ctx, created.ID); err != nil {
		t.Fatalf("delete track: %v", err)
	}

	_, err = repo.GetByID(ctx, created.ID)
	if err == nil {
		t.Fatalf("expected ErrNotFound after delete")
	}
	if err != library.ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestTrackRepositoryDuplicateFilePath(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "tracks.db")
	store, err := Open(dbPath)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Fatalf("close store: %v", err)
		}
	})

	repo := store.TrackRepository()
	ctx := context.Background()

	filePath := "library/duplicate.opus"

	_, err = repo.Create(ctx, library.CreateTrackParams{
		Title:    "Track One",
		FilePath: filePath,
	})
	if err != nil {
		t.Fatalf("first create: %v", err)
	}

	_, err = repo.Create(ctx, library.CreateTrackParams{
		Title:    "Track Two",
		FilePath: filePath,
	})
	if err == nil {
		t.Fatalf("expected error for duplicate file path")
	}
}
