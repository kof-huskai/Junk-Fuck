<div align="center">

# 🧹 JUNKFUCK

**Deep Windows Junk Scanner & Cleaner — Desktop App**

![Go](https://img.shields.io/badge/Go-1.23+-00ADD8?logo=go&logoColor=white)
![Wails](https://img.shields.io/badge/Wails-v2-DF0000)
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

It is a complete rewrite of the original Python CLI: a **Go core** (all
filesystem logic), a **Wails v2** desktop shell and a **React/TypeScript**
UI. **Python is no longer required.**

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
- 📦 **Auto releases** — tag-driven builds, GitHub Releases and Telegram notifications

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
- **Wails CLI** (`go install github.com/wailsapp/wails/v2/cmd/wails@latest`)
- Windows 10/11 with WebView2 runtime

### Run in dev mode

```bash
wails dev
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
| Regenerate Wails bindings | `wails generate module` (after changing `app.go`) |
| Build EXE | `wails build -clean` |
| Build EXE with version | `wails build -clean -ldflags "-X main.Version=4.0.0"` |

> ⚠️ The Go binary embeds the built frontend (`go:embed frontend/dist`), so
> run `npm run build` before any `go build`/`go test` in a fresh checkout
> (CI does this automatically).

---

## 🏗️ Architecture

```
app.go / main.go          Wails desktop layer (async scans, events, sessions)
internal/
├── classifier/           explicit junk rules (extensions, folders, tokens)
├── protection/           protected paths & apps (backend-enforced)
├── scanner/              read-only async walk, progress, cancellation
├── cleaner/              validated deletion + dry-run
├── filesystem/           canonicalization, reparse points, helpers
├── report/               cleanup report model
└── platform/             OS version / elevation info
frontend/                 React + TypeScript + Tailwind (EN/FA, RTL)
docs/MODERNIZATION-SPEC.md  the migration specification (source of truth)
legacy/python/            original Python CLI (reference only)
```

Key principle: the **frontend never decides what is safe to delete**. All
filesystem-sensitive operations live in the Go core and are covered by
automated safety tests.

---

## 🚀 Release process

Releases are **tag-driven**. Pushing a `v*` tag triggers:

1. Tests (Go + frontend build) — must pass
2. Production build (`wails build`, amd64) + SHA-256 checksums
3. GitHub Release with the EXE + checksums attached
4. Rich Telegram announcement + EXE upload (optional)

```bash
git tag v4.0.0
git push origin v4.0.0
```

Workflows:

- `ci.yml` — tests on every push / PR
- `build-exe.yml` — manual EXE build (Actions → Run workflow)
- `release.yml` — tag-driven production release + Telegram

## 📣 Telegram release notifications (optional)

The release workflow can post the new release to a Telegram channel directly
via the Bot API (plain `curl`, no third-party action). Configure two
repository secrets:

| Secret | Value |
| --- | --- |
| `TELEGRAM_BOT_TOKEN` | Bot token from **@BotFather** (`/newbot`) |
| `TELEGRAM_CHAT_ID` | Channel id, e.g. `@junkfuck_channel` |

Setup: add the bot as **Administrator** of the channel with **Post Messages**
permission, then add the secrets under
**Settings → Secrets and variables → Actions**.

The notification is best-effort: if Telegram is unreachable the GitHub
Release still succeeds (the step is isolated with `continue-on-error`).

---

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
