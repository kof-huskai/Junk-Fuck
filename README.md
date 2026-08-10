<div align="center">

# 🧹 JUNKFUCK

**Deep Windows Junk Scanner & Cleaner — Desktop App**

![Go](https://img.shields.io/badge/Go-1.23+-00ADD8?logo=go&logoColor=white)
![Wails](https://img.shields.io/badge/Wails-v3-DF0000)
![React](https://img.shields.io/badge/React-19-61DAFB?logo=react&logoColor=white)
![Platform](https://img.shields.io/badge/Platform-Windows-0078D6?logo=windows&logoColor=white)
![Version](https://img.shields.io/badge/version-4.0.0-9B59B6)
![CI](https://github.com/kof-huskai/Junk-Fuck/actions/workflows/ci.yml/badge.svg)
![Release](https://github.com/kof-huskai/Junk-Fuck/actions/workflows/release.yml/badge.svg)
![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen)

**English** | [**فارسی (Persian)**](README.fa.md)

</div>

> ⚠️ **Heads up:** JunkFuck permanently deletes files. It always asks for your
> explicit confirmation before removing anything, and every deletion is
> validated by the backend — but please review each item carefully:
> **you are responsible for your own data.**

**JunkFuck** is a fast, honest Windows desktop application that deep-scans
your drives for junk — temporary files, caches, logs, crash dumps, backups
and partial downloads — classifies them, and cleans them **only after your
explicit confirmation**. No auto-delete, no dark patterns.

Built as a native Windows application: a **Go core** (all filesystem
logic), a **Wails v3** desktop shell and a **React/TypeScript** UI.
No runtime dependencies — just run the EXE.

> 💾 **No build needed to use it:** grab `JunkFuck.exe` from the
> [Releases page](https://github.com/kof-huskai/Junk-Fuck/releases) and run
> it on Windows 10/11 (WebView2 runtime included in recent Windows updates).

---

## ✨ Features

- 🔍 **Deep, non-destructive scanning** — read-only walk; nothing is modified during a scan
- 🗂️ **Structured classification** — Temporary Files, Cache, Logs, Crash Dumps, Backups, Partial Downloads, Build Artifacts, Editor Temp, Other
- 🛡️ **Backend-enforced safety** — protected system paths, protected applications, and deletion only from the current scan's candidate set
- ✅ **Explicit confirmation** — nothing is deleted without your selection + final confirmation
- 🧪 **Dry run** — preview exactly what would be deleted without touching anything
- ⏹️ **Cancellable async scans** — live progress, current path, counters; the UI never freezes
- 🔎 **Rich results table** — search, filter by category, sort, select all / none, protected items shown but not selectable
- 📝 **History** — last cleanup report (deleted / skipped / failed / space freed)
- 🌐 **English & Persian (RTL)** — switchable from the sidebar
- 🎨 **Dark-first modern UI** — Tailwind CSS, shadcn-style components
- 📦 **Auto releases** — tag-driven builds and GitHub Releases
- 🔄 **In-app updates** — official Wails v3 updater checks GitHub Releases, verifies checksums and installs on demand

---

## 🛡️ Safety model (enforced in Go, not just in the UI)

1. Protected paths and protected application directories are **never deleted**.
2. Only candidates from the **current scan session** can be deleted — arbitrary paths from the UI are rejected.
3. Every candidate is **re-validated** before deletion: re-stat, protection check, drive-root check, reparse-point check, identity check.
4. A directory that **contains** a protected path is refused.
5. Symbolic links / reparse points are never deleted as directories.
6. Deleting always requires **selection + final confirmation**.
7. **Dry run** performs the full validation without modifying anything.
8. Failing safely: locked or missing items are reported, never forced.

Protected by default: `C:\Windows`, `C:\Program Files`, `C:\ProgramData`,
drive roots, and the data directories of Discord, browsers, IDEs, Steam,
Spotify, Telegram, WhatsApp, Docker, OBS and many more.

---

## 🖼️ Screenshots

*Coming soon — the app is under active development.*

---

## ✅ Supported systems

- **Windows 10 / Windows 11** (x64). WebView2 runtime required
  (preinstalled on current Windows 10/11 updates; otherwise installed
  automatically by Microsoft).

## 📦 Installation

Download `JunkFuck.exe` from the latest
[release](https://github.com/kof-huskai/Junk-Fuck/releases) and run it.

> ⚠️ Windows SmartScreen may warn that the EXE is unsigned. Click
> **More info → Run anyway** — it's the same open-source code you can read in
> this repo. (Code signing is on the roadmap, not yet available.)

## 🧑‍💻 Development

### Prerequisites

- **Go 1.23+**
- **Node.js 20+** (npm)
- **Wails v3 CLI** (`go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-beta.6`)
- Windows 10/11 with WebView2 runtime

> **Wails v3 is pinned at `v3.0.0-beta.6`** (pre-GA). Version bumps are
> deliberate: update `go.mod`, reinstall `wails3`, regenerate bindings and
> re-verify the build.

### Run in dev mode

```bash
wails3 dev
```

This builds the Go backend, starts the Vite dev server and launches the app
with hot reload.

### Commands

| Task | Command |
| --- | --- |
| Run tests | `go test ./...` |
| Vet | `go vet ./...` |
| Format check | `gofmt -l .` |
| Build frontend | `cd frontend && npm ci && npm run build` |
| Regenerate Wails bindings | `wails3 generate bindings -clean=true -ts` (after changing a service) |
| Build EXE | `wails3 build` |
| Build EXE with version | `wails3 build VERSION=4.0.0` |
| Generate icons | `wails3 generate icons -input appicon.png -windowsfilename windows/icon.ico` (in `build/`) |

> ⚠️ The Go binary embeds the built frontend (`go:embed frontend/dist`), so
> run `npm run build` before any `go build`/`go test` in a fresh checkout
> (CI does this automatically).

---

## 🏗️ Architecture

```
main.go / services/       Wails v3 layer (app wiring, services, updater)
internal/  (pure Go core, no Wails imports)
├── classifier/           explicit junk rules (extensions, folders, tokens)
├── protection/           protected paths & apps (backend-enforced)
├── scanner/              read-only async walk, progress, cancellation
├── cleaner/              validated deletion + dry-run
├── filesystem/           canonicalization, reparse points, helpers
├── report/               cleanup report model
└── platform/             OS version / elevation info
frontend/                 React + TypeScript + Tailwind (EN/FA, RTL) + v3 bindings
build/                    wails3 build config (config.yml, windows/, icons)
Taskfile.yml              wails3 tasks (build / dev / generate)
docs/MODERNIZATION-SPEC.md  the migration specification (source of truth)
```

```
React / TypeScript
    ↓ @wailsio/runtime (bindings + events)
services/ (Wails v3)
    ↓
internal/ (pure Go core)
    ↓
Filesystem / Windows APIs
```

Key principle: the **frontend never decides what is safe to delete**. All
filesystem-sensitive operations live in the Go core and are covered by
automated safety tests.

---

## 🚀 Release process

Releases are **tag-driven**. Pushing a `v*` tag triggers:

1. Tests (Go + frontend build) — must pass
2. Production build (`wails3 build VERSION=vX.Y.Z`, amd64) — the version is
   injected via `-ldflags` from the tag (single source of truth)
3. SHA-256 checksums
4. GitHub Release with the EXE + checksums attached

The GitHub Release is also the **update source**: the in-app updater reads
`JunkFuck-*.exe` + `SHA256SUMS` from it.

```bash
git tag v4.0.0
git push origin v4.0.0
```

Workflows:

- `ci.yml` — tests on every push / PR
- `build-exe.yml` — manual EXE build (Actions → Run workflow)
- `release.yml` — tag-driven production release


## 🧠 How junk is detected

| Category | Examples |
| --- | --- |
| Temporary Files | `.tmp`, `.temp`, `.cache`, `temp`/`tmp` folders, `~`-suffixed files |
| Cache | `cache` folders, `.thumbcache`, thumbnails |
| Logs | `.log`, `.etl`, `.evtx`, `logs` folders |
| Crash Dumps | `.dmp`, `.dump`, `.hdmp`, `.mdmp`, `.wer` |
| Backups | `.bak`, `.backup`, `.old` |
| Partial Downloads | `.crdownload`, `.part`, `.download` |
| Build Artifacts | `.pyc`, `.class`, `.o`, `.obj`, `__pycache__` |
| Editor Temp | `.swp`, `.swo` |

Rules are explicit and tested — deliberately **no** broad substring matching
(words like `old` or `copy` are not junk markers).

## ❓ FAQ

**Is this a virus?** No. It is open-source; it scans read-only and only
deletes what you explicitly confirm.

**Does it need Administrator rights?** No — it runs as a normal user. Some
locked folders (e.g. `System Volume Information`) may not be readable without
elevation; the UI explains this without forcing elevation.

**Can it clean other drives?** Yes — scan targets are configurable (default
`C:\`), as long as they are not protected paths.

**Why is Discord protected?** Messaging apps keep caches you may want to
preserve (messages, media, login state). JunkFuck skips them.

---

## 🤝 Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) — safety rules first, then workflow
and style.

## 📜 License

Distributed under the **MIT License**. See the [LICENSE](LICENSE) file.

---

<div align="center">

Made with 🧹 and good intentions. Happy cleaning!

</div>
