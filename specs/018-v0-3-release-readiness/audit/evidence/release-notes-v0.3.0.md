# Gameplane v0.3.0 release notes (draft)

Draft for the maintainer (T068). Not published and not final: v0.3.0 has no go decision yet (see [../report.md](../report.md)). The highlights are copied from the Highlights of the `0.3.0-rc.1` and `0.3.0-rc.2` sections of `CHANGELOG.md`; the "Not live-verified" section is copied word for word from the report (FR-002, SC-001); the pre-v1 caveats are copied from `docs/roadmap.md`, "Wanted for v1, not blocking". Two highlights pointed at other `CHANGELOG.md` sections ("above"); they now name `CHANGELOG.md`. The rc.2 section of the changelog is marked Unreleased and `v0.3.0-rc.2` has not been tagged (RC-TAG-2 is PENDING). The release notes must be re-checked against the report when the audit finishes.

## Highlights

### Since v0.3.0-rc.1 (from the 0.3.0-rc.2 changelog section)

- **Agent Prometheus metrics move to a dedicated listener (port 9090):** the
  agent now serves `/metrics` on its own unauthenticated `:9090` listener
  instead of the mTLS control port, where every agent scrape target was
  reported "down"; the chart's `PodMonitor` scrapes it by name (#476).
- **Owner-only server operations now need the owner or an admin:** see
  Upgrade Notes in `CHANGELOG.md`.
- **Helm chart fixes:** `helm upgrade` works for releases not named
  `gameplane` and with `--reuse-values` from v0.2.0-beta.8, a reinstall over
  leftover CRDs updates them (#443), and `helm uninstall` keeps the games
  Namespace and the GameServers in it (#425).
- **Playit tunnels:** the tunnel NetworkPolicy now lets playit reach its
  relay (#488), and the playit-assigned address shows up in the
  GameServer's status (#447).
- **API fixes:** validation errors return 400/409 instead of 500 (#477), a
  required console-action parameter sent empty is rejected instead of taking
  its default (#491), and large file, log and capture transfers are no longer
  cut off by the 60 s timeout and 1 MiB body cap (#444).
- **Backup, restore and wipe:** a failed data wipe is reported instead of
  acknowledged (#436), and backup/restore quiesce-lifecycle fixes stop a
  server being left quiesced; a restore now also removes files created
  after the snapshot (#454).
- **Uploads and backups:** dashboard file and mod uploads over 1 MiB are no
  longer rejected (#426), and backup and restore Jobs can reach their
  repository when NetworkPolicies are enabled (#432).
- Further security hardening across the API, operator, agent and release
  pipeline (see Security hardening in `CHANGELOG.md`).

### In v0.3.0-rc.1 (from the 0.3.0-rc.1 changelog section)

- **Every existing GameServer pod restarts once on upgrade** due to the network
  capture feature adding an emptyDir volume; plan your upgrade window accordingly.
  This interrupts active player sessions.
- **User theme customization:** presets (Modern Pink, Legacy Orange) + custom
  accent/surface colors with WCAG AA contrast guarantee, custom CSS overlay, and
  safe-mode suspension on demand. Accessed via Settings → Theme & Appearance.
- **13 new top-steam-game modules** (FiveM, Farming Simulator 25, Euro Truck
  Simulator 2, Mount & Blade Bannerlord, tModLoader, Team Fortress 2, BeamMP,
  Left 4 Dead 2, The Isle, ARK, Arma Reforger, Hell Let Loose, Squad) with
  Gameplane-owned container images (`fivem`, `farming-simulator-25`,
  `euro-truck-simulator-2`, `beammp`).
- **Network capture for protocol reverse-engineering:** admin-only, on-demand
  packet capture from running game pods as ephemeral sidecar containers,
  downloaded as PCAPNG files (Wireshark-compatible).
- **Easy module building toolkit:** `gp-module` CLI (`init`, `validate`, `preview`,
  `package`) for scaffolding, authoring, and publishing custom game modules;
  web dashboard UI with live validation and cluster installation.
- **Install-time configuration:** Helm-seeded OIDC role mappings (no
  `bootstrap-admin` needed for OIDC-only installs) and default StorageClass for
  game-data PVCs.
- **Share link expiry options:** no-expiry, 15/30/60/90-day presets, or custom
  date; lifts the old 90-day cap that outlived long-running survival servers.
- **Console fixes:** non-default port handling (port-forward scenarios) and
  guarded `send()` before socket open.
- **Existing PVC support:** Helm `api.storage.existingClaim` lets installs
  point SQLite to a pre-existing PersistentVolumeClaim.

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

- **8 of 486 inventory rows in `audit/inventory.md` are still `untested`** (UPG 5, NODE 3); 24 of 502 with the `INV-SEC-*` rows. 459 rows (WEB 145, API 146, CRD 35, AGT 36, AUX 13, HELM 28, MOD 56) are `pass` on the strength of a maintainer attestation, not per-row evidence: the formal per-row live round did not run (OD-029) and the 2026-10-04 informal live sweep on `master` did not record per-row outcomes, so on 2026-10-05 the maintainer attested that "all screen and feature where tested live in a cluster" (OD-030). The attestation does not cover `UPG`, `NODE`, `SEC` or the 19 `blocked` rows.
- The 16 security-control rows (`INV-SEC-001` to `INV-SEC-016`) are kept in `audit/held/inventory-SEC.md`, not in `audit/inventory.md` (OD-019; that file has been in git since 8249db80) and are all `untested` there, so no active violation attempt is recorded for them.
- No upgrade or node round has run (T061, T062, T065): the beta.8 to RC upgrade (`UPG`, 5 rows) and the node behaviour rows (`NODE`, 3 rows) are `untested`.
- 229 findings are `fixed-unverified`: their fixes merged but have not been re-verified on a release candidate (T058 has not run).

## Pre-v1 caveats

This is a pre-v1 release. These items are wanted for v1 and are not blocking v0.3.0:

### Production-readiness hardening (planned)

- A documented backup/restore **drill** — a runbook an operator can follow.
  The *coverage* half of this item is already done and was stale here:
  `TestRestore_RoundTrip` (`test/e2e/restore_e2e_test.go`) writes a marker into
  a server's PVC, backs it up, wipes it, restores, and asserts the bytes came
  back — against the real in-cluster restic-server that `ensureResticRepo`
  provisions. What is missing is the human-facing runbook, not the test.
- Resource-limit guidance sized from real workloads rather than defaults.

### Postgres driver: production readiness (experimental)

SQLite is the only production-tested driver. The Postgres driver (build tag
`-tags postgres`) now works end to end (#518): migrations 001–012 have
hand-written Postgres equivalents (`api/internal/db/migrations/postgres/`),
every later migration is one portable file both drivers run
(`migrations/common/`), the Postgres connection rewrites `?` placeholders to
`$n`, and runtime timestamps are generated in Go. The `api (postgres)` CI job
builds the api with the tag and runs `api/internal/db`'s tests against a real
PostgreSQL server. It stays **experimental** until the rest lands: running the
handler/auth/audit test suites and the kind e2e and upgrade suites against
Postgres, a published image built with `-tags postgres`, and multi-replica
safety (the user-management lock and the audit hash chain assume a single API
process).
