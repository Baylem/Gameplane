# Storage Pool Inventory Implementation Plan

> For implementers: execute the agreed steps in order; parent review and CI gate production changes.

**Goal:** Report configured game-data pool capacity instead of the node system disk, with truthful provisioned-storage labels and unknown states.

**Architecture:** An optional administrator Node annotation selects an exact OpenEBS Local PV LVM thin pool. One read-only resolver serves both cluster stats and fleet inventory using the selected cluster client. Unconfigured clusters retain the existing node-ephemeral calculation.

**Tech stack:** Existing Go Kubernetes typed/dynamic clients and React/TypeScript.

**Spec:** This accepted change brief extends `api/specs.md` inventory and `web/specs.md` fleet-storage contracts. Base: v0.4.1 commit `657462a4d8ec916305b8499995e98c026fc1157c`.

## Global constraints

- No DB migration, registration/configuration UI, CRD change, operator fork, metrics dashboard or pool management.
- The Node annotation is administrator-owned. Readers gain no mutation or Secret permissions.
- No source-capacity number is configured or hardcoded; read the selected pool's live reported size.
- No physical-used metric is added. Existing `usedStorageBytes` remains logical Bound-PV provisioned bytes.
- Only local compile checks are allowed. Test/lint suites run in CI only; preserve user authorship/signatures and add no AI attribution.

## Frozen annotation and response contract

Node annotation key: `gameplane.local/storage-pool`. Value is one JSON object:

```json
{"provider":"openebs-lvm","namespace":"openebs","volumeGroup":"openebs-chicago-1","thinPool":"openebs-chicago-1_thinpool","storageClasses":["openebs-lvm"]}
```

All fields are required; provider is exactly `openebs-lvm`; reject unknown fields, invalid names, empty class lists, malformed JSON and duplicate/ambiguous references. The annotated Node's name is the exact LVMNode name; no arbitrary target name or URL is accepted.

Read only `local.openebs.io/v1alpha1`, namespace from the annotation, resource `lvmnodes`, name from that Node. Resolve the exact top-level `volumeGroups[name].thinPools[name].size`, accepting valid positive Kubernetes quantity strings or integers. The actual source has no `spec`/`status` wrapper or freshness field. The controller writes only when values change: age/resourceVersion cannot prove staleness and no age heuristic is introduced.

Existing stats fields remain, plus `storageCapacitySource`:

```json
{"nodes":1,"totalStorageBytes":343597383680,"usedStorageBytes":0,"storageCapacitySource":"openebs-lvm"}
```

Per-cluster source is `node-ephemeral` or `openebs-lvm`; frontend aggregation may produce `mixed`. Missing source from an older API is interpreted as legacy node-ephemeral. No Node annotations means the existing node-ephemeral capacity and all-Bound-PV behavior. With any configured references, sum only the explicitly referenced local pools; never add unannotated nodes' system disks.

Provisioned bytes in configured mode include only Bound PVs in the configured storage classes with CSI driver `local.csi.openebs.io`, matching `openebs.io/volgroup` and actual affinity to a referenced Node. Exclude Released/unbound PVs and other classes/VGs/nodes; count each PV once. Preserve Kubernetes node-affinity OR/AND semantics and reject malformed/ambiguous metadata rather than silently invent allocation.

Configured lookup/parse failures are unavailable: cluster stats returns the existing error path; fleet omits that inventory item and reports partial coverage. Preserve authorization error handling and generic error messages. Never substitute EPHEMERAL, another cluster, a partial pool sum or zero. Named read-only LVMNode GET permission is sufficient.

## Exact implementation surfaces

- New `api/internal/handlers/storage_inventory.go`: validated annotation, named dynamic lookup, pool/PV scope resolution and source metadata; no writes.
- `api/internal/handlers/cluster.go:160-179,386-440`: add response discriminator and call the shared resolver.
- `api/internal/handlers/fleet.go:567-627`: use the same resolver; retain existing per-cluster issue semantics.
- New `api/internal/handlers/storage_inventory_test.go`: handler-route regression cases using existing fixtures/fake clients; tests must compile against unchanged production code for the red CI phase.
- Web owner: `web/src/types.ts`, `web/src/lib/fleet.ts`, `web/src/lib/utils.ts`, `web/src/routes/Dashboard.tsx` and their existing tests. Keep a storage-only IEC formatter so 343597383680 bytes reads 320 GiB without changing every existing byte display. Captions distinguish pool, node-ephemeral and mixed reported capacity; numerator is provisioned, never physical usage.
- Update owning `api/specs.md`/`web/specs.md` and read-only inventory documentation with the final contract; parent owns RBAC/deployment and changelog integration.

## Review focus and regression cases

1. Correct pool versus system disk: both `/cluster/stats?cluster=remote` and `/fleet/inventory?cluster=remote` return pool 320 GiB while the node reports 100 GiB; source metadata is correct and lookup is only GET of the annotated LVMNode.
2. Configured failure: missing/forbidden pool, invalid quantity, malformed annotation and wrong provider produce errors/partial coverage with no node/home/zero fallback and no leaked upstream details.
3. Numerator scope: a matching 7 GiB Bound PV is counted; Released, wrong class/VG/node and unrelated CSI PVs are excluded. Multiple references never double-count a PV.
4. Authority/selection: cluster-wide inventory grant is required before pool reads; namespace/server grants do not probe the source. Remote selection never queries Home; unconfigured legacy behavior is unchanged.
5. Unknowns/units: fleet does not turn omitted failures into known zero; mixed sources retain honest coverage. Storage copy says provisioned and displays IEC units; existing overcommit behavior remains meaningful.

## Ordered steps

- [x] Write route-level regression tests only, using current handler interfaces and raw JSON assertions for the new discriminator.
- [x] Parent commits tests and observes expected red in scoped fork CI; do not run tests locally.
- [x] After parent approval, implement the shared API resolver and both call sites; run only `go build ./...` in `api/` locally.
- [ ] Web owner implements the agreed storage metadata/caption/IEC changes in parallel; parent reviews the combined diff and runs CI.
- [ ] Parent applies only reviewed read-only RBAC/annotation and publishes/deploys approved immutable artifacts; verify Chicago 320 GiB, legacy behavior, unknown error handling and unchanged resource ownership.

Implementation status: API compile-only check passed. Red regression CI run
44020592868/job133829048522 confirmed the original capacity/scope failures.
Combined final CI, artifact publication and deployment remain pending.
