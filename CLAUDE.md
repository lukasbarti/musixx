# musixx

Local-only web app for playing and managing a personal music library. Go HTTP server,
Templ-rendered HTML, SQLite metadata, Opus files on disk, yt-dlp for acquisition.

Project charter (goals, constraints, LLM guardrails): `GOALS_AND_RESTRICTIONS.md`.
Read it — notably: **always ask before adding or upgrading a dependency**, and
**don't remove existing features unless told to**.

## Commands

Everything goes through `just` — add a recipe rather than running ad-hoc commands.

| Command | What |
|---|---|
| `just build` | Bundle JS/CSS/Web Awesome, then `go build ./...` |
| `just generate` | Same bundling + `templ generate` (run after editing `.templ`) |
| `just run` | Live-reload dev server via air (port 8080) |
| `just test` | `gotestsum` |
| `just fmt` | `gofmt` over `cmd internal main.go` |

`templ`, `air`, and `gotestsum` are Go tool dependencies (`go tool ...` in `go.mod`) —
no separate install.

There is no `just` recipe for TypeScript typechecking; esbuild strips types without
checking them. Run `npx tsc --noEmit` after touching `web/assets/src/`.

`build-platform` / `dist` are PowerShell-only and fail on Linux. `windows-shell` is set,
so all other recipes use `sh` on Unix.

## Architecture

- `cmd/` — Cobra CLI. Only `serve` exists.
- `internal/app` — wires config → sqlite store → downloader → server.
- `internal/config` — Viper. Keys `http.port`, `library.music_dir`, `storage.database_path`,
  `tools.ytdlp_path`; env prefix `MUSIXX_` (e.g. `MUSIXX_STORAGE_DATABASE_PATH`).
- `internal/library` — domain entities + repository interfaces.
- `internal/storage/sqlite` — repositories; migrations in `internal/storage/sqlite/migrations/*.sql`,
  `go:embed`-ed and applied on open, tracked in a `schema_migrations` table. Driver is
  `ncruces/go-sqlite3` (WASM, so `CGO_ENABLED=0` builds work).
- `internal/server` — `net/http` mux, server-rendered pages, plain form POSTs. No JSON API.
- `web/components`, `web/ui` — Templ sources. `*_templ.go` are generated; never edit them.
- `web/assets/src` — TypeScript + CSS sources.

Assets are served from disk (`http.Dir("web/assets")`), not embedded — the binary needs
`web/assets/` beside it at runtime. The Dockerfile copies it explicitly.

> `docs/architecture.md` is aspirational and largely wrong: it describes `internal/media`,
> `internal/metadata`, `internal/playback`, `internal/workers`, `internal/server/views`,
> top-level `migrations/` and `assets/`, and `download`/`migrate`/`config` commands — none
> of which exist. Trust the tree, not that file.

## Frontend

**Web Awesome 3** (`@awesome.me/webawesome`) provides the UI as custom elements
(`<wa-button>`, `<wa-input>`, `<wa-select>`, `<wa-textarea>`, `<wa-slider>`, `<wa-callout>`,
`<wa-card>`, `<wa-icon>`). Components are cherry-picked in `web/assets/src/app.ts` — importing
a new `<wa-*>` tag into a template **also requires adding its import there**, or the element
silently stays undefined.

Theme classes live on `<html>` in `web/components/layout.templ`:
`wa-theme-default wa-palette-default wa-brand-blue wa-cloak`. `wa-cloak` is pure CSS
(`:has(:not(:defined))`) and hides custom elements until they register.

Style with WA design tokens (`--wa-color-*`, `--wa-space-*`, `--wa-font-*`) in
`web/assets/src/app.css` — no hardcoded hex values.

### Frontend landmines

- **`web/assets/src/wa-setup.ts` must stay the first import in `app.ts`.** `setBasePath()` /
  `setIconPath()` must run before any component module evaluates, and ES imports hoist above
  module bodies — hence the separate file.
- **Icons are self-hosted and enumerated.** `wa-icon` fetches from the Font Awesome CDN by
  default, which would break the local-only requirement. Only the icons in the `wa_icons`
  list in the `justfile` are vendored into `web/assets/wa/icons/solid/`. Using any other
  icon name renders nothing until you add it to that list.
- **Never assign `textContent` to a `<wa-button>`** — it destroys slotted `<wa-icon>`
  children. See `setToggleText()` in `player/controller.ts`, which updates a label span and
  the icon's `name` separately.
- **`<wa-slider>` values are numbers, not strings** (unlike `<input type="range">`).
- **The queue panel is deliberately not `<wa-drawer>`.** `wa-drawer` is always modal (native
  `<dialog>` + backdrop + focus trap); the panel is non-modal, pushes content, and persists
  its open state. It stays hand-rolled in `player/panel.ts` + `app.css`.
- **`.queue-panel__drag-handle` is load-bearing** — SortableJS is configured with it as
  `handle`, and renaming it breaks reordering silently.
- **`data-action="play-track"` / `"queue-track"` and the `data-track-*` attributes are the
  contract** between templates and `player/controller.ts` (see `constants.ts`). Emit them via
  the shared `TrackControls` templ in `web/ui/tracks.templ`.
- **`tsconfig.json` sets `skipLibCheck: true`** because Web Awesome's own `slider.d.ts` has a
  variance error. Don't remove it expecting a clean typecheck.
- **Turbo keeps the player alive across navigation** (`data-turbo-permanent` on `#player-bar`
  and `#queue-panel`). `preventTurboFouce()` in `app.ts` hooks `turbo:before-render` to stop
  components flashing unstyled; `controller.ts` hooks the same event to carry the
  `queue-open` body class onto the new body. Anything causing a full page reload stops
  playback.

## Conventions

- Templ files use 2-space indentation.
- Wrap outbound errors with context: `fmt.Errorf("do thing: %w", err)`.
- Pages are server-rendered on plain form POSTs; WA form controls are form-associated
  (`ElementInternals` + `setFormValue`), so `name`/`value` reach Go handlers unchanged.
- Form repopulation after a validation error uses the `value` attribute (maps to
  `defaultValue` on WA inputs), including `<wa-textarea value="...">` — not child text.
