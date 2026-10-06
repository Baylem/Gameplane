# Audit Rounds

One section per round (contracts/audit-records.md). Database snapshots are stored off-git; only their path is recorded here.

## Round setup / teardown checklist (OD-021)

Every round that runs `agent.md`/`api.md`/`modules.md`/`web.md`/`crd.md` rows needs some or all of these fixtures, per `procedures/conventions.md`. Record what was actually created/removed under that round's **Cleanup** field. Not every round needs every item — only create what that round's rows require.

Setup (before running rows):
1. `audit018-restic`: derive a Deployment + Service named `audit018-restic` from `test/e2e/fixtures/restic-server.yaml` (renamed from `gameplane-test-restic`, labeled `gameplane.io/audit: "018"`), then create the `audit018-restic` Secret in `gameplane-games` pointing `repo` at `rest:http://audit018-restic.gameplane-system.svc:8000/`, and an `audit018-restic` NetworkPolicy in `gameplane-games` (labeled `gameplane.io/audit: "018"`) allowing egress to pods labeled `app.kubernetes.io/name: audit018-restic` in `gameplane-system` on TCP 8000, since `default-deny-egress` (networkPolicies enabled) would otherwise block backup Job pods from reaching it (OD-021 items 9/16).
2. `audit018-games2` namespace plus one small `audit018-` GameServer in it (item 14).
3. `capture.enabled=true`: `helm upgrade` override, previous value recorded here (item 17).
4. `audit018-registry` (in-cluster OCI registry) plus an `audit018-` OCI ModuleSource; push one signed and one unsigned bundle (items 6/21).
5. CSI snapshot support: install `csi-driver-host-path` and the snapshot controller/CRDs (item 20).
6. `audit018-collab` account: role `audit018-norole` (no permissions), user `audit018-collab`, collaborator grant on one `audit018-` server; credentials in `~/gameplane-audit-018/collab.env` (item 15).

Teardown (after running rows, reverse order):
1. Remove the `audit018-collab` collaborator grant, delete the `audit018-collab` user, delete the `audit018-norole` role, delete `~/gameplane-audit-018/collab.env`.
2. Uninstall the CSI snapshot controller/CRDs and `csi-driver-host-path`.
3. Delete the `audit018-` OCI ModuleSource, then the `audit018-registry` Deployment/Service.
4. Restore `capture.enabled` to its prior value (`helm upgrade --set capture.enabled=false`, or the round's recorded prior value); snapshot-diff the config it affects.
5. Delete the `audit018-games2` GameServer, then the `audit018-games2` namespace.
6. Delete the `audit018-restic` Secret, then the `audit018-restic` NetworkPolicy, then the `audit018-restic` Deployment/Service.

## rc.0

Pre-RC round: component reviews, inventory enumeration and the known-bug import. No release candidate is deployed.

- **Tag**: none (pre-RC)
- **Helm overrides vs baseline**: none
- **DB snapshot location (off-git)**: `~/gameplane-audit-018/db-snapshots/`
- **kubelab connectivity (T004, OD-007)**: ok on 2026-09-23. `getent hosts kubelab-api` → `10.43.153.36 kubelab-control.kubelab.svc.cluster.local`. `kubectl get nodes -o wide` → `kubelab-control` (control-plane), `kubelab-worker-1`, `kubelab-worker-2`, all Ready, k3s `v1.36.2+k3s1`. Helm release `gameplane` in namespace `gameplane-system`, revision 6, chart `gameplane-0.2.0-beta.8`.
- **Rows run**: 0
- **New findings**: see [findings.md](findings.md)
- **Cleanup**: Delete `audit018-admin` account (see test resources table).
- **Admin bootstrap (T012, OD-015)**: `audit018-admin` was created via `bootstrap-admin` on 2026-09-24 15:15 UTC (`kubectl exec -n gameplane-system deploy/gameplane-api -- /api bootstrap-admin --username audit018-admin --password-stdin`, no `--force`, no existing user touched). Written straight into the live DB, so there is no API audit event for its creation. Password kept off-git in `~/gameplane-audit-018/admin.env` (mode 600). To be deleted at cleanup.

### Test resources

| Name | Kind | Created by | Round | Removed |
|------|------|------------|-------|---------|
| audit018-admin | User | bootstrap-admin cmd | rc.0 | pending |

## rc.1

Release candidate round that never became installable. The tag was pushed, but the release run was cancelled before the operator image, the chart and the GitHub release were published. No rows were run against rc.1; it is superseded by rc.2 (RC-TAG-2 in [OPEN-DECISIONS.md](../OPEN-DECISIONS.md)).

- **Tag**: `v0.3.0-rc.1`, annotated and unsigned (OD-013), tag object `7d9c1653b4bb5ba0dc5454a3ad6cc4954e9f895f`, tagger date 2026-09-24T19:52:15Z, pointing at commit `c44cb1798a996ea834a86ff65b9e20abab3c401d`. Approval: RC-TAG-1 (approved and tagged 2026-09-24).
- **Helm overrides vs baseline**: none (never deployed).
- **DB snapshot location (off-git)**: none (never deployed).
- **Rows run**: 0
- **New findings**: [F-257](findings.md) (S2, fixed by #435, merged 2026-09-24T22:45:26Z)
- **Cleanup**: nothing was created on kubelab for this round.

### Release run

`release.yaml` run [36051057887](https://github.com/ValgulNecron/Gameplane/actions/runs/36051057887), event `push`, head `c44cb1798a996ea834a86ff65b9e20abab3c401d`. Conclusion: **cancelled**. Attempt 1 (started 2026-09-24T19:52:19Z) was cancelled at 20:22:52Z; attempt 2 (re-run of the failed jobs) was cancelled the same way at 20:54:00Z. No other `release.yaml` run exists for this tag.

| Job | Conclusion |
|-----|------------|
| build & push images: agent, api, audit-syslog-bridge, capture-sidecar, mcp-server, sentinel, telemetry-receiver, web | success |
| build & push images: tunnel-frp, tunnel-playit, tunnel-tailscale | success |
| build & push images: operator | cancelled |
| push & sign module bundles | success |
| package Helm chart | skipped |
| publish GitHub release | skipped |

### Published vs not published

- Published (job success; per-tag presence in GHCR and `cosign verify --key cosign.pub` not checked, see below): every component image except the operator, and the module bundles.
- Not published: the `operator` image `v0.3.0-rc.1`; chart `oci://ghcr.io/valgulnecron/charts/gameplane:0.3.0-rc.1` (packaging job skipped, `needs: images`); the GitHub release (`gh api releases/tags/v0.3.0-rc.1` returns 404; the newest release is `v0.2.0-beta.8`).
- Because the chart and the GitHub release do not exist, the prerelease-flag and chart-signature checks of contracts/rc-deploy.md §1 are moot for rc.1.

### Deviation

F-257: the multi-arch operator image build exceeds the job's `timeout-minutes: 30` under QEMU emulation, so a tag push cannot complete. Fixed by #435 (cross-compile Go images). Whether the fix works end to end is verified by the rc.2 release run, not by rc.1.

### Verdict

rc.1 is not installable (no operator image, no chart) and is not deployed. It is superseded by rc.2 (RC-TAG-2). The `v0.3.0-rc.1` tag stays in place; do not move or reuse it.

### Not checked in this record

- `cosign verify --key cosign.pub` on the images that did build: the cloud session has no cosign and the GitHub REST API does not list GHCR tags for this repository.
- That no `0.3` image tag was created or moved: not checkable through the REST API here.

### Test resources

None created.

## live-sweep 2026-10-04

Informal live feature sweep of `master`, recorded after the fact from the descriptions of #548 and #549 (both merged 2026-10-05). It was not run as a formal per-row round (see OD-029 for how it is counted): per-row inventory outcomes (the T025-T033 format) were **not** recorded, so this sweep does not replace the formal per-row round, which is still pending rc.2.

- **Target**: `master` at the time (not an RC tag; RC-TAG-2 is still pending).
- **Cluster**: kubelab, 3-node k3s.
- **Method**: browser-driven smoke test of the dashboard plus the admin bootstrap guide (`docs/install.md`), and a feature sweep (the PR text gives no more detail).
- **Run by**: session [session_01NsN6bnAMRM9XQtrTQ6KC7U](https://claude.ai/code/session_01NsN6bnAMRM9XQtrTQ6KC7U).
- **Rows run**: not recorded.
- **Helm overrides vs baseline / DB snapshot location**: not recorded.
- **Fixes**: [#548](https://github.com/ValgulNecron/Gameplane/pull/548) (events replay, server uptime, sign-in audit; merged 2026-10-05T00:19:26Z) and [#549](https://github.com/ValgulNecron/Gameplane/pull/549) (everything else fixed; merged 2026-10-05T01:22:53Z).
- **New findings**: [F-273 to F-293](findings.md). Status `fixed-unverified` for the fixed ones (to be verified live on an RC), `open` for the deferred ones.
- **Cleanup / test resources**: not recorded.

### Bugs found

#549 numbers the bugs B1 to B20. The PR texts name B1-B12, B15, B16 and B20; the four findings fixed by #548 carry no B-number there. B13, B14 and B17-B19 are not described in either PR, so they are not recorded here. Mapping of B2/B5 and B4/B12 to individual symptoms is not stated.

| ID | Short description | Fix | Status | Finding |
|----|-------------------|-----|--------|---------|
| (none) | `/events` replayed every existing object as ADDED on connect | #548 | fixed-unverified | F-273 |
| (none) | `status.startedAt` not refreshed after a pod restart (uptime showed age) | #548 | fixed-unverified | F-274 |
| (none) | Successful sign-ins audited as `anonymous` | #548 | fixed-unverified | F-275 |
| (none) | Stopped tab includes Failed servers (intentional, documented in `web/src/lib/servers.ts:15-16`) | none | not-a-defect | F-276 |
| B1 | Uploaded files and mods unreadable by the game container (agent temp-file mode) | #549 | fixed-unverified | F-277 |
| B10 | Logs tab did not replay recent history | #549 | fixed-unverified | F-278 |
| B2, B5 | Backup/restore Jobs: template fsGroup, unbounded retries and deadline | #549 | fixed-unverified | F-279 |
| B6 | Deleting a BackupSchedule cascade-deleted its backups | #549 | fixed-unverified | F-280 |
| B8 | Server not restarted after a world wipe | #549 | fixed-unverified | F-281 |
| B20 | Expected races logged as reconciler errors | #549 | fixed-unverified | F-282 |
| B4, B12 | Argon2 login burst OOM-killed the API; user timestamps not RFC 3339 | #549 | fixed-unverified | F-283 |
| B7 | Revoked share links not shown as Revoked | #549 | fixed-unverified | F-284 |
| B15 | No per-node pod usage on the Cluster page (RBAC widening, see OD-028) | #549 | fixed-unverified | F-285 |
| B16 | `instanceName` required | #549 | fixed-unverified | F-286 |
| B9 | Hidden-tab event streams exhausted the 6-connections-per-origin limit | #549 | fixed-unverified | F-287 |
| B11 | Small dashboard fixes (proxy HTML errors, game version, transfer audit label, next-backup time, backup repo Secret key, image placeholder) | #549 | fixed-unverified | F-288 |
| B3 | Terraria and tModLoader TCP readiness probe | deferred | open | F-289 |

### Deferred

- **B3** (F-289): needs a separate `gameplane-module` PR plus an e2e update.
- **Design-first items** (need a Pencil design PR before code): 21c "starting up" vs "waking up" copy on the public share page (F-290); the AdminSettings `instanceName` hint text (F-291); 18c Files folder selection/delete (F-292); 21h console history (F-293).

### Not recorded

- Per-row inventory outcomes (T025-T033 format). The maintainer decided on 2026-10-05 ([OD-029](../OPEN-DECISIONS.md)) that this sweep on the latest `master` commit stands in for the formal round on rc.2, so T025-T033 are withdrawn; T034 (cleanup) stays open.
- Evidence files, request/response logs and the rows that were run.
- Whether #548/#549 coverage and CI results hold on an RC build: `fixed-unverified` until re-verified live.

### Test resources

Not recorded.

## v0.3.0

Release record for the public `v0.3.0` tag (T073). The maintainer chose to skip rc.2 and tag `v0.3.0` directly after the go decision ([report.md#decision](report.md#decision)).

- **Tag**: `v0.3.0`, annotated, tag object `d6f7a9afc1e2480da928b6418b8dcb438bb5866b`, tagger valgulnecron, 2026-10-06T04:03:41Z, pointing at commit `040e10ee16253f6ead593bd076ed90835b68e2c1` (merge of [#587](https://github.com/ValgulNecron/Gameplane/pull/587)). GitHub reports the tag object as signature-verified; OD-013 asked for an unsigned tag, but a signed tag does not change the release pipeline.
- **Module ref**: the chart default `ref: v0.3.0` resolves to `gameplane-module` tag `v0.3.0` (tag object `bca8143ddaac15cd392fe709a2e288156564dd89`, commit `b42eea2f051b7b4b416758d7c9763843ac32c146`, the commit the `modules` submodule points at on master).

### Release run

`release.yaml` run [37411911079](https://github.com/ValgulNecron/Gameplane/actions/runs/37411911079), event `push`, head `040e10ee16253f6ead593bd076ed90835b68e2c1`, 2026-10-06T04:03:44Z to 04:10:54Z. Conclusion: **success**, every job green (the F-257 timeout from rc.1 did not recur).

| Job | Conclusion |
|-----|------------|
| build & push images: agent, api, audit-syslog-bridge, capture-sidecar, mcp-server, operator, sentinel, telemetry-receiver, web | success |
| build & push images: tunnel-frp, tunnel-playit, tunnel-tailscale | success |
| push & sign module bundles | success |
| package Helm chart | success |
| publish GitHub release | success |

### Checks (contracts/rc-deploy.md §1)

| Check | Result |
|-------|--------|
| Images signed | Each image job ran `sign image (keyed, logged)` and `verify signature & check published key`, both success. For `api`, the cosign signature manifest `sha256-ccbe2975….sig` exists in GHCR for the release digest. |
| Chart signed | `sign chart (keyed, logged)` and `verify chart signature` success; `oci://ghcr.io/valgulnecron/charts/gameplane` lists tag `0.3.0`. |
| GitHub release not prerelease | [Release v0.3.0](https://github.com/ValgulNecron/Gameplane/releases/tag/v0.3.0), published 2026-10-06T04:10:51Z: `prerelease: false`, `draft: false`, and it is the repository's latest release. |
| Notes from `## [0.3.0]` | The release body opens with the `## [0.3.0]` section text of `CHANGELOG.md` at the tag ("The first Gameplane release without a pre-release suffix…"). |
| `0.3` image tag | `ghcr.io/valgulnecron/gameplane/api` tags `0.3.0`, `v0.3.0`, `0.3` and `latest` all resolve to digest `sha256:ccbe29755dd72fe191f97f0a3bc36e13bd672d6578e659272c9cb0be64198ae3`. |
| CI on the tagged commit (RC-08) | The `ci` push run on `040e10e` ([37411539484](https://github.com/ValgulNecron/Gameplane/actions/runs/37411539484)) was cancelled by `cancel-in-progress` when later PRs merged into master. The PR head it merges ([#587](https://github.com/ValgulNecron/Gameplane/pull/587), `23c23401`) passed the full `ci` run (19 jobs passed, 0 failed); see RC-08 in [release-criteria.md](release-criteria.md) for the re-run. |

### Other tag-triggered workflows

- `screenshot refresh` run [37411911070](https://github.com/ValgulNecron/Gameplane/actions/runs/37411911070) failed (3 of 139 gallery shots: `mods-registry-browse`, `server-console`, `server-detail-logs`). It has failed on every run since 2026-09-06, including `v0.3.0-rc.1`; it does not publish release artifacts and is not a release criterion.

### Test resources

None created.
