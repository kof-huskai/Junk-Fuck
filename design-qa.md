# Junk-Fuck Design QA

Living QA record for the Junk-Fuck desktop app (Go + Wails v3 + React/TypeScript).

## Sources

- User-provided annotated Dashboard screenshot (external reference; lives in the
  design conversation, not saved into the repository).
- User feedback from the current implementation cycle (sidebar full-height bug,
  window too short, Results scrolling bug, Steam-blue accent, canonical icon,
  equal category boxes, icon missing from the app header/About).
- User feedback for the next version cycle: JetBrains Mono as the single UI
  font, curved Windows-11-inspired Sidebar with subtle shadow separation, a
  real updater-status row at the bottom of the Sidebar, and hidden-file/folder
  scanning verification.
- User feedback (pre-release follow-up): automatic background update check on
  startup (no modal, no auto-install) and a collapsible Sidebar with icon-only
  mode, hover temporary expansion and a persistent pin preference.
- Local QA evidence generated during this cycle (kept in the gitignored
  `.codegraph/` scratch dir, not committed):
  - `icon-256-src.png` — the 256×256 PNG frame extracted from the canonical
    `build/windows/icon.ico`.
  - `exe-icon.png` — the 32×32 icon extracted from the built `bin/JunkFuck.exe`.
  - inspection/extraction scripts used to verify ICO frames and the EXE icon.

## Visual target

Accepted design principles (values from the actual implementation):

- Compact desktop utility, dark application shell (`--color-bg: #0b0d12`).
- Primary accent `--color-accent: #66c0f4` (Steam-like) with derived tokens
  `--color-accent-hover: #8cd0f9`, `--color-accent-active: #3f9edb`,
  `--color-accent-soft: #16324b`, `--color-on-accent: #0b0d12`. No hardcoded
  accent hexes in components.
- JetBrains Mono as the single application UI font (bundled offline via
  `@fontsource/jetbrains-mono`, latin subsets 400/500/600/700); Persian falls
  back to bundled Vazirmatn via `html[lang="fa"]`.
- Radius hierarchy: cards 10px, buttons/controls 6px, small controls 4px;
  minimal pills/badges.
- Full-height sidebar, internal content scrolling only.
- Windows-11-inspired Sidebar: full height, modest 14px right-corner curve,
  low-opacity soft shadow for separation, and a compact updater-status row at
  the bottom that consumes the existing update service state (all
  download/install UI stays in Settings → Updates).
- Results category summary boxes share one equal-track grid geometry.

## Window and DPI QA

Implemented in `main.go`:

- Preferred logical size **1120 × 760**.
- Clamped to the current monitor's logical WorkArea (~92% fallback, never
  exceeding it), centered on the screen.
- User resize: **disabled** (`DisableResize`). Maximize: **disabled**
  (button state + un-maximize guard). Move/minimize/close: enabled.
- Monitor adaptation on move (debounced) shrinks the window only when it no
  longer fits the target screen.
- Native title bar themed to the app background via the Windows custom theme.

Tested in this environment:

- Production EXE launches on Windows 11 (WebView2 environment created,
  process stays alive).

Not verified in current environment (single fixed monitor, 100% DPI):

- Multi-monitor drag between differently sized screens.
- 125/150/200% DPI scaling.
- 1366×768 or 1024×768 WorkArea fallback rendering.
- Actual title-bar seam appearance on Windows 10 (custom caption colour is a
  Windows 11 feature; Win10 falls back to the standard dark title bar).

## Typography QA (JetBrains Mono)

- Bundled offline: `@fontsource/jetbrains-mono` latin subsets for weights
  400/500/600/700; the `@fontsource/geist-sans` and `@fontsource/geist-mono`
  packages were removed (package.json, built bundle and EXE — no `geist`
  strings remain anywhere in the build output).
- Computed body font-family in the built frontend: `"JetBrains Mono",
  "Segoe UI Variable", ...` — verified in a real browser.
- Metric compensation for the wider/taller mono face: body `font-size:
  13.5px`, `line-height: 1.45`, `letter-spacing: -0.01em`; buttons/inputs/
  selects reset letter-spacing. Verified no clipping in the sidebar nav,
  buttons, headings or the Results table.
- Persian: unchanged — `html[lang="fa"]` switches the whole UI to bundled
  Vazirmatn (JetBrains Mono has no Persian glyph coverage).

## Hidden scanning QA

Verified behavior (no scanner rewrite was needed — the walk already visits
hidden content):

- `internal/scanner` walks with `filepath.WalkDir`, which lists hidden files
  and folders — there is no dot-prefix or attribute filtering anywhere.
- The classifier is name/extension based and is attribute-agnostic: hidden is
  metadata, never junk.
- Protection prunes protected subtrees regardless of hidden state.

Proof tests added (all passing; run on Windows using the real
`FILE_ATTRIBUTE_HIDDEN` via `SetFileAttributes`, no-op on other platforms):

- Hidden-attribute `.tmp` file is discovered as a candidate; the scan leaves
  it byte-identical (read-only).
- Junk inside a hidden directory is discovered; the hidden directory itself
  is not junk just because it is hidden.
- Dot-prefixed hidden junk folder (`.cache`) is discovered with its contents
  counted toward the candidate size.
- Hidden normal files (dot-prefixed `.keep.me` and attribute-hidden
  `notes.txt`) are never candidates.
- A protected subtree inside a hidden folder is pruned from the scan while
  junk beside it is still discovered.
- Dry run never deletes hidden candidates.
- An arbitrary hidden path outside the scan session is refused by the cleaner.

Files: `internal/scanner/scanner_test.go`, `internal/cleaner/cleaner_test.go`.

## Sidebar QA (curve, shadow, update status)

- Full height preserved: the aside still stretches the whole window height.
- Curved geometry verified in the built frontend: computed `border-radius:
  0 14px 14px 0` (modest Windows-11-style right-corner curve; flush left
  edge, not a floating card).
- Separation: `box-shadow: 4px 0 16px -6px rgba(0,0,0,0.55)` (low opacity,
  soft blur) plus the existing 1px border hairline; painted above the main
  content via `relative z-10`.
- Update-status row at the bottom (`mt-auto`, border-t separator): consumes
  the existing update service state (`store.updateState`) — states rendered:
  initial "not checked", "up to date", "update available vX.Y.Z" (click →
  Settings), "update ready — restart to apply" (click → Settings), "check
  failed" (click → retry). No duplicate download/install UI. The label uses
  `dir="auto"` so Persian renders RTL correctly.
- Limitation (by design): the spec's "Downloading %" state is not
  representable because `UpdateService` only exposes a blocking
  `InstallUpdate` call — no download progress is emitted. The sidebar only
  consumes existing service state, so it shows nothing during the download.

Not verified in this environment:

- The live "update available" state against a newer GitHub release (requires
  an actual newer release to exist; the state rendering is code-verified).

## Startup update check QA

- Exactly one background check per launch: the store runs it once in the
  provider mount effect, guarded by a ref so React StrictMode's double
  invocation cannot fire two checks (and no page/sidebar mount triggers
  another). Sidebar and Settings read the same `store.updateState`.
- Non-blocking: the check runs after first render via the existing Wails
  binding (`UpdateService.CheckForUpdates`); the UI is interactive
  immediately. `updateChecking` starts `true` so the Sidebar shows
  "Checking…" from the first frame (no "not checked" flash).
- No startup modal / no auto-install: an up-to-date result only sets the
  shared state; an available update only lights up the Sidebar (click →
  Settings → Updates). Installation still requires explicit user action and
  is refused while a scan/cleanup is running (existing `coreActive` guard).
- Network failure: the check's `catch` pulls the backend snapshot
  (`refreshUpdate`); the app continues normally with a quiet
  "Update check failed" state. Manual retry lives in Settings.
- State mapping: idle/not-checked, checking (`updateChecking`), up-to-date,
  available, installed(ready), error — matching what `UpdateService`
  actually emits. A "downloading %" state is not representable (blocking
  `InstallUpdate` emits no progress) and is intentionally not invented.

Verified in this environment:

- Unit tests (vitest) for the pure sidebar model; frontend build; EXE
  launches. The live GitHub check itself is a manual checklist item (the
  runner here has network access, but the transition was not observed in a
  headed session this cycle).

## Collapsible Sidebar QA

- Model (pure, unit-tested in `src/lib/sidebar.ts`): one persisted
  preference `userCollapsed` + transient `hovered`;
  `effectiveCollapsed = userCollapsed && !hovered`. Pinned-expanded ignores
  hover entirely; pinned-collapsed temporarily expands on hover.
- Widths: expanded `w-56` (224px), collapsed `w-14` (56px), animated via
  `transition-[width]` 150ms. Curved geometry + shadow preserved in both
  states.
- Layout-driven: the aside is a flex sibling, so main content shrinks as it
  expands — no absolute overlay.
- Live-verified in the built frontend: collapsed 56px → click Expand →
  224px → click Collapse → 56px immediately (the pin click clears the
  transient hover, so a collapse is visible instantly even with the pointer
  over the sidebar).
- Tooltips: collapsed nav buttons get native `title` + `aria-label`;
  collapsed update status shows the state text as its tooltip (e.g.
  "Update available — v4.2.0").
- Persistence: `jf.sidebarCollapsed` in localStorage (existing settings
  store), lazy-initialized before first paint (no expanded→collapsed flash);
  the transient hover is never persisted.
- A11y: the pin control has `aria-label`/`title`/focus ring; nav icons stay
  keyboard reachable in collapsed mode.
- Anti-flicker: hover handlers live on the aside itself; the left edge is
  anchored so the pointer stays inside during the width transition.

Not verified in this environment (browser tooling was flaky this cycle):

- Live hover-expansion width measurement (mouseenter/mouseout synthetic
  events did not reliably exercise React's handlers; the state logic is
  unit-tested and the handlers/classes are present in the built bundle).
  Manual checklist item.
- Results page interaction during a live sidebar width transition with a
  large result set (manual checklist item).

## Dashboard QA

- Sidebar fills the full window height: verified in the built frontend
  (`aside` height == `window.innerHeight`).
- No subtitle under the title; Status card is plain text (no dot/badge);
  Safety card is plain text with a subtle shield icon.
- 2-column card grid; Scan Target panel shows the backend-detected drive
  (`C:\ — Local Disk` style labels).
- Start Scan is a compact rectangular primary button (accent, dark text for
  contrast).
- No floating language button; language lives in Settings.

## Scanner QA

- Drive dropdown fed by the backend (`ListDrives`), friendly labels
  (`Local Disk`, `Removable Disk`, …), refresh button, backend validation of
  the selected root before every scan.
- Progress card with status line, progress bar and stats.
- Scan → Results auto-navigation implemented for successful completion only.

Not verified in this environment:

- End-to-end scan with real drive data and the automatic Results transition
  (requires a manual scan run — see the manual test checklist).

## Results QA

- Scroll ownership fixed: the app shell never scrolls; the page container
  scrolls, and the Results list has its own internal scroll region with the
  header/toolbar kept visible (`min-h-0` chain).
- Category summary boxes use an equal-track grid (`grid-cols-2 md:grid-cols-4`,
  i.e. `minmax(0,1fr)`), so every box has identical width/height/padding/radius
  regardless of label length; clicking a box filters the list.
- Empty state differentiates "no scan yet" from "scan completed with zero junk".

Not verified in this environment:

- Wheel/scrollbar/keyboard behaviour with a large real result set (requires a
  completed scan — manual checklist item).

## Icon QA

- Canonical artwork: `build/windows/icon.ico` (user-supplied; an uncommitted
  working-tree change — treated as sacred, never regenerated).
- ICO contents inspected: **9 frames, all 32bpp** — 16, 24, 32, 48, 64, 72,
  96, 128 (BMP-DIB) and 256 (PNG). All required sizes present (16/24/32/48/256),
  so no regeneration was needed.
- One canonical artwork source: the 256×256 PNG frame extracted from the ICO
  (`.codegraph/icon-256-src.png`, scratch). Every derivative is byte-identical
  or pixel-derived from it — fingerprints (avg RGB) verified identical
  (3,8,12) at every level:
  - `build/appicon.png` — re-aligned to the canonical artwork (it previously
    held the Wails gray template, avg RGB 192,192,192, which would have
    overwritten the canonical ICO if `wails3 generate icons` ran).
  - `embed/icon-256.png` + `embed/icon-32.png` — staged by the Wails build
    from `appicon.png` (verified canonical, not the Wails "W").
  - `frontend/src/assets/app-icon.png` — bundled frontend derivative (no
    runtime parsing of the ICO).
  - `frontend/public/icon-32.png` — favicon; serves 200 and is copied to
    `dist/`. (The browser's automatic `/favicon.ico` request 404s, but no
    page references it — harmless.)
- Executable: `bin/JunkFuck.exe` embeds the canonical icon — re-verified on
  the latest build by extracting it (32×32, avg RGB 3,8,12 = the dark logo).
- Title bar / taskbar / Alt+Tab: uses the EXE's embedded icon (native Wails/
  Windows behaviour).
- Sidebar branding (top-left, 18×18 px, next to "JunkFuck") and the About
  page identity header (72×72 px) both render the bundled canonical asset —
  verified in a real browser against the built frontend (images load, no
  broken-image icons; About shows icon + name + version).
- NSIS installer: `build/windows/nsis/project.nsi` references `..\icon.ico`
  (same canonical file) for installer + uninstaller.

Not verified in this environment:

- Taskbar/Alt+Tab rendering on a live desktop session (native Windows
  behaviour; manual checklist item).
- A packaged installer build was not produced (the release pipeline ships
  portable EXEs; NSIS config inspected only).

## Issue log

Severity scale: P0 = release-blocking, P1 = major, P2 = minor, P3 = cosmetic.

- P2 — Geist → JetBrains Mono swap changes component metrics.
  Resolution — centralized font tokens + body metric adjustments
  (13.5px / line-height 1.45 / -0.01em); verified no clipping in the build.
- Verified (no defect) — hidden files/folders were already discovered by the
  walk; added proof tests (real Windows hidden attributes + dot-prefix)
  covering discovery, classifier neutrality, protection pruning, dry-run and
  candidate validation.
- P1 — Sidebar background stopped before the bottom of the window.
  Resolution — established the full height chain
  (`html, body, #root { height: 100% }`, app-shell `h-full`, sidebar `h-full`)
  and verified `aside` height == viewport height.
- P1 — Results content could not scroll inside the fixed-height shell.
  Resolution — moved scroll ownership to the page container and the Results
  list container, corrected `min-h-0` constraints; the shell itself never
  scrolls.
- P2 — Accent colors were inconsistent / hardcoded hexes.
  Resolution — centralized `#66c0f4` and derived states (hover/active/soft/
  on-accent) in the theme tokens; removed scattered literals.
- P2 — Results category/summary boxes (new) were content-dependent.
  Resolution — built them as an equal-track grid so geometry is uniform.
- P2 — `build/appicon.png` was the Wails gray template and could overwrite
  the canonical user ICO through the icon pipeline.
  Resolution — re-aligned `appicon.png` to the canonical artwork.
- P2 — App identity was not consistently the canonical artwork in the UI
  (no real icon in the sidebar header or About, placeholder risk).
  Resolution — one bundled frontend derivative (`src/assets/app-icon.png`)
  used for the sidebar branding and About identity header; favicon derived
  from the same source; all surfaces verified against the same fingerprint
  (no Wails "W" or placeholder remains).
- P3 — Drive labels exposed internal enum words.
  Resolution — friendly localized labels (`Local Disk`, …) via i18n tokens.

## Functional verification

Commands actually executed (this cycle):

- `npm install` — added `@fontsource/jetbrains-mono`, removed geist packages.
- `npm test` (vitest, new) — passed: 6/6 pure sidebar-state tests
  (`src/lib/sidebar.test.ts`). Wired into CI (`ci.yml`).
- `npm run build` (tsc --noEmit + vite build) — passed; JetBrains Mono
  woff2 files in the bundle, no Geist references.
- `gofmt -l .` — clean.
- `go vet ./...` — passed.
- `go test ./...` — passed (incl. the new hidden-scanning proof tests in
  scanner + cleaner, run on Windows with real hidden attributes).
- `wails3 build VERSION="4.1.0"` — passed repeatedly across review fix
  cycles; the current candidate artifact is `bin/JunkFuck.exe` (12,092,416
  bytes, SHA256 `9e17974cb7e100d5b7fe4671556acaaed38f8d6eed9ee2e97fe16771098d1f6f`).
- EXE inspection: `4.1.0` version injected, `jetbrains-mono-latin` assets
  embedded, zero `geist` strings; app launches on Windows 11 and stays
  running.
- Browser (built frontend): body font `JetBrains Mono`, aside
  `border-radius: 0 14px 14px 0`.
- Browser (built frontend, collapsible sidebar): collapsed 56px → click
  Expand → 224px → click Collapse → 56px immediately (pin control verified).

Not executed in this environment: live scan → Results transition, large-result
scrolling, multi-monitor/DPI matrix, packaged installer build, live
"update available" state against a newer real release, live hover-expansion
width measurement (browser tooling flaky; logic unit-tested).

## Remaining findings

- P1 (verification gap) — scan → Results auto-navigation, Results scrolling
  with real data, and the live "update available" sidebar state must be
  confirmed by a manual run of the test build.
- P2 (verification gap) — multi-monitor and 125–200% DPI behaviour untested
  here (single monitor, 100% DPI).
- P3 — Windows 10 shows the standard dark title bar (no custom caption colour
  API); expected platform limitation, not a defect.
- P2 (change note) — the Geist → JetBrains Mono swap changes component
  metrics (wider/taller face); compensated via centralized font tokens and
  body metric adjustments, verified against the built UI.
- P2 (fixed in review) — clicking "Collapse sidebar" while the pointer was
  over the sidebar did not visibly collapse it (hover-expansion kept it
  open). Resolution — the pin click clears the transient hover state, so a
  collapse is instant; the pin control now reflects the pinned preference.
- P3 (fixed in review) — collapsed-mode update-status tooltip described the
  action, not the state. Resolution — the tooltip now shows the state label
  ("Update available — v4.2.0") when collapsed.

## Result

blocked — implementation and automated verification passed, but the candidate
build is awaiting the user's manual validation before it may be released.
