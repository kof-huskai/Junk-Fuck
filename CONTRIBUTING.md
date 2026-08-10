# 🤝 Contributing to JUNKFUCK

First off — thank you for wanting to contribute! 🧹 JUNKFUCK is a small project with a big responsibility (it *deletes files*), so a few guidelines help us keep things safe and maintainable.

## Table of contents

- [Code of conduct](#code-of-conduct)
- [Ways to contribute](#ways-to-contribute)
- [Setting up the project](#setting-up-the-project)
- [Safety rules (please read!)](#safety-rules-please-read)
- [Development workflow](#development-workflow)
- [Style guide](#style-guide)
- [Testing your changes](#testing-your-changes)
- [Pull request checklist](#pull-request-checklist)

---

## Code of conduct

Be kind, be constructive. This is a friendly open-source project — treat others the way you'd like to be treated. Harassment, trolling and personal attacks are not welcome in any form.

## Ways to contribute

There's something for everyone, not just code:

- 🐛 **Report bugs** — use the [bug report template](.github/ISSUE_TEMPLATE/bug_report.md) and include your OS/Python versions and console output.
- 💡 **Suggest features** — use the [feature request template](.github/ISSUE_TEMPLATE/feature_request.md).
- 🧠 **Add junk patterns** — new extensions, folders or detection keywords are small, valuable, low-risk contributions.
- 📚 **Improve docs** — README translations, better wording, clearer examples.
- 🎨 **Polish the UI** — the console output (colors, tables, spinner) is a core part of the experience.

## Setting up the project

```bash
# 1. Fork the repo, then clone your fork
git clone https://github.com/<your-username>/Junk-Fuck.git
cd Junk-Fuck

# 2. Add the original repo as "upstream"
git remote add upstream https://github.com/kof-huskai/Junk-Fuck.git

# 3. (Recommended) virtual environment
python -m venv venv
venv\Scripts\activate

# 4. Install the only dependency
pip install colorama
```

## Safety rules (please read!)

**This tool permanently deletes files.** These rules exist to protect users and are non-negotiable:

1. **Never weaken the protection lists.** `PROTECTED_APPS`, `PROTECTED_PATHS` and the `_is_protected()` logic keep system folders and apps (Discord, browsers, IDEs, games…) safe. Removing entries or "simplifying" this logic is a breaking change.
2. **Every deletion stays opt-in.** The interactive per-item confirmation is a core design decision. Don't add silent auto-delete modes without a very good reason and a discussion first.
3. **Don't broaden junk detection recklessly.** Adding `*` wildcards or overly generic keywords (like `file`, `data`) can match legitimate files. Prefer specific extensions/folder names.
4. **Never delete more than the scan reports.** If you change scan behavior, the report table and final summary must stay accurate.
5. **Test on throwaway data**, not your main drive. See [Testing](#testing-your-changes).

## Development workflow

1. **Create a branch** with a descriptive name:
   ```bash
   git checkout -b feat/support-d-drive
   git checkout -b fix/protected-folder-crash
   git checkout -b docs/improve-readme
   ```
2. **Make small, focused commits** with clear messages:
   ```bash
   git commit -m "feat: add support for scanning additional drives"
   git commit -m "fix: handle locked cache folders without crashing"
   git commit -m "docs: clarify the safety section in the README"
   ```
   (Conventional prefixes like `feat:`, `fix:`, `docs:`, `refactor:`, `test:` are appreciated.)
3. **Keep your branch up to date** with upstream `main` before opening a PR:
   ```bash
   git fetch upstream
   git rebase upstream/main
   ```
4. **Push and open a pull request** using the [PR template](.github/PULL_REQUEST_TEMPLATE.md). Describe *what* you changed and *why*.

## Style guide

The codebase is small — consistency beats cleverness:

- **Python 3.8+**, with type hints on public functions (e.g. `List[str]`, `Optional[...]`).
- **Dataclasses** for structured data (`JunkItem`) — don't introduce plain dicts where a dataclass fits.
- **Enums / class constants** for fixed sets (like `JUNK_EXTENSIONS`) — keep them grouped and commented.
- Existing patterns to follow:
  - Errors are **caught narrowly** (`except (PermissionError, OSError)`) and never swallowed silently.
  - Console output goes through `ConsoleUI` helpers (`print_center`, `print_table`), **not** raw `print()`.
  - Spinner state is managed via `start_spinner()` / `stop_spinner()`.
- No new third-party dependencies without discussing them first — `colorama` is the only one today.

## Testing your changes

- **Run the tool** and make sure it starts, scans and exits cleanly:
  ```bash
  python junkfuck.py
  ```
- **For detection changes**, create a throwaway test folder with fake junk (e.g. `my_test/__pycache__/`, `my_test/old.log`, `my_test/temp`), and verify the scanner lists them in the right categories.
- **For deletion changes**, use files you're happy to lose. Never test deletion logic on real project folders or your personal data.
- **Check the final report** — deleted/skipped/failed counts and space freed should be sensible.

## Pull request checklist

Before submitting, make sure you can tick every box:

- [ ] I read [CONTRIBUTING.md](CONTRIBUTING.md) and the README's safety section
- [ ] My change follows the existing code style
- [ ] I did **not** remove or weaken any built-in protections
- [ ] I tested the script (scan at minimum) and it runs without errors
- [ ] I described what I changed and why in the PR description

---

Questions? Open a [discussion](https://github.com/kof-huskai/Junk-Fuck/discussions) — happy to help. Thanks again for contributing! 💚
