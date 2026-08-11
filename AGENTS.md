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
- Remote whitelist rules never delete anything: the rules system is
  protection-only (see Safety boundary).

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
- Hidden state is metadata, never junk: the scanner walks hidden files and
  folders (real Windows attributes on Windows, dot-prefixed names elsewhere),
  and hidden content passes through exactly the same classifier, protection
  and candidate-validation rules as visible content. Hidden never implies
  deletable.
- Privilege elevation never happens silently; the app reports elevation status
  (`internal/platform/info_windows.go`) and the UI surfaces it. The only
  elevation path is an explicit user action (`RelaunchElevated`, UAC "runas"
  prompt) that relaunches the app and closes the old instance; a cancelled
  UAC prompt leaves the current instance running.
- The remote whitelist (RULES) is a WHITELIST/PROTECTION system only:
  - Remote data may ADD protection; it must never add deletion authority
    (the rule schema has no field capable of expressing it — `protection`
    only accepts `deny-delete`).
  - Hard-coded core safety always wins; a missing/malicious/corrupted remote
    never disables it.
  - The bundled whitelist (`rules/*.json`, embedded via the root `rules`
    package) keeps the app fully safe offline. Remote updates come from the
    SAME repository's `rules/` path on GitHub
    (`raw.githubusercontent.com/kof-huskai/Junk-Fuck/main/rules`) — no
    separate repository exists. Version convention: bump `rulesVersion` in
    `rules/manifest.json` whenever the bundled rules change (the app
    compares explicit rules versions, not git SHAs). Because `main` is the
    live remote source for every released client, merge rule changes
    conservatively (whitelist rules can only add protection, but
    over-broad rules can disable legitimate cleaning). Keep
    `minimumAppVersion` in `rules/manifest.json` consistent with the app
    version in `build/config.yml` so older clients are not silently
    blocked from rules updates.
  - Rule updates are validated (schema, app-version compatibility, sha256,
    rule structure, no wildcards/traversal/bare roots) and swapped in
    atomically; a failed update keeps the last valid ruleset.
  - Rule updates are distinct from app updates: they are never application
    binary updates.

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
  Hidden files and folders are discovered too (Windows hidden attributes and
  dot-prefixed names are both walked); hidden state is metadata, never junk.
- `internal/classifier` — rules that decide whether a path is junk and its
  category.
- `internal/protection` — protected roots and skip rules; supports an
  additive `DynamicProtector` (wired to the whitelist engine).
- `rules/` — the bundled whitelist DATA (manifest + category JSON), embedded
  into the binary via the root `rules` package; also the remote update path.
- `internal/whitelist` — the protection-whitelist engine: versioned
  manifest, declarative rules, validation, embedded-bundle loading, validated
  cache + remote updater (same-repo raw GitHub, TTL 24h, atomic cache write,
  manifest-first version gating — a remote version that is not newer
  downloads nothing).
- `internal/cleaner` — validated deletion against a scan session snapshot.
- `internal/filesystem` — portable FS helpers (compare keys, read-only attr).
- `internal/model` — shared data types (dependency-free package).
- `internal/platform` — Windows info + drive enumeration (`ListDrives`), with
  `_windows.go` / `_other.go` build-tagged files.
- `internal/report` — cleanup report model.
- `services/` — Wails services: scanner (scan control, drive list, validation),
  cleaner, settings (app info/version, elevated relaunch), updater, rules
  (whitelist status + refresh; background TTL check wired in `main.go`).
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
- JetBrains Mono is the single application UI font (bundled offline via
  `@fontsource/jetbrains-mono`, latin subsets for weights 400/500/600/700);
  Persian switches to bundled Vazirmatn via `html[lang="fa"]`.
- Dark theme with centralized tokens in `frontend/src/index.css`; primary
  accent `#66c0f4` with derived tokens — never hardcode accent hexes in
  components.
- Compact desktop-utility visual language: restrained radii (cards 10px,
  controls 6px, small 4px), minimal pills/badges.
- Results category boxes share one equal-track grid geometry.
- Windows-11-inspired Sidebar: full-height, modest right-corner curve, a
  low-opacity soft shadow for separation, and a compact updater-status row at
  the bottom that consumes the existing update service state (download/install
  UI lives only in Settings → Updates).
- Collapsible Sidebar: expanded `w-56` / collapsed `w-14` icon rail, hover
  temporary expansion (only while user-pinned collapsed), and a persisted
  pin preference (`jf.sidebarCollapsed`). Pure state model in
  `frontend/src/lib/sidebar.ts` (unit-tested); hover handlers live on the
  aside so the left-anchored width transition never flickers. Layout-driven:
  main content shrinks as the sidebar grows — never an overlay.
- Startup update check: exactly one background check per launch (store mount
  effect, ref-guarded against StrictMode double-invocation) via the existing
  `UpdateService.CheckForUpdates`; no modal, no auto-install; Sidebar and
  Settings share the same `store.updateState`/`updateChecking`.
- Sidebar motion is one coordinated layout transition: the app shell is a
  grid (`grid-template-columns: var(--sidebar-width) minmax(0,1fr)` animated
  180ms `cubic-bezier(0.2,0,0,1)`); main content moves at exactly the same
  pace as the sidebar (never an overlay). Nav labels stay mounted and fade
  (opacity + small translate) while the shrinking column clips them; the icon
  column uses constant padding so icons stay optically fixed. Reduced-motion
  is honored globally.
- Scanner admin hint: one quiet hint per scan, shown only when the backend
  classifies a scan error as an access/permission denial
  (`ScanError.Permission`) and the app is not already elevated; "Run as
  administrator" is an inline link that triggers `RelaunchElevated`.
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
   goroutine emitting `scan:progress` events. Unreadable paths are recorded as
   `ScanError`s; access/permission denials carry `Permission: true` so the UI
   can offer its one-time admin hint (the error count stays accurate).
3. Completion: the backend emits `scan:done` with `scanId`, `cancelled` and
   `error`; the frontend store persists final candidates/errors, then signals
   completion.
4. Results: the shell auto-navigates to Results **only** on successful
   completion (including zero-result scans). Cancelled or failed scans stay on
   the Scanner page.
5. Last-scan summary: on a real successful terminal state the backend records
   the canonical `model.ScanSummary` (RFC3339 UTC timestamp, target, item
   count, reclaimable bytes, error count), persisted atomically to
   `%LOCALAPPDATA%\JunkFuck\last-scan.json` and reloaded at startup. It is
   the single source of truth for the Dashboard; cancelled/failed scans never
   overwrite it. Never build a second Dashboard-side "last scan" state.

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
- Whitelist cache lives in the per-user cache dir (`%LOCALAPPDATA%\JunkFuck\rules`)
  — a validated, versioned copy of the remote ruleset; never commit it.

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
