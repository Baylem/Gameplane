# Open Decisions: 027-contributor-docs-layout-map

Unsettled items. Do not treat any of the below as decided until a maintainer rules on it here.

## Settled (valgul, 2026-10-20)

- **OD-1 (FR-001..FR-004): canonical agent instructions file.** `AGENTS.md` is the real file in both repos. Main repo: the content moves from `CLAUDE.md` into `AGENTS.md` (regular file), `CLAUDE.md` becomes a symlink to `AGENTS.md`, and the unused `AGENT.md` symlink is deleted. Website already matches (`AGENTS.md` real, `CLAUDE.md` link), so it is unchanged. (Premise corrected before asking: `AGENT.md` and `AGENTS.md` were already symlinks to `CLAUDE.md`, not copies.)

- **OD-2 (FR-005..FR-008): spec and design homes.** Two written roles: `specs/NNN-<slug>/` for changes (speckit), `<module>/specs.md` for how a module works today (names unchanged, constitution IV unchanged). `docs/superpowers/` moves with `git mv` to `docs/archive/superpowers/`, each file gets a "historical, superseded" banner. FR-008 does not apply. (Premise corrected before asking: the 19 live `specs.md` files are constitution IV component docs, not a competing spec home.)
- **OD-3 (FR-009, FR-011, FR-012): Go module layout.** Long-running services: thin `cmd/main.go` + `internal/<concern>/` (as `api`, `operator`, `agent`, `capture-sidecar` already do). Libraries stay one flat root package; CLIs use `cmd/<binary>/` (stated on the card as common to every option).

## Open (asked 2026-10-20)

- **OD-4 (FR-010): target layout for `web/src`.** Question: where does a big screen's code live? Proposed: one folder per screen under `routes/<Screen>/` holding the screen, its sections and their tests; `components/` only for parts shared by two or more screens.
- **OD-5 (FR-013): CI enforcement of the layout.** Question: does a CI check enforce it, and when? Proposed: a folder-shape check written by the split spec, switched on once modules conform; this spec adds no check.
