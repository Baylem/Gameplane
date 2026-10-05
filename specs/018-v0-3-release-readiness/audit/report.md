# v0.3.0 Release Readiness Report

Built from the audit records at the end of the audit (T066). Readable cold in under 15 minutes (SC-008).

**State of this report (2026-10-05, master `c79715a6`): the audit is not finished.** It is a snapshot of what the records say, not a go case. 1 of 8 release criteria is Met. The formal live round did not run (OD-029), the upgrade and node rounds have not run, and no finding is `verified` (T058 has not run). `v0.3.0-rc.2` **has not been tagged or pushed**: RC-TAG-2 in [OPEN-DECISIONS.md](../OPEN-DECISIONS.md#rc-tag-2-publish-v030-rc2--pending) still reads PENDING, and its tag target is the #496 merge commit `6750ba87`: `gh api repos/ValgulNecron/Gameplane/git/refs/tags/v0.3.0-rc.2` returns 404 on 2026-10-05, and only `v0.3.0-rc.1` exists (never installable, see [rounds.md](rounds.md#rc1)).

## Decision

**Go**, with the open criteria accepted as known risk (maintainer decision, T070).

- Go/no-go: **go**
- Maintainer: valgulnecron
- Date: 2026-10-05
- Accepted risk: the release criteria marked Not met / Partially below (RC-01, RC-03..RC-08) are not blocking for v0.3.0 by this decision; the gaps stay listed under [Not live-verified](#not-live-verified) and in the release notes. The cold read ([evidence/cold-read.md](evidence/cold-read.md)) recommended no-go; the maintainer chose go.

## Release criteria

Evidence as of 2026-10-05. `Met` here is filled only from the records; the maintainer's decision (T070) will set the `Met` column in `audit/release-criteria.md`, which stays `pending` until then.

| ID | Criterion | Source | Met | Reason |
|---|---|---|---|---|
| RC-01 | Every inventory row has an outcome, zero rows `fail`, and every `blocked` row names its prerequisite and is listed as not live-verified | SC-001 | **Not met** | 467 of 486 rows are `untested` and no row is `pass`; no row is `fail`. 19 rows are `blocked`, each names its prerequisite, and they are listed under Not live-verified. The criterion needs an outcome on every row. |
| RC-02 | Every component has a `complete` review record | SC-002 | **Met** | `audit/coverage.md` has 27 component rows, all `complete`. The 15 `go.work` entries, `web/`, `charts/gameplane/`, `deploy/`, `hack/`, `.github/workflows/`, `.github/actions/`, `images/`, `docs/`, root docs, `design-export/`, `modules/` and `website/` are all there. The records say a review happened; findings from them are counted under RC-03. |
| RC-03 | Zero findings in `imported`, `open`, `fixing` or `fixed-unverified` | SC-003, SC-004 | **Not met** | 291 findings are in a blocking status: 25 `imported`, 37 `open`, 0 `fixing`, 229 `fixed-unverified`. None is `verified`. |
| RC-04 | Each FR-008 boundary has at least one active violation attempt recorded | SC-007 | **Not met** | The 16 `INV-SEC-*` rows (in `audit/held/inventory-SEC.md`, OD-019) are all `untested`, so no violation attempt is recorded for any FR-008 boundary. |
| RC-05 | Live beta.8 → RC upgrade with zero data loss and zero lost accounts | SC-005, OD-005 | **Not met** | The upgrade round has not run (T061, T065). The 5 `UPG` rows are `untested`. CI `e2e-upgrade` from beta.8 exists but is not the live upgrade this criterion asks for. |
| RC-06 | The baseline snapshot matches the post-cleanup snapshot, with zero `audit018-` resources remaining | SC-006 | **Not met** | No post-cleanup snapshot comparison is recorded. The `audit018-admin` test account is still `pending` removal in [rounds.md](rounds.md), the live sweep's test resources are `not recorded`, and T034 (cleanup) is open. |
| RC-07 | Every `not-a-defect` and `out-of-scope` closure is justified, and each out-of-scope one cites a roadmap line | SC-009 | **Partially** | Both closures that exist are accounted for: F-276 (`not-a-defect`) carries a Justification citing `web/src/lib/servers.ts:15-16`, and F-181 (`closed-already-fixed`) cites #467. No finding is `out-of-scope`, so no roadmap line is needed yet. Not complete: the open findings have not been triaged, so more closures are expected. |
| RC-08 | CI is green on the tagged commit | FR-017 | **Not met** | No `v0.3.0` tag exists. `v0.3.0-rc.2` is not tagged (RC-TAG-2 is PENDING), so there is no tagged commit with a green CI run. `v0.3.0-rc.1` was tagged but its release run was cancelled. |

## Totals

How each number was produced: findings from the `| F-NNN |` rows of `audit/findings.md` (awk, split on `|`, Severity and Status columns); inventory from the `| INV-` rows of `audit/inventory.md` and `audit/held/inventory-SEC.md` (awk, Outcome column); coverage from `audit/coverage.md`. The T059 loop command `grep -cE '\| (imported|open|fixing|fixed-unverified) \|' audit/findings.md` returns **291**.

### Findings: 293 total (F-001 to F-293, no gaps)

| Severity | imported | open | fixing | fixed-unverified | verified | closed-already-fixed | not-a-defect | out-of-scope | Total |
|---|---|---|---|---|---|---|---|---|---|
| S1 | 6 | 1 | 0 | 5 | 0 | 0 | 0 | 0 | 12 |
| S2 | 1 | 2 | 0 | 10 | 0 | 0 | 0 | 0 | 13 |
| S3 | 13 | 15 | 0 | 91 | 0 | 0 | 0 | 0 | 119 |
| S4 | 5 | 19 | 0 | 123 | 0 | 1 | 1 | 0 | 149 |
| Total | 25 | 37 | 0 | 229 | 0 | 1 | 1 | 0 | 293 |

### Inventory: 486 rows in `inventory.md` plus 16 `INV-SEC-*` rows in `held/inventory-SEC.md` = 502

| Outcome | `inventory.md` (486) | With `INV-SEC-*` (502) |
|---|---|---|
| pass | 0 | 0 |
| fail | 0 | 0 |
| blocked | 19 | 19 |
| n/a | 0 | 0 |
| untested | 467 | 483 |

No row is `pass`. The formal per-row round did not run (OD-029), so most rows stay `untested`; they are reported as such.

### Coverage

27 of 27 required component rows in `audit/coverage.md` are `complete` (100%): 15 `go.work` modules, plus `web/`, `charts/gameplane/`, `deploy/`, `hack/`, `.github/workflows/`, `.github/actions/`, `images/`, `docs/`, root docs, `design-export/`, `modules/`, `website/` (OD-022 added the actions and images rows). Complete means reviewed, not that its findings are fixed.

### Rounds

| Round | State |
|---|---|
| rc.0 | pre-RC review and inventory enumeration; 0 rows run |
| rc.1 | `v0.3.0-rc.1` tagged 2026-09-24, release run cancelled (F-257), never installable; 0 rows run |
| live-sweep 2026-10-04 | informal sweep of `master` on kubelab; F-273 to F-293; per-row outcomes not recorded (OD-029) |
| rc.2 | RC-TAG-2 PENDING; tag target `6750ba87` (#496 merge commit), **not tagged or pushed** |
| upgrade, node (T061, T062, T065) | not run |
| re-verification of fixed-unverified findings (T058) | not run |

## Open findings

Must be empty for a go. It is not: 291 findings are in a blocking status.

- **229 `fixed-unverified`** (fix merged, not re-verified on an RC): S1 5, S2 10, S3 91, S4 123. Listed in [findings.md](findings.md); re-verification (T058) has not run.
- **25 `imported`** and **37 `open`** are listed below (62 rows). Several `imported` rows say their fix is merged or that they are believed not-a-defect; none has been re-checked and closed.

| ID | Severity | Status | Component | Title |
|---|---|---|---|---|
| F-001 | S3 | imported | web/ | Dependency bump (#406) broke upload-dialog unit tests and keyboard-focus e2e |
| F-002 | S4 | imported | web/ | Visual-diff residuals after feature 014 conformance (7 screens) |
| F-003 | S4 | imported | web/ | Design exports mismatched shipped UI: DMnEi mislabel, J5pjJ3 composite frame, oversized chips/badges |
| F-004 | S3 | imported | web/ | e2e `page.route()` overrides silently no-op under mock-target MSW service worker |
| F-005 | S3 | imported | web/ | Live share-links e2e failure traced to missing `/shares` rule in Vite dev proxy |
| F-006 | S1 | imported | .github/ | CI `dump-cluster-state` redaction misses quoted/JSON-embedded secret values |
| F-007 | S2 | imported | web/ | Console WebSocket rejected (403) on non-default ports; `send()` before open threw |
| F-008 | S3 | imported | web/ | Module catalog tag filters unioned instead of intersected |
| F-009 | S4 | imported | web/ | Vertical tab list rendered as an oval |
| F-010 | S3 | imported | web/ | Console terminal grew without bound |
| F-011 | S3 | imported | web/ | Preset share links didn't expire at end of day like custom dates |
| F-012 | S1 | imported | api/, operator/, charts/gameplane/, images/ | Security-audit hardening: RBAC, share/tunnel credential handling, GameServer reconciler validation (bundled fix) |
| F-013 | S1 | imported | .github/ | CI `dump-cluster-state`: redact quoted/JSON-embedded secret values (fix for F-006) |
| F-014 | S1 | imported | api/ | Share links not scoped to their originating cluster in multi-cluster installs |
| F-015 | S1 | imported | api/, operator/ | GameServer env-var Secret/ConfigMap references not restricted to server-owned objects |
| F-016 | S3 | imported | charts/gameplane/ | Audit syslog bridge had no ingress NetworkPolicy restricting senders to the API |
| F-017 | S3 | imported | images/ | steamcmd base image referenced by a mutable `:latest` tag instead of a pinned digest |
| F-018 | S1 | imported | api/, operator/ | Tunnel credential Secret delete/update/mount lacked ownership checks |
| F-019 | S3 | imported | charts/gameplane/ | Telemetry receiver had no ingress NetworkPolicy restricting senders to the API |
| F-020 | S4 | imported | modules/, docs/ | Nuclear Option's assumed UDP 7777 game-join port is not bound on the live server |
| F-021 | S4 | imported | docs/ | Helm-seeded OIDC role mappings documented while the feature sits under CHANGELOG Unreleased |
| F-022 | S3 | imported | operator/, agent/, web/ | `cli` rcon.protocol semantics: confirm the implementation matches the recorded Option A ruling |
| F-023 | S3 | imported | modules/ | Open question: Factorio template's `rcon.protocol: source` vs. an earlier contract claim of `none` |
| F-024 | S3 | imported | modules/ | Open question: Project Zomboid template's `rcon.protocol: source` vs. an earlier contract claim of `none` |
| F-025 | S3 | imported | agent/ | Open question: generic `rest` rcon wire contract (FiveM txAdmin / Farming Simulator 25) not yet given a final ruling |
| F-027 | S3 | open | docs/ | `docs/install.md` has no rollback procedure |
| F-028 | S3 | open | charts/gameplane/ | `charts/gameplane/values.yaml` git module source still pinned to `ref: v0.2.0-beta.6` |
| F-037 | S4 | open | website/ | website games page lists 16 games but 30 now shipped |
| F-038 | S4 | open | website/ | website comparison.mdx says "16 official modules" but 30 now shipped |
| F-039 | S4 | open | website/ | website VERSION constant still beta.7; beta.8 published 2026-08-22 |
| F-115 | S1 | open | web | New file truncates existing file |
| F-117 | S3 | open | web | Settings save merges entire spec, losing concurrent changes |
| F-118 | S2 | open | web | Tunnel create blocked by missing Secret |
| F-119 | S2 | open | web | Tunnel enable flows fail validation |
| F-120 | S3 | open | web | Share links resolve hangs on extended wake |
| F-122 | S3 | open | web | Settings dirty state persists on tab switch |
| F-123 | S3 | open | web | Revoked share links show as active |
| F-124 | S3 | open | web | Node selector never applied to running server |
| F-127 | S3 | open | web | Default namespace not used on server create |
| F-128 | S4 | open | web | Modules hash navigation fails |
| F-129 | S4 | open | web | Add cluster links to unusable page |
| F-131 | S4 | open | web | Invalid sections don't block save |
| F-132 | S4 | open | web | Reset text editor preserves old content |
| F-133 | S4 | open | web | Invite dialog retains previous entry |
| F-134 | S4 | open | web | Identity provider tab contradicts code |
| F-139 | S4 | open | web | specs.md contradicts implemented features |
| F-140 | S4 | open | web | specs.md lists unimplemented UI |
| F-219 | S3 | open | charts/gameplane/ | Default module catalog omits 14 spec-015 modules |
| F-222 | S4 | open | charts/gameplane/ | Module source git.ref doc says main, but values pins v0.2.0-beta.6 |
| F-260 | S4 | open | operator | GameServer.Stopped and Restore.Resuming phases declared but never assigned |
| F-264 | S3 | open | .github/actions/ | Dependabot never bumps the action pins inside `.github/actions/`, and they already trail the workflows |
| F-265 | S4 | open | .github/actions/ | docker-bake.hcl header names `e2e-images` as its caller; bake runs in `build-e2e-images` |
| F-266 | S3 | open | images/ | steam-install.sh reports a failed update as success once the game is installed, and never retries it |
| F-267 | S3 | open | images/ | images/README.md Dockerfile template runs `chmod` as the non-root user and fails to build |
| F-268 | S4 | open | images/ | Documented local `docker run -v game-data:/data` gives a root-owned volume the game user can't write |
| F-269 | S4 | open | images/ | images/README.md promises exponential backoff; steam-install.sh uses a fixed delay |
| F-270 | S4 | open | images/ | images/README.md trigger list omits `pull_request`, and its manual `docker build -f Dockerfile` commands fail from the repo root |
| F-289 | S3 | open | modules/ | Terraria and tModLoader servers lack a TCP readiness probe |
| F-290 | S4 | open | web | Public share page: "starting up" vs "waking up" copy (item 21c) |
| F-291 | S4 | open | web | AdminSettings `instanceName` hint text |
| F-292 | S3 | open | web | Files tab: folder selection and delete (item 18c) |
| F-293 | S3 | open | web | Console history (item 21h) |

## Not live-verified

Rows are not live-verified if they are `blocked` (listed below, with the prerequisite and the alternative tested) or still `untested` (counted after the table). Counts come from `awk` over the Outcome column of `audit/inventory.md` (see Totals).

### Blocked rows (19)

| ID | Capability | Prerequisite and alternative |
|---|---|---|
| INV-CRD-029 | Cluster register and health check | OD-021 item 19: blocked, no second live cluster available to the audit; anchor renamed to match. Verified live instead: an unregistered `?cluster=` selector gets HTTP 400 `cluster not permitted` from the RBAC allow-list (`scope.ResolveCluster`) before dispatch. The `rejectRemoteCluster` guard (`resources.go:104-110`) itself only fires for a *registered* remote cluster, which the audit lacks; it returns 501 with body `httperr.RemoteClusterNotImplemented` (`resources.go:106`, `httperr.go:33-34`), per OD-021 item 19 (#430) — see procedures/crd.md step 4 |
| INV-CRD-030 | Cluster health check failure (Unhealthy) | OD-021 item 19: blocked, cascades from INV-CRD-029 (no second live cluster to register first) |
| INV-AUX-009 | Supervise frp (forward proxy) relay client | blocked candidate: needs external frp server; alternative: test operator tunnel injection via envtest (api-agent bucket) |
| INV-AUX-010 | Supervise Tailscale relay client | blocked candidate: needs Tailscale account and auth key; alternative: test operator injection |
| INV-AUX-011 | Supervise playit relay client | blocked candidate: needs playit.gg account and secret key; alternative: test operator injection |
| INV-AUX-012 | Accept HTTP audit events and relay to syslog collector | blocked candidate: needs external RFC 5424 syslog receiver; alternative: test with local netcat listener (api-auth bucket) |
| INV-AUX-013 | Format and send RFC 5424 syslog records | blocked candidate: needs external syslog collector |
| INV-MOD-033 | Create and start DayZ server from template | blocked candidate: needs node memory ≥6Gi per pod; no lighter alternative available in rcon/battleye category (DayZ is the only module) |
| INV-MOD-034 | Join DayZ server via A2S query | blocked candidate: needs node memory ≥6Gi per pod; no lighter alternative available in rcon/battleye category (DayZ is the only module) |
| INV-MOD-035 | DayZ server BattlEye RCON console | blocked candidate: needs node memory ≥6Gi per pod; no lighter alternative available in rcon/battleye category (DayZ is the only module) |
| INV-MOD-036 | Backup and restore DayZ server | blocked candidate: needs node memory ≥6Gi per pod; no lighter alternative available in rcon/battleye category (DayZ is the only module) |
| INV-MOD-049 | Create and start Satisfactory server from template | blocked candidate: needs node memory ≥6Gi per pod; alternative: FiveM (rcon/rest + HTTP REST probe, 4Gi) |
| INV-MOD-050 | Join Satisfactory server via HTTP REST | blocked candidate: needs node memory ≥6Gi per pod; alternative: FiveM (rcon/rest + HTTP REST probe, 4Gi) |
| INV-MOD-051 | Satisfactory server satisfactory RCON console | blocked candidate: needs node memory ≥6Gi per pod; alternative: FiveM (rcon/rest + HTTP REST probe, 4Gi) |
| INV-MOD-052 | Backup and restore Satisfactory server | blocked candidate: needs node memory ≥6Gi per pod; alternative: FiveM (rcon/rest + HTTP REST probe, 4Gi) |
| INV-MOD-053 | Create and start Ark Survival Ascended server from template | blocked candidate: needs node memory ≥10Gi per pod; alternative: CS2 (rcon/source + Steam A2S probe, 2Gi) |
| INV-MOD-054 | Join Ark Survival Ascended server via TCP RCON probe | blocked candidate: needs node memory ≥10Gi per pod; alternative: CS2 (rcon/source + Steam A2S probe, 2Gi) |
| INV-MOD-055 | Ark Survival Ascended server Source RCON console | blocked candidate: needs node memory ≥10Gi per pod; alternative: CS2 (rcon/source + Steam A2S probe, 2Gi) |
| INV-MOD-056 | Backup and restore Ark Survival Ascended server | blocked candidate: needs node memory ≥10Gi per pod; alternative: CS2 (rcon/source + Steam A2S probe, 2Gi) |

### Untested rows

- **467 of 486 inventory rows in `audit/inventory.md` are still `untested`** (WEB 145, API 146, CRD 35, AGT 36, AUX 13, HELM 28, MOD 56, UPG 5, NODE 3). The formal per-row live round did not run (OD-029): the 2026-10-04 informal live sweep on `master` stands in for it, and it did not record per-row outcomes, so no row is marked `pass`.
- The 16 security-control rows (`INV-SEC-001` to `INV-SEC-016`) are kept in `audit/held/inventory-SEC.md`, not in `audit/inventory.md` (OD-019; that file has been in git since 8249db80) and are all `untested` there, so no active violation attempt is recorded for them.
- No upgrade or node round has run (T061, T062, T065): the beta.8 to RC upgrade (`UPG`, 5 rows) and the node behaviour rows (`NODE`, 3 rows) are `untested`.
- 229 findings are `fixed-unverified`: their fixes merged but have not been re-verified on a release candidate (T058 has not run).

## By component

Counts are from the Component column of `audit/findings.md`; a finding naming several components counts under each. Review record, method, tier and date are from `audit/coverage.md`.

| Component | Review record | Method | Reviewer tier | Review date | imported | open | fixed-unverified | closed / not-a-defect |
|---|---|---|---|---|---|---|---|---|
| agent/ | complete | code-review | opus → opus (independent, OD-020) | 2026-09-24 | 2 | 0 | 15 | 0 |
| api/ | complete | code-review | opus → opus (independent, OD-020) | 2026-09-24 | 4 | 0 | 40 | 0 |
| audit-syslog-bridge/ | complete | code-review | opus → opus (independent, OD-020) | 2026-09-24 | 0 | 0 | 6 | 0 |
| capture-sidecar/ | complete | code-review | opus → opus (independent, OD-020) | 2026-09-24 | 0 | 0 | 8 | 0 |
| gameaction/ | complete | code-review | opus → opus (independent, OD-020) | 2026-09-24 | 0 | 0 | 4 | 0 |
| gameproto/ | complete | code-review | opus → opus (independent, OD-020) | 2026-09-24 | 0 | 0 | 6 | 0 |
| gp-module/ | complete | code-review | opus → opus (independent, OD-020) | 2026-09-24 | 0 | 0 | 11 | 0 |
| mcp-server/ | complete | code-review | opus → opus (independent, OD-020) | 2026-09-24 | 0 | 0 | 8 | 0 |
| netguard/ | complete | code-review | opus → opus (independent, OD-020) | 2026-09-24 | 0 | 0 | 3 | 0 |
| operator/ | complete | code-review | opus → opus (independent, OD-020) | 2026-09-24 | 4 | 1 | 37 | 0 |
| sentinel/ | complete | code-review | opus → opus (independent, OD-020) | 2026-09-24 | 0 | 0 | 8 | 1 |
| svcutil/ | complete | code-review | opus → opus (independent, OD-020) | 2026-09-24 | 0 | 0 | 2 | 0 |
| telemetry-receiver/ | complete | code-review | opus → opus (independent, OD-020) | 2026-09-24 | 0 | 0 | 4 | 0 |
| test/e2e/ | complete | code-review | sonnet → opus | 2026-09-23 | 0 | 0 | 3 | 0 |
| tunnel/ | complete | code-review | opus → opus (independent, OD-020) | 2026-09-24 | 0 | 0 | 7 | 0 |
| web/ | complete | code-review | opus → opus (independent, OD-020) | 2026-09-24 | 11 | 21 | 24 | 1 |
| charts/gameplane/ | complete | code-review | opus → opus (independent, OD-020) | 2026-09-24 | 3 | 3 | 20 | 0 |
| deploy/ | complete | code-review | opus → opus (independent, OD-020) | 2026-09-24 | 0 | 0 | 6 | 0 |
| hack/ | complete | code-review | opus → opus (independent, OD-020) | 2026-09-24 | 0 | 0 | 3 | 0 |
| .github/workflows/ | complete | code-review | opus → opus (independent, OD-020) | 2026-09-24 | 0 | 0 | 11 | 0 |
| .github/actions/ | complete | code-review | sonnet → opus | 2026-10-04 | 0 | 2 | 0 | 0 |
| images/ | complete | code-review | sonnet → opus | 2026-10-04 | 2 | 5 | 0 | 0 |
| docs/ | complete | code-review | sonnet → opus | 2026-09-23 | 2 | 1 | 7 | 0 |
| root docs | complete | code-review | sonnet → opus | 2026-09-23 | 0 | 0 | 3 | 0 |
| design-export/ | complete | code-review | sonnet → opus | 2026-09-23 | 0 | 0 | 0 | 0 |
| modules/ | complete | consistency-only | sonnet → opus | 2026-09-23 | 3 | 1 | 0 | 0 |
| website/ | complete | consistency-only | sonnet → opus | 2026-09-23 | 0 | 3 | 0 | 0 |

Findings on components without their own coverage row (not shown above): `.github/` (2 `imported`: F-006, F-013; 2 `fixed-unverified`).

## By area

Inventory outcomes by area code, from `audit/inventory.md` (awk on the ID prefix and Outcome column).

| Area | Rows | untested | blocked | pass | fail | n/a |
|---|---|---|---|---|---|---|
| WEB | 145 | 145 | 0 | 0 | 0 | 0 |
| API | 146 | 146 | 0 | 0 | 0 | 0 |
| CRD | 37 | 35 | 2 | 0 | 0 | 0 |
| AGT | 36 | 36 | 0 | 0 | 0 | 0 |
| AUX | 18 | 13 | 5 | 0 | 0 | 0 |
| HELM | 28 | 28 | 0 | 0 | 0 | 0 |
| MOD | 68 | 56 | 12 | 0 | 0 | 0 |
| UPG | 5 | 5 | 0 | 0 | 0 | 0 |
| NODE | 3 | 3 | 0 | 0 | 0 | 0 |
| SEC (`held/inventory-SEC.md`) | 16 | 16 | 0 | 0 | 0 | 0 |
| **Total** | 502 | 483 | 19 | 0 | 0 | 0 |

## Proposed E2E additions

Counted from the `**Automatable?**` field of each `### <slug>` procedure under `audit/procedures/*.md` (python script, same files): 356 procedures carry the field, 300 say `yes`, 56 say `no`, deferred or n/a. The bucket is the first bucket name written in the field; where a procedure names two (for example "api-auth or api-mods"), the full wording is in the procedure. `web-api` is a new bucket proposed in OD-021 item 12 that does not exist yet in `test/e2e/buckets.sh`.

| Proposed bucket | Procedures |
|---|---|
| api-agent | 77 |
| web-api | 74 |
| api-auth | 53 |
| api-mods | 35 |
| operator | 28 |
| api-rbac | 23 |
| api-roles | 6 |
| upgrade | 3 |
| bot-heavy | 1 |

Procedures by bucket (area file and slug; the link goes to the procedure):

**api-agent** (77): [agent:console-source](procedures/agent.md#console-source), [agent:console-telnet](procedures/agent.md#console-telnet), [agent:console-websocket](procedures/agent.md#console-websocket), [agent:console-battleye](procedures/agent.md#console-battleye), [agent:console-satisfactory](procedures/agent.md#console-satisfactory), [agent:console-palworld](procedures/agent.md#console-palworld), [agent:console-nuclearoption](procedures/agent.md#console-nuclearoption), [agent:console-rest](procedures/agent.md#console-rest), [agent:console-pty](procedures/agent.md#console-pty), [agent:files-list](procedures/agent.md#files-list), [agent:files-read](procedures/agent.md#files-read), [agent:files-write](procedures/agent.md#files-write), [agent:files-upload](procedures/agent.md#files-upload), [agent:files-download](procedures/agent.md#files-download), [agent:files-mkdir](procedures/agent.md#files-mkdir), [agent:files-delete](procedures/agent.md#files-delete), [agent:logs-tail](procedures/agent.md#logs-tail), [agent:logs-download](procedures/agent.md#logs-download), [agent:players-list](procedures/agent.md#players-list), [agent:players-banned](procedures/agent.md#players-banned), [agent:players-kick](procedures/agent.md#players-kick), [agent:players-ban](procedures/agent.md#players-ban), [agent:players-unban](procedures/agent.md#players-unban), [agent:players-whitelist](procedures/agent.md#players-whitelist), [agent:players-whitelist-add](procedures/agent.md#players-whitelist-add), [agent:players-whitelist-remove](procedures/agent.md#players-whitelist-remove), [agent:quiesce-pause](procedures/agent.md#quiesce-pause), [agent:quiesce-resume](procedures/agent.md#quiesce-resume), [agent:lifecycle-stop](procedures/agent.md#lifecycle-stop), [agent:actions-run](procedures/agent.md#actions-run), [agent:status-metrics](procedures/agent.md#status-metrics), [agent:mods-list](procedures/agent.md#mods-list), [agent:mods-install](procedures/agent.md#mods-install), [agent:mods-upload](procedures/agent.md#mods-upload), [agent:mods-remove](procedures/agent.md#mods-remove), [agent:heartbeat-metrics](procedures/agent.md#heartbeat-metrics), [aux:sentinel-minecraft-status-ping](procedures/aux.md#sentinel-minecraft-status-ping), [aux:sentinel-minecraft-join-wake](procedures/aux.md#sentinel-minecraft-join-wake), [aux:sentinel-terraria-generic-udp](procedures/aux.md#sentinel-terraria-generic-udp), [aux:capture-sidecar-start-stop](procedures/aux.md#capture-sidecar-start-stop), [aux:mcp-server-read-only-check](procedures/aux.md#mcp-server-read-only-check), [crd:gameserver-create-from-template](procedures/crd.md#gameserver-create-from-template), [crd:gameserver-restart](procedures/crd.md#gameserver-restart), [crd:gameserver-wipe-data-volume](procedures/crd.md#gameserver-wipe-data-volume), [web:servers-start-server](procedures/web.md#servers-start-server), [web:servers-stop-server](procedures/web.md#servers-stop-server), [web:servers-restart-server](procedures/web.md#servers-restart-server), [web:servers-wake-sleeping-server](procedures/web.md#servers-wake-sleeping-server), [web:server-detail-lifecycle-start](procedures/web.md#server-detail-lifecycle-start), [web:server-detail-lifecycle-stop](procedures/web.md#server-detail-lifecycle-stop), [web:server-detail-lifecycle-restart](procedures/web.md#server-detail-lifecycle-restart), [web:server-detail-lifecycle-wake](procedures/web.md#server-detail-lifecycle-wake), [web:server-detail-console-send-command](procedures/web.md#server-detail-console-send-command), [web:server-detail-logs-view-game-logs](procedures/web.md#server-detail-logs-view-game-logs), [web:server-detail-files-browse](procedures/web.md#server-detail-files-browse), [web:server-detail-files-view-file](procedures/web.md#server-detail-files-view-file), [web:server-detail-files-edit-file](procedures/web.md#server-detail-files-edit-file), [web:server-detail-files-upload-file](procedures/web.md#server-detail-files-upload-file), [web:server-detail-files-create-file](procedures/web.md#server-detail-files-create-file), [web:server-detail-files-create-directory](procedures/web.md#server-detail-files-create-directory), [web:server-detail-files-delete-file](procedures/web.md#server-detail-files-delete-file), [web:server-detail-mods-list](procedures/web.md#server-detail-mods-list), [web:server-detail-mods-install-url](procedures/web.md#server-detail-mods-install-url), [web:server-detail-mods-install-registry](procedures/web.md#server-detail-mods-install-registry), [web:server-detail-mods-upload](procedures/web.md#server-detail-mods-upload), [web:server-detail-mods-remove](procedures/web.md#server-detail-mods-remove), [web:server-detail-modpacks-install](procedures/web.md#server-detail-modpacks-install), [web:server-detail-players-view](procedures/web.md#server-detail-players-view), [web:server-detail-players-kick](procedures/web.md#server-detail-players-kick), [web:server-detail-players-ban](procedures/web.md#server-detail-players-ban), [web:server-detail-players-unban](procedures/web.md#server-detail-players-unban), [web:server-detail-players-whitelist-add](procedures/web.md#server-detail-players-whitelist-add), [web:server-detail-players-whitelist-remove](procedures/web.md#server-detail-players-whitelist-remove), [web:create-server-wizard-pick-template](procedures/web.md#create-server-wizard-pick-template), [web:create-server-wizard-configure](procedures/web.md#create-server-wizard-configure), [web:create-server-wizard-network](procedures/web.md#create-server-wizard-network), [web:create-server-wizard-review-create](procedures/web.md#create-server-wizard-review-create)

**web-api** (74): [web:dashboard-view-recent-activity](procedures/web.md#dashboard-view-recent-activity), [web:dashboard-view-recent-backups](procedures/web.md#dashboard-view-recent-backups), [web:servers-filter-by-namespace](procedures/web.md#servers-filter-by-namespace), [web:server-detail-logs-view-pod-logs](procedures/web.md#server-detail-logs-view-pod-logs), [web:server-detail-mods-browse-registry](procedures/web.md#server-detail-mods-browse-registry), [web:server-detail-mods-check-updates](procedures/web.md#server-detail-mods-check-updates), [web:server-detail-modpacks-browse](procedures/web.md#server-detail-modpacks-browse), [web:server-detail-backups-create-now](procedures/web.md#server-detail-backups-create-now), [web:server-detail-backups-view-list](procedures/web.md#server-detail-backups-view-list), [web:server-detail-backups-restore](procedures/web.md#server-detail-backups-restore), [web:server-detail-backups-schedule-create](procedures/web.md#server-detail-backups-schedule-create), [web:server-detail-backups-schedule-delete](procedures/web.md#server-detail-backups-schedule-delete), [web:server-detail-settings-general](procedures/web.md#server-detail-settings-general), [web:server-detail-settings-version](procedures/web.md#server-detail-settings-version), [web:server-detail-settings-resources](procedures/web.md#server-detail-settings-resources), [web:server-detail-settings-networking](procedures/web.md#server-detail-settings-networking), [web:server-detail-settings-environment](procedures/web.md#server-detail-settings-environment), [web:server-detail-settings-lifecycle](procedures/web.md#server-detail-settings-lifecycle), [web:server-detail-settings-scheduled-backups](procedures/web.md#server-detail-settings-scheduled-backups), [web:server-detail-settings-network-capture](procedures/web.md#server-detail-settings-network-capture), [web:server-detail-settings-placement](procedures/web.md#server-detail-settings-placement), [web:server-detail-settings-access](procedures/web.md#server-detail-settings-access), [web:server-detail-settings-sharelinks-create](procedures/web.md#server-detail-settings-sharelinks-create), [web:server-detail-settings-sharelinks-view](procedures/web.md#server-detail-settings-sharelinks-view), [web:server-detail-settings-sharelinks-revoke](procedures/web.md#server-detail-settings-sharelinks-revoke), [web:server-detail-settings-danger-delete](procedures/web.md#server-detail-settings-danger-delete), [web:server-detail-settings-danger-transfer](procedures/web.md#server-detail-settings-danger-transfer), [web:modules-catalog-browse](procedures/web.md#modules-catalog-browse), [web:modules-search](procedures/web.md#modules-search), [web:modules-filter-by-source](procedures/web.md#modules-filter-by-source), [web:modules-filter-by-category](procedures/web.md#modules-filter-by-category), [web:modules-install](procedures/web.md#modules-install), [web:modules-upgrade](procedures/web.md#modules-upgrade), [web:modules-uninstall](procedures/web.md#modules-uninstall), [web:modules-upload-custom](procedures/web.md#modules-upload-custom), [web:users-list-view](procedures/web.md#users-list-view), [web:users-invite](procedures/web.md#users-invite), [web:users-edit-role](procedures/web.md#users-edit-role), [web:users-reset-password](procedures/web.md#users-reset-password), [web:users-delete](procedures/web.md#users-delete), [web:users-manage-roles](procedures/web.md#users-manage-roles), [web:admin-settings-general](procedures/web.md#admin-settings-general), [web:admin-settings-auth](procedures/web.md#admin-settings-auth), [web:admin-settings-backup-destinations](procedures/web.md#admin-settings-backup-destinations), [web:admin-settings-module-sources](procedures/web.md#admin-settings-module-sources), [web:admin-settings-mod-registries](procedures/web.md#admin-settings-mod-registries), [web:admin-settings-notifications](procedures/web.md#admin-settings-notifications), [web:admin-settings-telemetry](procedures/web.md#admin-settings-telemetry), [web:admin-settings-updates](procedures/web.md#admin-settings-updates), [web:admin-settings-about](procedures/web.md#admin-settings-about), [web:theme-settings-import](procedures/web.md#theme-settings-import), [web:backups-list-view](procedures/web.md#backups-list-view), [web:backups-filter-by-server](procedures/web.md#backups-filter-by-server), [web:backups-filter-by-phase](procedures/web.md#backups-filter-by-phase), [web:backups-backup-now](procedures/web.md#backups-backup-now), [web:backups-restore](procedures/web.md#backups-restore), [web:backups-view-detail](procedures/web.md#backups-view-detail), [web:backups-schedules-create](procedures/web.md#backups-schedules-create), [web:backups-schedules-edit](procedures/web.md#backups-schedules-edit), [web:backups-schedules-toggle-suspend](procedures/web.md#backups-schedules-toggle-suspend), [web:backups-schedules-delete](procedures/web.md#backups-schedules-delete), [web:backups-restores-view](procedures/web.md#backups-restores-view), [web:audit-log-view](procedures/web.md#audit-log-view), [web:audit-log-filter-by-status-class](procedures/web.md#audit-log-filter-by-status-class), [web:audit-log-filter-by-method](procedures/web.md#audit-log-filter-by-method), [web:audit-log-filter-by-actor](procedures/web.md#audit-log-filter-by-actor), [web:audit-log-pagination](procedures/web.md#audit-log-pagination), [web:audit-log-verify-integrity](procedures/web.md#audit-log-verify-integrity), [web:admin-logs-view-api](procedures/web.md#admin-logs-view-api), [web:admin-logs-view-operator](procedures/web.md#admin-logs-view-operator), [web:admin-logs-tail-option](procedures/web.md#admin-logs-tail-option), [web:admin-logs-follow](procedures/web.md#admin-logs-follow), [web:share-server-start](procedures/web.md#share-server-start), [web:share-server-view-status](procedures/web.md#share-server-view-status)

**api-auth** (53): [api:public-healthz-get](procedures/api.md#public-healthz-get), [api:public-metrics-get](procedures/api.md#public-metrics-get), [api:public-auth-providers-list](procedures/api.md#public-auth-providers-list), [api:public-auth-login](procedures/api.md#public-auth-login), [api:public-auth-logout](procedures/api.md#public-auth-logout), [api:public-auth-oidc-provider-start](procedures/api.md#public-auth-oidc-provider-start), [api:audit-read-paginated](procedures/api.md#audit-read-paginated), [api:audit-verify-chain](procedures/api.md#audit-verify-chain), [api:audit-export-csv](procedures/api.md#audit-export-csv), [api:audit-export-json](procedures/api.md#audit-export-json), [api:config-read-all](procedures/api.md#config-read-all), [api:config-write-section](procedures/api.md#config-write-section), [api:auth-provider-secret-put](procedures/api.md#auth-provider-secret-put), [api:auth-provider-secret-delete](procedures/api.md#auth-provider-secret-delete), [api:notifications-secret-put](procedures/api.md#notifications-secret-put), [api:registry-secret-put](procedures/api.md#registry-secret-put), [api:users-list](procedures/api.md#users-list), [api:users-create](procedures/api.md#users-create), [api:users-me](procedures/api.md#users-me), [api:users-preferences-get](procedures/api.md#users-preferences-get), [api:users-preferences-put](procedures/api.md#users-preferences-put), [api:users-preferences-reset](procedures/api.md#users-preferences-reset), [api:users-get](procedures/api.md#users-get), [api:users-update](procedures/api.md#users-update), [api:users-reset-password](procedures/api.md#users-reset-password), [api:users-bindings-list](procedures/api.md#users-bindings-list), [api:users-bindings-add](procedures/api.md#users-bindings-add), [api:users-delete](procedures/api.md#users-delete), [api:roles-list](procedures/api.md#roles-list), [api:roles-permissions-catalog](procedures/api.md#roles-permissions-catalog), [api:cluster-view](procedures/api.md#cluster-view), [api:cluster-info](procedures/api.md#cluster-info), [api:cluster-stats](procedures/api.md#cluster-stats), [api:cluster-join-node](procedures/api.md#cluster-join-node), [api:cluster-download-kubeconfig](procedures/api.md#cluster-download-kubeconfig), [api:clusters-list](procedures/api.md#clusters-list), [api:modules-list-installed](procedures/api.md#modules-list-installed), [api:modules-list-sources](procedures/api.md#modules-list-sources), [api:modules-catalog](procedures/api.md#modules-catalog), [api:backup-destinations-list](procedures/api.md#backup-destinations-list), [api:events-sse](procedures/api.md#events-sse), [api:system-logs-api](procedures/api.md#system-logs-api), [api:system-logs-operator](procedures/api.md#system-logs-operator), [aux:audit-syslog-bridge-blocked-candidate](procedures/aux.md#audit-syslog-bridge-blocked-candidate), [aux:telemetry-receiver-opt-in](procedures/aux.md#telemetry-receiver-opt-in), [helm:audit-webhook-sink](procedures/helm.md#audit-webhook-sink), [helm:syslog-bridge-audit](procedures/helm.md#syslog-bridge-audit), [helm:audit-stdout-logging](procedures/helm.md#audit-stdout-logging), [helm:web-dashboard-ui](procedures/helm.md#web-dashboard-ui), [helm:ingress-configuration](procedures/helm.md#ingress-configuration), [helm:image-tag-override](procedures/helm.md#image-tag-override), [web:login-username-password](procedures/web.md#login-username-password), [web:login-safe-mode](procedures/web.md#login-safe-mode)

**api-mods** (35): [crd:backup-create-and-run](procedures/crd.md#backup-create-and-run), [crd:backup-failure-and-phase](procedures/crd.md#backup-failure-and-phase), [crd:restore-create-and-suspend-server](procedures/crd.md#restore-create-and-suspend-server), [crd:restore-running-to-succeeded](procedures/crd.md#restore-running-to-succeeded), [crd:backup-schedule-create](procedures/crd.md#backup-schedule-create), [crd:backup-schedule-tick-creates-backup](procedures/crd.md#backup-schedule-tick-creates-backup), [crd:backup-schedule-retention-prunes](procedures/crd.md#backup-schedule-retention-prunes), [crd:module-create-pending](procedures/crd.md#module-create-pending), [crd:module-pulling-to-ready](procedures/crd.md#module-pulling-to-ready), [crd:module-bad-signature-fails](procedures/crd.md#module-bad-signature-fails), [crd:modulesource-oci-sync](procedures/crd.md#modulesource-oci-sync), [crd:modulesource-git-sync-error](procedures/crd.md#modulesource-git-sync-error), [crd:backup-volumesnapshot-strategy](procedures/crd.md#backup-volumesnapshot-strategy), [crd:restore-volumesnapshot-strategy](procedures/crd.md#restore-volumesnapshot-strategy), [helm:telemetry-receiver-bundled](procedures/helm.md#telemetry-receiver-bundled), [helm:mcp-server-deployment](procedures/helm.md#mcp-server-deployment), [helm:default-module-source](procedures/helm.md#default-module-source), [helm:module-signature-verification](procedures/helm.md#module-signature-verification), [helm:upload-module-source](procedures/helm.md#upload-module-source), [modules:garrys-mod](procedures/modules.md#garrys-mod), [modules:farming-simulator-25](procedures/modules.md#farming-simulator-25), [modules:beammp](procedures/modules.md#beammp), [modules:valheim](procedures/modules.md#valheim), [modules:dont-starve-together](procedures/modules.md#dont-starve-together), [modules:tmodloader](procedures/modules.md#tmodloader), [modules:terraria](procedures/modules.md#terraria), [modules:factorio](procedures/modules.md#factorio), [modules:dayz](procedures/modules.md#dayz), [modules:palworld](procedures/modules.md#palworld), [modules:fivem](procedures/modules.md#fivem), [modules:satisfactory](procedures/modules.md#satisfactory), [modules:ark-survival-ascended](procedures/modules.md#ark-survival-ascended), [modules:cs2](procedures/modules.md#cs2), [modules:minecraft-java](procedures/modules.md#minecraft-java), [modules:rust](procedures/modules.md#rust)

**operator** (28): [crd:gameserver-phase-pending-to-starting](procedures/crd.md#gameserver-phase-pending-to-starting), [crd:gameserver-phase-starting-to-running](procedures/crd.md#gameserver-phase-starting-to-running), [crd:gameserver-suspend](procedures/crd.md#gameserver-suspend), [crd:gameserver-unsuspend-wake](procedures/crd.md#gameserver-unsuspend-wake), [crd:gameserver-version-switch](procedures/crd.md#gameserver-version-switch), [crd:gameserver-idle-auto-sleep](procedures/crd.md#gameserver-idle-auto-sleep), [crd:gameserver-idle-wake-window](procedures/crd.md#gameserver-idle-wake-window), [crd:gameserver-delete-no-finalizer](procedures/crd.md#gameserver-delete-no-finalizer), [crd:gameserver-failed-phase-crash-loop](procedures/crd.md#gameserver-failed-phase-crash-loop), [crd:gametemplate-create](procedures/crd.md#gametemplate-create), [crd:networkcapture-create-pending](procedures/crd.md#networkcapture-create-pending), [crd:networkcapture-start-running](procedures/crd.md#networkcapture-start-running), [crd:networkcapture-stop-completed](procedures/crd.md#networkcapture-stop-completed), [crd:networkcapture-failed-sidecar-crash](procedures/crd.md#networkcapture-failed-sidecar-crash), [crd:networkcapture-expired-auto-delete](procedures/crd.md#networkcapture-expired-auto-delete), [helm:existing-storage-claim](procedures/helm.md#existing-storage-claim), [helm:operator-leader-election](procedures/helm.md#operator-leader-election), [nodes:scheduling](procedures/nodes.md#scheduling), [nodes:drain](procedures/nodes.md#drain), [web:dashboard-view-fleet-summary](procedures/web.md#dashboard-view-fleet-summary), [web:dashboard-view-cluster-resources](procedures/web.md#dashboard-view-cluster-resources), [web:servers-list-view](procedures/web.md#servers-list-view), [web:servers-filter-by-status](procedures/web.md#servers-filter-by-status), [web:servers-search-by-name](procedures/web.md#servers-search-by-name), [web:servers-filter-by-game](procedures/web.md#servers-filter-by-game), [web:server-detail-overview-tab](procedures/web.md#server-detail-overview-tab), [web:server-detail-events-view](procedures/web.md#server-detail-events-view), [web:cluster-view-nodes](procedures/web.md#cluster-view-nodes)

**api-rbac** (23): [api:servers-list](procedures/api.md#servers-list), [api:servers-create](procedures/api.md#servers-create), [api:servers-get](procedures/api.md#servers-get), [api:servers-update](procedures/api.md#servers-update), [api:templates-list](procedures/api.md#templates-list), [api:templates-get](procedures/api.md#templates-get), [api:backups-list](procedures/api.md#backups-list), [api:schedules-list](procedures/api.md#schedules-list), [api:restores-list](procedures/api.md#restores-list), [api:capture-enable](procedures/api.md#capture-enable), [api:servers-start](procedures/api.md#servers-start), [api:servers-stop](procedures/api.md#servers-stop), [api:servers-restart](procedures/api.md#servers-restart), [api:servers-clone](procedures/api.md#servers-clone), [api:servers-wipe-data](procedures/api.md#servers-wipe-data), [api:servers-delete](procedures/api.md#servers-delete), [api:files-list](procedures/api.md#files-list), [api:files-read](procedures/api.md#files-read), [api:files-upload](procedures/api.md#files-upload), [api:players-list](procedures/api.md#players-list), [helm:cluster-operations-feature](procedures/helm.md#cluster-operations-feature), [helm:network-policies-enforcement](procedures/helm.md#network-policies-enforcement), [helm:pod-security-enforcement](procedures/helm.md#pod-security-enforcement)

**api-roles** (6): [api:public-shares-resolve](procedures/api.md#public-shares-resolve), [api:public-shares-start](procedures/api.md#public-shares-start), [api:shares-create](procedures/api.md#shares-create), [api:shares-list](procedures/api.md#shares-list), [api:servers-collaborators-set](procedures/api.md#servers-collaborators-set), [api:servers-transfer](procedures/api.md#servers-transfer)

**upgrade** (3): [helm:crd-auto-apply-hook](procedures/helm.md#crd-auto-apply-hook), [upgrade:upgrade-to-rc](procedures/upgrade.md#upgrade-to-rc), [upgrade:restart](procedures/upgrade.md#restart)

**bot-heavy** (1): [modules:nuclear-option](procedures/modules.md#nuclear-option)


## Status-wording change set

Copied from [contracts/status-wording.md](../contracts/status-wording.md) (T067 copy step) on 2026-10-05. That contract was last re-checked on 2026-10-04 on master `6750ba87`; it was not re-run or edited here. Applied as one commit only after the go decision (T071). Re-run the re-check command once more before applying.

### Wording

| Replace | With |
|---|---|
| `Status: **beta** (\`v0.2.0-beta.8\`)` and variants | `Status: **pre-v1 release** (\`v0.3.0\`)` |
| `## Beta Status & Limitations` | `## Pre-v1 Status & Limitations` (update the `#beta-status--limitations` anchor at `README.md:10`) |
| `Gameplane is currently in **beta**` | `Gameplane is a **pre-v1 release**` |
| `**Status:** Beta (\`v0.2.0-beta.8\`)` in CLAUDE.md | `**Status:** Pre-v1 release (\`v0.3.0\`)` |
| `**Status:** beta (v0.2.0-beta.8)` in module `specs.md` | `**Status:** pre-v1 (v0.3.0)` |
| roadmap "between beta and a v1 GA" | "between v0.3 and a v1 GA" |

The new wording must not claim v1, "stable" or production support (OD-003). The remaining caveats stay in `docs/roadmap.md` under "Wanted for v1, not blocking".

### Version markers → `0.3.0`

- `charts/gameplane/Chart.yaml:5-6`: `version` and `appVersion`
- `web/package.json:4`, and the regenerated `web/package-lock.json` top-level version
- `README.md:8,39,61` (also the `#beta-status--limitations` link at `README.md:10`, the `## Beta Status & Limitations` heading at `README.md:37`, and "rolling beta" at `README.md:216`)
- `CLAUDE.md:5` (was `:6`; the line now reads `**Status:** beta \`v0.2.0-beta.8\`; v1 scope feature-complete, stabilizing.`, so the table row for CLAUDE.md matches by meaning, not by literal text)
- `docs/roadmap.md:3,6,223` (was `:211`; line 6 and 223 hold the "between beta and a v1 GA" wording)
- `docs/install.md:14` (example version), `docs/install.md:34` (edge-channel heading wording, was `:27`), `docs/install.md:636` ("Releases up to `0.2.0-beta.8`", new: reword by meaning, the fact stays true)
- `docs/dependencies.md:29,33,320` (was `:26,28,226`)
- `telemetry-receiver/README.md:9,28`, and `telemetry-receiver/specs.md:53,65` (new: the example payload and metric carry `0.2.0-beta.7`)
- `web/specs.md:745` (was `:740`, the `package.json` description line)
- Status lines in `*/specs.md`: `agent`, `api`, `audit-syslog-bridge`, `capture-sidecar`, `gameaction`, `gameproto`, `mcp-server`, `netguard`, `operator`, `sentinel`, `telemetry-receiver`, `tunnel`, `web`, `test/e2e`, and `test/e2e/internal/**/spec.md`. Keep each file's existing qualifiers, such as "probe depth measured" and "in-progress". Some `test/e2e/internal/*/spec.md` status lines carry no version (`**Status:** beta; probe depth measured: ...`, 13 files, plus one `**Status:** Beta (implemented, not yet CI-verified)` and one `**Status:** beta (heavy-set, hand-run only)`): change the bare word to `pre-v1` and keep the qualifier. The other `test/e2e` status lines, including `test/e2e/specs.md:3` and `test/e2e/internal/specs.md:3`, carry `(v0.2.0-beta.8)` and follow the module `specs.md` row of the Wording table.
- `.github/workflows/publish-edge.yaml:3`: the comment says "beta images". Change it to "edge images".

### Coupled changes, in the same release but not wording

- The upgrade baseline goes from `0.2.0-beta.5` to `0.2.0-beta.8`: `deploy/kind/upgrade.sh:35`, `.github/workflows/ci.yaml:869-881,1307`, `.claude/agents/ci-triager.md:66`. **Already landed on master** (re-check 2026-10-04): all three now read `0.2.0-beta.8`. The fixture `charts/gameplane/testdata/upgrade-from-v0.2.0-beta.8-values.yaml` is a frozen copy of beta.8's values (its `ref: v0.2.0-beta.6` at `:431` is intentionally historical, leave it).
- The default git module source `ref: v0.2.0-beta.6` (`charts/gameplane/values.yaml:576`, was `:473`) moves to the `gameplane-module` tag tested with v0.3.0.
- `hack/check-doc-versions.sh` only recognises `-beta.N` versions: see lines 19, 94, 103 and 151, where the single-digit `[0-9]` pattern disagrees with the `[0-9]+` in the header. Without a fix it can't catch stale `0.3.0` or `-rc.N` strings. It has to accept `X.Y.Z` and `X.Y.Z-(beta|rc).N` before the wording commit. This is seeded as a finding. **Already landed on master** (re-check 2026-10-04): `hack/check-doc-versions.sh:19-26,110-121` now matches bare `v?0.X.Y` and `-(beta|rc).N` (OD-011). Confirm in the T071 pass that the script has no remaining single-digit patterns.
- `hack/check-links.sh:60-67,211`: the anchor self-test hard-codes `"Beta Status & Limitations"` -> `beta-status--limitations`. Keep it as a pure slug-algorithm example (it does not read the README), or update it with the heading; either way `make check-links` must pass after the README rename.
- `CHANGELOG.md` gets a `## [0.3.0]` section. The "Unreleased" entries, including the OIDC Helm role mappings flagged in 012 OD-8, move into it or into the right RC sections.

### Left as-is (historical or unrelated)

- Lines marked `<!-- doc-versions: historical -->` (including `README.md:232`, `docs/install.md:61,694`, `docs/oidc.md:50,91,383`, `docs/roadmap.md:18,36,41,179`), "shipped v0.2.0-beta.X" roadmap headings, and past CHANGELOG sections.
- Chart comments about `<= 0.2.0-beta.5` behaviour (`charts/gameplane/templates/api.yaml:231,332`, was `:219,315`; `crd-apply-hook.yaml:115`, was `:71`).
- Kubernetes "This is a beta field" text in generated CRDs, `service.beta.kubernetes.io` annotations, and test fixtures that use "beta" as a server name.
- Fixtures and tooling that use these strings on purpose: `hack/testdata/check-doc-versions/**` (the checker's own pass/fail fixtures), `charts/gameplane/testdata/upgrade-from-v0.2.0-beta.8-values.yaml`. (`hack/check-links.sh:60-67,211` is not left as-is: see Coupled changes above.)
- Roadmap headings "(shipped v0.2.0-beta.8)" (`docs/roadmap.md:76,100,143,189,204`), `.github/workflows/release.yaml:265` (comment about past release pages), `deploy/kind/upgrade.sh:80` ("beta.5 published no web").
- `web/specs.md:455,510` ("released/beta/unstable" version channel in the picker, a product term), `docs/comparison-sources.md:24,307` (24 quotes the README status line as verified evidence: re-verify after the README change; 307 is Agones's own "Beta" label).
- Test fixtures using "beta"/"Beta"/"alpha" as names, in `*_test.go`, `*.test.tsx`, `web/src/test/handlers.ts:165`, `web/e2e/specs/restoreFlow.spec.ts:51`, `web/src/routes/tabs/settings/Networking.tsx:450` (`service.beta.kubernetes.io`), `web/src/test/screenshotData.ts:1088` (sample log line, a version string in demo data: bump only if the screenshot dataset is refreshed).
- Web tests that assert no version or "beta" text leaks onto public pages (`web/src/routes/Share.test.tsx:137,559`, `web/e2e/specs/slice5.spec.ts:124`). These are login-privacy guards and must stay.
- `reddit.md` (`:6,23`) and `IDEA.md` (`:10`). They're maintainer notes. Flag them to the maintainer, but they're not part of the product docs.

### Re-check command

```sh
git grep -n -E '0\.2\.0-beta\.[0-9]+|\b[Bb]eta\b' -- ':!specs/**' ':!CHANGELOG.md' ':!**/package-lock.json' ':!*.pen' ':!design-export/**'
```
Every remaining hit must fall under "Left as-is".

### Re-checked 2026-10-04 on master 6750ba87

Re-ran the command above on a checkout with master `6750ba87` merged in (T067). 256 matching lines, against 239 when the same command is run at the 2026-09-23 commit `652df873`. Almost all of the +17 are test fixtures and tooling that fall under "Left as-is". Product-facing changes:

- **Moved:** `CLAUDE.md:6`->`5` (and its wording changed), `docs/roadmap.md:211`->`223`, `docs/install.md:27`->`34`, `docs/dependencies.md:26,28,226`->`29,33,320`, `web/specs.md:740`->`745`, `charts/gameplane/values.yaml:473`->`576`, `.github/workflows/ci.yaml:968-970`->`869-881,1307`, `deploy/kind/upgrade.sh:36`->`35`, `.claude/agents/ci-triager.md:65`->`66`, `charts/gameplane/templates/api.yaml:219,315`->`231,332`, `crd-apply-hook.yaml:71`->`115`. The list above carries the new numbers.
- **New hits to change:** `docs/install.md:636`, `telemetry-receiver/specs.md:53,65`, the version-less `Status: beta` lines in 15 `test/e2e/internal/*/spec.md` files, `README.md:10,37,216` (listed explicitly now).
- **New hits left as-is:** `hack/testdata/check-doc-versions/**`, `charts/gameplane/testdata/upgrade-from-v0.2.0-beta.8-values.yaml`, `docs/comparison-sources.md:24,307`, `web/specs.md:455,510`, `release.yaml:265`, `deploy/kind/upgrade.sh:80`, the new test-fixture names. `hack/check-links.sh:60-67,211` is a coupled change (see above).
- **Resolved on master:** the `check-doc-versions.sh` pattern fix and the upgrade baseline bump (both noted above).
- **Still outstanding:** the roadmap/README/CLAUDE.md/Chart/package.json/specs.md wording and version bumps, which are applied only after the go decision. Re-run once more before T071 and copy the final set into `audit/report.md` once T066 creates it.
