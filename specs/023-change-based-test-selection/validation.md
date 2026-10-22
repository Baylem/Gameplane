# Validation results

Results of the [quickstart](./quickstart.md) scenarios. Each scenario was a throwaway
draft PR based on PR 2's branch (`claude/project-thread-63yei6-pr2`, head `82b77ab`),
allowed by OD-11. The drafts were closed and their branches deleted after the check.

## T030: scenarios 1–4 and 12 (2026-10-22)

All five runs were queued for about 65 minutes behind other PRs' runs before any job
started, and every later job waited for a runner too. Wall time therefore says nothing
about SC-001 yet; it is measured again after rollout on an idle queue. Runner-minutes
are the sum of the durations of jobs that ran (success or failure).

| # | Draft PR | Change | Scope | Runner-min | Result |
|---|---|---|---|---|---|
| 1 | #780 | comment in `agent/internal/console/console.go` | partial | 170 | selection broader than the quickstart row, see below |
| 2 | #781 | comment in `sentinel/main.go` | partial | 61 | RBAC and auth suites skipped (US1-2 met for those); 11 operator tests ran |
| 3 | #782 | comment in `charts/gameplane/values.yaml` | partial, every suite selected | 83 (arm64 only) | every e2e suite selected, 78 of 78 operator tests (US1-3 met) |
| 4 | #783 | comment in `test/e2e/api_agent_e2e_test.go` | partial | 104 | `api-agent` ran 7 of 7 (US1-4 met) |
| 12 | #784 | new file `tools/x.sh` | full: `unmapped path tools/x.sh` | 146 (still running at close) | everything selected; `e2e bucket coverage` failed with `no selection rule maps these paths: ["tools/x.sh"]` (FR-012 met) |

Every summary listed each suite with a decision and a reason (SC-005 met).

### Selections in detail

**Scenario 1 (agent).** Selected: `operator` 55 of 78, `api-agent` 7 of 7, `api-mods`
6 of 7, `api-rbac` 2 of 8, `api-roles` 1 of 7, `optional-components` 1 of 1,
`multicluster` 2 of 6, `upgrade` 1 of 1, web live, all six game-bot joins. Skipped:
`api-auth`, `telemetry`, `ratelimit`, `postgres`. `capture-overhead` was selected in
the summary but its job stayed skipped, because its PR gate still uses the
`capture_overhead` path filter.

The quickstart row expects `upgrade`, `api-rbac`, `api-roles`, `api-mods` and
`capture-overhead` to skip, and spec US1-1 names multicluster and upgrade as examples
of skipped suites. They ran because `suite-components.json` maps `TestMultiCluster_`,
`TestUpgrade_`, `TestAPI_Lifecycle*`, `TestAPI_OwnerCollaboratorAccess` and the mod
tests to the `agent` component: each of them starts a game server, and the agent runs
in that pod. The rows and the mapping disagree; see "Open item" below.

**Scenario 2 (sentinel).** Selected: `operator` 11 of 78 (the ten `TestHelmInstall_`
tests, mapped to every component, plus `TestGameServer_IdleNeverSleepsWithoutAPlayerCount`)
and the three `WakeOnConnect` game-bot joins. Skipped: `api-auth`, `api-rbac`,
`api-roles`, `api-agent`, `api-mods`, `telemetry`, `ratelimit`, `multicluster`,
`upgrade`, web live, the other three bots. Spec US1-2 says the operator suite does not
run; 11 of its tests ran, because `TestHelmInstall_` is mapped to `*`.

**Scenario 3 (chart).** Every suite was selected with "changed:
charts/gameplane/values.yaml". The scope reads `partial` because the chart is a
`universal` path rather than a full-run trigger, which gives the same suites.

**Scenario 4 (e2e test file).** Selected: `api-agent` 7 of 7 plus the tests that share
the file's helpers: `operator` 16 of 78, `api-rbac` 2 of 8, `api-mods` 5 of 7,
`optional-components`, three game-bot joins.

**Scenario 12 (unmapped path).** `all: true`, reason `unmapped path tools/x.sh`,
every gate on, and the `e2e bucket coverage` job failed on the unmapped path as
designed.

### Failures unrelated to selection

- `lint` failed on scenarios 1, 2 and 4 with `gofmt`: the throwaway comment line was
  appended without a blank line. It does not affect the selection result.
- Scenario 2: `e2e operator / amd64` failed in `TestHelmInstall_MetricsNotOnPublicPort`
  (`helm_install_e2e_test.go:239`, metrics port not serving within 1m30s). The change
  was a comment in `sentinel/main.go`, and the same test passed on arm64 in the same run.
- Scenarios 3 and 12: the amd64 `build e2e images` job was cancelled at its 15-minute
  timeout (layer-cache miss), so only the arm64 e2e jobs ran.
- `TestCISelection_*` ran in scenarios 3 and 12 (operator, arm64) and passed in
  scenario 3; scenario 12's operator job was still running at close.

### OD-12

Scenarios 1 and 2 selected more than the quickstart rows and spec US1-1/US1-2 said, and
the cause is the agreed `suite-components.json` mapping, not a code defect. Settled as
OD-12 in [OPEN-DECISIONS.md](./OPEN-DECISIONS.md): the mapping stays, and US1-1, US1-2
and quickstart rows 1-2 were reworded to match it. With the new wording, scenarios 1 and
2 pass.
