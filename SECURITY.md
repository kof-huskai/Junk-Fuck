# 🔒 Security Policy

JUNKFUCK is a tool that **permanently deletes files**, so security matters here more than in a typical utility. Thank you for taking the time to report issues responsibly.

## Supported versions

| Version | Supported |
| --- | --- |
| 4.0.0 (current) | ✅ Supported |
| < 4.0.0 | ❌ Not supported |

Only the latest release receives security fixes. If you're on an older version, please upgrade before reporting.

## Reporting a vulnerability

**Please do NOT open a public issue for security problems.** Details of a flaw in a file-deleting tool can be abused — report it privately instead:

- **Preferred:** GitHub's [Private vulnerability reporting](https://github.com/kof-huskai/Junk-Fuck/security/advisories) (if enabled on the repo).
- **Alternative:** Email the maintainer directly. The address is available on their GitHub profile.

### What to include in your report

- **Description** — what the issue is and why it's a problem
- **Impact** — what an attacker could do (e.g. "protected paths can be bypassed", "untrusted input causes deletion of arbitrary files")
- **Steps to reproduce** — minimal, concrete steps
- **Affected versions**
- **Suggested fix (optional)** — a patch or recommendation is always welcome

### What happens next

1. **Acknowledgment** — we'll reply within **72 hours** to confirm we received your report.
2. **Investigation** — we'll reproduce, triage and develop a fix.
3. **Coordinated disclosure** — we'll work with you on when and how to publish. Credit goes to the reporter unless you prefer to stay anonymous.

## Security-relevant areas of the code

These are the parts most likely to matter if you're auditing the project:

- **`internal/protection`** — the core safety gate. Anything that changes how protected paths/apps are matched affects safety directly.
- **`internal/cleaner`** — the validated deletion path. It must always respect the protection checks and the scan-session scope.
- **`internal/classifier`** — detection rules. Overly broad patterns could match legitimate user files.
- **`internal/scanner`** — the read-only walk; it must never modify the filesystem.
- **`services/cleanup_service.go`** — the Wails-facing boundary. It must never accept an arbitrary path from the frontend.
- **Updater** (`services/update_service.go`) — update URLs are never accepted from the frontend; downloads are verified against the official Wails v3 updater + GitHub release checksums.
- **Dependencies** — Go standard library + Wails v3 + React; keep them updated deliberately (Wails pinned at `v3.0.0-beta.6`).

## General safety notes for users

- Always **review the items** the tool lists before confirming deletion — the tool asks for a reason.
- Run **as Administrator only when needed**; a normal-user session is safer and usually sufficient.
- The tool is designed to protect system paths and common apps, but **no list is perfect** — when in doubt, skip the item.
- Use the latest version from the [Releases page](https://github.com/kof-huskai/Junk-Fuck/releases) or via the in-app updater.

Thanks for helping keep JUNKFUCK safe for everyone. 🛡️
