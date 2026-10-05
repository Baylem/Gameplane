# Review: .github/actions/

- **Date**: 2026-10-04
- **Reviewer tier**: sonnet (verification pending)
- **Checked against**: `.github/dependabot.yml`, `.github/zizmor.yml`, `docker-bake.hcl`, `specs/done_008-hardened-github-actions/spec.md` (FR-001 to FR-021), `docs/contributing.md` ("CI and workflows")

## Scope reviewed

Read in full:
- `.github/actions/build-e2e-images/action.yml`, `dump-cluster-state/action.yml`, `e2e-images/action.yml`, `go-cache/action.yml`
- every caller in `.github/workflows/ci.yaml` (`build-images`, `build-images-arm64`, the `go-cache` calls, the `e2e-images` loads and the seven `dump-cluster-state` calls)

Read in part, to cross-check a claim:
- `docker-bake.hcl` (target names, tags, groups)
- `.github/dependabot.yml` (every `package-ecosystem` and `directory` line)
- `.github/workflows/{images,release,publish-edge}.yaml` (action pins only)
- `go.work` (Go version)

Not reviewed: no action was run. The Dependabot behaviour below comes from its documented directory rules, not from a Dependabot run.

## Method

1. Read each action top to bottom, then each caller's `with:` inputs against the action's inputs and defaults.
2. Compared every third-party `uses:` SHA and version comment in the four actions with the same action's pin in the workflows.
3. Checked which directories `dependabot.yml` lists for the `github-actions` ecosystem.
4. Compared the bake group and tags with the `docker save` list and the `gameplane-test/*:e2e` names.

## Observations (no finding)

- Every `uses:` in the four actions is a full commit SHA with a version comment (spec 008 SC-001).
- `go-cache` takes `key-suffix` as an input and has no matrix expression, as its description requires. Its default `go-version: '1.26'` is compatible with `go.work` (`go 1.26.0`).
- `build-e2e-images` bakes the six-target `e2e` group plus `e2e-gameprobe`, and the seven tags in `docker save` match the `docker-bake.hcl` tags.
- `dump-cluster-state` never fails the job: every command is `|| true` or the step ends `exit 0`.
- The redaction regex is identical in all six steps, so a fix to one is a fix to all only if all six are edited. Contents of the redaction are covered by F-006/F-013 and not re-reviewed here.

## Candidate findings

### C-github-actions-01: Dependabot never updates the third-party actions pinned inside `.github/actions/*`

- **Location**: `.github/dependabot.yml:416-417` (the only `github-actions` entry, `directory: "/"`), compared with the pins at `.github/actions/build-e2e-images/action.yml:18,20,39,59`, `e2e-images/action.yml:16`, `go-cache/action.yml:22,27`
- **Category**: release-pipeline / maintenance
- **Suggested severity**: S3
- **Observation**:
  1. The `github-actions` ecosystem has one entry with `directory: "/"`. For a composite action, Dependabot only scans a `.github/actions/<name>` directory when it is listed.
  2. The pins have already drifted from the workflows: `docker/setup-buildx-action` is `v4.1.0` (`d7f5e7f5…`) at `build-e2e-images/action.yml:18` but `v4.4.1` (`f87e5991…`) in `ci.yaml:349`, `images.yaml:60,220`, `publish-edge.yaml:101` and `release.yaml:57`. `actions/upload-artifact` is `v7.0.0` (`bbbca2dd…`) at `build-e2e-images/action.yml:59` but `v7.0.1` (`043fb46d…`) in every workflow.
  3. Four more pins (`docker/bake-action`, `actions/download-artifact`, `actions/setup-go`, `actions/cache`) are likely to drift the same way. Only `actions/cache` has a copy in the workflows to compare (`v6.1.0`, equal today).
- **Expected**: Spec 008 SC-001 pins every action by SHA. A pin that no bot bumps stays at its old version until someone notices.
- **Actual**: Pins inside the composite actions are frozen. They are used by every e2e job and by every Go lint and test leg.

### C-github-actions-02: `dump-cluster-state` hard-codes `gameplane-system` for operator/API logs and ignores the `namespaces` input

- **Location**: `.github/actions/dump-cluster-state/action.yml:71,74,89,103,117`
- **Category**: correctness
- **Suggested severity**: S4
- **Observation**: The operator, API, helm-history and jobs steps always use `-n gameplane-system`. The `namespaces` input only affects the pod describe step and the game-pod log step. The latter skips `gameplane-system` on purpose (`:141-153`). A caller that passes only `ingress-nginx` (`ci.yaml:1530`) therefore gets `describe pods` for ingress-nginx but no container logs for its pods. A caller that passes nothing (multicluster A and B, `ci.yaml:1273-1281`) gets no game-pod logs and no logs for any pod outside `gameplane-system`, even though the multicluster bucket creates game pods.
- **Expected**: The input description says "Comma-separated list of namespaces to describe pods in". That is true, but the logs for the same pods are a gap on exactly the failures this step exists to diagnose.
- **Actual**: A failing multicluster or ingress job has less diagnostic output than the sibling jobs. No data is lost silently: the step still exits 0.

### C-github-actions-03: `dump-cluster-state` prints a `--previous` error for every container that has never restarted

- **Location**: `.github/actions/dump-cluster-state/action.yml:74,187,209`
- **Category**: diagnostics quality
- **Suggested severity**: S4
- **Observation**: `kubectl logs --previous` exits non-zero with "previous terminated container … not found" for a container that has not restarted. `2>&1 | redact` sends that text into the log. The game-pod step does this for every container of every pod, so a failed run buries the real logs in dozens of identical error lines.
- **Expected**: Check `.status.containerStatuses[].restartCount` or `.lastState` before asking for `--previous`.
- **Actual**: Noise only. No failure.

### C-github-actions-04: The same 3-line `redact()` function is pasted into six steps

- **Location**: `.github/actions/dump-cluster-state/action.yml:40-45,63-68,81-86,95-100,109-114,124-129`
- **Category**: maintainability
- **Suggested severity**: S4
- **Observation**: Six byte-identical copies of the `sed` redactor. F-006/F-013 already needed a fix in this function, and each future change needs six matching edits. A copy that is missed redacts less than its neighbours with no signal.
- **Expected**: One definition (a small script file in the action directory, or a single step that writes a function file to `$RUNNER_TEMP` and each later step sources).
- **Actual**: Six copies.

### C-github-actions-05: The `e2e-images` loader has no fallback when the artifact is missing, contradicting its own description

- **Location**: `.github/actions/e2e-images/action.yml:6-7,16-19`
- **Category**: correctness / documentation
- **Suggested severity**: S4
- **Observation**: The description says "`e2e.sh`'s build-fallback still covers a missing image". But `actions/download-artifact` fails the job when the named artifact does not exist, and nothing sets `continue-on-error`. So a missing or expired artifact never reaches `e2e.sh`. Today every job that loads the tar `needs` the build job and shares its `images` gate, so the case does not arise in CI. The comment would mislead a maintainer who relies on it.
- **Expected**: Either the comment drops the claim, or the download step tolerates a missing artifact.
- **Actual**: Stale comment.

### C-github-actions-06: `docker-bake.hcl` header names the wrong action and the wrong target count

- **Location**: `docker-bake.hcl:1-3`, compared with `.github/actions/build-e2e-images/action.yml:2-9`
- **Category**: documentation
- **Suggested severity**: S4
- **Observation**: The header says the file is "used by .github/actions/e2e-images" (that action only downloads and loads the tar; `build-e2e-images` runs bake). It also says `bake --load e2e` builds "all six", while the action's own description says "all seven" (the seventh, gameprobe, is baked in a second step outside the group).
- **Expected**: Header points at `build-e2e-images`.
- **Actual**: Stale reference.

### C-github-actions-07: `upload-artifact` for the image tar uses the default `if-no-files-found: warn`

- **Location**: `.github/actions/build-e2e-images/action.yml:59-63`
- **Category**: correctness
- **Suggested severity**: S4
- **Observation**: If `docker save` wrote to the wrong path the upload would only warn. The consumers then fail at `download-artifact`, one job away from the cause, in all nine heavy e2e jobs. `docker save` runs under `bash` with default `-e` so a save failure fails first. The residual case is a path mismatch between the two steps, which are 4 lines apart, so the risk is low. Other uploads in the workflows set `if-no-files-found: warn` explicitly (`ci.yaml:1410`), so this one is the only one that relies on the default.
- **Expected**: `if-no-files-found: error`.
- **Actual**: Default (`warn`).
