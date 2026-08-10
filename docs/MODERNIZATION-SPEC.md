# Junk-Fuck Desktop Modernization

**Status:** In progress (branch `feat/wails-desktop`)
**Last updated:** 2026-08-11
**Version:** 4.0.0 (first desktop release)

## Context

Junk-Fuck is currently a Python 3.8+ command-line tool that deep-scans the
`C:\` drive for junk files, classifies them, and deletes them one-by-one after
interactive confirmation. It has built-in protection for system paths and
known applications, a colored console UI, and a CI/CD pipeline that builds
Windows EXEs (x64/x86/arm64) via PyInstaller and publishes releases plus a
Telegram notification.

The goal of this migration is to replace the Python CLI with a modern,
lightweight **Windows desktop application** built with:

- **Go** for the backend/core (all filesystem-sensitive logic)
- **Wails v2** for the desktop application layer
- **React + TypeScript + Tailwind CSS** for the frontend
- **GitHub Actions** for CI/build/release
- **Telegram Bot API** for release notifications/uploads

The final application must **not require Python** on the user's machine.

References (architecture inspiration only, never copied):

- `WhiteVPN-Desktop` — Wails project organization reference.
- `AI-Tricks-101` — engineering workflow principles (role, dictionary,
  guardrails, spec-driven development, measurement criteria, iterative
  implementation).

## Current Architecture

```
junkfuck.py            single-file Python CLI (~650 LOC)
  ├── ConsoleUI        centered colored output, spinner, tables
  ├── Spinner          async spinner thread
  ├── JunkCleaner
  │   ├── PROTECTED_APPS   name-based app protection
  │   ├── PROTECTED_PATHS  hardcoded Windows system paths
  │   ├── JUNK_EXTENSIONS  extension-based junk rules
  │   ├── JUNK_FOLDERS     folder-name-based junk rules
  │   ├── _is_protected()  path containment checks
  │   ├── scan_drive()     os.walk over C:\
  │   ├── print_log_table()
  │   ├── confirm_and_delete()  per-item y/n confirmation
  │   └── delete_item()
tests/                pytest suite (93 tests)
.github/workflows/    ci.yml, release.yml, build-exe.yml (PyInstaller)
```

Known weaknesses being fixed by this migration:

1. Deletion safety is a runtime convention, not an enforced contract — any
   caller of `delete_item` can delete anything not protected.
2. `_is_protected` is lexical-only; symlinks/junctions are not resolved.
3. Protection uses substring name matching (`if app in name`) which can
   over-protect but also misses case/encoding variations.
4. A junk **directory** is never checked for whether it *contains* a
   protected path before deletion.
5. Deletion has no revalidation against the scan snapshot (TOCTOU).
6. The whole scan runs synchronously; the UI freezes for large drives.
7. Keyword matching for files (`'old'`, `'copy'`, `'~'`) is dangerously broad.

## Target Architecture

```
/
├── app.go                    Wails App (Startup/Shutdown, bound methods)
├── main.go                   Wails entrypoint
├── go.mod / go.sum
├── wails.json
├── internal/
│   ├── classifier/           junk rules + categories
│   ├── cleaner/              validated deletion + dry-run
│   ├── filesystem/           walk, size, canonicalize, symlink detection
│   ├── platform/             Windows-specific info (elevation, OS)
│   ├── protection/           protected paths + apps
│   ├── report/               cleanup report model
│   └── scanner/              async scan + progress + cancellation
├── frontend/
│   ├── src/
│   │   ├── components/       UI components
│   │   ├── i18n/             en/fa dictionaries + RTL support
│   │   ├── lib/              wails bindings, formatting, store
│   │   ├── pages/            Dashboard, Scanner, Results, History, Settings, About
│   │   ├── types/            shared types
│   │   └── App.tsx
│   ├── package.json
│   └── ...
├── legacy/python/            original Python implementation (reference only)
├── tests/                    (Go tests live beside their packages)
├── docs/MODERNIZATION-SPEC.md
└── .github/workflows/
```

## Dictionary

| Term | Meaning |
| --- | --- |
| **Junk Candidate** | A file or directory that matches an explicitly defined, tested junk rule. |
| **Protected Path** | A filesystem path the backend must refuse to delete regardless of frontend behavior. |
| **Scan** | A read-only filesystem operation that discovers potential junk. |
| **Clean** | Deletion of explicitly selected junk candidates after confirmation. |
| **Dry Run** | The entire cleanup decision process run without modifying the filesystem. |
| **Core** | The Go packages containing scanning, classification, protection, validation, deletion and reporting logic. Must not depend on React/Wails. |
| **Desktop Layer** | The Wails-specific layer exposing controlled backend methods to the frontend. |
| **Release Build** | A production executable generated from a version tag. |
| **Scan Session** | An immutable snapshot of candidates produced by one scan; deletions are only allowed against it. |

## Goals

- Replace Python with a Go core + Wails desktop app (no Python runtime needed).
- Enforce safety in the Go backend, with automated safety tests.
- Responsive UI: async scanning, live progress, cancellation.
- Explicit user confirmation before any deletion; dry-run support.
- Tag-driven releases with GitHub Releases + Telegram notification.
- Simpler CI than today (no duplicated full builds across workflows).

## Non-Goals

- Port every cosmetic behavior of the Python UI (colors, ASCII banner).
- Replicate dangerous broad matching rules (`old`, `copy` keywords).
- Auto-delete anything, ever.
- Multi-arch (x86/arm64) release builds in v4.0.0 — amd64 only, workflow is
  structured so extra architectures can be added later.
- Code signing (requires certificates).
- Admin-elevation requests (explicitly out of scope; we explain in the UI).

## Functional Requirements

- FR-1: The app must not auto-scan on startup; scans are user-initiated.
- FR-2: Scans run asynchronously, expose progress, support cancellation, and
  continue past access-denied directories (recording errors).
- FR-3: Scan targets are user-selectable; the default target is `C:\`.
- FR-4: Candidates are classified into structured categories with a human
  readable reason for classification.
- FR-5: Results are shown in a sortable/filterable table with selection.
- FR-6: Cleanup requires selection + a final confirmation summary.
- FR-7: Dry-run must be available and must never modify the filesystem.
- FR-8: The app must work without Administrator rights; if elevation would
  allow more scanning, explain it in the UI without forcing it.
- FR-9: English and Persian localization with RTL layout support.

## Safety Requirements

SR-1: Never delete protected paths (enforced in Go).
SR-2: Never delete outside the candidate set of the current scan session.
SR-3: Never delete a candidate whose identity changed since the scan
      (re-stat + re-canonicalize + re-protection-check before deletion).
SR-4: Require explicit user selection.
SR-5: Require a final confirmation before destructive cleanup.
SR-6: Support dry-run.
SR-7: Never silently escalate privileges.
SR-8: Never bypass Windows filesystem protections.
SR-9: Never recursively delete an arbitrary directory supplied directly by
      frontend input — only scan-session candidates may be deleted.
SR-10: Canonicalize and validate paths before deletion.
SR-11: Refuse to delete symbolic links / reparse points as directories.
SR-12: Fail safely — report failures instead of forcing deletion.
SR-13: Never delete a directory that is an ancestor of a protected path.

## Backend Architecture

- `internal/classifier` — pure rule engine. Input: path + is-dir. Output:
  category + reason. Rules are an explicit table (extensions and folder
  names); no broad substring matching. Never touches the filesystem except
  through injected `os.Lstat` for symlink awareness (testable).
- `internal/scanner` — walks targets read-only, collects candidates,
  accumulates directory sizes during descent, honors a `context.Context` for
  cancellation, reports progress via a callback, records per-path errors.
- `internal/protection` — builds protected path set from explicit Windows
  paths + environment probe (injectable for tests). Containment checks are
  canonical and case-insensitive; also resolves symlinks where possible.
- `internal/cleaner` — validates a selection against a scan-session snapshot,
  re-canonicalizes, re-checks protection + symlink + ancestor rules, then
  deletes; returns a structured report. Dry-run performs validation only.
- `internal/filesystem` — size helpers, canonicalization, reparse-point
  detection (Windows) / symlink detection.
- `internal/report` — cleanup result model (deleted/skipped/failed/bytes).
- `internal/platform` — OS version, admin detection (Windows-specific, safe
  no-op elsewhere).

All filesystem-sensitive operations go through the Core. The Desktop Layer
(`app.go`) only wires Core to Wails events/bindings.

## Frontend Architecture

- Vite + React + TypeScript + Tailwind CSS v4 (CSS-first config, dark-first).
- Hand-rolled shadcn-style components (button, card, checkbox, input,
  dialog) implemented with Tailwind — no radix/shadcn dependency bloat.
- Localization: small dictionary-based i18n layer (`i18n/`) with `en` and
  `fa` dictionaries and `dir="rtl"` when Persian is selected.
- State: React Context + hooks (no state library dependency).
- Backend calls: Wails-generated bindings (`frontend/wailsjs/...`), committed
  to the repository; events via `@wailsio/runtime`.
- Pages: Dashboard, Scanner, Results, History, Settings, About — only pages
  with real functionality.

## Wails API Contract

| Method | Input | Output | Notes |
| --- | --- | --- | --- |
| `StartScan(targets []string)` | scan targets | `scanID string` | async; emits `scan:progress` + `scan:done` events |
| `CancelScan(scanID)` | scan id | — | cancels the scan context |
| `GetScanState(scanID)` | scan id | progress snapshot | polling fallback |
| `GetCandidates(scanID)` | scan id | `[]Candidate` | snapshot from session |
| `Cleanup(dryRun, scanID, selected)` | dry-run flag, session id, selected paths | `Report` | validates against session |
| `GetProtectedPaths()` | — | `[]string` | informational |
| `GetSystemInfo()` | — | OS version, admin bool | informational |
| `GetAppInfo()` | — | name, version | from build metadata |

Events:

- `scan:progress` → `{ scanID, scannedFiles, candidates, errors, currentPath }`
- `scan:done` → `{ scanID, ok, error? }`

## Scan Algorithm

1. Resolve and canonicalize each target.
2. Walk with `filepath.WalkDir` (top-down), checking `context.Context` at
   every level for cancellation.
3. Prune protected directories from descent.
4. For each file: if it matches a junk extension rule → candidate
   (size = file size). If its parent is already a junk-directory candidate,
   it is not separately added (parent covers it).
5. For each directory: if it matches a junk folder-name rule → candidate and
   keep descending to accumulate its total size; flag its children as
   covered.
6. Access-denied / walk errors are recorded as `ScanError` and scanning
   continues.
7. Progress callback fires periodically (time- and count-based).

## Deletion Algorithm

1. `Cleanup` receives a session id + selected canonical paths.
2. Look up the scan session; reject if missing (stale/unknown session).
3. For each selected path:
   a. Must exist in the session's candidate set.
   b. Re-canonicalize (`filepath.Clean` + absolute).
   c. Re-check protection (including ancestors of protected paths).
   d. Refuse symlink/reparse-point deletions.
   e. Re-stat: file must still be a file; dir must still be a dir.
   f. Never delete drive roots.
4. If dry-run: report would-delete results; touch nothing.
5. Otherwise delete (files with `os.Remove`, dirs with `os.RemoveAll` after
   making contents writable where needed). Record per-item result.
6. Return structured report.

## Protected Path Rules

- Explicit Windows system paths (`C:\Windows`, `C:\Program Files`, ...).
- Environment-derived paths: `SYSTEMROOT`, `WINDIR`, `ProgramFiles`,
  `ProgramFiles(x86)`, `ProgramData`, `APPDATA`, `LOCALAPPDATA`,
  `USERPROFILE`.
- Any drive root (`X:\`).
- Known application directories under the user profile (Discord, browsers,
  IDEs, Steam, Spotify, Telegram, WhatsApp, Docker, OBS, Razer/Logitech...).
- Path containment: candidate is protected if it equals or is under a
  protected path; a candidate directory that *contains* a protected path is
  also refused (SR-13).

## State Model

```
AppState (frontend)
├── language
├── settings (scan targets, last used)
└── scanState
    ├── currentScanId
    ├── progress
    ├── candidates
    ├── selection
    └── lastReport (history)
```

Backend keeps an in-memory map of scan sessions (id → candidates + status).
Sessions expire after cleanup or when replaced by a new scan.

## UX Flow

1. Dashboard → click "Start Scan" (default target `C:\`, configurable).
2. Scanner page → live progress, current path, counters, Cancel button.
3. Results page → selectable table (Select All / None, category filter,
   search, sort). Protected candidates visible but not selectable.
4. Cleanup dialog → summary (count, size, categories, permanent-delete
   warning) → explicit confirm.
5. History page → last report (deleted/skipped/failed/bytes).
6. Settings → language, scan targets, dry-run toggle.
7. About → version, safety notes, link to repository.

## Error Handling

- Scan path errors → collected, scan continues, surfaced in results footer.
- Telegram failures → isolated with `continue-on-error`, never invalidate the
  GitHub Release.
- Deletion failures → reported per item; never abort the whole cleanup.
- Frontend/backend communication errors → surfaced as user-visible toasts
  with actionable text.

## Testing Strategy

Go tests (no real drive access; temporary directories only):

- classification by extension / directory / safe files not classified
- protected path rejection (incl. env-probe injection)
- path normalization & case-insensitivity
- parent/child protected path behavior
- dry-run never deletes
- selected files deleted inside a temp fixture; unselected remain
- directory cleanup behavior
- deletion outside the scan candidate set rejected
- symlink/reparse-point refusal
- cancellation of long scans (practical subset)
- locked/unavailable path handling (practical subset)

Frontend: type-checked build (`tsc --noEmit`) and production build.

## CI/CD Strategy

| Trigger | Workflow | Checks |
| --- | --- | --- |
| push / PR to main | `ci.yml` | gofmt, go vet, go test, frontend `tsc` + `vite build` |
| manual | `build-exe.yml` (workflow_dispatch) | one amd64 Wails build, uploaded as artifact |
| tag `v*` | `release.yml` | tests → Wails build → checksums → GitHub Release → Telegram |

No workflow performs a full production build twice. Production build happens
only in `release.yml`.

## Release Strategy

- Semantic tags: `v4.0.0` for the first desktop release.
- On tag: tests must pass, production build must succeed, checksums
  generated, GitHub Release created, artifacts attached, then Telegram
  notified.
- Minimum token permissions (`contents: write` only where needed).
- Release is never published if tests or build fail.

## Telegram Release Integration

- Direct Telegram Bot API via `curl` on the runner; no third-party action.
- Secrets: `TELEGRAM_BOT_TOKEN`, `TELEGRAM_CHAT_ID` (never logged; skipped
  with a warning when unset).
- Uses `--form-string` for `chat_id` so `@channelname` values are sent
  literally (curl `-F` would misread `@` as a file — exit 26).
- HTML `parse_mode` for clean rich messages.
- `continue-on-error` on the notification step: Telegram failure must not
  invalidate the successful GitHub Release.
- The release EXE is uploaded as a document when small enough (< 50 MB).

## Measurement Criteria

- [ ] `go build ./...` succeeds (or `go test` passes for all packages)
- [ ] Safety tests pass (`go test ./internal/...`)
- [ ] Frontend production build succeeds
- [ ] Production operation does not require Python
- [ ] Scanning is read-only (test-verified)
- [ ] Cleanup requires explicit user action (backend-enforced)
- [ ] Protected paths enforced by backend code (test-verified)
- [ ] Cancellation implemented in the architecture (tested where practical)
- [ ] CI is simpler than the previous pipeline (3 workflows, no duplicated builds)
- [ ] Production tags can create GitHub Releases (workflow in place)
- [ ] Telegram integration requires only GitHub Secrets
- [ ] READMEs accurately describe the new architecture
- [ ] No secrets committed
- [ ] No unrelated architecture introduced

## Expected Behavior

- Start scan → read-only scan → classified candidates → protected rejected →
  results shown → selection → confirmation → revalidated deletion → report.
- `git tag v4.0.0` → tests → build → GitHub Release → Telegram.
- Telegram down → Release still valid; Telegram step reports failure only.
- Permission denied on one directory → recorded; scan continues.
- Frontend requests deletion of a path outside the scan → backend rejects.

## Wrong Behavior

- One giant Go file; filesystem logic in React; trusting raw frontend paths.
- Auto-cleaning after scan; deleting without explicit confirmation.
- Testing against the real `C:\`; disabling safety checks to pass tests.
- Requiring Administrator to launch; dozens of unnecessary abstractions.
- Copying WhiteVPN source/branding; keeping Python as hidden runtime.
- Three workflows doing the same full build; hardcoding Telegram secrets.
- Telegram failure destroying an already-created Release.
- Stopping after writing this specification.

## Guardrails

- Safety > visual polish.
- Correctness > number of features.
- Simple architecture > clever architecture.
- Backend-enforced rules > frontend-only validation.
- Tests > assumptions.
- Inspect existing behavior before replacing it.
- Never silently omit an existing safety feature.
- Never claim a command passed unless it was actually executed.

## Migration Plan

1. ✅ Repository & environment inspection
2. ✅ Specification
3. Go module + Core packages (classifier, protection, filesystem, scanner,
   cleaner, report, platform)
4. Core tests + `gofmt`/`go vet`/`go test` green
5. Wails layer (`main.go`, `app.go`, `wails.json`) + generated bindings
6. Frontend (Vite + React + TS + Tailwind, i18n, pages)
7. Frontend build verification
8. GitHub Actions rewrite (ci, release, build-exe)
9. Legacy Python moved to `legacy/python/`
10. Documentation (README, README.fa, CONTRIBUTING)
11. Full verification loop; review; fix
12. Push branch; open PR; user reviews and merges

## Decisions Log

- **Desktop version:** 4.0.0 — major architectural rewrite (Python CLI → Go
  desktop). First desktop release tag: `v4.0.0`.
- **Release architectures:** amd64 only for v4.0.0 (Wails + WebView2 arm64/
  x86 toolchains add CI fragility; workflow is structured to add them later).
- **shadcn/ui:** adopted as *style*, implemented with hand-rolled Tailwind
  components — no radix dependency tree.
- **Legacy Python:** preserved under `legacy/python/` for behavioral
  reference; not part of the production build. Removal decision documented at
  the end of the migration.
- **Broad keyword matching dropped:** `old`, `copy`, `~` substring rules from
  the Python version are not ported (dangerous). Rules are explicit and
  tested.
