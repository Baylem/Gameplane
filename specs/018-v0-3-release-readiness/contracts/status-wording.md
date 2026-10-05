# Contract: v0.3.0 Status Wording and Version Markers

Covers FR-015 and FR-016. Applied as one commit, only after the go decision. The locations were found with `git grep` on 2026-09-23. Re-run the grep before applying (see the end of this file).

## Wording

| Replace | With |
|---|---|
| `Status: **beta** (\`v0.2.0-beta.8\`)` and variants | `Status: **pre-v1 release** (\`v0.3.0\`)` |
| `## Beta Status & Limitations` | `## Pre-v1 Status & Limitations` (update the `#beta-status--limitations` anchor at `README.md:10`) |
| `Gameplane is currently in **beta**` | `Gameplane is a **pre-v1 release**` |
| `**Status:** Beta (\`v0.2.0-beta.8\`)` in CLAUDE.md | `**Status:** Pre-v1 release (\`v0.3.0\`)` |
| `**Status:** beta (v0.2.0-beta.8)` in module `specs.md` | `**Status:** pre-v1 (v0.3.0)` |
| roadmap "between beta and a v1 GA" | "between v0.3 and a v1 GA" |

The new wording must not claim v1, "stable" or production support (OD-003). The remaining caveats stay in `docs/roadmap.md` under "Wanted for v1, not blocking".

## Version markers → `0.3.0`

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

## Coupled changes, in the same release but not wording

- The upgrade baseline goes from `0.2.0-beta.5` to `0.2.0-beta.8`: `deploy/kind/upgrade.sh:35`, `.github/workflows/ci.yaml:869-881,1307`, `.claude/agents/ci-triager.md:66`. **Already landed on master** (re-check 2026-10-04): all three now read `0.2.0-beta.8`. The fixture `charts/gameplane/testdata/upgrade-from-v0.2.0-beta.8-values.yaml` is a frozen copy of beta.8's values (its `ref: v0.2.0-beta.6` at `:431` is intentionally historical, leave it).
- The default git module source `ref: v0.2.0-beta.6` (`charts/gameplane/values.yaml:576`, was `:473`) moves to the `gameplane-module` tag tested with v0.3.0.
- `hack/check-doc-versions.sh` only recognises `-beta.N` versions: see lines 19, 94, 103 and 151, where the single-digit `[0-9]` pattern disagrees with the `[0-9]+` in the header. Without a fix it can't catch stale `0.3.0` or `-rc.N` strings. It has to accept `X.Y.Z` and `X.Y.Z-(beta|rc).N` before the wording commit. This is seeded as a finding. **Already landed on master** (re-check 2026-10-04): `hack/check-doc-versions.sh:19-26,110-121` now matches bare `v?0.X.Y` and `-(beta|rc).N` (OD-011). Confirm in the T071 pass that the script has no remaining single-digit patterns.
- `hack/check-links.sh:60-67,211`: the anchor self-test hard-codes `"Beta Status & Limitations"` -> `beta-status--limitations`. Keep it as a pure slug-algorithm example (it does not read the README), or update it with the heading; either way `make check-links` must pass after the README rename.
- `CHANGELOG.md` gets a `## [0.3.0]` section. The "Unreleased" entries, including the OIDC Helm role mappings flagged in 012 OD-8, move into it or into the right RC sections.

## Left as-is (historical or unrelated)

- Lines marked `<!-- doc-versions: historical -->` (including `README.md:232`, `docs/install.md:61,694`, `docs/oidc.md:50,91,383`, `docs/roadmap.md:18,36,41,179`), "shipped v0.2.0-beta.X" roadmap headings, and past CHANGELOG sections.
- Chart comments about `<= 0.2.0-beta.5` behaviour (`charts/gameplane/templates/api.yaml:231,332`, was `:219,315`; `crd-apply-hook.yaml:115`, was `:71`).
- Kubernetes "This is a beta field" text in generated CRDs, `service.beta.kubernetes.io` annotations, and test fixtures that use "beta" as a server name.
- Fixtures and tooling that use these strings on purpose: `hack/testdata/check-doc-versions/**` (the checker's own pass/fail fixtures), `charts/gameplane/testdata/upgrade-from-v0.2.0-beta.8-values.yaml`. (`hack/check-links.sh:60-67,211` is not left as-is: see Coupled changes above.)
- Roadmap headings "(shipped v0.2.0-beta.8)" (`docs/roadmap.md:76,100,143,189,204`), `.github/workflows/release.yaml:265` (comment about past release pages), `deploy/kind/upgrade.sh:80` ("beta.5 published no web").
- `web/specs.md:455,510` ("released/beta/unstable" version channel in the picker, a product term), `docs/comparison-sources.md:24,307` (24 quotes the README status line as verified evidence: re-verify after the README change; 307 is Agones's own "Beta" label).
- Test fixtures using "beta"/"Beta"/"alpha" as names, in `*_test.go`, `*.test.tsx`, `web/src/test/handlers.ts:165`, `web/e2e/specs/restoreFlow.spec.ts:51`, `web/src/routes/tabs/settings/Networking.tsx:450` (`service.beta.kubernetes.io`), `web/src/test/screenshotData.ts:1088` (sample log line, a version string in demo data: bump only if the screenshot dataset is refreshed).
- Web tests that assert no version or "beta" text leaks onto public pages (`web/src/routes/Share.test.tsx:137,559`, `web/e2e/specs/slice5.spec.ts:124`). These are login-privacy guards and must stay.
- `reddit.md` (`:6,23`) and `IDEA.md` (`:10`). They're maintainer notes. Flag them to the maintainer, but they're not part of the product docs.

## Re-check command

```sh
git grep -n -E '0\.2\.0-beta\.[0-9]+|\b[Bb]eta\b' -- ':!specs/**' ':!CHANGELOG.md' ':!**/package-lock.json' ':!*.pen' ':!design-export/**'
```
Every remaining hit must fall under "Left as-is".

## Re-checked 2026-10-04 on master 6750ba87

Re-ran the command above on a checkout with master `6750ba87` merged in (T067). 256 matching lines, against 239 when the same command is run at the 2026-09-23 commit `652df873`. Almost all of the +17 are test fixtures and tooling that fall under "Left as-is". Product-facing changes:

- **Moved:** `CLAUDE.md:6`->`5` (and its wording changed), `docs/roadmap.md:211`->`223`, `docs/install.md:27`->`34`, `docs/dependencies.md:26,28,226`->`29,33,320`, `web/specs.md:740`->`745`, `charts/gameplane/values.yaml:473`->`576`, `.github/workflows/ci.yaml:968-970`->`869-881,1307`, `deploy/kind/upgrade.sh:36`->`35`, `.claude/agents/ci-triager.md:65`->`66`, `charts/gameplane/templates/api.yaml:219,315`->`231,332`, `crd-apply-hook.yaml:71`->`115`. The list above carries the new numbers.
- **New hits to change:** `docs/install.md:636`, `telemetry-receiver/specs.md:53,65`, the version-less `Status: beta` lines in 15 `test/e2e/internal/*/spec.md` files, `README.md:10,37,216` (listed explicitly now).
- **New hits left as-is:** `hack/testdata/check-doc-versions/**`, `charts/gameplane/testdata/upgrade-from-v0.2.0-beta.8-values.yaml`, `docs/comparison-sources.md:24,307`, `web/specs.md:455,510`, `release.yaml:265`, `deploy/kind/upgrade.sh:80`, the new test-fixture names. `hack/check-links.sh:60-67,211` is a coupled change (see above).
- **Resolved on master:** the `check-doc-versions.sh` pattern fix and the upgrade baseline bump (both noted above).
- **Still outstanding:** the roadmap/README/CLAUDE.md/Chart/package.json/specs.md wording and version bumps, which are applied only after the go decision. Re-run once more before T071 and copy the final set into `audit/report.md` once T066 creates it.
