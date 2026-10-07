# Cold-read check (T069)

Date: 2026-10-05
Model tier: sonnet

## (1) Go or no-go

**No-go.** v0.3.0 should not be released now.

## (2) Reasons

- **Release criteria (Release criteria table):** only 1 of 8 is Met (RC-02, component coverage). RC-07 is Partially met. The other 6 are Not met.
- **Open findings (Totals, Open findings):** 291 of 293 findings are in a blocking status: 25 `imported`, 37 `open`, 229 `fixed-unverified`. None is `verified`. This fails RC-03. The open set includes S1 and S2 items, for example F-115 (S1: new file truncates existing file), F-118 and F-119 (S2: tunnel create and enable flows fail), and F-014, F-015 and F-018 (S1 security items, still `imported`).
- **Inventory (Totals, Not live-verified):** 467 of 486 rows are `untested` and 0 are `pass`. 19 rows are `blocked`. The formal live round did not run (OD-029). This fails RC-01.
- **Security boundaries (RC-04):** all 16 `INV-SEC-*` rows are `untested`, so no violation attempt is recorded for any FR-008 boundary.
- **Upgrade (RC-05, Rounds):** the beta.8 to RC upgrade round and the node round have not run. Zero data loss is unproven.
- **Cleanup (RC-06):** no post-cleanup snapshot comparison exists, and the `audit018-admin` account is still pending removal.
- **Tag and CI (RC-08, header, Rounds):** `v0.3.0-rc.2` is not tagged or pushed (RC-TAG-2 is PENDING). `v0.3.0-rc.1` was never installable because its release run was cancelled. No tagged commit has a green CI run.
- **Re-verification (Rounds):** T058, which re-verifies the 229 `fixed-unverified` findings, has not run.
- **Decision section:** the maintainer's decision is still pending. The report itself says it is "not a go case".

What would change this: a tagged and installable RC with green CI, the re-verification, upgrade and security rounds run, and the blocking findings closed or triaged.

## (3) Time taken

About 6 minutes. This is an estimate: roughly 4 to 5 minutes reading, most of it skimming the long tables and the status-wording change set, and 1 to 2 minutes deciding. The no-go was clear from the header and the criteria table.
