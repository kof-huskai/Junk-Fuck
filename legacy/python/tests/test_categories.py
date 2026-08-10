"""Tests for junk categorization."""

import pytest


@pytest.mark.parametrize(
    ("name", "is_folder", "expected"),
    [
        ("test.tmp", False, "Temp"),
        ("cache.temp", False, "Temp"),
        ("app.log", False, "Logs"),
        ("events.etl", False, "Logs"),
        ("file.bak", False, "Backup"),
        ("old.backup", False, "Backup"),
        ("crash.dmp", False, "Crash Dumps"),
        ("dump.hdmp", False, "Crash Dumps"),
        ("download.part", False, "Partial Downloads"),
        ("file.crdownload", False, "Partial Downloads"),
        ("module.pyc", False, "Build Artifacts"),
        ("module.o", False, "Build Artifacts"),
        ("scratch.swp", False, "Editor Temp"),
        ("cache", True, "Cache"),
        ("temp", True, "Temp"),
        ("logs", True, "Logs"),
        ("backup", True, "Backup"),
        ("trash", True, "Trash"),
        ("thumbnails", True, "Thumbnails"),
        ("prefetch", True, "Prefetch"),
        ("random_junk_folder", True, "Junk Folder"),
    ],
)
def test_categories(cleaner, name, is_folder, tmp_path, expected):
    path = tmp_path / name
    if is_folder:
        path.mkdir()
    else:
        path.write_text("x")
    assert cleaner._get_category(str(path), is_folder) == expected
