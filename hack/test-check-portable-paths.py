#!/usr/bin/env python3
"""Portable filename regression fixtures; run only in CI."""

import importlib.util
from pathlib import Path
import unittest
from unittest.mock import patch


spec = importlib.util.spec_from_file_location(
    "check_portable_paths", Path(__file__).with_name("check-portable-paths.py")
)
checker = importlib.util.module_from_spec(spec)
spec.loader.exec_module(checker)


class PortablePathTests(unittest.TestCase):
    def test_valid_paths(self):
        for path in (
            "specs/audit/procedures/auxiliary-services.md",
            "docs/auxiliary.md", "CONSOLE.md", "somewhere/com10.txt",
            "lpt0/config.yaml", "README.md", "nested/ordinary name.txt",
            "nested/unicode-café.md", "nested/.gitkeep",
        ):
            with self.subTest(path=path):
                self.assertEqual(checker.path_errors(path), [])

    def test_reserved_device_components_with_extensions_and_mixed_case(self):
        for name in (
            "CON", "prn.md", "aux.md", "NuL.tar.gz", "com1.txt",
            "COM9", "lpt1.md", "LPT9", "COM¹.txt", "lpt²", "COM³",
            "CON .txt",
        ):
            for path in (name, f"nested/{name}/ordinary.md"):
                with self.subTest(path=path):
                    self.assertTrue(checker.path_errors(path))

    def test_invalid_characters_and_control_characters(self):
        for character in '<>:"\\|?*\x01\x1f':
            with self.subTest(character=character):
                self.assertTrue(checker.path_errors(f"nested/bad{character}name.md"))

    def test_trailing_dot_and_space_in_any_component(self):
        for path in ("bad.", "bad ", "nested./file.md", "nested /file.md"):
            with self.subTest(path=path):
                self.assertTrue(checker.path_errors(path))

    def test_nul_delimited_git_paths_include_newlines_without_splitting_them(self):
        with patch.object(checker.subprocess, "check_output", return_value=b"ok.md\0nested/bad\nname.md\0") as git:
            paths = checker.tracked_paths()
        self.assertEqual(paths, ["ok.md", "nested/bad\nname.md"])
        self.assertIn("-z", git.call_args.args[0])
        self.assertTrue(checker.path_errors(paths[1]))

    def test_repository_scan_fails_on_nonportable_paths(self):
        with patch.object(checker, "tracked_paths", return_value=["ok.md", "docs/AUX.md"]), patch("builtins.print"):
            self.assertEqual(checker.main(), 1)
        with patch.object(checker, "tracked_paths", return_value=["ok.md"]), patch("builtins.print"):
            self.assertEqual(checker.main(), 0)


if __name__ == "__main__":
    unittest.main()
