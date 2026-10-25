"""Behavioral coverage for the changed-lines coverage gate (run in CI)."""

import contextlib
import io
import json
import os
import subprocess
import tempfile
import unittest
from pathlib import Path
from unittest.mock import patch

from check_changed_coverage import check, expand, go_config, main, vitest_config


GIT = ["git", "-c", "user.name=test", "-c", "user.email=test@example.com",
       "-c", "commit.gpgsign=false", "-c", "core.hooksPath=/dev/null",
       # background gc could still be writing .git when the temp dir is removed
       "-c", "gc.auto=0", "-c", "maintenance.auto=false"]
COVERAGE_YML = """threshold:
  file: 0
  total: 80

exclude:
  paths:
    - ^cmd/  # wiring
"""
VITEST_CONFIG = """export default defineConfig({
  test: {
    exclude: ["e2e/**"],
    coverage: {
      exclude: ["src/main.tsx", "**/*.test.{ts,tsx}"],
      thresholds: {
        lines: 92,
        functions: 76,
      },
    },
  },
});
"""
GO_FUNC = "package mod\n// added comment\nfunc F() int {\n\treturn 1\n}\n"


class ChangedCoverageTests(unittest.TestCase):
    """Check added-line coverage against Go profiles and Istanbul JSON."""

    def setUp(self):
        """Create a git repository with one Go module and a web tree."""
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.git("init", "-q")
        self.write("mod/go.mod", "module example.com/mod\n")
        self.write("mod/.testcoverage.yml", COVERAGE_YML)
        self.write("mod/a.go", "package mod\n")
        self.write("mod/cmd/main.go", "package main\n")
        self.write("web/vitest.config.ts", VITEST_CONFIG)
        self.write("web/src/a.ts", "export const a = 1;\n")
        self.base = self.commit()
        self.head = self.base

    def git(self, *args):
        """Run git in the fixture repository and return its trimmed output."""
        return subprocess.run([*GIT, *args], cwd=self.root, check=True, capture_output=True,
                              text=True).stdout.strip()

    def write(self, name, content):
        """Write a fixture file and return its path."""
        path = self.root / name
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(content, encoding="utf-8")
        return path

    def commit(self):
        """Commit every fixture change and return the new commit id."""
        self.git("add", "-A")
        self.git("commit", "-q", "-m", "change")
        return self.git("rev-parse", "HEAD")

    def run_main(self, *args):
        """Run the gate and return (exit code, stdout, step summary)."""
        summary = self.root / "summary.md"
        out = io.StringIO()
        with patch.dict(os.environ, {"GITHUB_STEP_SUMMARY": str(summary)}), \
                contextlib.redirect_stdout(out):
            code = main(["--root", str(self.root), "--base", self.base, "--head", self.head,
                         *args])
        text = summary.read_text(encoding="utf-8") if summary.exists() else ""
        return code, out.getvalue(), text

    def go_change(self, count):
        """Add a comment and a three-line function block; return a profile path."""
        self.write("mod/a.go", GO_FUNC)
        self.head = self.commit()
        profile = self.write("profile.out",
                             f"mode: atomic\nexample.com/mod/a.go:3.14,5.2 1 {count}\n")
        return str(profile)

    def test_covered_go_lines_pass(self):
        code, out, summary = self.run_main("--go-profile", self.go_change(1), "--module", "mod")
        self.assertEqual(code, 0)
        self.assertIn("3/3 added statement lines covered", out)
        self.assertIn("changed-lines coverage", summary)

    def test_uncovered_go_lines_fail_and_are_listed(self):
        code, out, _ = self.run_main("--go-profile", self.go_change(0), "--module", "mod")
        self.assertEqual(code, 1)
        self.assertIn("0/3 added statement lines covered", out)
        for number in (3, 4, 5):
            self.assertIn(f"uncovered: mod/a.go:{number}", out)

    def test_zero_statement_lines_pass(self):
        self.write("mod/a.go", "package mod\n// only a comment\n")
        self.head = self.commit()
        profile = self.write("profile.out", "mode: atomic\n")
        code, out, _ = self.run_main("--go-profile", str(profile), "--module", "mod")
        self.assertEqual(code, 0)
        self.assertIn("no added statement lines", out)

    def test_go_tests_and_excluded_paths_are_ignored(self):
        self.write("mod/cmd/main.go", "package main\nfunc main() {\n}\n")
        self.write("mod/a_test.go", "package mod\nfunc helper() {\n}\n")
        self.head = self.commit()
        profile = self.write("profile.out", "mode: atomic\n"
                             "example.com/mod/cmd/main.go:2.13,3.2 0 0\n"
                             "example.com/mod/a_test.go:2.15,3.2 0 0\n")
        code, out, _ = self.run_main("--go-profile", str(profile), "--module", "mod")
        self.assertEqual(code, 0)
        self.assertIn("no added statement lines", out)

    def test_istanbul_json_for_web(self):
        self.write("web/src/a.ts", "export const a = 1;\nexport const b = 2;\n")
        self.head = self.commit()
        source = str(self.root / "web/src/a.ts")
        for hits, want in ((0, 1), (1, 0)):
            with self.subTest(hits=hits):
                report = self.write("web/coverage/coverage-final.json", json.dumps({source: {
                    "path": source,
                    "statementMap": {"0": {"start": {"line": 2, "column": 0},
                                           "end": {"line": 2, "column": 20}}},
                    "s": {"0": hits},
                }}))
                code, out, _ = self.run_main("--vitest-json", str(report))
                self.assertEqual(code, want, out)

    def test_threshold_comes_from_testcoverage_yml(self):
        self.assertEqual(go_config(self.root / "mod"), (80.0, ["^cmd/"]))

    def test_threshold_comes_from_vitest_config(self):
        self.assertEqual(vitest_config(self.root / "web"),
                         (92.0, ["src/main.tsx", "**/*.test.ts", "**/*.test.tsx"]))

    def test_missing_threshold_fails(self):
        self.write("mod/.testcoverage.yml", "threshold:\n  file: 0\n")
        with self.assertRaises(ValueError):
            go_config(self.root / "mod")
        code, out, _ = self.run_main("--go-profile", self.go_change(1), "--module", "mod")
        self.assertEqual(code, 1)
        self.assertIn("::error::", out)
        self.write("web/vitest.config.ts", "export default defineConfig({ test: {} });\n")
        with self.assertRaises(ValueError):
            vitest_config(self.root / "web")

    def test_share_at_threshold_passes(self):
        added = {"f.go": {1, 2}}
        statements = {"f.go": {1: True, 2: False}}
        self.assertEqual(check(added, statements, 50, lambda _: False)[0], 0)
        self.assertEqual(check(added, statements, 51, lambda _: False)[0], 1)

    def test_brace_expansion(self):
        self.assertEqual(expand("**/*.test.{ts,tsx}"), ["**/*.test.ts", "**/*.test.tsx"])
        self.assertEqual(expand("src/main.tsx"), ["src/main.tsx"])


if __name__ == "__main__":
    unittest.main()
