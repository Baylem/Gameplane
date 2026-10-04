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
