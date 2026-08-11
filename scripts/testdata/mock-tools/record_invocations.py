#!/usr/bin/env python3
"""Invocation-recording shim used by tests.

Replaces a binary that the harness MUST NOT call directly (e.g.
`govulncheck`) and writes a line to scripts/testdata/mock-tools/invocations.log
every time it is invoked. Tests assert the log is empty after a harness run,
proving the harness never bypassed the Makefile target.

Usage in tests:
    record_invocations = tmp_path / "record_invocations.py"
    record_invocations.write_text(Path(THIS_FILE).read_text())
    subprocess.run([sys.executable, str(record_invocations), "--version"])
    assert (log_path).read_text() == ""  # never invoked by harness
"""
from __future__ import annotations

import os
import sys
from pathlib import Path

LOG = Path(os.environ.get("INVOCATIONS_LOG", "/dev/null"))


def main() -> int:
    LOG.parent.mkdir(parents=True, exist_ok=True)
    with LOG.open("a", encoding="utf-8") as f:
        f.write(f"invoked: argv={sys.argv!r} cwd={os.getcwd()}\n")
    # Exit 0 with no output to simulate a successful no-op.
    return 0


if __name__ == "__main__":
    sys.exit(main())