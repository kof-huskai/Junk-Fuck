"""Tests for the deletion path — safe, throwaway temp dirs only.

These tests never touch the real C: drive; they only delete files inside
pytest's temporary directories. The protected-path case is covered with a
fake path inside C:\\Windows and is Windows-only.
"""

import platform

import pytest

from junkfuck import JunkItem


def test_delete_file(cleaner, tmp_path):
    target = tmp_path / "junk.tmp"
    target.write_text("junk")
    item = JunkItem(
        path=str(target),
        name="junk.tmp",
        size=target.stat().st_size,
        is_folder=False,
        category="Temp",
    )
    assert cleaner.delete_item(item) is True
    assert not target.exists()
    assert cleaner.deleted_count == 1
    assert cleaner.failed_count == 0


def test_delete_folder_recursively(cleaner, tmp_path):
    folder = tmp_path / "cache"
    folder.mkdir()
    (folder / "inner.tmp").write_text("junk")
    item = JunkItem(
        path=str(folder), name="cache", size=0, is_folder=True, category="Cache"
    )
    assert cleaner.delete_item(item) is True
    assert not folder.exists()
    assert cleaner.deleted_count == 1


def test_delete_missing_file_is_reported_as_failure(cleaner, tmp_path):
    missing = tmp_path / "gone.tmp"
    item = JunkItem(
        path=str(missing),
        name="gone.tmp",
        size=0,
        is_folder=False,
        category="Temp",
    )
    assert cleaner.delete_item(item) is False
    assert cleaner.failed_count == 1


@pytest.mark.skipif(
    platform.system() != "Windows", reason="Windows-specific protected paths"
)
def test_protected_path_is_never_deleted(cleaner):
    protected = r"C:\Windows\System32\definitely-not-deletable.tmp"
    item = JunkItem(
        path=protected,
        name="definitely-not-deletable.tmp",
        size=0,
        is_folder=False,
        category="Temp",
    )
    assert cleaner.delete_item(item) is False
    assert cleaner.skipped_count == 1
    assert cleaner.deleted_count == 0


def test_scan_finds_only_junk_in_a_temp_tree(cleaner, tmp_path):
    (tmp_path / "cache").mkdir()
    (tmp_path / "test.tmp").write_text("junk")
    (tmp_path / "normal.txt").write_text("important")
    items = cleaner.scan_drive(str(tmp_path))
    names = {item.name for item in items}
    assert "cache" in names
    assert "test.tmp" in names
    assert "normal.txt" not in names
