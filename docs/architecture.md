# Musixx Architecture Overview

## Top-Level Structure
- `cmd/`: Cobra-based CLI entry points, including commands such as `serve`, `download`, `migrate`, and `config`. Commands parse CLI flags and delegate into internal services.
- `internal/app`: Bootstrap logic wiring configuration, storage, background workers, and the HTTP server. Provides `Run()` helpers invoked by CLI commands.
- `internal/config`: Configuration layer built on Viper. Loads default values, merges environment variables and CLI flags, exposes typed settings (storage paths, HTTP port, yt-dlp binary location).
- `internal/library`: Core domain models (track, album, playlist) and service interfaces for CRUD, search, and organization tasks.
- `internal/storage/sqlite`: SQLite-backed repository implementations, schema definitions, and migration helpers.
- `internal/media`: Audio pipeline utilities for invoking yt-dlp, handling ffmpeg/Opus conversion, storing files on disk, and validating audio artifacts.
- `internal/metadata`: Provider abstraction for external metadata enrichment services. Ships with a stub implementation; concrete adapters (e.g., MusicBrainz) can be plugged in later.
- `internal/server`: HTTP server exposing the web UI and API endpoints. Handles routing, middleware, authentication hooks, WebSocket upgrades, and progress notifications.
- `internal/server/views`: Generated Go templates (via Templ) powering the UI. Source Templ files live under `web/ui` and `web/components`.
- `internal/playback`: Streaming utilities that read Opus files, manage concurrent playback sessions, and prepare data for the browser audio player.
- `internal/workers`: Background job orchestration handling downloads, metadata lookups, waveform generation, and other asynchronous tasks.
- `assets/`: Static files served alongside the web UI (CSS, JS, images).
- `migrations/`: SQL migration files executed to evolve the SQLite schema.

## Workflow Conventions
- All build, run, and test workflows are executed through `just` recipes defined in the repository `justfile`.
- CLI commands should surface long-running operations (downloads, metadata imports) with progress indicators in the UI.
- Cross-package interactions must flow through explicit interfaces to keep components testable and swappable.
- Background workers communicate with the main application via channels and shared service interfaces, avoiding direct package coupling.

## Future Considerations
- Introduce caching for metadata lookups to minimize network calls once a provider is selected.
- Evaluate job persistence (e.g., BoltDB, SQLite) if background tasks need durability across restarts.
- Add health-check and diagnostic endpoints for local monitoring if desired.
