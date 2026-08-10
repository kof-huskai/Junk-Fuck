"""Tests for the built-in safety protections."""

import os
import platform

import pytest

pytestmark = pytest.mark.skipif(
    platform.system() != "Windows",
    reason="Protected-path checks are Windows-specific",
)


class TestProtectedPaths:
    @pytest.mark.parametrize(
        "path",
        [
            r"C:\Windows",
            r"C:\Windows\System32",
            r"C:\Windows\System32\drivers\etc\hosts",
            r"C:\Windows\SysWOW64",
            r"C:\Program Files",
            r"C:\Program Files (x86)",
            r"C:\ProgramData",
            r"C:\Users\All Users",
            r"C:\System Volume Information",
            r"C:\$Recycle.Bin",
            r"C:\Recovery",
            r"C:\Boot",
            r"C:\Documents and Settings",
        ],
    )
    def test_system_paths_are_protected(self, cleaner, path):
        assert cleaner._is_protected(path) is True

    @pytest.mark.parametrize(
        "app",
        [
            "discord",
            "chrome",
            "firefox",
            "vscode",
            "pycharm",
            "steam",
            "spotify",
            "telegram",
            "whatsapp",
            "docker",
        ],
    )
    def test_known_apps_are_protected_by_name(self, cleaner, app):
        path = os.path.join(r"C:\Users\demo\AppData\Local", app)
        assert cleaner._is_protected(path) is True

    def test_drive_roots_are_protected(self, cleaner):
        assert cleaner._is_protected("C:\\") is True
        assert cleaner._is_protected("D:\\") is True

    def test_regular_user_path_is_not_protected(self, cleaner, tmp_path):
        assert cleaner._is_protected(str(tmp_path)) is False

    def test_junk_file_inside_protected_path_is_never_flagged(self, cleaner):
        candidates = [
            r"C:\Windows\System32\notepad.exe",
            r"C:\Program Files\Microsoft Office\root\office16\WINWORD.EXE",
        ]
        found = False
        for path in candidates:
            if os.path.isfile(path):
                assert cleaner._is_junk_file(path) is False
                found = True
        assert found, "None of the candidate protected files exist on this runner"
