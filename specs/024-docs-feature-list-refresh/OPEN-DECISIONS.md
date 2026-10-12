# Open Decisions: 024-docs-feature-list-refresh

Unsettled items. Do not treat any of the below as decided until a maintainer rules on it here.

## Open

OD-1 (FR-001): where the feature catalog lives.
- A (recommended): `docs/features.md` in the main repo is the source; the website gets a `features` docs page with the same content; the README links to it.
- B: root `FEATURES.md` next to the README, plus the website page.
- C: main repo only, no website page.

OD-2 (FR-005, FR-013): how much in-progress work the roadmap shows.
- A (recommended): an "In progress" section with one line per open spec (002, 009, 013, 019, 021, 023) and its plain-words status and blocker.
- B: only user-visible features in progress (019 language packs, 021 settings, 002 Nuclear Option); CI and test specs (009, 013, 023) left out.
- C: shipped and v1 items only, no in-progress list.

OD-3 (FR-009): how far the website change goes.
- A (recommended): text in the docs pages only (new features page, roadmap page, stale-fact fixes); landing page untouched.
- B: also refresh the landing page feature cards, which first needs a design pass in `website.pen` on the maintainer's machine.
