# Implementation Plan: Split big packages and files

**Branch**: `claude/project-thread-oxpa6z` | **Date**: 2026-10-20 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `specs/028-split-big-packages/spec.md`

## Summary

Apply spec 027's layout to the code with behaviour-preserving moves only. Two Go packages
of 146 and 150 files become one sub-package per resource (OD-2); web screens move to
`web/src/pages/` and anything over 500 lines is split into sections (027 OD-4); the five
small Go services get `cmd/main.go` + `internal/<concern>/` (027 OD-3); every other
production file over 500 lines is split inside its package (OD-3). A last PR adds
`hack/check_layout.py` to `make lint` (027 OD-5, OD-4). Each PR is one package, screen
group or service, made by a reviewed script from a scout's move list, proven by a full CI
run, a test-name parity check and a test-edit check. Order (OD-1): web, api, operator,
then the services. Details: [research.md](./research.md).

## Technical Context

**Language/Version**: Go (go.work, 17 modules); TypeScript strict, React 19 (web); Python 3 stdlib for the two new `hack/` scripts

**Primary Dependencies**: none added. Moves use `git mv`, `gofmt -r`/`gopls rename` for exported identifiers, and plain text edits for imports

**Storage**: N/A

**Testing**: existing suites only, run in CI with `ci: full-run`: Go unit + envtest, every e2e bucket on amd64 and arm64, Vitest, Playwright mock, visual-diff, screenshots, image builds, lint. New unit tests only for the two new scripts (`hack/test_move_parity.py`, `hack/test_check_layout.py`)

**Target Platform**: unchanged (Kubernetes; browser dashboard)

**Project Type**: refactor inside the monorepo

**Performance Goals**: no runtime change. CI: envtest job time grows by at most 2 minutes per module from extra `TestMain`s (R5), else stop and ask

**Constraints**: FR-001/FR-002 (move only), OD-6/OD-9 (test edits), FR-009 (coverage minimums), CLAUDE.md rules 8, 11–14, 16–18; `.pen` files never touched; no `//nolint`, `eslint-disable` or widened exclusions

**Scale/Scope**: ~30 PRs: web ≈ 9, api ≈ 10, operator ≈ 7, other-module file splits ≈ 7, services 5, layout check 1, docs/archive 1. Production files over 500 lines today: 34 Go, 17 web

## Constitution Check

| Principle | Status | Notes |
|---|---|---|
| **I. E2E-Tested Delivery** | Pending OD-10 (recommend exempt) | No behaviour changes, so there is no new user path. Every PR runs the full existing e2e suite with `ci: full-run`. If valgul rules otherwise, a task adds a test under `test/e2e/`. |
| **II. Design-First** | Pass | No visual change. Web splits must keep DOM identical (visual-diff); `design.pen` untouched. |
| **III. Best Practice** | Pass | No suppressions; path-based lint and coverage rules move with their file, same scope (R8). Go errors untouched. |
| **IV. Spec-Driven** | Pass | Spec, plan, tasks before code; every PR updates the touched module's `specs.md` paths (FR-015, 027 OD-2). Archive to `specs/done_028-…` after the last merge. |
| **V. Workflows & Subagents** | Pass | Per PR: scout (haiku) writes the move-list brief; one haiku agent runs the move script; reviewer one tier up (sonnet) checks the diff against the brief (rules 13, 17, 18). Web PRs add a sonnet browser smoke test. |
| **VI. CI Bears the Heavy Lifting** | Pass | Locally only `go build ./...`, `npx tsc --noEmit`, `python3 -m py_compile`. |

## Phasing

Each phase is a run of PRs; a PR starts after the previous one in its phase merges
(moves on the same tree conflict). Phase 0 first, then OD-1 order.

0. **Tooling PR**: `hack/move_parity.py` (`names`, `test-edits`) + `hack/test_move_parity.py`, wired into CI next to `make test-ci-scope`; the brief contract ([contracts/move-brief.md](./contracts/move-brief.md)).
1. **Web** (US4): `types.ts` split (OD-5); route files per OD-7; one PR per big screen (AdminSettings, CreateServer, ThemeSettings, Users, Servers, Backups, Dashboard); ServerDetail and its tabs per OD-8; one PR moving the remaining small screens; one PR for `lib/endpoints.ts` and the four big components.
2. **API** (US2): helper package first, then one PR per group in R3, leaf groups first (`mods`, `shares`, `capture`, `settings`, `modules`, `clusters`, `servers`, `access`); then split `api/cmd/main.go`; then `api/internal/audit`, `api/internal/auth/oidc.go`.
3. **Operator** (US3): helper package; `cluster`, `networkcapture`, `module`, `backup`, `gameserver` (largest last, after the guard coupling is resolved in the helper package); then `operator/cmd/main.go`.
4. **Other modules** (FR-018): `capture-sidecar/internal/httpserver`, `agent/internal/{mods,rcon,files,players}`, `i18n/cmd/i18n-check` file splits.
5. **Services** (US1): `audit-syslog-bridge`, `mcp-server`, `telemetry-receiver`, then `tunnel` and `sentinel` once #721–#723 and #729 merge.
6. **Layout check** (US5): `hack/check_layout.py` + tests + `make lint` target (R9).
7. **Close**: `docs/agent-architecture.md` sweep, archive the spec.

## Per-PR procedure

1. List open PRs touching the paths; if any, ask valgul whether to wait (R10).
2. Scout (haiku) writes `specs/028-split-big-packages/briefs/<pr>.md` per the brief contract: move list, identifiers to export, path references, `specs.md` lines, open-PR check.
3. One haiku agent runs the move script from the brief, then `go build ./...` / `npx tsc --noEmit` only.
4. `hack/move_parity.py names` and `test-edits` run locally (static scans, not tests) and in CI.
5. Reviewer (sonnet) checks `git diff -M` against the brief; fixes go to a new haiku agent.
6. Push, open PR with `type: refactor` + one `area:` label and `ci: full-run`; PR body carries the parity report ([contracts/parity-report.md](./contracts/parity-report.md)), no attribution block or session link.
7. Before calling it ready: CI green on both architectures, coverage gates green, every CodeRabbit comment answered.

Every helper agent's brief repeats the binding rules: CLAUDE.md applies; explicit `model:` starting at haiku; never call `mcp__hearthbot__` tools; never run test or lint suites locally; no test edits outside OD-6/OD-9; signed commits.

## Project Structure

### Documentation (this feature)

```text
specs/028-split-big-packages/
├── spec.md
├── OPEN-DECISIONS.md
├── plan.md
├── research.md
├── quickstart.md
├── contracts/
│   ├── move-brief.md
│   ├── parity-report.md
│   └── layout-check.md
├── briefs/            # one scout brief per PR, added during implementation
├── checklists/requirements.md
└── tasks.md
```

### Source Code (target, per 027 and OD-2..OD-8)

```text
api/internal/handlers/{capture,access,shares,clusters,modules,mods,servers,settings,<helper>}/
operator/internal/controller/{gameserver,backup,networkcapture,module,cluster,<helper>}/
web/src/pages/<Screen>.tsx, web/src/pages/<Screen>/<Section>.tsx
web/src/routes/        # per OD-7
{tunnel,sentinel,telemetry-receiver,audit-syslog-bridge,mcp-server}/cmd/main.go + internal/<concern>/
hack/move_parity.py, hack/check_layout.py (+ test_*.py)
```

## Complexity Tracking

| Choice | Why | Simpler option rejected because |
|---|---|---|
| New `TestMain` per Go sub-package | Required by Go once tests live in several packages | One shared envtest package would mean rewriting test setup (beyond OD-6) |
| Two new `hack/` scripts | Parity and test-edit proof must be mechanical for ~30 PRs | Manual review of every moved test file does not scale and is what rule 18 forbids |
