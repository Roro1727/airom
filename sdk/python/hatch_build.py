"""Build hook: compile the ``airom`` binary into the wheel.

The binary is installed as a **script**, not as package data: hatchling places it
in ``<name>-<version>.data/scripts/``, which pip copies into the environment's
``bin/`` (``Scripts\\`` on Windows) and marks executable. So ``pip install airom``
gives you a real ``airom`` command on PATH — the actual Go binary, with no Python
shim and no interpreter startup — as well as the importable library.

Wheels are therefore platform-specific, and this hook stamps the wheel tag. It
needs the Go toolchain and the repository checkout (the module root is two levels
up from this file).

Opt out with ``AIROM_SKIP_BUNDLE=1`` — the resulting wheel is pure-Python and the
SDK falls back to ``$AIROM_BINARY`` or ``airom`` on ``PATH`` at runtime.

Cross-compile by setting ``GOOS``/``GOARCH`` (both are forwarded to ``go build``);
set ``AIROM_WHEEL_TAG`` to override the platform tag when doing so.
"""

from __future__ import annotations

import os
import shutil
import subprocess
import sys
import sysconfig
from pathlib import Path

from hatchling.builders.hooks.plugin.interface import BuildHookInterface

HERE = Path(__file__).parent
# sdk/python/hatch_build.py -> sdk/python -> sdk -> <repo root>
REPO_ROOT = HERE.parent.parent
# Staged outside the package: the binary ships as a script, not package data.
BUILD_DIR = HERE / "build" / "bin"


def _exe_name() -> str:
    goos = os.environ.get("GOOS") or sys.platform
    return "airom.exe" if goos in ("win32", "windows") else "airom"


def _git(*args: str) -> str:
    """Run a git command in the checkout, or return "" if it is unavailable.

    Building from an exported tarball (no .git) must still work, so a failure
    here is never fatal — the field just falls back to "unknown".
    """
    try:
        r = subprocess.run(
            ["git", *args], cwd=REPO_ROOT, capture_output=True, text=True, check=True
        )
        return r.stdout.strip()
    except (subprocess.CalledProcessError, FileNotFoundError, OSError):
        return ""


def _wheel_tag() -> str:
    if tag := os.environ.get("AIROM_WHEEL_TAG"):
        return tag
    # Not pure-Python, but ABI-independent: the payload is a standalone binary,
    # so the wheel works on any CPython for this platform.
    plat = sysconfig.get_platform().replace("-", "_").replace(".", "_")
    return f"py3-none-{plat}"


class AiromBuildHook(BuildHookInterface):
    PLUGIN_NAME = "custom"

    def initialize(self, version: str, build_data: dict) -> None:
        if self.target_name != "wheel":
            return

        if os.environ.get("AIROM_SKIP_BUNDLE"):
            self.app.display_waiting("AIROM_SKIP_BUNDLE set — building a pure-Python wheel")
            return

        if not (REPO_ROOT / "go.mod").is_file():
            self.app.display_warning(
                f"no go.mod under {REPO_ROOT} — building without a bundled binary "
                "(the SDK will fall back to $AIROM_BINARY or PATH)"
            )
            return

        if shutil.which("go") is None:
            self.app.display_warning(
                "the Go toolchain was not found — building without a bundled binary "
                "(set AIROM_SKIP_BUNDLE=1 to silence this)"
            )
            return

        BUILD_DIR.mkdir(parents=True, exist_ok=True)
        out = BUILD_DIR / _exe_name()

        env = dict(os.environ)
        env["CGO_ENABLED"] = "0"  # invariant P8: the release binary is always static

        # Stamp the version, exactly as the Makefile and goreleaser do. Without
        # this the binary reports "dev" — and since ToolInfo is embedded in every
        # AIBOM it produces, a pip-installed airom would emit documents whose
        # provenance claims tool.version "dev". The wheel and the binary are
        # released together, so the package version is the honest answer.
        #
        # NB: the `version` argument of initialize() is the BUILD TARGET version
        # ("standard"/"editable"), not the package version — that is
        # self.metadata.version.
        ldflags = [
            "-s",
            "-w",
            f"-X main.version={self.metadata.version}",
            f"-X main.commit={_git('rev-parse', '--short', 'HEAD') or 'unknown'}",
            f"-X main.date={_git('show', '-s', '--format=%cI', 'HEAD') or 'unknown'}",
        ]

        cmd = [
            "go", "build", "-trimpath",
            "-ldflags", " ".join(ldflags),
            "-o", str(out), "./cmd/airom",
        ]
        self.app.display_info(f"bundling airom: {' '.join(cmd)} (in {REPO_ROOT})")
        try:
            subprocess.run(cmd, cwd=REPO_ROOT, env=env, check=True)
        except subprocess.CalledProcessError as e:
            raise RuntimeError(f"failed to build the airom binary: {e}") from e

        out.chmod(0o755)
        build_data["pure_python"] = False
        build_data["tag"] = _wheel_tag()
        # -> <name>-<version>.data/scripts/airom -> the environment's bin/ dir.
        build_data["shared_scripts"] = {str(out): _exe_name()}
