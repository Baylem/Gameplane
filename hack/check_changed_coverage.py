"""Changed-lines coverage gate for partial CI runs (spec 023 FR-009, OD-1, OD-5).

Reads added lines from `git diff -U0`, intersects them with a Go cover profile or a
Vitest (Istanbul) coverage-final.json, and fails when the covered share of added
statement lines is below the module's own threshold. Python stdlib only.
"""

import argparse
import fnmatch
import json
import os
import re
import subprocess
import sys
from pathlib import Path


ROOT = Path(__file__).resolve().parent.parent
PROFILE_LINE = re.compile(r"^(.+):(\d+)\.\d+,(\d+)\.\d+ (\d+) (\d+)$")
HUNK = re.compile(r"^@@ -\S+ \+(\d+)(?:,(\d+))? @@")


def added_lines(root, base, head, prefix):
    """Return {repo path: set of added line numbers} for files under prefix."""
    diff = subprocess.run(
        ["git", "diff", "-U0", "--no-color", "--no-renames", base, head, "--", prefix],
        cwd=root, check=True, capture_output=True, text=True).stdout
    lines, current = {}, None
    for line in diff.splitlines():
        if line.startswith("+++ "):
            target = line[4:]
            current = target[2:] if target.startswith("b/") else None
        elif current and line.startswith("@@"):
            match = HUNK.match(line)
            if match:
                start, count = int(match[1]), int(match[2] or "1")
                lines.setdefault(current, set()).update(range(start, start + count))
    return lines


def expand(glob):
    """Expand {a,b} alternatives in a glob into plain fnmatch patterns."""
    match = re.search(r"\{([^{}]*)\}", glob)
    if not match:
        return [glob]
    return [pattern for option in match[1].split(",")
            for pattern in expand(glob[:match.start()] + option + glob[match.end():])]


def go_config(module_dir):
    """Return (threshold.total, exclude.paths regexes) from a module's .testcoverage.yml."""
    text = (module_dir / ".testcoverage.yml").read_text(encoding="utf-8")
    total, excludes, section = None, [], None
    for raw in text.splitlines():
        line = raw.split("#", 1)[0].rstrip()
        if not line.strip():
            continue
        if not line.startswith((" ", "-")):
            section = line.split(":", 1)[0].strip()
            continue
        item = line.strip()
        if section == "threshold" and item.startswith("total:"):
            total = float(item.split(":", 1)[1])
        elif section == "exclude" and item.startswith("- "):
            excludes.append(item[2:].strip().strip("'\""))
    if total is None:
        raise ValueError(f"{module_dir.name}/.testcoverage.yml has no threshold.total")
    return total, excludes


def vitest_config(web_dir):
    """Return (coverage.thresholds.lines, expanded coverage.exclude globs) from vitest.config.ts."""
    text = (web_dir / "vitest.config.ts").read_text(encoding="utf-8")
    start = text.find("coverage:")
    coverage = text[start:] if start >= 0 else ""
    lines = re.search(r"thresholds:\s*\{[^}]*?\blines:\s*(\d+(?:\.\d+)?)", coverage)
    if not lines:
        raise ValueError("web/vitest.config.ts has no coverage.thresholds.lines")
    block = re.search(r"\bexclude:\s*\[(.*?)\]", coverage, re.DOTALL)
    globs = re.findall(r'"([^"]+)"', block[1]) if block else []
    return float(lines[1]), [pattern for glob in globs for pattern in expand(glob)]


def mark(lines, first, last, covered):
    """Mark lines first..last; a line is covered when any block covering it ran."""
    for number in range(first, last + 1):
        lines[number] = lines.get(number, False) or covered


def go_statements(profile, module_path, module):
    """Return {repo path: {line: covered}} from a Go cover profile."""
    result = {}
    for line in Path(profile).read_text(encoding="utf-8").splitlines():
        match = PROFILE_LINE.match(line)
        if not match or not match[1].startswith(module_path + "/"):
            continue
        path = module + "/" + match[1][len(module_path) + 1:]
        mark(result.setdefault(path, {}), int(match[2]), int(match[3]), int(match[5]) > 0)
    return result


def vitest_statements(report, root):
    """Return {repo path: {line: covered}} from an Istanbul coverage-final.json."""
    data = json.loads(Path(report).read_text(encoding="utf-8"))
    result = {}
    for name, entry in data.items():
        path = Path(entry.get("path", name))
        if not path.is_absolute():
            path = root / path
        try:
            relative = path.resolve().relative_to(root.resolve()).as_posix()
        except ValueError:
            continue
        lines = result.setdefault(relative, {})
        counts = entry.get("s", {})
        for key, location in entry.get("statementMap", {}).items():
            mark(lines, location["start"]["line"], location["end"]["line"],
                 counts.get(key, 0) > 0)
    return result


def check(added, statements, threshold, excluded):
    """Return (exit code, one-line result, uncovered file:line list)."""
    total, missing = 0, []
    for path in sorted(added):
        if excluded(path):
            continue
        known = statements.get(path, {})
        for number in sorted(added[path]):
            if number in known:
                total += 1
                if not known[number]:
                    missing.append(f"{path}:{number}")
    if total == 0:
        return 0, "changed-lines coverage: no added statement lines, passes", []
    covered = total - len(missing)
    share = 100 * covered / total
    line = (f"changed-lines coverage: {covered}/{total} added statement lines covered "
            f"({share:.1f}%, minimum {threshold:g}%)")
    return (0 if share >= threshold else 1), line, missing


def go_inputs(root, module, profiles, base, head):
    """Return (threshold, added lines, statements, excluded) for one Go module."""
    threshold, patterns = go_config(root / module)
    text = (root / module / "go.mod").read_text(encoding="utf-8")
    match = re.search(r"^module\s+(\S+)", text, re.MULTILINE)
    if not match:
        raise ValueError(f"{module}/go.mod has no module directive")
    statements = {}
    for profile in profiles:
        for path, lines in go_statements(profile, match[1], module).items():
            merged = statements.setdefault(path, {})
            for number, covered in lines.items():
                mark(merged, number, number, covered)

    def excluded(path):
        """Skip tests, non-Go files and the module's own coverage exclusions."""
        relative = path[len(module) + 1:]
        directory = relative.rpartition("/")[0]
        return (not relative.endswith(".go") or relative.endswith("_test.go")
                or any(re.search(p, relative) or re.search(p, directory) for p in patterns))

    return threshold, added_lines(root, base, head, module), statements, excluded


def web_inputs(root, report, base, head):
    """Return (threshold, added lines, statements, excluded) for the dashboard."""
    threshold, globs = vitest_config(root / "web")

    def excluded(path):
        """Skip tests, non-TypeScript files and vitest.config.ts coverage.exclude."""
        relative = path[len("web/"):]
        return (not relative.endswith((".ts", ".tsx"))
                or relative.endswith((".test.ts", ".test.tsx"))
                or any(fnmatch.fnmatch(relative, glob) for glob in globs))

    return (threshold, added_lines(root, base, head, "web/src"),
            vitest_statements(report, root), excluded)


def main(argv=None):
    """Parse arguments, run the gate and return the exit code."""
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--base", required=True)
    parser.add_argument("--head", required=True)
    parser.add_argument("--go-profile", action="append", default=[])
    parser.add_argument("--module")
    parser.add_argument("--vitest-json")
    parser.add_argument("--root", default=str(ROOT))
    args = parser.parse_args(argv)
    root = Path(args.root)
    if not args.vitest_json and not (args.go_profile and args.module):
        parser.error("pass --vitest-json, or --go-profile with --module")
    try:
        if args.vitest_json:
            inputs = web_inputs(root, args.vitest_json, args.base, args.head)
        else:
            inputs = go_inputs(root, args.module.strip("/"), args.go_profile, args.base,
                               args.head)
    except (OSError, ValueError, KeyError, subprocess.CalledProcessError) as exc:
        print(f"::error::changed-lines coverage: {exc}")
        return 1
    threshold, added, statements, excluded = inputs
    code, line, missing = check(added, statements, threshold, excluded)
    print(line)
    for item in missing:
        print(f"uncovered: {item}")
    summary = os.environ.get("GITHUB_STEP_SUMMARY")
    if summary:
        with open(summary, "a", encoding="utf-8") as output:
            output.write(line + "\n")
    return code


if __name__ == "__main__":
    sys.exit(main())
