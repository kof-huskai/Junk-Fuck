# Junk-Fuck Design QA

Living QA record for the Junk-Fuck desktop app (Go + Wails v3 + React/TypeScript).

## Sources

- User-provided annotated Dashboard screenshot (external reference; lives in the
  design conversation, not saved into the repository).
- User feedback from the current implementation cycle (sidebar full-height bug,
  window too short, Results scrolling bug, Steam-blue accent, canonical icon,
  equal category boxes, icon missing from the app header/About).
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
- Geist Sans UI font (bundled offline), Geist Mono for technical values;
  Persian falls back to bundled Vazirmatn.
- Radius hierarchy: cards 10px, buttons/controls 6px, small controls 4px;
  minimal pills/badges.
- Full-height sidebar, internal content scrolling only.
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

- `npm run build` (tsc --noEmit + vite build) — passed.
- `gofmt -l .` — clean.
- `go build ./...` — passed.
- `go vet ./...` — passed.
- `go test ./...` — passed (classifier, cleaner, filesystem, protection,
  scanner).
- `wails3 build VERSION="4.0.1"` — passed; produced `bin/JunkFuck.exe`
  (12,234,240 bytes, SHA256 `8c7a1e1263e67b8b1ec12242aa465997ec0602614c20ba0b8afd981a65080346`).
  (The earlier 12,240,896-byte build was superseded by the icon-integration
  rebuild; the temp `JunkFuck.exe~` artifact observed mid-cycle was a stale
  build temp after a file-lock race, removed before the final build.)
- Launched `bin/JunkFuck.exe` on Windows 11 — starts and stays running.
- Browser (built frontend) — sidebar full height, `--color-accent: #66c0f4`,
  Geist font, no dashboard subtitle, no floating language button, plain
  Status/Safety cards.
- Browser (built frontend) — sidebar branding icon (18 px) and About icon
  (72 px) render the canonical artwork; no broken images; favicon and bundled
  asset both serve HTTP 200.
- EXE icon extraction — canonical dark logo embedded (32×32, avg RGB 3,8,12).

Not executed in this environment: live scan → Results transition, large-result
scrolling, multi-monitor/DPI matrix, packaged installer build.

## Remaining findings

- P1 (verification gap) — scan → Results auto-navigation and Results scrolling
  with real data must be confirmed by a manual run of the test build.
- P2 (verification gap) — multi-monitor and 125–200% DPI behaviour untested
  here (single monitor, 100% DPI).
- P3 — Windows 10 shows the standard dark title bar (no custom caption colour
  API); expected platform limitation, not a defect.

## Result

blocked — implementation and automated verification passed, but the build is
awaiting the user's manual validation before it may be released.
