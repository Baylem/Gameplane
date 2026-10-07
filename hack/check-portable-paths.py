#!/usr/bin/env python3
"""Reject tracked filenames that cannot be checked out on Windows."""

import os
from pathlib import Path
import re
import subprocess


RESERVED = re.compile(r"(?:CON|PRN|AUX|NUL|COM[1-9¹²³]|LPT[1-9¹²³])", re.IGNORECASE)
INVALID = re.compile(r'[\x00-\x1f<>:"\\|?*]')


def path_errors(path: str) -> list[str]:
    errors = []
    for component in path.split("/"):
        # Device names remain reserved with any extension. Windows also
        # strips spaces immediately before the extension for device lookup.
        basename = component.split(".", 1)[0].rstrip(" ")
        if RESERVED.fullmatch(basename):
            errors.append(f"reserved device component {component!r}")
        if INVALID.search(component):
            errors.append(f"invalid characters in component {component!r}")
        if component.endswith((".", " ")):
            errors.append(f"trailing dot or space in component {component!r}")
    return errors


def tracked_paths() -> list[str]:
    # NUL separators preserve spaces, newlines and non-ASCII Git filenames;
    # scan the entire index, including docs-only changes and nested names.
    output = subprocess.check_output(
        ["git", "ls-files", "-z"], cwd=Path(__file__).resolve().parent.parent
    )
    return [os.fsdecode(path) for path in output.split(b"\0") if path]


def main() -> int:
    failures = [(path, path_errors(path)) for path in tracked_paths()]
    failures = [(path, errors) for path, errors in failures if errors]
    for path, errors in failures:
        print(f"Nonportable tracked path {path!r}: {'; '.join(errors)}")
    if failures:
        return 1
    print("All tracked filenames are portable to Windows.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
