package downloader

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Request describes a download job initiated by the user.
type Request struct {
	URL   string
	Title string
}

// Result captures the filesystem location of the downloaded asset.
type Result struct {
	FilePath string
}

// Service represents a component capable of acquiring media from remote sources.
type Service interface {
	Download(ctx context.Context, req Request) (Result, error)
}

// YTDLP implements Service via the yt-dlp CLI.
type YTDLP struct {
	binary   string
	mediaDir string
}

// NewYTDLP constructs a downloader backed by the yt-dlp executable.
func NewYTDLP(binary, mediaDir string) (*YTDLP, error) {
	binary = strings.TrimSpace(binary)
	if binary == "" {
		return nil, errors.New("yt-dlp binary path must not be empty")
	}

	mediaDir = strings.TrimSpace(mediaDir)
	if mediaDir == "" {
		return nil, errors.New("media directory must not be empty")
	}

	if err := os.MkdirAll(mediaDir, 0o755); err != nil {
		return nil, fmt.Errorf("ensure media directory: %w", err)
	}

	return &YTDLP{binary: binary, mediaDir: mediaDir}, nil
}

// Download stores the remote audio as an Opus file and returns its path.
func (d *YTDLP) Download(ctx context.Context, req Request) (Result, error) {
	if strings.TrimSpace(req.URL) == "" {
		return Result{}, errors.New("download url must not be empty")
	}

	base := slugify(req.Title)
	if base == "" {
		base = "track"
	}

	base = d.ensureUniqueBase(base)
	outputPattern := filepath.Join(d.mediaDir, base) + ".%(ext)s"
	finalPath := filepath.Join(d.mediaDir, base) + ".opus"

	var buf bytes.Buffer
	cmd := exec.CommandContext(
		ctx,
		d.binary,
		"--ignore-errors",
		"--no-playlist",
		"--no-progress",
		"--restrict-filenames",
		"--cookies", "cookies.txt",
		"--extract-audio",
		"--audio-format", "opus",
		"--audio-quality", "0",
		"--output", outputPattern,
		req.URL,
	)
	cmd.Stdout = &buf
	cmd.Stderr = &buf

	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return Result{}, fmt.Errorf("yt-dlp failed: %w: %s", err, strings.TrimSpace(buf.String()))
		}
		return Result{}, fmt.Errorf("execute yt-dlp: %w", err)
	}

	if _, err := os.Stat(finalPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Result{}, fmt.Errorf("yt-dlp completed but file missing at %s", finalPath)
		}
		return Result{}, fmt.Errorf("stat downloaded file: %w", err)
	}

	return Result{FilePath: finalPath}, nil
}

func (d *YTDLP) ensureUniqueBase(base string) string {
	// Ensure consistency if slugify returned empty.
	if base == "" {
		base = "track"
	}

	candidate := base
	suffix := 1

	for {
		path := filepath.Join(d.mediaDir, candidate) + ".opus"
		if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
			return candidate
		}
		candidate = fmt.Sprintf("%s-%02d", base, suffix)
		suffix++
	}
}

func slugify(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return ""
	}

	var builder strings.Builder
	lastDash := false

	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
			builder.WriteRune(r)
			lastDash = false
		case r >= '0' && r <= '9':
			builder.WriteRune(r)
			lastDash = false
		default:
			if !lastDash && builder.Len() > 0 {
				builder.WriteByte('-')
				lastDash = true
			}
		}
	}

	slug := strings.Trim(builder.String(), "-")
	if slug == "" {
		// Fallback guarantees uniqueness with timestamp.
		return fmt.Sprintf("track-%d", time.Now().Unix())
	}
	return slug
}
