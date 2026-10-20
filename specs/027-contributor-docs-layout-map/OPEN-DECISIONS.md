# Open Decisions: 027-contributor-docs-layout-map

Unsettled items. Do not treat any of the below as decided until a maintainer rules on it here.

## Settled (valgul, 2026-10-20)

- **OD-1 (FR-001..FR-004): canonical agent instructions file.** `AGENTS.md` is the real file in both repos. Main repo: the content moves from `CLAUDE.md` into `AGENTS.md` (regular file), `CLAUDE.md` becomes a symlink to `AGENTS.md`, and the unused `AGENT.md` symlink is deleted. Website already matches (`AGENTS.md` real, `CLAUDE.md` link), so it is unchanged. (Premise corrected before asking: `AGENT.md` and `AGENTS.md` were already symlinks to `CLAUDE.md`, not copies.)

## Open (asked 2026-10-20)

- **OD-2 (FR-005..FR-008): spec and design homes.** Premise corrected: the 19 live `specs.md` files are living component reference docs required by constitution Principle IV and enforced by `hack/check-specs.sh`, not a competing feature-spec home. The real stray home is `docs/superpowers/` (5 files, stale "not yet implemented" status for shipped features). Question: what happens to `docs/superpowers/`, and do the `specs.md` files keep their name? Proposed: two written roles (`specs/NNN-*` for changes, `<module>/specs.md` for how a module works today); move `docs/superpowers/` to `docs/archive/` with a "historical, superseded" banner.
- **OD-3 (FR-009, FR-011, FR-012): target layout for Go modules.** Question: which shape do long-running services take? Proposed: `cmd/main.go` (thin wiring) + `internal/<concern>/`, as `api`, `operator`, `agent` and `capture-sidecar` already do; libraries stay one flat root package; CLIs use `cmd/<binary>/`.
- **OD-4 (FR-010): target layout for `web/src`.** Question: where does a big screen's code live? Proposed: one folder per screen under `routes/<Screen>/` holding the screen, its sections and their tests; `components/` only for parts shared by two or more screens.
- **OD-5 (FR-013): CI enforcement of the layout.** Question: does a CI check enforce it, and when? Proposed: a folder-shape check written by the split spec, switched on once modules conform; this spec adds no check.
