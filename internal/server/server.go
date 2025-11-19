package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/a-h/templ"
	"github.com/lukasbarti/musixx/internal/config"
	"github.com/lukasbarti/musixx/internal/downloader"
	"github.com/lukasbarti/musixx/internal/library"
	"github.com/lukasbarti/musixx/web/ui"
)

// Server wraps the HTTP listener and handlers for musixx.
type Server struct {
	httpServer   *http.Server
	trackRepo    library.TrackRepository
	playlistRepo library.PlaylistRepository
	downloader   downloader.Service
}

// New constructs a server configured with routing stubs.
func New(trackRepo library.TrackRepository, playlistRepo library.PlaylistRepository, dl downloader.Service) (*Server, error) {
	mux := http.NewServeMux()
	mux.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir("web/assets"))))

	s := &Server{trackRepo: trackRepo, playlistRepo: playlistRepo, downloader: dl}
	mux.HandleFunc("/", s.handleLibraryPage)
	mux.HandleFunc("/tracks/new", s.handleNewTrack)
	mux.HandleFunc("/tracks", s.handleTracks)
	mux.HandleFunc("/tracks/", s.handleTrackByID)
	mux.HandleFunc("/playlists", s.handlePlaylists)
	mux.HandleFunc("/playlists/", s.handlePlaylistByID)
	mux.HandleFunc("/playlists/tracks", s.handlePlaylistTracks)
	mux.HandleFunc("/media/", s.handleMedia)

	httpSrv := &http.Server{
		Addr:    fmt.Sprintf(":%d", config.HTTPPort()),
		Handler: mux,
	}

	s.httpServer = httpSrv

	return s, nil
}

// Run starts the HTTP server and blocks until the context is canceled or the server exits.
func (s *Server) Run(ctx context.Context) error {
	errCh := make(chan error, 1)

	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("listen and serve: %w", err)
			return
		}
		errCh <- nil
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := s.httpServer.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("shutdown server: %w", err)
		}

		if err := <-errCh; err != nil {
			return fmt.Errorf("server run error: %w", err)
		}

		return context.Canceled

	case err := <-errCh:
		if err != nil {
			return fmt.Errorf("server run error: %w", err)
		}
		return nil
	}
}

func (s *Server) handleLibraryPage(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	tab := parseLibraryTab(r.URL.Query().Get("tab"))
	s.renderLibraryPage(w, r, tab, ui.PlaylistFormData{}, nil, nil)
}

func parseLibraryTab(raw string) ui.LibraryTab {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case string(ui.LibraryTabPlaylists):
		return ui.LibraryTabPlaylists
	default:
		return ui.LibraryTabTracks
	}
}

func (s *Server) renderLibraryPage(w http.ResponseWriter, r *http.Request, tab ui.LibraryTab, playlistForm ui.PlaylistFormData, playlistErrors []string, trackErrors []string) {
	data, err := s.loadLibraryPageData(r.Context(), tab, playlistForm, playlistErrors, trackErrors)
	if err != nil {
		log.Printf("load library data: %v", err)
		http.Error(w, "failed to load library", http.StatusInternalServerError)
		return
	}

	if err := s.renderHTML(w, r, ui.LibraryPage(data)); err != nil {
		log.Printf("render library page: %v", err)
		http.Error(w, "failed to render page", http.StatusInternalServerError)
	}
}

func (s *Server) loadLibraryPageData(ctx context.Context, tab ui.LibraryTab, playlistForm ui.PlaylistFormData, playlistErrors []string, trackErrors []string) (ui.LibraryPageData, error) {
	tracks, err := s.trackRepo.List(ctx)
	if err != nil {
		return ui.LibraryPageData{}, fmt.Errorf("list tracks: %w", err)
	}

	playlists, err := s.playlistRepo.List(ctx)
	if err != nil {
		return ui.LibraryPageData{}, fmt.Errorf("list playlists: %w", err)
	}

	return ui.LibraryPageData{
		Tracks:         tracks,
		Playlists:      playlists,
		ActiveTab:      tab,
		PlaylistForm:   playlistForm,
		PlaylistErrors: playlistErrors,
		TrackErrors:    trackErrors,
	}, nil
}

func (s *Server) loadPlaylistDetails(ctx context.Context, playlistID int64) (library.Playlist, []library.PlaylistTrackEntry, error) {
	playlist, err := s.playlistRepo.GetByID(ctx, playlistID)
	if err != nil {
		return library.Playlist{}, nil, fmt.Errorf("get playlist %d: %w", playlistID, err)
	}

	tracks, err := s.playlistRepo.ListTracks(ctx, playlistID)
	if err != nil {
		return library.Playlist{}, nil, fmt.Errorf("list tracks for playlist %d: %w", playlistID, err)
	}

	return playlist, tracks, nil
}

func (s *Server) handleNewTrack(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.renderNewTrackForm(w, r, ui.TrackFormData{}, nil)
	case http.MethodPost:
		if err := r.ParseForm(); err != nil {
			s.renderNewTrackForm(w, r, ui.TrackFormData{}, []string{"Invalid form submission."})
			return
		}

		form := ui.TrackFormData{
			Title:     strings.TrimSpace(r.FormValue("title")),
			Artist:    strings.TrimSpace(r.FormValue("artist")),
			Album:     strings.TrimSpace(r.FormValue("album")),
			Duration:  strings.TrimSpace(r.FormValue("duration_seconds")),
			FilePath:  strings.TrimSpace(r.FormValue("file_path")),
			SourceURL: strings.TrimSpace(r.FormValue("source_url")),
		}

		var validationErrors []string
		if form.Title == "" {
			validationErrors = append(validationErrors, "Title is required.")
		}

		var durationPtr *int
		if form.Duration != "" {
			dur, err := strconv.Atoi(form.Duration)
			if err != nil || dur < 0 {
				validationErrors = append(validationErrors, "Duration must be a non-negative integer.")
			} else {
				v := dur
				durationPtr = &v
			}
		}

		finalPath := form.FilePath
		if finalPath == "" && form.SourceURL != "" {
			if s.downloader == nil {
				log.Printf("download requested but no downloader configured")
				validationErrors = append(validationErrors, "Download service is not configured on this server.")
			} else {
				result, err := s.downloader.Download(r.Context(), downloader.Request{URL: form.SourceURL, Title: form.Title})
				if err != nil {
					log.Printf("download track: %v", err)
					validationErrors = append(validationErrors, "Unable to download audio with yt-dlp.")
				} else {
					finalPath = result.FilePath
					form.FilePath = finalPath
				}
			}
		}

		if finalPath == "" {
			validationErrors = append(validationErrors, "Provide either a file path or a source URL to download.")
		}

		if len(validationErrors) > 0 {
			s.renderNewTrackForm(w, r, form, validationErrors)
			return
		}

		_, err := s.trackRepo.Create(r.Context(), library.CreateTrackParams{
			Title:           form.Title,
			Artist:          optionalString(form.Artist),
			Album:           optionalString(form.Album),
			DurationSeconds: durationPtr,
			FilePath:        finalPath,
			SourceURL:       optionalString(form.SourceURL),
		})
		if err != nil {
			if isUniqueConstraintError(err) {
				s.renderNewTrackForm(w, r, form, []string{"A track already exists for that file path."})
				return
			}
			log.Printf("create track via form: %v", err)
			http.Error(w, "failed to save track", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/", http.StatusSeeOther)
	default:
		w.Header().Set("Allow", strings.Join([]string{http.MethodGet, http.MethodPost}, ", "))
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleTracks(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.listTracks(w, r)
	case http.MethodPost:
		s.createTrack(w, r)
	default:
		w.Header().Set("Allow", strings.Join([]string{http.MethodGet, http.MethodPost}, ", "))
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *Server) handlePlaylists(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.listPlaylists(w, r)
	case http.MethodPost:
		contentType := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
		if strings.HasPrefix(contentType, "application/json") {
			s.createPlaylistJSON(w, r)
		} else {
			s.createPlaylistForm(w, r)
		}
	default:
		w.Header().Set("Allow", strings.Join([]string{http.MethodGet, http.MethodPost}, ", "))
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *Server) handlePlaylistTracks(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		contentType := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
		if strings.HasPrefix(contentType, "application/json") {
			s.addTrackToPlaylistJSON(w, r)
		} else {
			s.addTrackToPlaylistForm(w, r)
		}
	default:
		w.Header().Set("Allow", http.MethodPost)
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *Server) handlePlaylistByID(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/playlists/")
	if path == "" {
		http.NotFound(w, r)
		return
	}

	segments := strings.Split(path, "/")
	if len(segments) == 0 || segments[0] == "" {
		http.NotFound(w, r)
		return
	}

	id, err := strconv.ParseInt(segments[0], 10, 64)
	if err != nil || id <= 0 {
		s.writeError(w, http.StatusBadRequest, "invalid playlist id")
		return
	}

	if len(segments) > 1 && segments[1] != "" {
		http.NotFound(w, r)
		return
	}

	switch r.Method {
	case http.MethodGet:
		if prefersHTML(r) {
			s.renderPlaylistPage(w, r, id)
		} else {
			s.getPlaylist(w, r, id)
		}
	default:
		w.Header().Set("Allow", http.MethodGet)
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *Server) renderPlaylistPage(w http.ResponseWriter, r *http.Request, playlistID int64) {
	playlist, tracks, err := s.loadPlaylistDetails(r.Context(), playlistID)
	if err != nil {
		if errors.Is(err, library.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		log.Printf("load playlist %d: %v", playlistID, err)
		http.Error(w, "failed to load playlist", http.StatusInternalServerError)
		return
	}

	data := ui.PlaylistPageData{
		Playlist: playlist,
		Tracks:   tracks,
	}

	if err := s.renderHTML(w, r, ui.PlaylistPage(data)); err != nil {
		log.Printf("render playlist page: %v", err)
		http.Error(w, "failed to render page", http.StatusInternalServerError)
	}
}

func (s *Server) getPlaylist(w http.ResponseWriter, r *http.Request, id int64) {
	playlist, tracks, err := s.loadPlaylistDetails(r.Context(), id)
	if err != nil {
		if errors.Is(err, library.ErrNotFound) {
			s.writeError(w, http.StatusNotFound, "playlist not found")
			return
		}
		log.Printf("load playlist %d: %v", id, err)
		s.writeError(w, http.StatusInternalServerError, "failed to load playlist")
		return
	}

	s.writeJSON(w, http.StatusOK, playlistDetailsResponse{Playlist: playlist, Tracks: tracks})
}

func (s *Server) createPlaylistForm(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		s.renderLibraryPage(w, r, ui.LibraryTabPlaylists, ui.PlaylistFormData{}, []string{"Invalid form submission."}, nil)
		return
	}

	form := ui.PlaylistFormData{
		Name:        strings.TrimSpace(r.FormValue("name")),
		Description: strings.TrimSpace(r.FormValue("description")),
	}

	var validationErrors []string
	if form.Name == "" {
		validationErrors = append(validationErrors, "Name is required.")
	}

	if len(validationErrors) > 0 {
		s.renderLibraryPage(w, r, ui.LibraryTabPlaylists, form, validationErrors, nil)
		return
	}

	_, err := s.playlistRepo.Create(r.Context(), library.CreatePlaylistParams{
		Name:        form.Name,
		Description: optionalString(form.Description),
	})
	if err != nil {
		log.Printf("create playlist: %v", err)
		http.Error(w, "failed to create playlist", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/?tab=playlists", http.StatusSeeOther)
}

func (s *Server) createPlaylistJSON(w http.ResponseWriter, r *http.Request) {
	var payload createPlaylistRequest
	if err := decodeJSON(r, &payload); err != nil {
		s.writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid json: %v", err))
		return
	}

	name := strings.TrimSpace(payload.Name)
	if name == "" {
		s.writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	var description *string
	if payload.Description != nil {
		description = optionalString(strings.TrimSpace(*payload.Description))
	}

	playlist, err := s.playlistRepo.Create(r.Context(), library.CreatePlaylistParams{
		Name:        name,
		Description: description,
	})
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("create playlist: %v", err))
		return
	}

	w.Header().Set("Location", fmt.Sprintf("/playlists/%d", playlist.ID))
	s.writeJSON(w, http.StatusCreated, playlist)
}

func (s *Server) listPlaylists(w http.ResponseWriter, r *http.Request) {
	playlists, err := s.playlistRepo.List(r.Context())
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("list playlists: %v", err))
		return
	}

	s.writeJSON(w, http.StatusOK, playlists)
}

func (s *Server) addTrackToPlaylistForm(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		s.renderLibraryPage(w, r, ui.LibraryTabTracks, ui.PlaylistFormData{}, nil, []string{"Invalid form submission."})
		return
	}

	rawPlaylistID := strings.TrimSpace(r.FormValue("playlist_id"))
	rawTrackID := strings.TrimSpace(r.FormValue("track_id"))

	var errorsList []string
	if rawPlaylistID == "" {
		errorsList = append(errorsList, "Select a playlist.")
	}
	if rawTrackID == "" {
		errorsList = append(errorsList, "Invalid track selection.")
	}

	playlistID, playlistErr := strconv.ParseInt(rawPlaylistID, 10, 64)
	if playlistErr != nil || playlistID <= 0 {
		errorsList = append(errorsList, "Invalid playlist selection.")
	}

	trackID, trackErr := strconv.ParseInt(rawTrackID, 10, 64)
	if trackErr != nil || trackID <= 0 {
		errorsList = append(errorsList, "Invalid track selection.")
	}

	if len(errorsList) > 0 {
		s.renderLibraryPage(w, r, ui.LibraryTabTracks, ui.PlaylistFormData{}, nil, errorsList)
		return
	}

	if _, err := s.playlistRepo.GetByID(r.Context(), playlistID); err != nil {
		if errors.Is(err, library.ErrNotFound) {
			s.renderLibraryPage(w, r, ui.LibraryTabTracks, ui.PlaylistFormData{}, nil, []string{"Playlist not found."})
			return
		}
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("load playlist: %v", err))
		return
	}

	if _, err := s.trackRepo.GetByID(r.Context(), trackID); err != nil {
		if errors.Is(err, library.ErrNotFound) {
			s.renderLibraryPage(w, r, ui.LibraryTabTracks, ui.PlaylistFormData{}, nil, []string{"Track not found."})
			return
		}
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("load track: %v", err))
		return
	}

	if err := s.playlistRepo.AddTrack(r.Context(), playlistID, trackID); err != nil {
		if isUniqueConstraintError(err) {
			s.renderLibraryPage(w, r, ui.LibraryTabTracks, ui.PlaylistFormData{}, nil, []string{"Track is already in that playlist."})
			return
		}
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("add track to playlist: %v", err))
		return
	}

	http.Redirect(w, r, "/?tab=tracks", http.StatusSeeOther)
}

func (s *Server) addTrackToPlaylistJSON(w http.ResponseWriter, r *http.Request) {
	var payload addTrackToPlaylistRequest
	if err := decodeJSON(r, &payload); err != nil {
		s.writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if payload.PlaylistID <= 0 {
		s.writeError(w, http.StatusBadRequest, "playlist_id is required")
		return
	}
	if payload.TrackID <= 0 {
		s.writeError(w, http.StatusBadRequest, "track_id is required")
		return
	}

	if _, err := s.playlistRepo.GetByID(r.Context(), payload.PlaylistID); err != nil {
		if errors.Is(err, library.ErrNotFound) {
			s.writeError(w, http.StatusNotFound, "playlist not found")
			return
		}
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("load playlist: %v", err))
		return
	}

	if _, err := s.trackRepo.GetByID(r.Context(), payload.TrackID); err != nil {
		if errors.Is(err, library.ErrNotFound) {
			s.writeError(w, http.StatusNotFound, "track not found")
			return
		}
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("load track: %v", err))
		return
	}

	if err := s.playlistRepo.AddTrack(r.Context(), payload.PlaylistID, payload.TrackID); err != nil {
		if isUniqueConstraintError(err) {
			s.writeError(w, http.StatusConflict, "track already exists in playlist")
			return
		}
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("add track to playlist: %v", err))
		return
	}

	s.writeJSON(w, http.StatusCreated, addTrackToPlaylistResponse{PlaylistID: payload.PlaylistID, TrackID: payload.TrackID})
}

func (s *Server) handleTrackByID(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/tracks/")
	if path == "" {
		s.writeError(w, http.StatusNotFound, "track not found")
		return
	}

	segments := strings.Split(path, "/")
	if len(segments) == 0 || segments[0] == "" {
		s.writeError(w, http.StatusNotFound, "track not found")
		return
	}

	id, err := strconv.ParseInt(segments[0], 10, 64)
	if err != nil {
		if len(segments) == 1 {
			s.writeError(w, http.StatusBadRequest, "invalid track id")
			return
		}
		http.NotFound(w, r)
		return
	}

	if len(segments) == 1 {
		switch r.Method {
		case http.MethodGet:
			s.getTrack(w, r, id)
		case http.MethodPut, http.MethodPatch:
			s.updateTrack(w, r, id)
		case http.MethodDelete:
			s.deleteTrack(w, r, id)
		default:
			w.Header().Set("Allow", strings.Join([]string{http.MethodGet, http.MethodPut, http.MethodPatch, http.MethodDelete}, ", "))
			s.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
		return
	}

	if len(segments) > 2 {
		http.NotFound(w, r)
		return
	}

	action := segments[1]
	switch action {
	case "edit":
		s.handleTrackEdit(w, r, id)
	case "delete":
		s.handleTrackDeleteHTML(w, r, id)
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) handleTrackEdit(w http.ResponseWriter, r *http.Request, id int64) {
	switch r.Method {
	case http.MethodGet:
		track, err := s.trackRepo.GetByID(r.Context(), id)
		if err != nil {
			if errors.Is(err, library.ErrNotFound) {
				http.NotFound(w, r)
				return
			}
			log.Printf("load track for edit: %v", err)
			http.Error(w, "failed to load track", http.StatusInternalServerError)
			return
		}

		form := trackFormData(track)
		s.renderEditTrackForm(w, r, id, form, nil)
	case http.MethodPost:
		existing, err := s.trackRepo.GetByID(r.Context(), id)
		if err != nil {
			if errors.Is(err, library.ErrNotFound) {
				http.NotFound(w, r)
				return
			}
			log.Printf("load track for update: %v", err)
			http.Error(w, "failed to load track", http.StatusInternalServerError)
			return
		}

		if err := r.ParseForm(); err != nil {
			s.renderEditTrackForm(w, r, existing.ID, trackFormData(existing), []string{"Invalid form submission."})
			return
		}

		form := ui.TrackFormData{
			Title:     strings.TrimSpace(r.FormValue("title")),
			Artist:    strings.TrimSpace(r.FormValue("artist")),
			Album:     strings.TrimSpace(r.FormValue("album")),
			Duration:  strings.TrimSpace(r.FormValue("duration_seconds")),
			FilePath:  existing.FilePath,
			SourceURL: strings.TrimSpace(r.FormValue("source_url")),
		}

		var validationErrors []string
		if form.Title == "" {
			validationErrors = append(validationErrors, "Title is required.")
		}

		var durationPtr *int
		if form.Duration != "" {
			dur, err := strconv.Atoi(form.Duration)
			if err != nil || dur < 0 {
				validationErrors = append(validationErrors, "Duration must be a non-negative integer.")
			} else {
				v := dur
				durationPtr = &v
			}
		}

		if len(validationErrors) > 0 {
			s.renderEditTrackForm(w, r, id, form, validationErrors)
			return
		}

		title := form.Title
		params := library.UpdateTrackParams{
			ID:              id,
			Title:           &title,
			Artist:          optionalString(form.Artist),
			Album:           optionalString(form.Album),
			DurationSeconds: durationPtr,
			SourceURL:       optionalString(form.SourceURL),
		}

		if _, err := s.trackRepo.Update(r.Context(), params); err != nil {
			if errors.Is(err, library.ErrNotFound) {
				http.NotFound(w, r)
				return
			}
			log.Printf("update track via form: %v", err)
			http.Error(w, "failed to update track", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/", http.StatusSeeOther)
	default:
		w.Header().Set("Allow", strings.Join([]string{http.MethodGet, http.MethodPost}, ", "))
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleTrackDeleteHTML(w http.ResponseWriter, r *http.Request, id int64) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := s.trackRepo.Delete(r.Context(), id); err != nil {
		if errors.Is(err, library.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		log.Printf("delete track via form: %v", err)
		http.Error(w, "failed to delete track", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (s *Server) handleMedia(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	idSegment := strings.TrimPrefix(r.URL.Path, "/media/")
	if idSegment == "" || strings.Contains(idSegment, "/") {
		http.NotFound(w, r)
		return
	}

	id, err := strconv.ParseInt(idSegment, 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	track, err := s.trackRepo.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, library.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		log.Printf("load track for media: %v", err)
		http.Error(w, "failed to load media", http.StatusInternalServerError)
		return
	}

	file, err := os.Open(track.FilePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			http.NotFound(w, r)
			return
		}
		log.Printf("open media file: %v", err)
		http.Error(w, "failed to open media file", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		log.Printf("stat media file: %v", err)
		http.Error(w, "failed to read media file", http.StatusInternalServerError)
		return
	}

	ext := strings.ToLower(filepath.Ext(track.FilePath))
	contentType := mime.TypeByExtension(ext)
	if contentType == "" {
		switch ext {
		case ".opus":
			contentType = "audio/ogg; codecs=opus"
		case ".ogg":
			contentType = "audio/ogg"
		default:
			contentType = "application/octet-stream"
		}
	}

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "no-store")
	http.ServeContent(w, r, filepath.Base(track.FilePath), info.ModTime(), file)
}

func (s *Server) listTracks(w http.ResponseWriter, r *http.Request) {
	tracks, err := s.trackRepo.List(r.Context())
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("list tracks: %v", err))
		return
	}

	s.writeJSON(w, http.StatusOK, tracks)
}

func (s *Server) getTrack(w http.ResponseWriter, r *http.Request, id int64) {
	track, err := s.trackRepo.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, library.ErrNotFound) {
			s.writeError(w, http.StatusNotFound, "track not found")
			return
		}
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("get track: %v", err))
		return
	}

	s.writeJSON(w, http.StatusOK, track)
}

func (s *Server) createTrack(w http.ResponseWriter, r *http.Request) {
	var payload createTrackRequest
	if err := decodeJSON(r, &payload); err != nil {
		s.writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if strings.TrimSpace(payload.Title) == "" {
		s.writeError(w, http.StatusBadRequest, "title is required")
		return
	}

	filePath := strings.TrimSpace(payload.FilePath)
	sourceURL := ""
	if payload.SourceURL != nil {
		sourceURL = strings.TrimSpace(*payload.SourceURL)
	}

	if filePath == "" && sourceURL == "" {
		s.writeError(w, http.StatusBadRequest, "either file_path or source_url must be provided")
		return
	}
	if payload.DurationSeconds != nil && *payload.DurationSeconds < 0 {
		s.writeError(w, http.StatusBadRequest, "duration_seconds cannot be negative")
		return
	}

	if filePath == "" && sourceURL != "" {
		if s.downloader == nil {
			s.writeError(w, http.StatusInternalServerError, "download service not configured")
			return
		}
		result, err := s.downloader.Download(r.Context(), downloader.Request{URL: sourceURL, Title: payload.Title})
		if err != nil {
			log.Printf("download track via api: %v", err)
			s.writeError(w, http.StatusInternalServerError, "failed to download audio")
			return
		}
		filePath = result.FilePath
	}

	track, err := s.trackRepo.Create(r.Context(), library.CreateTrackParams{
		Title:           payload.Title,
		Album:           payload.Album,
		Artist:          payload.Artist,
		DurationSeconds: payload.DurationSeconds,
		FilePath:        filePath,
		SourceURL:       optionalString(sourceURL),
	})
	if err != nil {
		if isUniqueConstraintError(err) {
			s.writeError(w, http.StatusConflict, "track already exists for this file path")
			return
		}
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("create track: %v", err))
		return
	}

	w.Header().Set("Location", fmt.Sprintf("/tracks/%d", track.ID))
	s.writeJSON(w, http.StatusCreated, track)
}

func (s *Server) updateTrack(w http.ResponseWriter, r *http.Request, id int64) {
	var payload updateTrackRequest
	if err := decodeJSON(r, &payload); err != nil {
		s.writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if payload.DurationSeconds != nil && *payload.DurationSeconds < 0 {
		s.writeError(w, http.StatusBadRequest, "duration_seconds cannot be negative")
		return
	}

	track, err := s.trackRepo.Update(r.Context(), library.UpdateTrackParams{
		ID:              id,
		Title:           payload.Title,
		Album:           payload.Album,
		Artist:          payload.Artist,
		DurationSeconds: payload.DurationSeconds,
		SourceURL:       payload.SourceURL,
	})
	if err != nil {
		if errors.Is(err, library.ErrNotFound) {
			s.writeError(w, http.StatusNotFound, "track not found")
			return
		}
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("update track: %v", err))
		return
	}

	s.writeJSON(w, http.StatusOK, track)
}

func (s *Server) deleteTrack(w http.ResponseWriter, r *http.Request, id int64) {
	if err := s.trackRepo.Delete(r.Context(), id); err != nil {
		if errors.Is(err, library.ErrNotFound) {
			s.writeError(w, http.StatusNotFound, "track not found")
			return
		}
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("delete track: %v", err))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) renderNewTrackForm(w http.ResponseWriter, r *http.Request, form ui.TrackFormData, errs []string) {
	if err := s.renderHTML(w, r, ui.TrackFormPage(form, errs)); err != nil {
		log.Printf("render track form: %v", err)
		http.Error(w, "failed to render page", http.StatusInternalServerError)
	}
}

func (s *Server) renderEditTrackForm(w http.ResponseWriter, r *http.Request, id int64, form ui.TrackFormData, errs []string) {
	if err := s.renderHTML(w, r, ui.TrackEditPage(id, form, errs)); err != nil {
		log.Printf("render track edit form: %v", err)
		http.Error(w, "failed to render page", http.StatusInternalServerError)
	}
}

func (s *Server) renderHTML(w http.ResponseWriter, r *http.Request, component templ.Component) error {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := component.Render(r.Context(), w); err != nil {
		return fmt.Errorf("render component: %w", err)
	}
	return nil
}

func prefersHTML(r *http.Request) bool {
	accept := r.Header.Get("Accept")
	if accept == "" {
		return true
	}

	for _, part := range strings.Split(accept, ",") {
		mediaType := strings.SplitN(part, ";", 2)[0]
		mediaType = strings.ToLower(strings.TrimSpace(mediaType))
		switch mediaType {
		case "text/html", "application/xhtml+xml", "*/*":
			return true
		}
	}

	return false
}

func optionalString(value string) *string {
	if value == "" {
		return nil
	}
	v := value
	return &v
}

func trackFormData(track library.Track) ui.TrackFormData {
	var duration string
	if track.DurationSeconds != nil {
		duration = strconv.Itoa(*track.DurationSeconds)
	}

	return ui.TrackFormData{
		Title:     track.Title,
		Artist:    derefString(track.Artist),
		Album:     derefString(track.Album),
		Duration:  duration,
		FilePath:  track.FilePath,
		SourceURL: derefString(track.SourceURL),
	}
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func (s *Server) writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if payload == nil {
		return
	}
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Printf("encode json response: %v", err)
	}
}

func (s *Server) writeError(w http.ResponseWriter, status int, message string) {
	s.writeJSON(w, status, errorResponse{Error: message})
}

func decodeJSON(r *http.Request, dst any) error {
	defer r.Body.Close()
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		return fmt.Errorf("invalid JSON payload: %w", err)
	}
	return nil
}

func isUniqueConstraintError(err error) bool {
	return strings.Contains(strings.ToLower(err.Error()), "unique constraint")
}

type createTrackRequest struct {
	Title           string  `json:"title"`
	Album           *string `json:"album"`
	Artist          *string `json:"artist"`
	DurationSeconds *int    `json:"duration_seconds"`
	FilePath        string  `json:"file_path"`
	SourceURL       *string `json:"source_url"`
}

type updateTrackRequest struct {
	Title           *string `json:"title"`
	Album           *string `json:"album"`
	Artist          *string `json:"artist"`
	DurationSeconds *int    `json:"duration_seconds"`
	SourceURL       *string `json:"source_url"`
}

type errorResponse struct {
	Error string `json:"error"`
}

type createPlaylistRequest struct {
	Name        string  `json:"name"`
	Description *string `json:"description"`
}

type addTrackToPlaylistRequest struct {
	PlaylistID int64 `json:"playlist_id"`
	TrackID    int64 `json:"track_id"`
}

type addTrackToPlaylistResponse struct {
	PlaylistID int64 `json:"playlist_id"`
	TrackID    int64 `json:"track_id"`
}

type playlistDetailsResponse struct {
	Playlist library.Playlist             `json:"playlist"`
	Tracks   []library.PlaylistTrackEntry `json:"tracks"`
}
