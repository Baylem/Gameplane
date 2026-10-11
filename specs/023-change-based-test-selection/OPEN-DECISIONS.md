# Open Decisions: 023-change-based-test-selection

Unsettled items. Do not treat any of the below as decided until a maintainer rules on it here.

## Settled (user, 2026-10-09)
OD-1 (FR-009): module coverage minimums are enforced on full runs only (default-branch pushes, forced full runs); partial PR runs check coverage of the lines they changed.
OD-2 (FR-010): every selected end-to-end suite keeps running on both amd64 and arm64 on PRs.
OD-3 (US3, FR-014): each shared CI config file runs only the jobs it feeds plus workflow validation; only the selection script and its tests, Makefile, go.work/go.work.sum and composite actions still force the full suite.

## Settled (user, 2026-10-11)
OD-4 (Principle I): a Go e2e test is required for this feature even though it is CI-only. Plan research R13 defines it: `TestCISelection_*` in `test/e2e/`, registered in the `operator` bucket.
OD-5 (FR-009): changed lines on a partial run must meet the module's own coverage minimum (`total` from its `.testcoverage.yml`; web uses the Vitest `lines` threshold, 92).
OD-6 (plan R2): a selected e2e bucket runs only its affected tests; full runs still run every test of every bucket.

## OD-7 (plan R9, FR-007): how to force a full run on a PR — **Open**
Plan default: the label `ci: full-run` (read on each run, no `labeled` trigger, because auto-label would start extra runs) plus a manual `workflow_dispatch`. Options: (a) label + manual run; (b) label only; (c) manual run only.

## OD-8 (plan R4): how Go package dependencies are found — **Open**
Plan default: scan import lines in Python, so the `changes` job stays Go-free (~1 min). Options: (a) import scanning; (b) `go list -deps -test` in the `changes` job (exact, but installs Go and downloads modules there).

## OD-9 (plan R6, R7): dashboard checks that stay whole when web changes — **Open**
Plan default: on a partial run only Vitest is narrowed (`vitest related`); ESLint, `tsc` build and the Playwright mock suite still run whole. Options: (a) as planned; (b) also narrow Playwright mock to specs touching the changed screens.

## OD-10 (plan Phasing): rollout shape — **Open**
Plan default: five PRs (summary table first, then e2e, unit, CI-config mapping, docs). Options: (a) five PRs; (b) one PR.
