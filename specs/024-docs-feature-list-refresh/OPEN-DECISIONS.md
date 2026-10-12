# Open Decisions: 024-docs-feature-list-refresh

Unsettled items. Do not treat any of the below as decided until a maintainer rules on it here.

## Settled (user, 2026-10-12)
OD-1 (FR-001): the feature catalog is a root `FEATURES.md` next to `README.md`; the website gets a features docs page with the same content, and the README links to it.
OD-2 (FR-005, FR-013): the roadmap has an "In progress" section with one line per open spec (002, 009, 013, 019, 021, 023 at the time of writing), each with its plain-words status and blocker.
OD-3 (FR-009): the website change is docs-page text only (new features page, roadmap page, stale-fact fixes); the landing page is untouched.

## Open

OD-4 (plan Constitution Check, Principle I): does this spec need a Go e2e test in `test/e2e/`?
- A (recommended): no; take the documented exception spec 012 used for docs work, verified by the sync check's fixture tests and the existing doc gates in CI.
- B: yes, as spec 023 OD-4 ruled for CI-only work; add a `test/e2e/` test that runs the sync check against the repo, registered in a bucket.
