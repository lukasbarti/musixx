# Musixx Project Charter

## Goals
- Provide a local-only web application for playing and managing a personal music library.
- Serve a browser-based UI backed by a Go HTTP server.
- Store audio tracks on disk in the Opus format for efficient quality-to-size balance.
- Persist all track metadata in a SQLite database for reliable querying and updates.
- Let users stream tracks through the UI and perform core library management tasks (browse, search, edit metadata, organize collections).
- Enable downloading new songs through an integrated yt-dlp workflow initiated from a form.

## Feature Requirements
- Offer a download form that accepts a source URL plus user-supplied fields such as track name and album.
- Integrate yt-dlp to fetch audio, convert it to Opus, and register the track and metadata in the library.
- Provide hooks for optional automatic metadata enrichment via an external service (service to be selected later).
- Expose management actions in the UI (e.g., edit metadata, delete tracks, organize playlists or folders).

## Technology Stack
- Language: Go.
- Configuration management: Viper.
- Command-line interface scaffolding: Cobra.
- HTML templating: Templ.
- Database: SQLite.
- Media acquisition: yt-dlp.

## Implementation Notes
- Design an abstraction layer for metadata providers so an external lookup service can be plugged in once chosen.
- Keep the metadata schema expressive enough for future enhancements (e.g., genres, artwork, track numbers).
- Ensure the web UI can handle concurrent playback and administrative actions without blocking.
- Wrap outbound errors with context using `fmt.Errorf` or `errors.New` so callers receive actionable diagnostics.

## Build & Run Workflow
- Use the `just` command runner for every build, run, or test action to keep workflows scripted and reproducible.
- Add or update targets in the `Justfile` whenever a required build, run, or test workflow is not yet represented.

## LLM Collaboration Guardrails
- Always consult the user before installing or upgrading any dependency to avoid version drift.
- Do not remove or deprecate existing features unless explicitly instructed by the user beforehand.

## Outstanding Questions
- Evaluate and select the preferred metadata lookup service (e.g., MusicBrainz, Discogs, or similar) prior to implementation.
