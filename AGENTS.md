# Junk-Fuck development guide

Guide for working on the Junk-Fuck repository. It describes the repository as
it actually is — read the code before trusting any summary here.

## Project overview

Junk-Fuck is a Windows junk scanner/cleaner with a native desktop UI:

- **Go core** — scanning, classification, protection and cleanup logic under
  `internal/` (no Wails dependency).
- **Wails v3 desktop layer** — `main.go` creates the app, window and services;
  services live in `services/`.
- **React + TypeScript frontend** — `frontend/` (Vite + Tailwind CSS v4).
- **Windows-first** — Win32 APIs via `golang.org/x/sys/windows`; other
  platforms get stubs so the tree still builds.
- **Delivery** — GitHub Actions workflows under `.github/workflows/`; releases
  are tag-driven and distributed through GitHub Releases (+ optional Telegram).

What it does:

- Scans user-selected Windows drives/folders for junk (temporary files, caches,
  logs, crash dumps, backups, partial downloads, build artifacts, editor temp,
  other junk), read-only.
- Lets the user select candidates and delete them with explicit confirmation.
- Provides history of cleanup runs, settings, and a Wails v3 updater wired to
  GitHub Releases.

What it does NOT do:

- No arbitrary `delete(path)` API exposed to the frontend — deletion only ever
  happens against the candidate set produced by the current scan session.
- No silent privilege elevation, no registry/system-service changes.
- No background/continuous scanning or polling; scans are explicit user actions.

## Working agreements

- Be concise; clearly distinguish verified behavior from assumptions.
- Finish authorized work end to end, including verification.
- Never claim a build or test passed without actually executing it.
- Keep changes focused on the requested task; do not refactor unrelated code.
- Preserve unrelated user work (e.g. `build/windows/icon.ico` is the user's
  uncommitted canonical artwork — do not regenerate or replace it).
- Do not push, publish releases, delete remote resources or perform other
  destructive remote operations unless explicitly authorized. Never publish a
  release automatically at the end of an implementation task.
- Treat filesystem deletion code as safety-critical (see Safety boundary).
- Wails v3 is a beta framework — verify API contracts against the vendored
  module (`go doc` / module cache) before relying on new options.
- Never expose credentials or machine-specific absolute paths in code, docs or
  examples.

## Safety boundary

Project-specific invariants (paths are real):

- Scanning is read-only: `internal/scanner/scanner.go` only walks and
  classifies; it never deletes.
- Cleanup is a separate, explicit operation: `internal/cleaner/cleaner.go`.
- The frontend is never trusted with paths: `services/scanner_service.go`
  validates every scan target against the live backend drive list before a
  scan starts (`validateTarget`).
- Deletion requires scanned candidates: `internal/cleaner` operates only on
  the candidate snapshot produced by the active scan session
  (`internal/model.ScanSession`), never on arbitrary frontend paths.
- Protected paths are backend-enforced: `internal/protection/protection.go`
  lists protected roots; the scanner skips them and the cleaner refuses them
  (`protected` flag on candidates).
- Selected candidates are revalidated against the session before deletion
  (safety SR-2 in the cleaner).
- Dry-run must remain safe: a dry run never touches disk.
- Tests must never delete real user files — test fixtures live under `t.TempDir()`.
- Windows system protections (protected roots) must not be bypassed.
- Privilege elevation never happens silently; the app reports elevation status
  (`internal/platform/info_windows.go`) and the UI surfaces it.

## Architecture

Trust boundary:

```
React/TypeScript frontend
        │  Wails service bindings (frontend/bindings/, generated)
        ▼
Wails v3 services (services/) — thin, stateful API layer
        ▼
Go core (internal/) — pure Go, no Wails imports
        ▼
filesystem / Win32 APIs (golang.org/x/sys/windows, os)
```

Modules:

- `internal/scanner` — filesystem walk, progress, candidate collection.
- `internal/classifier` — rules that decide whether a path is junk and its
  category.
- `internal/protection` — protected roots and skip rules.
- `internal/cleaner` — validated deletion against a scan session snapshot.
- `internal/filesystem` — portable FS helpers (compare keys, read-only attr).
- `internal/model` — shared data types (dependency-free package).
- `internal/platform` — Windows info + drive enumeration (`ListDrives`), with
  `_windows.go` / `_other.go` build-tagged files.
- `internal/report` — cleanup report model.
- `services/` — Wails services: scanner (scan control, drive list, validation),
  cleaner, settings (app info/version), updater.
- `main.go` — window creation, theming, fixed-size monitor-aware sizing,
  updater registration.
- `frontend/` — React app: `src/App.tsx` shell, `src/lib/store.tsx` shared
  state (drives, selected root, scan lifecycle), `src/pages/*` pages,
  `src/components/ui.tsx` primitives, `src/i18n/*` en/fa dictionaries.

The pure Go core stays independent of Wails (no `pkg/application` imports in
`internal/`); services translate between the core and Wails.

## UI and layout

- Fixed user-resize behavior: the window cannot be resized or maximized by the
  user; the application sizes it to the current monitor's logical WorkArea
  (preferred 1120×760, ~92% fallback) and adapts when moved to a smaller screen
  (`main.go`).
- Full-height sidebar: `html, body, #root { height: 100% }` chain; the app
  shell never scrolls (`overflow: hidden`); scrolling belongs to the page
  container and per-page scroll regions (e.g. the Results list).
- Geist typography (bundled via `@fontsource`); Persian uses bundled Vazirmatn
  via `html[lang="fa"]`.
- Dark theme with centralized tokens in `frontend/src/index.css`; primary
  accent `#66c0f4` with derived tokens — never hardcode accent hexes in
  components.
- Compact desktop-utility visual language: restrained radii (cards 10px,
  controls 6px, small 4px), minimal pills/badges.
- Results category boxes share one equal-track grid geometry.
- Localization is en/fa with RTL switching; language selection lives in
  Settings (no floating switcher).
- UI layout must be verified in the running Wails application, not only in a
  browser preview (browser previews cannot exercise native window/DPI
  behavior).

## Scan lifecycle

1. Drive selection: the Scanner dropdown is fed by the backend
   (`ScannerService.ListDrives` → `platform.ListDrives`), ordered system/fixed
   first. The selected root is shared single state in the frontend store.
2. Scan: `StartScan` validates the target backend-side, then runs in a
   goroutine emitting `scan:progress` events.
3. Completion: the backend emits `scan:done` with `scanId`, `cancelled` and
   `error`; the frontend store persists final candidates/errors, then signals
   completion.
4. Results: the shell auto-navigates to Results **only** on successful
   completion (including zero-result scans). Cancelled or failed scans stay on
   the Scanner page.

## Build and test

Prerequisites: Go 1.25+, Node 22+, the pinned Wails v3 CLI
(`go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-beta.6`), npm.

- Dev: `wails3 dev` (runs the Vite dev server + rebuilds Go).
- Frontend install: `cd frontend && npm ci`
- Frontend checks/build: `cd frontend && npm run build` (tsc + vite).
- Go formatting: `gofmt -l .` (CI fails on unformatted files).
- Go vet: `go vet ./...`
- Go tests (incl. safety tests): `go test ./...`
- Regenerate bindings after changing Go service/model signatures:
  `wails3 generate bindings -clean=true -ts -i` (committed under
  `frontend/bindings/`).
- Production build: `wails3 build VERSION="x.y.z"` → `bin/JunkFuck.exe`
  (builds the frontend, stages assets into `embed/`, generates the syso icon
  resource, and injects the version).
- Package/installer: `wails3 package` (NSIS installer; the release pipeline
  currently ships portable EXEs instead).

When the owner requests it, a release-quality build must be manually tested
before it may be published.

## Release process

Actual flow (see `.github/workflows/release.yml`, `ci.yml`, `build-exe.yml`):

1. Development changes.
2. Tests: CI (`ci.yml`) runs frontend build, `gofmt -l`, `go vet`, `go test`.
3. Production test build (local or `build-exe.yml` workflow_dispatch).
4. User validation of the artifact.
5. Explicit user approval.
6. Tag + push (`v*`), which triggers `release.yml`: builds amd64 + arm64 EXEs
   with the version from the tag, generates `SHA256SUMS`, creates the GitHub
   Release, and (if secrets exist) posts to Telegram.

**Publishing must never happen automatically during ordinary implementation
tasks.** Only the owner's explicit approval authorizes the tag push and release.
Deleting a remote release/tag is a destructive action that also requires
explicit authorization.

## Repository hygiene

- Do not commit generated temporary binaries, caches or logs; release
  artifacts belong in `bin/` (gitignored) or as GitHub Release assets.
- `embed/` is deliberately tracked: it stages the built frontend for
  `go:embed` (gitignored `frontend/dist` cannot be embedded directly). Keep it
  regenerated by the build task; its contents are build output.
- Generated frontend bindings (`frontend/bindings/`) are tracked — regenerate
  and commit them when Go APIs change.
- Preserve licenses/notices and existing user changes.
- Keep secrets out of source; no machine-specific absolute paths.
- Avoid destructive git commands; never force-push shared branches.
- Do not accidentally commit test scan data or fixtures that reference real
  user paths.
- The canonical app artwork is `build/windows/icon.ico` (user-supplied,
  uncommitted). `build/appicon.png` is the icon-pipeline source and must stay
  aligned to that artwork so `wails3 generate icons` never replaces it.

## Design QA requirements

User-facing changes must be verified for: typography, spacing, icon rendering,
sidebar full height, Results scrolling, category card geometry,
selected/hover/focus states, DPI and monitor WorkArea behavior, fixed-window
behavior, the scan flow, dark-theme contrast, and `#66c0f4` accent
consistency. Record findings in `design-qa.md` (repository root) with real
evidence; never fabricate test results.
