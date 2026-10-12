# Coverage Gap Record

**Summary**: 5/24 closed · 0 closed unproven · 2 in progress · 16 open · 1 unit-only by design
**Last updated**: 2026-10-12

| ID | Category | Gap | Story | FR | Priority | Status | Evidence | Notes |
|---|---|---|---|---|---|---|---|---|
| G-01 | static | govulncheck gate | US1 | FR-001 | P1 | closed | `vuln (go)` https://github.com/GameplanePanel/Gameplane/actions/runs/38312206591 | Green on #673 after the Go patch, cosign v3 and rekor v1.5.4 bumps (T021); the proof shows the gate fails on a vendored vulnerable fixture. |
| G-02 | static | container image scan gate | US1 | FR-001 | P1 | open |  |  |
| G-03 | static | Dockerfile lint gate | US1 | FR-001 | P1 | closed | `workflow lint` https://github.com/GameplanePanel/Gameplane/actions/runs/38286000942 | First run on #674 flagged DL3008/DL3018/DL4006 in 13 Dockerfiles; pinned apk/apt versions and pipefail SHELL fixed them, and every game image still builds (run 38286000940). |
| G-04 | static | dependency-review gate | US1 | FR-001 | P1 | open |  |  |
| G-05 | static | codegen / tidy drift gate | US1 | FR-001 | P1 | closed | `codegen drift` https://github.com/GameplanePanel/Gameplane/actions/runs/38189397259 | First run on #672 found no drift; the proof shows the job fails on injected drift. |
| G-06 | static | submodule pointer freshness gate | US1 | FR-001 | P1 | open |  |  |
| G-07 | e2e-dashboard | live: notification webhook config | US2 | FR-002 | P1 | open |  |  |
| G-08 | e2e-dashboard | live: user create + role assignment | US2 | FR-002 | P1 | open |  |  |
| G-09 | e2e-dashboard | live: custom role + binding | US2 | FR-002 | P1 | open |  |  |
| G-10 | e2e-dashboard | live: backup restore | US2 | FR-002 | P1 | open |  |  |
| G-11 | e2e-dashboard | live: mod registry config | US2 | FR-002 | P1 | open |  |  |
| G-12 | e2e-dashboard | live: login, create/delete server, settings | US2 | FR-002 | P1 | closed | `web/e2e/specs/live/*` specs (`login-and-shell`, `createAndCleanupServer`, `servers-core`, `server-settings`) in job `e2e web live`: https://github.com/GameplanePanel/Gameplane/tree/master/web/e2e/specs/live |  |
| G-13 | e2e-optional | MCP server | US3 | FR-003 | P2 | open |  |  |
| G-14 | e2e-optional | audit syslog bridge | US3 | FR-003 | P2 | open |  |  |
| G-15 | e2e-optional | tunnel (frp) supervision | US3 | FR-003 | P2 | open |  |  |
| G-16 | e2e-optional | tunnel (tailscale, playit) | US3 | FR-003 | P2 | unit-only-by-design |  | Tailscale and playit need external accounts and network reachability that CI cannot provide; covered by unit tests only (research R7). |
| G-17 | e2e-optional | telemetry receiver | US3 | FR-003 | P2 | closed | `TestTelemetryLifecycle` in bucket `telemetry` (job `e2e telemetry`), spec 022: https://github.com/GameplanePanel/Gameplane/blob/master/test/e2e/telemetry_e2e_test.go |  |
| G-18 | e2e-optional | disabled state + no interference | US3 | FR-003 | P2 | open |  |  |
| G-19 | unit | `secrets_managed.go` 100 % | US4 | FR-004 | P2 | open |  |  |
| G-20 | unit | `ConsoleShell.test.tsx` ≥ 80 % lines | US4 | FR-004 | P2 | open |  |  |
| G-21 | unit | `CloneServerDialog` ≥ 80 % lines + cancel/error | US4 | FR-004 | P2 | in-progress |  | test exists (`CloneServerDialog.test.tsx`); coverage to verify in T053. |
| G-22 | unit | `ServerActionsMenu` ≥ 80 % lines + cancel/error | US4 | FR-004 | P2 | in-progress |  | test exists (`ServerActionsMenu.test.tsx`); coverage to verify in T053. |
| G-23 | e2e-postgres | kind E2E on Postgres-backed API | US5 | FR-005 | P3 | open |  |  |
| G-24 | e2e-postgres | upgrade from frozen 001–012 baseline | US5 | FR-005 | P3 | open |  |  |
