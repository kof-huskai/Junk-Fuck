# Legacy Python implementation (reference only)

This directory preserves the original Python 3 CLI implementation of Junk-Fuck
(`junkfuck.py`, its pytest suite and packaging) exactly as it existed before
the desktop modernization (see `docs/MODERNIZATION-SPEC.md`).

It is kept for **behavioral reference**:

- junk detection rules (extensions, folder names, keywords)
- protected path / protected application lists
- deletion flow and per-item confirmation
- the 93-case pytest suite

It is **not** part of the production application. The desktop app is built
with Go + Wails and does not require Python at runtime.

## Running the legacy tool (optional)

```bash
python -m venv venv
venv\Scripts\activate
pip install -r requirements-dev.txt
python junkfuck.py          # interactive CLI
python -m pytest            # legacy test suite
```

> ⚠️ The legacy tool scans `C:\` and deletes on confirmation — run only on
> data you are prepared to lose, and read `CONTRIBUTING.md`'s safety rules
> from the original repository history first.
