# Release Criteria: v0.3.0

Written before round 0 (FR-015). Columns are fixed by [contracts/audit-records.md](../contracts/audit-records.md#release-criteriamd). Every `Met` stays `pending` until the go/no-go decision (T070). Changing a criterion after this file is first committed needs a dated line under [Change log](#change-log) that cites the maintainer's decision.

| ID | Criterion | Source | Evidence | Met |
|----|-----------|--------|----------|-----|
| RC-01 | Every inventory row has an outcome, zero rows `fail`, and every `blocked` row names its prerequisite and is listed as not live-verified | SC-001 | [report.md#totals](report.md#totals), [report.md#not-live-verified](report.md#not-live-verified) | not met (accepted risk, T070) |
| RC-02 | Every component has a `complete` review record | SC-002 | [coverage.md](coverage.md), [report.md#by-component](report.md#by-component) | met |
| RC-03 | Zero findings in `imported`, `open`, `fixing` or `fixed-unverified` | SC-003, SC-004 | [report.md#open-findings](report.md#open-findings), [findings.md](findings.md) | not met (accepted risk, T070) |
| RC-04 | Each FR-008 boundary has at least one active violation attempt recorded | SC-007 | [inventory.md#sec](inventory.md#sec) | not met (accepted risk, T070) |
| RC-05 | Live beta.8 → RC upgrade with zero data loss and zero lost accounts | SC-005, OD-005 | [inventory.md#upg](inventory.md#upg) | not met (accepted risk, T070) |
| RC-06 | The baseline snapshot matches the post-cleanup snapshot, with zero `audit018-` resources remaining | SC-006 | [rounds.md](rounds.md), [kubelab-baseline.md](kubelab-baseline.md) | not met (accepted risk, T070) |
| RC-07 | Every `not-a-defect` and `out-of-scope` closure is justified, and each out-of-scope one cites a roadmap line | SC-009 | [findings.md](findings.md) | partially (accepted risk, T070) |
| RC-08 | CI is green on the tagged commit | FR-017 | [rounds.md#v030](rounds.md#v030) | pending (set after the v0.3.0 tag, T073) |

## Change log

- 2026-09-24: RC-03 now also counts `imported` findings as blocking, matching data-model.md's blocking statuses. Maintainer decision, OD-010 in [../OPEN-DECISIONS.md](../OPEN-DECISIONS.md#od-010-rc-03-leaves-out-the-imported-status--resolved-2026-09-24).
- 2026-10-05: `Met` filled at the go/no-go decision (T070, maintainer valgulnecron: go, with Not met criteria accepted as known risk; see [report.md#decision](report.md#decision)). Criteria themselves unchanged.
- 2026-10-05: RC-01 `Met` unchanged (`not met (accepted risk, T070)`) after OD-030: the maintainer attested that all screens and features were tested live on kubelab on 2026-10-04, and the 459 `untested` WEB/API/CRD/AGT/AUX/HELM/MOD rows are recorded as `pass`, but 8 `UPG`/`NODE` rows and the 16 `INV-SEC-*` rows are still `untested`, so not every row has an outcome. The criterion text is unchanged. Maintainer decision, OD-030 in [../OPEN-DECISIONS.md](../OPEN-DECISIONS.md#od-030-screen-and-feature-rows-recorded-as-pass-from-the-live-sweep--resolved-2026-10-05).
