"""Locating the ``airom`` binary."""

from __future__ import annotations

import os
import shutil
import sys
import sysconfig
from pathlib import Path

from .errors import BinaryNotFoundError

__all__ = ["find_binary", "bundled_binary"]

_ENV_VAR = "AIROM_BINARY"

# Legacy location: the binary used to ship as package data. It now installs as a
# script (into the environment's bin/), so this only matters for an older wheel.
_LEGACY_BIN_DIR = Path(__file__).parent / "_bin"


def _exe_name() -> str:
    return "airom.exe" if sys.platform == "win32" else "airom"


def _scripts_dirs() -> list[Path]:
    """Directories pip may have installed the ``airom`` script into.

    Consulting these directly (rather than relying on ``PATH``) means the SDK
    still resolves the binary when the environment is not activated — e.g.
    calling ``/path/to/venv/bin/python`` without sourcing its activate script.
    """
    out: list[Path] = []
    try:
        if p := sysconfig.get_path("scripts"):
            out.append(Path(p))
    except (KeyError, OSError):  # pragma: no cover - exotic sysconfig setups
        pass
    try:  # pip install --user
        if p := sysconfig.get_path("scripts", f"{os.name}_user"):
            out.append(Path(p))
    except (KeyError, OSError):  # pragma: no cover
        pass
    return out


def bundled_binary() -> Path | None:
    """The ``airom`` executable shipped with this installation, if present."""
    exe = _exe_name()
    for d in (*_scripts_dirs(), _LEGACY_BIN_DIR):
        p = d / exe
        if p.is_file() and os.access(p, os.X_OK):
            return p
    return None


def find_binary(explicit: str | os.PathLike[str] | None = None) -> str:
    """Resolve the ``airom`` executable.

    Resolution order:

    1. ``explicit`` — the ``binary=`` argument, if given.
    2. The binary this installation shipped: the environment's scripts dir
       (where the wheel puts it), then the legacy ``airom/_bin/`` location.
    3. ``$AIROM_BINARY``.
    4. ``airom`` on ``PATH``.

    Raises:
        BinaryNotFoundError: if no executable is found, with a message
            explaining every option.
    """
    if explicit is not None:
        p = Path(explicit)
        if not p.is_file():
            raise BinaryNotFoundError(f"binary={p!s}: no such file")
        return str(p)

    if (b := bundled_binary()) is not None:
        return str(b)

    if env := os.environ.get(_ENV_VAR):
        p = Path(env)
        if not p.is_file():
            raise BinaryNotFoundError(f"{_ENV_VAR}={env!r}: no such file")
        return str(p)

    if found := shutil.which("airom"):
        return found

    raise BinaryNotFoundError(
        "the 'airom' binary was not found. This installation did not ship one "
        "(installing from an sdist does not), and it is not on PATH.\n"
        "Fix it with any of:\n"
        "  • pip install a platform wheel — it bundles the binary and puts\n"
        "    'airom' on your PATH\n"
        "  • go install github.com/airomhq/airom/cmd/airom@latest   "
        "(then ensure $(go env GOPATH)/bin is on PATH)\n"
        "  • download a release binary from https://github.com/airomhq/airom/releases\n"
        f"  • point {_ENV_VAR} at an existing binary\n"
        "  • pass binary='/path/to/airom' to the scan call"
    )
