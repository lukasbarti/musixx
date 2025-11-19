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

type stubPlaylistRepository struct {
	createFunc                func(ctx context.Context, params library.CreatePlaylistParams) (library.Playlist, error)
	getFunc                   func(ctx context.Context, id int64) (library.Playlist, error)
	listFunc                  func(ctx context.Context) ([]library.Playlist, error)
	deleteFunc                func(ctx context.Context, id int64) error
	addTrackFunc              func(ctx context.Context, playlistID, trackID int64) error
	listTracksForPlaylistFunc func(ctx context.Context, playlistID int64) ([]library.PlaylistTrackEntry, error)
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
	return newServerWithRepos(repo, stubPlaylistRepository{
		listFunc: func(ctx context.Context) ([]library.Playlist, error) {
			return nil, nil
		},
		createFunc: func(ctx context.Context, params library.CreatePlaylistParams) (library.Playlist, error) {
			panic("Create not implemented in stub")
		},
		getFunc: func(ctx context.Context, id int64) (library.Playlist, error) {
			panic("GetByID not implemented in stub")
		},
		deleteFunc: func(ctx context.Context, id int64) error {
			panic("Delete not implemented in stub")
		},
		addTrackFunc: func(ctx context.Context, playlistID, trackID int64) error {
			panic("AddTrack not implemented in stub")
		},
		listTracksForPlaylistFunc: func(ctx context.Context, playlistID int64) ([]library.PlaylistTrackEntry, error) {
			return nil, nil
		},
	})
}

func newServerWithRepos(trackRepo stubTrackRepository, playlistRepo stubPlaylistRepository) *Server {
	return &Server{
		trackRepo:    trackRepo,
		playlistRepo: playlistRepo,
		downloader: stubDownloader{
			downloadFunc: func(ctx context.Context, req downloader.Request) (downloader.Result, error) {
				panic("unexpected downloader invocation")
			},
		},
	}
}

func (s stubPlaylistRepository) Create(ctx context.Context, params library.CreatePlaylistParams) (library.Playlist, error) {
	if s.createFunc == nil {
		panic("Create not implemented in stub")
	}
	return s.createFunc(ctx, params)
}

func (s stubPlaylistRepository) GetByID(ctx context.Context, id int64) (library.Playlist, error) {
	if s.getFunc == nil {
		panic("GetByID not implemented in stub")
	}
	return s.getFunc(ctx, id)
}

func (s stubPlaylistRepository) List(ctx context.Context) ([]library.Playlist, error) {
	if s.listFunc == nil {
		panic("List not implemented in stub")
	}
	return s.listFunc(ctx)
}

func (s stubPlaylistRepository) ListTracks(ctx context.Context, playlistID int64) ([]library.PlaylistTrackEntry, error) {
	if s.listTracksForPlaylistFunc == nil {
		return nil, nil
	}
	return s.listTracksForPlaylistFunc(ctx, playlistID)
}

func (s stubPlaylistRepository) Delete(ctx context.Context, id int64) error {
	if s.deleteFunc == nil {
		panic("Delete not implemented in stub")
	}
	return s.deleteFunc(ctx, id)
}

func (s stubPlaylistRepository) AddTrack(ctx context.Context, playlistID, trackID int64) error {
	if s.addTrackFunc == nil {
		panic("AddTrack not implemented in stub")
	}
	return s.addTrackFunc(ctx, playlistID, trackID)
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
		return nil, nil
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

func TestHandlePlaylists_ListSuccess(t *testing.T) {
	now := time.Now().UTC()
	playlistRepo := stubPlaylistRepository{
		listFunc: func(ctx context.Context) ([]library.Playlist, error) {
			return []library.Playlist{{
				ID:        7,
				Name:      "Favorites",
				CreatedAt: now,
				UpdatedAt: now,
			}}, nil
		},
		createFunc: func(ctx context.Context, params library.CreatePlaylistParams) (library.Playlist, error) {
			panic("unexpected create invocation")
		},
		getFunc: func(ctx context.Context, id int64) (library.Playlist, error) {
			panic("unexpected get invocation")
		},
		deleteFunc: func(ctx context.Context, id int64) error {
			panic("unexpected delete invocation")
		},
		addTrackFunc: func(ctx context.Context, playlistID, trackID int64) error {
			panic("unexpected add track invocation")
		},
	}

	s := newServerWithRepos(stubTrackRepository{listFunc: func(ctx context.Context) ([]library.Track, error) {
		return nil, nil
	}}, playlistRepo)
	req := httptest.NewRequest(http.MethodGet, "/playlists", nil)
	rec := httptest.NewRecorder()

	s.handlePlaylists(rec, req)

	res := rec.Result()
	t.Cleanup(func() { _ = res.Body.Close() })

	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, res.StatusCode)
	}

	var playlists []library.Playlist
	if err := json.NewDecoder(res.Body).Decode(&playlists); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if len(playlists) != 1 || playlists[0].Name != "Favorites" {
		t.Fatalf("unexpected playlists response: %+v", playlists)
	}
}

func TestHandlePlaylists_CreateJSONSuccess(t *testing.T) {
	now := time.Now().UTC()
	playlistRepo := stubPlaylistRepository{
		createFunc: func(ctx context.Context, params library.CreatePlaylistParams) (library.Playlist, error) {
			if params.Name != "Chill" {
				t.Fatalf("expected name Chill, got %q", params.Name)
			}
			return library.Playlist{ID: 10, Name: params.Name, CreatedAt: now, UpdatedAt: now}, nil
		},
		listFunc: func(ctx context.Context) ([]library.Playlist, error) {
			return nil, nil
		},
		getFunc: func(ctx context.Context, id int64) (library.Playlist, error) {
			panic("unexpected get invocation")
		},
		deleteFunc: func(ctx context.Context, id int64) error {
			panic("unexpected delete invocation")
		},
		addTrackFunc: func(ctx context.Context, playlistID, trackID int64) error {
			panic("unexpected add track invocation")
		},
	}

	s := newServerWithRepos(stubTrackRepository{listFunc: func(ctx context.Context) ([]library.Track, error) {
		return nil, nil
	}}, playlistRepo)

	payload := createPlaylistRequest{Name: "Chill"}
	req := httptest.NewRequest(http.MethodPost, "/playlists", bytes.NewReader(mustJSON(payload)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	s.handlePlaylists(rec, req)

	res := rec.Result()
	t.Cleanup(func() { _ = res.Body.Close() })

	if res.StatusCode != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, res.StatusCode)
	}

	if location := res.Header.Get("Location"); location != "/playlists/10" {
		t.Fatalf("expected Location /playlists/10, got %q", location)
	}

	var playlist library.Playlist
	if err := json.NewDecoder(res.Body).Decode(&playlist); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if playlist.ID != 10 || playlist.Name != "Chill" {
		t.Fatalf("unexpected playlist response: %+v", playlist)
	}
}

func TestHandlePlaylists_CreateFormValidation(t *testing.T) {
	playlistRepo := stubPlaylistRepository{
		listFunc: func(ctx context.Context) ([]library.Playlist, error) {
			return nil, nil
		},
		createFunc: func(ctx context.Context, params library.CreatePlaylistParams) (library.Playlist, error) {
			panic("unexpected create invocation")
		},
		getFunc: func(ctx context.Context, id int64) (library.Playlist, error) {
			panic("unexpected get invocation")
		},
		deleteFunc: func(ctx context.Context, id int64) error {
			panic("unexpected delete invocation")
		},
		addTrackFunc: func(ctx context.Context, playlistID, trackID int64) error {
			panic("unexpected add track invocation")
		},
	}

	s := newServerWithRepos(stubTrackRepository{}, playlistRepo)
	req := httptest.NewRequest(http.MethodPost, "/playlists", strings.NewReader("name="))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	s.handlePlaylists(rec, req)

	res := rec.Result()
	t.Cleanup(func() { _ = res.Body.Close() })

	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, res.StatusCode)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}

	if !strings.Contains(string(body), "Name is required.") {
		t.Fatalf("expected validation error in body")
	}
}

func TestHandlePlaylistByID_RenderHTML(t *testing.T) {
	now := time.Now().UTC()
	playlist := library.Playlist{ID: 3, Name: "Road Trip", CreatedAt: now, UpdatedAt: now}
	track := library.Track{ID: 11, Title: "Sunset Drive", FilePath: "music/sunset-drive.opus", CreatedAt: now, UpdatedAt: now}

	playlistRepo := stubPlaylistRepository{
		listFunc: func(ctx context.Context) ([]library.Playlist, error) {
			return nil, nil
		},
		getFunc: func(ctx context.Context, id int64) (library.Playlist, error) {
			if id != playlist.ID {
				t.Fatalf("unexpected playlist id: %d", id)
			}
			return playlist, nil
		},
		listTracksForPlaylistFunc: func(ctx context.Context, playlistID int64) ([]library.PlaylistTrackEntry, error) {
			if playlistID != playlist.ID {
				t.Fatalf("unexpected playlist id for tracks: %d", playlistID)
			}
			return []library.PlaylistTrackEntry{{
				Track:    track,
				Position: 0,
				AddedAt:  now,
			}}, nil
		},
		createFunc: func(ctx context.Context, params library.CreatePlaylistParams) (library.Playlist, error) {
			panic("unexpected create invocation")
		},
		deleteFunc: func(ctx context.Context, id int64) error {
			panic("unexpected delete invocation")
		},
		addTrackFunc: func(ctx context.Context, playlistID, trackID int64) error {
			panic("unexpected add track invocation")
		},
	}

	s := newServerWithRepos(stubTrackRepository{}, playlistRepo)
	req := httptest.NewRequest(http.MethodGet, "/playlists/3", nil)
	rec := httptest.NewRecorder()

	s.handlePlaylistByID(rec, req)

	res := rec.Result()
	t.Cleanup(func() { _ = res.Body.Close() })

	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, res.StatusCode)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}

	if !strings.Contains(string(body), playlist.Name) {
		t.Fatalf("expected body to contain playlist name")
	}
	if !strings.Contains(string(body), track.Title) {
		t.Fatalf("expected body to contain track title")
	}
}

func TestHandlePlaylistByID_GetJSON(t *testing.T) {
	now := time.Now().UTC()
	playlist := library.Playlist{ID: 6, Name: "Focus", CreatedAt: now, UpdatedAt: now}
	track := library.Track{ID: 21, Title: "Laser", FilePath: "music/laser.opus", CreatedAt: now, UpdatedAt: now}

	playlistRepo := stubPlaylistRepository{
		listFunc: func(ctx context.Context) ([]library.Playlist, error) {
			return nil, nil
		},
		getFunc: func(ctx context.Context, id int64) (library.Playlist, error) {
			if id != playlist.ID {
				t.Fatalf("unexpected playlist id: %d", id)
			}
			return playlist, nil
		},
		listTracksForPlaylistFunc: func(ctx context.Context, playlistID int64) ([]library.PlaylistTrackEntry, error) {
			if playlistID != playlist.ID {
				t.Fatalf("unexpected playlist id for tracks: %d", playlistID)
			}
			return []library.PlaylistTrackEntry{{
				Track:    track,
				Position: 1,
				AddedAt:  now,
			}}, nil
		},
		createFunc: func(ctx context.Context, params library.CreatePlaylistParams) (library.Playlist, error) {
			panic("unexpected create invocation")
		},
		deleteFunc: func(ctx context.Context, id int64) error {
			panic("unexpected delete invocation")
		},
		addTrackFunc: func(ctx context.Context, playlistID, trackID int64) error {
			panic("unexpected add track invocation")
		},
	}

	s := newServerWithRepos(stubTrackRepository{}, playlistRepo)
	req := httptest.NewRequest(http.MethodGet, "/playlists/6", nil)
	req.Header.Set("Accept", "application/json")
	rec := httptest.NewRecorder()

	s.handlePlaylistByID(rec, req)

	res := rec.Result()
	t.Cleanup(func() { _ = res.Body.Close() })

	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, res.StatusCode)
	}

	var payload playlistDetailsResponse
	if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if payload.Playlist.ID != playlist.ID {
		t.Fatalf("unexpected playlist id in response: %+v", payload.Playlist)
	}
	if len(payload.Tracks) != 1 {
		t.Fatalf("expected 1 track, got %d", len(payload.Tracks))
	}
	if payload.Tracks[0].Track.ID != track.ID {
		t.Fatalf("unexpected track id: %+v", payload.Tracks[0])
	}
	if payload.Tracks[0].Position != 1 {
		t.Fatalf("unexpected track position: %d", payload.Tracks[0].Position)
	}
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
