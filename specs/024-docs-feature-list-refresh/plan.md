# Implementation Plan: Documentation Refresh, Feature Catalog, and Roadmap Status

**Branch**: `claude/project-thread-6pzhf7` | **Date**: 2026-10-12 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `specs/024-docs-feature-list-refresh/spec.md`

## Summary

Write a root `FEATURES.md` that lists every Gameplane capability, one row per capability, grouped by area, each with an evidence-based maturity label, its install default and a docs link. Mirror it as a website docs page and add a CI check that fails when the two drift. Refresh `docs/roadmap.md` (and the website roadmap page) to the current release, with what shipped in `v0.3.0` and since, and one line per open spec. Sweep the README, `docs/`, component READMEs, the module README and the website docs for stale facts. Website changes are docs text only. Research: [research.md](./research.md).

## Technical Context

**Language/Version**: Markdown (GFM) in the main and module repos; MDX in the website (Astro, strict TypeScript); Bash + awk for the sync check

**Primary Dependencies**: existing doc gates `hack/check-links.sh`, `hack/check-doc-versions.sh`; website content collection schema `src/content.config.ts` and nav `src/lib/docs-nav.ts`; GitHub Actions `ci.yaml` `docs and repository checks` job

**Storage**: N/A

**Testing**: fixture tests `hack/test-check-features-sync.sh` (`make test-features-sync`) in CI; existing `make check-links`, `make check-doc-versions` in CI; website CI `npm run lint`, `check`, `build`. Principle I treatment is pending OD-4.

**Target Platform**: GitHub (rendered Markdown), GitHub Pages website, GitHub Actions `ubuntu-latest`

**Project Type**: documentation plus one repository tooling script

**Performance Goals**: sync check runs in a few seconds inside the existing 10-minute docs job

**Constraints**: no doc gate relaxed (FR-011); competitor comparison columns unchanged (FR-014); CHANGELOG not rewritten; no `.pen` edits and no landing page change (OD-3); only compile/build checks locally (rule 8); submodule pointers bumped only after their PRs merge

**Scale/Scope**: about 100 to 150 catalog rows across ~18 areas; 22 `docs/*.md` files, root README, 4 component READMEs, the module README, ~90 website docs pages (only those carrying stale facts change, plus new `features.mdx` and `roadmap.mdx`); 6 open specs on the roadmap

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|---|---|---|
| I. E2E-Tested Delivery | PENDING OD-4 | Docs feature plus a CI-only check. Precedents differ: spec 012 (docs) took a justified exception; spec 023 (CI-only) was ruled to need a Go e2e test. Plan default until valgul answers OD-4: the spec 012 exception, verified by the sync check's fixture tests and the existing doc gates (see Complexity Tracking). |
| II. Design-First | PASS | No dashboard change. Website change is docs-page text only (OD-3); a new docs page is frontmatter-only per the website CLAUDE.md, so no `website.pen` change. No `.pen` file is read or touched. |
| III. Ecosystem Best Practice | PASS | Bash script follows the sibling doc gates; no suppressions; no lint config change. |
| IV. Spec-Driven | PASS | spec → clarify → plan done; tasks next. No module behaviour changes, so no `specs.md` updates. Archive to `done_024` after all PRs merge. |
| V. Delegate | PASS | Inventory and stale-fact sweep run as a Workflow: one `haiku` scout per area/fact class writes rows and evidence, one `sonnet` reviewer checks the combined diff against sources; every `agent()` sets `model`. |
| VI. CI Bears the Load | PASS | Sync check, its fixtures and doc gates are verified in CI only. Locally: website `npm run build`/`check` only. |

## Project Structure

### Documentation (this feature)

```text
specs/024-docs-feature-list-refresh/
├── spec.md
├── OPEN-DECISIONS.md
├── plan.md               # this file
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   ├── features-catalog.md
│   └── features-sync-check.md
├── checklists/requirements.md
├── feature-evidence.md   # implementation output: evidence per catalog row
├── stale-facts.md        # implementation output: one record per correction
└── tasks.md              # /speckit-tasks
```

### Source files touched

```text
GameplanePanel/Gameplane
├── FEATURES.md                                 # NEW catalog (OD-1)
├── README.md                                   # stale facts; Features → summary + link
├── docs/roadmap.md                             # status refresh (OD-2)
├── docs/*.md                                   # stale-fact fixes where found
├── audit-syslog-bridge/README.md, containers/README.md,
│   mcp-server/README.md, telemetry-receiver/README.md
├── hack/check-features-sync.sh                 # NEW (FR-015)
├── hack/test-check-features-sync.sh            # NEW fixture runner
├── hack/testdata/check-features-sync/{pass,fail}/   # NEW fixtures
├── hack/check-links.sh, hack/check-doc-versions.sh  # add FEATURES.md to audited lists
├── Makefile                                    # check-features-sync, test-features-sync, lint
├── .github/workflows/ci.yaml                   # docs job: init website submodule, run check; path filter
└── website, modules                            # submodule pointer bumps (after their PRs merge)

GameplanePanel/website
├── src/content/docs/features.mdx               # NEW mirror page
├── src/content/docs/roadmap.mdx                # status refresh
└── src/content/docs/*.mdx                      # stale-fact fixes where found

GameplanePanel/module
└── README.md                                   # only if module list/count is stale
```

**Structure Decision**: no new code project; one Bash tool joins the existing `hack/` doc gates, and content changes land in the three existing repos.

## Delivery

1. Spec docs PR (spec, plan, tasks) to `master`.
2. Draft `FEATURES.md`, `feature-evidence.md` and `stale-facts.md` on the main branch (not yet merged).
3. Module repo PR if needed; website PR with `features.mdx` copied from the draft catalog, roadmap page and stale fixes. Both merge first.
4. Main repo PR: everything else plus both pointer bumps. CI runs the sync check against the merged website page.
5. Archive `specs/024-…` to `specs/done_024-…` after merge.

Coordination: spec 023 is reshaping CI job selection; whichever lands second maps the new files into the other's structure (research R10).

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| Principle I exception (proposed, pending OD-4) | The feature changes no runtime path: catalog, roadmap and doc fixes describe shipped code, and the sync check is a repo lint that has no cluster behaviour to exercise. Verification is the fixture tests (every mismatch kind rejected, in-sync accepted) plus the existing doc gates in CI. | A Go e2e test in `test/e2e/` would boot a kind cluster to run a text comparison; it would test nothing the fixture tests do not. Same reasoning as `specs/done_012-docs-refresh-and-outreach/plan.md`. If valgul rules otherwise on OD-4, add the e2e test the ruling names. |
