"""Shared fixtures for the JUNKFUCK test suite.

JunkCleaner reads Windows environment variables (APPDATA, LOCALAPPDATA, ...)
to build its protected-path list. We clear those variables before creating a
cleaner so tests are hermetic and the test temp directories are NOT treated
as protected paths.
"""

import os
import sys

import pytest

# Make the repo root importable (works even when pytest is run as a plain
# `pytest` binary instead of `python -m pytest`).
_REPO_ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
if _REPO_ROOT not in sys.path:
    sys.path.insert(0, _REPO_ROOT)

_ENV_KEYS = (
    "SYSTEMROOT",
    "WINDIR",
    "ProgramFiles",
    "ProgramFiles(x86)",
    "ProgramData",
    "APPDATA",
    "LOCALAPPDATA",
    "USERPROFILE",
)


@pytest.fixture
def cleaner():
    """A JunkCleaner isolated from the host environment."""
    saved = {key: os.environ.get(key) for key in _ENV_KEYS}
    for key in _ENV_KEYS:
        os.environ.pop(key, None)

    from junkfuck import JunkCleaner, ConsoleUI

    instance = JunkCleaner(ConsoleUI())

    yield instance

    for key, value in saved.items():
        if value is None:
            os.environ.pop(key, None)
        else:
            os.environ[key] = value
