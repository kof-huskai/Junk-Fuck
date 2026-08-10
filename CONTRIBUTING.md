# 🤝 Contributing to JunkFuck

Thank you for wanting to contribute! 🧹 JunkFuck **deletes files**, so safety
rules come first. Please read them before touching anything.

## Table of contents

- [Safety rules (please read!)](#safety-rules-please-read)
- [Code of conduct](#code-of-conduct)
- [Setting up the project](#setting-up-the-project)
- [Project layout](#project-layout)
- [Development workflow](#development-workflow)
- [Testing](#testing)
- [Style guide](#style-guide)
- [Release process](#release-process)
- [Pull request checklist](#pull-request-checklist)

---

## Safety rules (please read!)

JunkFuck is a destructive filesystem utility. These rules are non-negotiable:

1. **Never weaken the protection lists.** `internal/protection` is the
   backend-enforced safety layer. Removing paths, apps or the
   ancestor-of-protected rule is a breaking change.
2. **Deletion stays session-scoped.** The cleaner (`internal/cleaner`) only
   deletes candidates from the current scan session. Never add a code path
   that deletes an arbitrary path.
3. **Every deletion stays opt-in.** Selection + final confirmation is a core
   design decision. Never add silent auto-delete modes.
4. **Never disable safety checks to make a test pass.** If a test conflicts
   with a safety rule, the test is wrong.
5. **Don't broaden junk detection recklessly.** Rules in
   `internal/classifier` are explicit and tested. No broad substring
   matching (words like `old`, `copy` are not junk markers).
6. **The frontend never decides safety.** UI code may display and select
   candidates, but all validation lives in the Go core.
7. **Test on throwaway data** — never against the real `C:\`.

## Code of conduct

Be kind, be constructive. Harassment, trolling and personal attacks are not
welcome in any form.

## Setting up the project

Prerequisites: **Go 1.23+**, **Node.js 20+**, **Wails v3 CLI**
(`go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-beta.6`), Windows 10/11.

> Wails v3 is pinned at `v3.0.0-beta.6` (pre-GA). Version bumps are deliberate:
> update `go.mod`, reinstall `wails3`, regenerate bindings, re-verify the build.

```bash
# 1. Fork the repo, then clone your fork
git clone https://github.com/<your-username>/Junk-Fuck.git
cd Junk-Fuck

# 2. Build the frontend (the Go binary embeds it - required before go build/test)
cd frontend
npm ci
npm run build
cd ..

# 3. Run the backend test suite
go test ./...
```

## Project layout

```
main.go / services/       Wails v3 layer (app wiring, services, updater)
internal/                 pure Go core — NO Wails imports allowed
├── classifier/           explicit junk rules
├── protection/           protected paths & apps
├── scanner/              async read-only scan, progress, cancellation
├── cleaner/              validated deletion + dry-run
├── filesystem/           path helpers, reparse points
├── report/               report model
└── platform/             OS version / elevation
frontend/                 React + TS + Tailwind (EN/FA + RTL) + v3 bindings
build/                    wails3 build config (config.yml, windows/, icons)
Taskfile.yml              wails3 tasks (build / dev / generate)
docs/MODERNIZATION-SPEC.md  architecture specification (source of truth)
```

## Development workflow

1. Create a descriptive branch (`feat/...`, `fix/...`, `docs/...`).
2. Make small, focused commits with conventional messages
   (`feat:`, `fix:`, `test:`, `ci:`, `docs:`).
3. Keep your branch rebased on `main`.
4. Open a PR describing *what* changed and *why*.

## Testing

```bash
go vet ./...
gofmt -l .                 # must print nothing
go test ./...              # safety tests included
cd frontend && npm run build   # typecheck + production build
```

Every change to `internal/classifier`, `internal/protection`,
`internal/cleaner` or `internal/scanner` **must** come with tests. The most
important tests are the safety ones: protected-path rejection, dry-run
non-modification, session-scoped deletion, revalidation.

## Style guide

- Go: standard `gofmt`, explicit error handling, no `panic`, package
  comments, no new dependencies without discussion.
- The Core (`internal/`) must stay independent of Wails/React — a future
  Wails upgrade must not touch scanner/cleaner/safety code.
- Frontend: TypeScript strict mode; types come from the Wails v3-generated
  bindings (`frontend/bindings/...`) — after changing a `services/*` struct,
  run `wails3 generate bindings -clean=true -ts` and commit the output.
- The updater (`services/update_service.go`) wraps the official Wails v3
  updater; never accept update URLs from the frontend.
- No dependency bloat: prefer hand-rolled components over huge libraries.

## Release process

Releases are tag-driven (`v*`). See [README.md](README.md#-release-process).
The pipeline: tests → `wails3 build VERSION=vX.Y.Z` → checksums → GitHub
Release → Telegram notification. Telegram is best-effort and must never
invalidate the release. The GitHub Release also feeds the in-app updater.

## Pull request checklist

- [ ] I read this file and the safety rules
- [ ] My change follows the existing style
- [ ] I did **not** weaken or bypass any backend safety rule
- [ ] Safety-relevant changes include tests
- [ ] `go test ./...`, `go vet ./...` and the frontend build pass
- [ ] I described what I changed and why

---

Questions? Open a discussion. Thanks again for contributing! 💚
