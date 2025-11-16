package server

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/lukasbarti/musixx/internal/downloader"
	"github.com/lukasbarti/musixx/internal/library"
)

type stubTrackRepository struct {
	createFunc func(ctx context.Context, params library.CreateTrackParams) (library.Track, error)
	getFunc    func(ctx context.Context, id int64) (library.Track, error)
	listFunc   func(ctx context.Context) ([]library.Track, error)
	updateFunc func(ctx context.Context, params library.UpdateTrackParams) (library.Track, error)
	deleteFunc func(ctx context.Context, id int64) error
}

type stubDownloader struct {
	downloadFunc func(ctx context.Context, req downloader.Request) (downloader.Result, error)
}

func (s stubDownloader) Download(ctx context.Context, req downloader.Request) (downloader.Result, error) {
	if s.downloadFunc == nil {
		panic("Download not implemented in stub")
	}
	return s.downloadFunc(ctx, req)
}

func newServerWithRepo(repo stubTrackRepository) *Server {
	return &Server{
		trackRepo: repo,
		downloader: stubDownloader{
			downloadFunc: func(ctx context.Context, req downloader.Request) (downloader.Result, error) {
				panic("unexpected downloader invocation")
			},
		},
	}
}

func (s stubTrackRepository) Create(ctx context.Context, params library.CreateTrackParams) (library.Track, error) {
	if s.createFunc == nil {
		panic("Create not implemented in stub")
	}
	return s.createFunc(ctx, params)
}

func (s stubTrackRepository) GetByID(ctx context.Context, id int64) (library.Track, error) {
	if s.getFunc == nil {
		panic("GetByID not implemented in stub")
	}
	return s.getFunc(ctx, id)
}

func (s stubTrackRepository) List(ctx context.Context) ([]library.Track, error) {
	if s.listFunc == nil {
		panic("List not implemented in stub")
	}
	return s.listFunc(ctx)
}

func (s stubTrackRepository) Update(ctx context.Context, params library.UpdateTrackParams) (library.Track, error) {
	if s.updateFunc == nil {
		panic("Update not implemented in stub")
	}
	return s.updateFunc(ctx, params)
}

func (s stubTrackRepository) Delete(ctx context.Context, id int64) error {
	if s.deleteFunc == nil {
		panic("Delete not implemented in stub")
	}
	return s.deleteFunc(ctx, id)
}

func TestHandleTracks_ListSuccess(t *testing.T) {
	now := time.Now().UTC()
	repo := stubTrackRepository{
		listFunc: func(ctx context.Context) ([]library.Track, error) {
			return []library.Track{{
				ID:        1,
				Title:     "Test Song",
				FilePath:  "tracks/test.opus",
				CreatedAt: now,
				UpdatedAt: now,
			}}, nil
		},
	}

	s := newServerWithRepo(repo)
	req := httptest.NewRequest(http.MethodGet, "/tracks", nil)
	rec := httptest.NewRecorder()

	s.handleTracks(rec, req)

	res := rec.Result()
	t.Cleanup(func() { _ = res.Body.Close() })

	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, res.StatusCode)
	}

	var tracks []library.Track
	if err := json.NewDecoder(res.Body).Decode(&tracks); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if len(tracks) != 1 || tracks[0].Title != "Test Song" {
		t.Fatalf("unexpected tracks response: %+v", tracks)
	}
}

func TestHandleTracks_CreateSuccess(t *testing.T) {
	now := time.Now().UTC()
	var captured library.CreateTrackParams

	repo := stubTrackRepository{
		createFunc: func(ctx context.Context, params library.CreateTrackParams) (library.Track, error) {
			captured = params
			return library.Track{
				ID:        10,
				Title:     params.Title,
				FilePath:  params.FilePath,
				CreatedAt: now,
				UpdatedAt: now,
			}, nil
		},
	}

	s := newServerWithRepo(repo)

	payload := createTrackRequest{
		Title:    "Another Song",
		FilePath: "tracks/another.opus",
	}

	body := mustJSON(payload)
	req := httptest.NewRequest(http.MethodPost, "/tracks", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	s.handleTracks(rec, req)

	res := rec.Result()
	t.Cleanup(func() { _ = res.Body.Close() })

	if res.StatusCode != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, res.StatusCode)
	}

	if location := res.Header.Get("Location"); location != "/tracks/10" {
		t.Fatalf("expected Location header /tracks/10, got %q", location)
	}

	if captured.Title != payload.Title || captured.FilePath != payload.FilePath {
		t.Fatalf("unexpected captured params: %+v", captured)
	}

	var track library.Track
	if err := json.NewDecoder(res.Body).Decode(&track); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if track.ID != 10 || track.Title != payload.Title {
		t.Fatalf("unexpected track response: %+v", track)
	}
}

func TestHandleTracks_CreateValidationError(t *testing.T) {
	repo := stubTrackRepository{}
	s := newServerWithRepo(repo)

	payload := createTrackRequest{FilePath: "tracks/bad.opus"}
	req := httptest.NewRequest(http.MethodPost, "/tracks", bytes.NewReader(mustJSON(payload)))
	rec := httptest.NewRecorder()

	s.handleTracks(rec, req)

	res := rec.Result()
	t.Cleanup(func() { _ = res.Body.Close() })

	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, res.StatusCode)
	}
}

func TestHandleTrackByID_GetNotFound(t *testing.T) {
	repo := stubTrackRepository{
		getFunc: func(ctx context.Context, id int64) (library.Track, error) {
			return library.Track{}, library.ErrNotFound
		},
	}
	s := newServerWithRepo(repo)

	req := httptest.NewRequest(http.MethodGet, "/tracks/42", nil)
	rec := httptest.NewRecorder()

	s.handleTrackByID(rec, req)

	res := rec.Result()
	t.Cleanup(func() { _ = res.Body.Close() })

	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, res.StatusCode)
	}
}

func TestHandleTrackByID_UpdateInvalidDuration(t *testing.T) {
	repo := stubTrackRepository{
		updateFunc: func(ctx context.Context, params library.UpdateTrackParams) (library.Track, error) {
			panic("update should not be called on invalid payload")
		},
	}
	s := newServerWithRepo(repo)

	dur := -5
	payload := updateTrackRequest{DurationSeconds: &dur}
	req := httptest.NewRequest(http.MethodPatch, "/tracks/1", bytes.NewReader(mustJSON(payload)))
	rec := httptest.NewRecorder()

	s.handleTrackByID(rec, req)

	res := rec.Result()
	t.Cleanup(func() { _ = res.Body.Close() })

	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, res.StatusCode)
	}
}

func TestHandleTrackByID_DeleteSuccess(t *testing.T) {
	var deletedID int64
	repo := stubTrackRepository{
		deleteFunc: func(ctx context.Context, id int64) error {
			deletedID = id
			return nil
		},
	}
	s := newServerWithRepo(repo)

	req := httptest.NewRequest(http.MethodDelete, "/tracks/55", nil)
	rec := httptest.NewRecorder()

	s.handleTrackByID(rec, req)

	res := rec.Result()
	t.Cleanup(func() { _ = res.Body.Close() })

	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, res.StatusCode)
	}

	if deletedID != 55 {
		t.Fatalf("expected delete to be called with id 55, got %d", deletedID)
	}
}

func TestHandleTrackEdit_GetSuccess(t *testing.T) {
	repo := stubTrackRepository{
		getFunc: func(ctx context.Context, id int64) (library.Track, error) {
			return library.Track{
				ID:        id,
				Title:     "Sample",
				FilePath:  "tracks/sample.opus",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}, nil
		},
	}

	s := newServerWithRepo(repo)
	req := httptest.NewRequest(http.MethodGet, "/tracks/5/edit", nil)
	rec := httptest.NewRecorder()

	s.handleTrackByID(rec, req)

	res := rec.Result()
	t.Cleanup(func() { _ = res.Body.Close() })

	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, res.StatusCode)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}

	if !strings.Contains(string(body), "Edit Track") {
		t.Fatalf("expected edit form in response body")
	}
}

func TestHandleTrackEdit_PostSuccess(t *testing.T) {
	var captured library.UpdateTrackParams
	repo := stubTrackRepository{
		getFunc: func(ctx context.Context, id int64) (library.Track, error) {
			return library.Track{
				ID:        id,
				Title:     "Original",
				FilePath:  "tracks/original.opus",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}, nil
		},
		updateFunc: func(ctx context.Context, params library.UpdateTrackParams) (library.Track, error) {
			captured = params
			return library.Track{ID: params.ID}, nil
		},
	}

	s := newServerWithRepo(repo)
	form := url.Values{}
	form.Set("title", "Updated")
	form.Set("artist", "Artist")
	form.Set("album", "Album")
	form.Set("duration_seconds", "180")
	form.Set("source_url", "https://example.com")

	req := httptest.NewRequest(http.MethodPost, "/tracks/7/edit", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	s.handleTrackByID(rec, req)

	res := rec.Result()
	t.Cleanup(func() { _ = res.Body.Close() })

	if res.StatusCode != http.StatusSeeOther {
		t.Fatalf("expected status %d, got %d", http.StatusSeeOther, res.StatusCode)
	}

	if location := res.Header.Get("Location"); location != "/" {
		t.Fatalf("expected redirect to /, got %q", location)
	}

	if captured.ID != 7 {
		t.Fatalf("expected update ID 7, got %d", captured.ID)
	}
	if captured.Title == nil || *captured.Title != "Updated" {
		t.Fatalf("expected title Updated, got %+v", captured.Title)
	}
	if captured.DurationSeconds == nil || *captured.DurationSeconds != 180 {
		t.Fatalf("expected duration 180, got %+v", captured.DurationSeconds)
	}
}

func TestHandleTrackDeleteHTML_Post(t *testing.T) {
	var deletedID int64
	repo := stubTrackRepository{
		deleteFunc: func(ctx context.Context, id int64) error {
			deletedID = id
			return nil
		},
	}

	s := newServerWithRepo(repo)
	req := httptest.NewRequest(http.MethodPost, "/tracks/9/delete", nil)
	rec := httptest.NewRecorder()

	s.handleTrackByID(rec, req)

	res := rec.Result()
	t.Cleanup(func() { _ = res.Body.Close() })

	if res.StatusCode != http.StatusSeeOther {
		t.Fatalf("expected status %d, got %d", http.StatusSeeOther, res.StatusCode)
	}
	if location := res.Header.Get("Location"); location != "/" {
		t.Fatalf("expected redirect to /, got %q", location)
	}
	if deletedID != 9 {
		t.Fatalf("expected deleted ID 9, got %d", deletedID)
	}
}

func TestHandleNewTrack_PostDownloadSuccess(t *testing.T) {
	var captured library.CreateTrackParams
	repo := stubTrackRepository{
		createFunc: func(ctx context.Context, params library.CreateTrackParams) (library.Track, error) {
			captured = params
			return library.Track{ID: 1, Title: params.Title, FilePath: params.FilePath}, nil
		},
	}
	dl := stubDownloader{
		downloadFunc: func(ctx context.Context, req downloader.Request) (downloader.Result, error) {
			return downloader.Result{FilePath: "music/downloaded.opus"}, nil
		},
	}

	s := &Server{trackRepo: repo, downloader: dl}
	form := url.Values{}
	form.Set("title", "Remote Track")
	form.Set("source_url", "https://example.com/watch")

	req := httptest.NewRequest(http.MethodPost, "/tracks/new", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	s.handleNewTrack(rec, req)

	res := rec.Result()
	t.Cleanup(func() { _ = res.Body.Close() })

	if res.StatusCode != http.StatusSeeOther {
		t.Fatalf("expected status %d, got %d", http.StatusSeeOther, res.StatusCode)
	}
	if captured.FilePath != "music/downloaded.opus" {
		t.Fatalf("expected downloaded file path, got %q", captured.FilePath)
	}
}

func TestHandleTracks_CreateDownload(t *testing.T) {
	var captured library.CreateTrackParams
	repo := stubTrackRepository{
		createFunc: func(ctx context.Context, params library.CreateTrackParams) (library.Track, error) {
			captured = params
			return library.Track{ID: 99}, nil
		},
	}
	dl := stubDownloader{
		downloadFunc: func(ctx context.Context, req downloader.Request) (downloader.Result, error) {
			return downloader.Result{FilePath: "music/api-download.opus"}, nil
		},
	}

	s := &Server{trackRepo: repo, downloader: dl}
	source := "https://example.com/audio"
	payload := createTrackRequest{Title: "API Track", SourceURL: &source}
	req := httptest.NewRequest(http.MethodPost, "/tracks", bytes.NewReader(mustJSON(payload)))
	rec := httptest.NewRecorder()

	s.handleTracks(rec, req)

	res := rec.Result()
	t.Cleanup(func() { _ = res.Body.Close() })

	if res.StatusCode != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, res.StatusCode)
	}
	if captured.FilePath != "music/api-download.opus" {
		t.Fatalf("expected downloaded path, got %q", captured.FilePath)
	}
}
func mustJSON(v any) []byte {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return data
}
