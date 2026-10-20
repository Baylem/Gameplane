"""Behavioral coverage for CI module and image selection (run in CI)."""

import json
import tempfile
import unittest
from pathlib import Path
from unittest.mock import patch
from subprocess import CompletedProcess

from ci_scope import (E2E_BUCKETS, FORCE_ALL_FILES, affected_modules, bot_matrix, changed_paths,
                      full_scope_reason, image_inputs, main, publish_paths, select_images, summarize)


class ScopeTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.write("go.work", "go 1.26.0\nuse (\n ./lib\n ./api\n ./test/e2e\n ./other\n)\n")
        for name in ("lib", "api", "test/e2e", "other"):
            self.write(f"{name}/go.mod", f"module example.com/{name}\ngo 1.26.0\n")
        self.write("api/main.go", 'package api\nimport "example.com/lib/helpers"\n')
        self.write("test/e2e/go.mod", "module example.com/test/e2e\nrequire example.com/api v0.0.0\n")

    def write(self, name, content):
        path = self.root / name
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(content, encoding="utf-8")

    def test_changed_library_selects_transitive_consumers_including_workspace_imports(self):
        self.assertEqual(affected_modules(self.root, ["lib/helpers/file.go"]),
                         ["api", "lib", "test/e2e"])

    def test_deleted_source_still_selects_its_module_and_consumers(self):
        self.assertEqual(affected_modules(self.root, ["lib/deleted.go"]),
                         ["api", "lib", "test/e2e"])

    def test_crd_files_select_api_envtests_and_transitive_consumers(self):
        self.write("go.work", "go 1.26.0\nuse (\n ./lib\n ./api\n ./test/e2e\n ./other\n ./operator\n)\n")
        self.write("operator/go.mod", "module example.com/operator\ngo 1.26.0\n")
        for path in ("operator/config/crd", "operator/config/crd/bases/gameplane.io_gameservers.yaml"):
            with self.subTest(path=path):
                self.assertEqual(affected_modules(self.root, [path]),
                                 ["api", "operator", "test/e2e"])

    def test_module_gitlink_and_files_select_validator_and_transitive_consumers(self):
        self.write("go.work", "go 1.26.0\nuse (\n ./lib\n ./api\n ./test/e2e\n ./other\n ./gp-module\n)\n")
        self.write("gp-module/go.mod", "module example.com/gp-module\ngo 1.26.0\n")
        self.write("api/module_test.go", 'package api\nimport "example.com/gp-module/validator"\n')
        for path in ("modules", "modules/cs2/template.yaml", "modules/cs2/README.md"):
            with self.subTest(path=path):
                self.assertEqual(affected_modules(self.root, [path]),
                                 ["api", "gp-module", "test/e2e"])
        self.assertEqual(affected_modules(self.root, ["modules-other/template.yaml"]), [])

    def test_docs_only_does_not_select_go_jobs(self):
        self.assertEqual(affected_modules(self.root, ["CLAUDE.md", "api/specs.md"]), [])

    def test_unrelated_module_stays_isolated(self):
        self.assertEqual(affected_modules(self.root, ["other/source.go"]), ["other"])

    def test_shared_inputs_and_dependency_changes_select_every_module(self):
        for path in ("go.work", "go.work.sum", "api/go.mod", "api/go.sum",
                     "Makefile", ".golangci.yml", ".github/workflows/ci.yaml",
                     ".github/actions/go-cache/action.yml", "hack/ci_scope.py",
                     "test/e2e/buckets.sh", "web/Dockerfile"):
            with self.subTest(path=path):
                self.assertEqual(affected_modules(self.root, [path]),
                                 ["api", "lib", "other", "test/e2e"])

    def test_unknown_diff_fails_open(self):
        self.assertIsNone(changed_paths(self.root, "missing-base", "HEAD"))
        self.assertEqual(affected_modules(self.root, None), ["api", "lib", "other", "test/e2e"])

    def test_master_push_runs_full_scope_even_for_docs_only_changes(self):
        output = self.root / "output"
        with patch("ci_scope.ROOT", self.root), patch("sys.argv", ["ci_scope.py", "modules"]), \
                patch.dict("os.environ", {"GITHUB_EVENT_NAME": "push", "GITHUB_REF": "refs/heads/master",
                                          "GITHUB_OUTPUT": str(output)}), \
                patch("ci_scope.changed_paths", return_value=["README.md"]):
            main()
        values = {key: json.loads(value) for key, value in
                  (line.split("=", 1) for line in output.read_text(encoding="utf-8").splitlines())}
        self.assertEqual(values, {
            "all": True, "modules": ["api", "lib", "other", "test/e2e"],
            "lint": True, "go": True, "e2e-unit": True,
            "lint-exclude": [], "go-exclude": [],
        })

    def test_non_master_push_events_keep_docs_only_scope_empty(self):
        for event, ref in (("pull_request", "refs/heads/master"),
                           ("push", "refs/heads/feature"), ("push", "refs/tags/master")):
            with self.subTest(event=event, ref=ref):
                output = self.root / "output"
                output.write_text("", encoding="utf-8")
                with patch("ci_scope.ROOT", self.root), patch("sys.argv", ["ci_scope.py", "modules"]), \
                        patch.dict("os.environ", {"GITHUB_EVENT_NAME": event, "GITHUB_REF": ref,
                                                  "GITHUB_OUTPUT": str(output)}), \
                        patch("ci_scope.changed_paths", return_value=["README.md"]):
                    main()
                values = {key: json.loads(value) for key, value in
                          (line.split("=", 1) for line in output.read_text(encoding="utf-8").splitlines())}
                self.assertEqual(values, {
                    "all": False, "modules": [], "lint": False, "go": False, "e2e-unit": False,
                    "lint-exclude": [{"module": "api"}, {"module": "lib"},
                                     {"module": "other"}, {"module": "test/e2e"}],
                    "go-exclude": [{"module": "api"}, {"module": "lib"}, {"module": "other"}],
                })

    def test_docker_copy_sources_select_only_affected_images(self):
        self.write("api/Dockerfile", "FROM golang\nCOPY lib/ ./lib/\nCOPY api/go.mod api/go.sum ./api/\nCOPY --from=build /out/api /api\n")
        self.write("other/Dockerfile", 'FROM golang\nCOPY ["other/", "/src/"]\n')
        images = [{"component": "api"}, {"component": "other"}]
        self.assertEqual(select_images(self.root, images, ["lib/deleted.go"]), [images[0]])
        self.assertEqual(select_images(self.root, images, ["other/new.go"]), [images[1]])
        self.assertEqual(select_images(self.root, images, ["docs/install.md"]), [])

    def test_image_configuration_or_unknown_diff_rebuilds_every_image(self):
        images = [{"component": "api"}, {"component": "other"}]
        for paths in (None, ["go.work"], [".dockerignore"], ["signing/cosign.pub"],
                      [".github/workflows/publish-edge.yaml"], ["hack/ci_scope.py"]):
            with self.subTest(paths=paths):
                self.assertEqual(select_images(self.root, images, paths), images)

    def test_explicit_dockerfile_and_continuations(self):
        self.write("tunnel/Dockerfile.frp", "FROM golang\nCOPY --chown=1:1 lib/go.mod \\\n lib/go.sum /src/\nCOPY tunnel/ /src/\n")
        self.assertEqual(image_inputs(self.root / "tunnel/Dockerfile.frp"), {"lib", "tunnel"})

    def test_unclassifiable_dockerfile_fails_open(self):
        self.write("other/Dockerfile", "FROM golang\nCOPY ${SOURCE} /src/\n")
        images = [{"component": "other"}]
        self.assertEqual(select_images(self.root, images, ["lib/source.go"]), images)

    def test_bot_shards_preserve_every_input_test_exactly_once(self):
        names = ["TestMinecraft", "TestTerraria", "TestNewGame", "TestWake"]
        self.assertEqual(bot_matrix(names), {"test": names})
        for invalid in ([], ["TestMinecraft", "TestMinecraft"], ["Test.*"], ["TestA\nTestB"]):
            with self.subTest(invalid=invalid), self.assertRaises(ValueError):
                bot_matrix(invalid)

    def test_publish_uses_successful_baseline_and_includes_reverted_intermediate_changes(self):
        base, head = "a" * 40, "b" * 40
        replies = [CompletedProcess([], 0, '{"workflow_runs":[{"head_sha":"' + base + '"}]}'),
                   CompletedProcess([], 0, b""),
                   CompletedProcess([], 0, b"api/main.go\0api/main.go\0web/src/App.tsx\0")]
        with patch("ci_scope.subprocess.run", side_effect=replies) as run:
            paths = publish_paths(self.root, "owner/repo", "master", "push", head)
        self.write("api/Dockerfile", "FROM golang\nCOPY api/ /src/\n")
        self.write("web/Dockerfile", "FROM node\nCOPY web/ /src/\n")
        images = [{"component": "api"}, {"component": "web"}]
        self.assertEqual(select_images(self.root, images, paths), images)
        # These arguments are the external API contract: cancelled runs cannot
        # be a baseline and net-zero reverts cannot hide partially moved tags.
        self.assertIn("status=success", run.call_args_list[0].args[0])
        self.assertIn("log", run.call_args_list[-1].args[0])
        self.assertIn(f"{base}..{head}", run.call_args_list[-1].args[0])

    def test_manual_publish_and_missing_successful_baseline_build_everything(self):
        self.assertIsNone(publish_paths(self.root, "owner/repo", "master", "workflow_dispatch", "a" * 40))
        with patch("ci_scope.subprocess.run", return_value=CompletedProcess([], 0, '{"workflow_runs":[]}')):
            self.assertIsNone(publish_paths(self.root, "owner/repo", "master", "push", "a" * 40))


def make_go_workspace(test_case):
    test_case.temp = tempfile.TemporaryDirectory()
    test_case.addCleanup(test_case.temp.cleanup)
    test_case.root = Path(test_case.temp.name)
    write_file(test_case.root, "go.work", "go 1.26.0\nuse (\n ./lib\n ./api\n ./test/e2e\n ./other\n)\n")
    for name in ("lib", "api", "test/e2e", "other"):
        write_file(test_case.root, f"{name}/go.mod", f"module example.com/{name}\ngo 1.26.0\n")
    write_file(test_case.root, "test/e2e/go.mod", "module example.com/test/e2e\nrequire example.com/api v0.0.0\n")


def write_file(root, name, content):
    path = root / name
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(content, encoding="utf-8")


ALL_MODULES = ["api", "lib", "other", "test/e2e"]
FULL_VALUES = {
    "all": True, "modules": ALL_MODULES, "lint": True, "go": True, "e2e-unit": True,
    "lint-exclude": [], "go-exclude": [],
}
EMPTY_INPUTS = {"SCOPE_OUTPUTS": "", "FILTER_OUTPUTS": "", "GATE_OUTPUTS": "", "BOT_MATRIX": ""}
GATE_KEYS = ("go", "web", "charts", "standalone", "e2e", "images", "weblive", "github", "specs",
             "docs", "capture_overhead", "submodules", "codegen", "kindsmoke")


class FullScopeTests(unittest.TestCase):
    def setUp(self):
        make_go_workspace(self)

    def read_output(self, output):
        return {key: json.loads(value) for key, value in
                (line.split("=", 1) for line in output.read_text(encoding="utf-8").splitlines())}

    def run_modules(self, env, changed=None):
        output = self.root / "output"
        output.write_text("", encoding="utf-8")
        with patch("ci_scope.ROOT", self.root), patch("sys.argv", ["ci_scope.py", "modules"]), \
                patch.dict("os.environ", {**env, "GITHUB_OUTPUT": str(output)}), \
                patch("ci_scope.changed_paths", return_value=changed) as changed_mock:
            main()
        return self.read_output(output), changed_mock

    def test_full_scope_reason_values(self):
        self.assertEqual(full_scope_reason({"GITHUB_EVENT_NAME": "push", "GITHUB_REF": "refs/heads/master"},
                                           ["README.md"]), "master push")
        self.assertIsNone(full_scope_reason({"GITHUB_EVENT_NAME": "push", "GITHUB_REF": "refs/heads/feature"},
                                            ["docs/install.md"]))
        self.assertEqual(full_scope_reason({"GITHUB_EVENT_NAME": "workflow_dispatch"}, ["README.md"]),
                         "manual run")
        self.assertEqual(full_scope_reason({"CI_FULL_RUN": "true"}, ["README.md"]), "label ci: full-run")
        self.assertEqual(full_scope_reason({}, None), "unknown base")
        self.assertIsNone(full_scope_reason({}, ["README.md"]))

    def test_selection_rule_changes_force_full_scope(self):
        for path in (*sorted(FORCE_ALL_FILES), ".github/actions/go-cache/action.yml"):
            with self.subTest(path=path):
                self.assertEqual(full_scope_reason({}, [path]), "selection-rule change")

    def test_event_reasons_take_precedence(self):
        master = {"GITHUB_EVENT_NAME": "push", "GITHUB_REF": "refs/heads/master", "CI_FULL_RUN": "true"}
        self.assertEqual(full_scope_reason(master, None), "master push")
        manual = {"GITHUB_EVENT_NAME": "workflow_dispatch", "CI_FULL_RUN": "true"}
        self.assertEqual(full_scope_reason(manual, None), "manual run")
        self.assertEqual(full_scope_reason({"CI_FULL_RUN": "true"}, None), "label ci: full-run")

    def test_modules_label_runs_full_scope_without_diff(self):
        values, changed = self.run_modules(
            {"GITHUB_EVENT_NAME": "pull_request", "GITHUB_REF": "refs/pull/7/merge", "CI_FULL_RUN": "true"},
            changed=["README.md"])
        changed.assert_not_called()
        self.assertEqual(values, FULL_VALUES)

    def test_modules_manual_run_runs_full_scope_without_diff(self):
        values, changed = self.run_modules(
            {"GITHUB_EVENT_NAME": "workflow_dispatch", "GITHUB_REF": "refs/heads/feature"},
            changed=["README.md"])
        changed.assert_not_called()
        self.assertEqual(values, FULL_VALUES)

    def test_selection_exception_falls_back_to_every_module(self):
        output = self.root / "output"
        output.write_text("", encoding="utf-8")
        with patch("ci_scope.ROOT", self.root), patch("sys.argv", ["ci_scope.py", "modules"]), \
                patch.dict("os.environ", {"GITHUB_EVENT_NAME": "pull_request",
                                          "GITHUB_REF": "refs/pull/7/merge", "CI_FULL_RUN": "false",
                                          "GITHUB_OUTPUT": str(output)}), \
                patch("ci_scope.changed_paths", return_value=["web/src/App.tsx"]), \
                patch("ci_scope.affected_modules", side_effect=RuntimeError("boom")):
            main()
        self.assertEqual(self.read_output(output), FULL_VALUES)

    def test_summary_on_master_push_is_full_scope(self):
        output = self.root / "output"
        summary = self.root / "summary.md"
        output.write_text("", encoding="utf-8")
        summary.write_text("", encoding="utf-8")
        with patch("ci_scope.ROOT", self.root), patch("sys.argv", ["ci_scope.py", "summary"]), \
                patch.dict("os.environ", {**EMPTY_INPUTS, "GITHUB_EVENT_NAME": "push",
                                          "GITHUB_REF": "refs/heads/master",
                                          "GITHUB_OUTPUT": str(output),
                                          "GITHUB_STEP_SUMMARY": str(summary)}), \
                patch("ci_scope.changed_paths") as changed:
            main()
        changed.assert_not_called()
        values = self.read_output(output)
        self.assertEqual(values["scope"], "full")
        self.assertEqual(values["reason"], "master push")
        self.assertIn("full: master push", summary.read_text(encoding="utf-8"))

    def test_summary_on_pull_request_is_partial_scope(self):
        output = self.root / "output"
        output.write_text("", encoding="utf-8")
        with patch("ci_scope.ROOT", self.root), patch("sys.argv", ["ci_scope.py", "summary"]), \
                patch.dict("os.environ", {**EMPTY_INPUTS, "GITHUB_EVENT_NAME": "pull_request",
                                          "GITHUB_REF": "refs/pull/7/merge", "CI_FULL_RUN": "false",
                                          "GITHUB_OUTPUT": str(output),
                                          "GITHUB_STEP_SUMMARY": str(self.root / "summary.md")}), \
                patch("ci_scope.changed_paths", return_value=["web/src/App.tsx"]):
            main()
        values = self.read_output(output)
        self.assertEqual(values["scope"], "partial")
        self.assertEqual(values["reason"], "")


class SelectionSummaryTests(unittest.TestCase):
    BOTS = ["TestMinecraft", "TestTerraria", "TestNewGame"]
    WEB_SCOPE = {"all": "false", "modules": "[]", "lint": "false", "go": "false", "e2e-unit": "false"}
    WEB_GATES = {"web": "true", "images": "true", "weblive": "true"}
    WEB_FILTERS = {"web": "true", "web_files": json.dumps(["web/src/App.tsx"])}

    def setUp(self):
        make_go_workspace(self)

    def summary(self, paths=None, reason=None, event="pull_request", scope=None, filters=None, gates=None):
        env = {
            "GITHUB_EVENT_NAME": event,
            "SCOPE_OUTPUTS": json.dumps(scope or {}),
            "FILTER_OUTPUTS": json.dumps(filters or {}),
            "GATE_OUTPUTS": json.dumps(gates or {}),
            "BOT_MATRIX": json.dumps({"test": self.BOTS}),
        }
        with patch("ci_scope.ROOT", self.root):
            return summarize(env, paths, reason)

    def web_only(self):
        return self.summary(paths=["web/src/App.tsx"], scope=self.WEB_SCOPE,
                            filters=self.WEB_FILTERS, gates=self.WEB_GATES)

    def test_rows_have_required_keys_and_unique_suites(self):
        rows, _ = self.web_only()
        for row in rows:
            with self.subTest(row=row):
                self.assertEqual(set(row), {"suite", "decision", "reason"})
                self.assertIn(row["decision"], {"run", "skip"})
        suites = [row["suite"] for row in rows]
        self.assertEqual(len(suites), len(set(suites)))

    def test_every_e2e_bucket_and_bot_test_has_a_row(self):
        rows, _ = self.web_only()
        suites = {row["suite"] for row in rows}
        self.assertEqual(len(E2E_BUCKETS), 8)
        for bucket in E2E_BUCKETS:
            name = "e2e api-auth + ratelimit tail" if bucket == "api-auth" else f"e2e {bucket}"
            self.assertIn(f"{name} (amd64, arm64)", suites)
        for test in self.BOTS:
            self.assertIn(f"e2e game bot / {test}", suites)

    def test_web_only_change_runs_web_and_skips_go_e2e(self):
        rows, _ = self.web_only()
        by_suite = {row["suite"]: row for row in rows}
        self.assertEqual(by_suite["web"],
                         {"suite": "web", "decision": "run", "reason": "changed: web/src/App.tsx"})
        self.assertEqual(by_suite["e2e operator (amd64, arm64)"],
                         {"suite": "e2e operator (amd64, arm64)", "decision": "skip",
                          "reason": "not affected"})

    def test_citation_lists_five_files_then_remainder(self):
        files = [f"web/src/f{i}.tsx" for i in range(7)]
        rows, _ = self.summary(paths=files, scope=self.WEB_SCOPE, gates=self.WEB_GATES,
                               filters={"web": "true", "web_files": json.dumps(files)})
        by_suite = {row["suite"]: row for row in rows}
        self.assertEqual(by_suite["web"]["reason"], "changed: " + ", ".join(files[:5]) + " (+2)")

    def test_partial_markdown_has_header_and_one_line_per_row(self):
        rows, md = self.web_only()
        lines = md.splitlines()
        self.assertEqual(lines[0], "### Test selection (partial)")
        self.assertEqual(lines[1], "")
        self.assertEqual(lines[2], "| Suite | Decision | Why |")
        self.assertEqual(lines[3], "|---|---|---|")
        self.assertEqual(len(lines) - 4, len(rows))
        self.assertTrue(md.endswith("\n"))
        self.assertNotIn("full: ", md)

    def test_full_scope_marks_run_rows_and_header(self):
        scope = {"all": "true", "modules": json.dumps(ALL_MODULES), "lint": "true",
                 "go": "true", "e2e-unit": "true"}
        gates = {key: "true" for key in GATE_KEYS}
        rows, md = self.summary(paths=None, reason="master push", event="push", scope=scope,
                                gates=gates, filters={})
        lines = md.splitlines()
        self.assertEqual(lines[0], "### Test selection (full)")
        self.assertIn("full: master push", lines)
        for row in rows:
            if row["decision"] == "run":
                self.assertTrue(row["reason"].startswith("full: "), row)
        skipped = {row["suite"] for row in rows if row["decision"] == "skip"}
        self.assertEqual(skipped, {"dependency review"})


if __name__ == "__main__":
    unittest.main()
