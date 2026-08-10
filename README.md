<div align="center">

# 🧹 JUNKFUCK

**Deep C: Drive Junk Scanner & Cleaner for Windows**

![Python](https://img.shields.io/badge/Python-3.8+-3776AB?logo=python&logoColor=white)
![Platform](https://img.shields.io/badge/Platform-Windows-0078D6?logo=windows&logoColor=white)
![Version](https://img.shields.io/badge/version-3.0.0-9B59B6)
![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen)

**English** | [**فارسی (Persian)**](README.fa.md)

---

> ⚠️ **Heads up:** This tool permanently deletes files. It always asks for your confirmation before removing anything, but please review each item carefully — **you are responsible for your own data.**

</div>

**JUNKFUCK** is a colorful, interactive command-line utility that deep-scans your **C: drive** for junk files — temporary files, caches, logs, crash dumps, backups, partial downloads and more — and lets you clean them up one by one, with **your confirmation for every single item**.

No bloated GUI, no dark patterns — just a fast, honest terminal tool that tells you exactly what it found and asks before it touches anything.

---

## ✨ Features

- 🔍 **Deep C: drive scan** — walks the whole drive looking for junk
- 🗂️ **Smart categorization** — junk is sorted into readable groups (Temp, Cache, Logs, Backups, Crash Dumps, Partial Downloads, Build Artifacts, …)
- 🛡️ **Built-in protection** — system paths and apps like Discord, browsers, IDEs and games are automatically protected from deletion
- ✅ **Confirm before every deletion** — nothing gets removed without a `y/n` from you
- 🎨 **Beautiful console UI** — centered, colored output with a live spinner animation
- 📊 **Table-formatted logs** — every found item is listed with name, size, type and category
- 📝 **Final report** — a clean summary of what was deleted, skipped and failed, plus space freed
- 🖥️ **DPI-aware** — crisp rendering on high-resolution displays
- ⏹ **Safe exit** — `Ctrl+C` cancels the operation cleanly at any time

---

## 📦 Requirements

- **Windows** (the tool is built around the C: drive and Windows paths)
- **Python 3.8+**
- [**colorama**](https://pypi.org/project/colorama/) — for colored terminal output

### Installation

```bash
# 1. Clone the repository
git clone https://github.com/kof-huskai/Junk-Fuck.git
cd Junk-Fuck

# 2. (Recommended) create and activate a virtual environment
python -m venv venv
venv\Scripts\activate

# 3. Install the only dependency
pip install colorama
```

---

## 🚀 Usage

```bash
python junkfuck.py
```

> 💡 **Tip:** Run your terminal **as Administrator** for a deeper scan — some folders are locked for normal users. The tool will warn you if it detects it isn't running with admin rights.

### What happens when you run it

1. A JUNKFUCK banner and scan status are displayed.
2. The tool walks `C:\` and collects every junk item it finds.
3. Results are printed in a table (name, size, type, category).
4. You are asked to confirm the cleanup, then **each item is confirmed individually**.
5. A final report shows deleted / skipped / failed counts and total space freed.

---

## 🛡️ Safety & Protection

JUNKFUCK is designed to be cautious by default:

- **Protected system paths** — `C:\Windows`, `System32`, `Program Files`, `ProgramData`, `System Volume Information`, `$Recycle.Bin`, and more are never touched.
- **Protected apps** — Discord, Slack, Telegram, browsers (Chrome, Firefox, Edge, Brave, …), IDEs (VS Code, PyCharm, IntelliJ, …), Steam, Spotify, Docker, OBS and many others are skipped automatically.
- **Interactive confirmation** — deleting is always opt-in, per item. You can skip anything you don't want removed.
- **Read-only safety net** — if a file is in use or locked, the tool reports the failure instead of forcing its way through.

> If you want even more safety, review the `JUNK_EXTENSIONS` and `JUNK_FOLDERS` lists in `junkfuck.py` before running.

---

## 🧠 How It Detects Junk

| Category | Examples |
| --- | --- |
| **Temp / Cache** | `.tmp`, `.temp`, `.cache`, `.part`, `temp`, `cache` folders |
| **Logs** | `.log`, `.etl`, `.evtx`, `logs` folders |
| **Backups** | `.bak`, `.backup`, `.old` |
| **Crash Dumps** | `.dmp`, `.dump`, `.hdmp`, `.mdmp`, `.wer` |
| **Partial Downloads** | `.crdownload`, `.download`, `.part` |
| **Build Artifacts** | `.pyc`, `.class`, `.o`, `.obj` |
| **Editor Temp** | `.swp`, `.swo`, `.~tmp` |
| **Junk Folders** | `__pycache__`, `Prefetch`, `Thumbnails`, `Trash`, `Backup`, … |

---

## ❓ FAQ

**Is this a virus?**
No. JUNKFUCK is an open-source script you can read in full — it only scans for junk patterns and deletes what you approve.

**Why does it say "Not running as Administrator"?**
Some folders (like `System Volume Information`) are locked unless the terminal is elevated. Run the terminal as admin for the fullest scan.

**Can it clean other drives?**
Currently the interactive scan targets `C:\`. The scanner is generic enough that other drives can be supported later.

**Why is Discord protected?**
Messaging apps keep data in caches that users often want to preserve (messages, media, login state). JUNKFUCK skips them so you don't accidentally nuke your data.

---

## 🤝 Contributing

Contributions are very welcome! Whether it's a bug report, a new junk pattern, or a full feature, here's how:

1. **Issues** — use the [bug report](.github/ISSUE_TEMPLATE/bug_report.md) or [feature request](.github/ISSUE_TEMPLATE/feature_request.md) templates so maintainers can help you faster.
2. **Pull requests** — please fill out the [pull request template](.github/PULL_REQUEST_TEMPLATE.md) and describe what you changed and why.

---

## 📜 License

Distributed under the **MIT License**. See the `LICENSE` file for details. *(If no license file is present yet, please ask the maintainer before reusing the code.)*

---

<div align="center">

Made with 🧹 and good intentions. Happy cleaning!

</div>
