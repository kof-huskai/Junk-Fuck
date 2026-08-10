"""Tests for the junk detection logic (files & folders)."""

import pytest


class TestJunkFileDetection:
    @pytest.mark.parametrize(
        "filename",
        [
            "test.tmp",
            "cache.temp",
            "app.log",
            "file.bak",
            "old.backup",
            "crash.dmp",
            "memory.dump",
            "data.cache",
            "thumb.thumbcache",
            "download.part",
            "file.crdownload",
            "module.pyc",
            "scratch.swp",
            "notes.txt~",
        ],
    )
    def test_known_junk_extensions(self, cleaner, tmp_path, filename):
        target = tmp_path / filename
        target.write_text("junk")
        assert cleaner._is_junk_file(str(target)) is True

    @pytest.mark.parametrize(
        "filename",
        [
            "my_temp_notes.txt",
            "tempfile.xyz",
            "backup_data.docx",
            "old_report.txt",
            "crash_report.md",
            "copy_of_notes.txt",
        ],
    )
    def test_junk_keywords_in_name(self, cleaner, tmp_path, filename):
        target = tmp_path / filename
        target.write_text("junk")
        assert cleaner._is_junk_file(str(target)) is True

    @pytest.mark.parametrize(
        "filename",
        [
            "normal_document.txt",
            "notes.docx",
            "photo.png",
            "report.pdf",
            "video.mp4",
            "data.xlsx",
            "archive.zip",
        ],
    )
    def test_regular_files_are_not_junk(self, cleaner, tmp_path, filename):
        target = tmp_path / filename
        target.write_text("important")
        assert cleaner._is_junk_file(str(target)) is False


class TestJunkFolderDetection:
    @pytest.mark.parametrize(
        "dirname",
        [
            "cache",
            "temp",
            "Temp",
            "logs",
            "backup",
            "__pycache__",
            ".cache",
            "Prefetch",
            "Trash",
            "thumbnails",
        ],
    )
    def test_known_junk_folders(self, cleaner, tmp_path, dirname):
        target = tmp_path / dirname
        target.mkdir()
        assert cleaner._is_junk_folder(str(target)) is True

    @pytest.mark.parametrize(
        "dirname",
        ["Documents", "MyMusic", "Projects", "normal_folder"],
    )
    def test_regular_folders_are_not_junk(self, cleaner, tmp_path, dirname):
        target = tmp_path / dirname
        target.mkdir()
        assert cleaner._is_junk_folder(str(target)) is False
