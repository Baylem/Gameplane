# Open Decisions: 028-split-big-packages

Unsettled items. Do not treat any of the below as decided until a maintainer rules on it here.

## Settled elsewhere (binding, not reopened here)

- Spec 027 OD-3 (Go layout), OD-4 (web `pages/` layout, 500-line screens), OD-5 (this spec adds the folder-shape check to `make lint`), OD-2 (`specs.md` stays per module). valgul, 2026-10-20.

## Open (asked as cards in the "Spec 028 split big packages" thread, 2026-10-20)

- **OD-1 (FR-013): split order.** Premise: the audit fixes (#712–#731) mostly touch the operator (backups, GameServer, templates), tunnel (#721–#723), sentinel (#729) and api (#726, #727); web is barely touched. Moves conflict with open fix PRs on the same files.
  Options: A web first, then api, then operator, small services after their audit fixes (fewest conflicts); B small services first as a pilot, then api, operator, web; C operator first (biggest package).
  Recommendation: A.
- **OD-2 (US2, US3, FR-002): sub-package boundaries for handlers and controllers.** Premise: both packages are one Go package with every test in the same package and one envtest `TestMain`. "By layer" (HTTP vs logic vs Kubernetes access) does not exist in the code today; handlers mix the three, so a layer split means rewriting logic, which FR-001/FR-002 forbid. Real options:
  A by resource (api: capture, users/roles/auth, clusters, modules, servers/lifecycle, config/notifications, …; operator: gameserver, backup/restore, networkcapture, module/template, cluster), shared helpers in one internal helper package;
  B no sub-packages: keep one package each and only split big files under the ceiling (pure moves, no test edits, but the 146/150-file packages stay);
  C by layer (needs logic rewrite; out of this spec's scope).
  Recommendation: A, done one resource group per PR, smallest leaf groups first.
- **OD-3 (FR-018, SC-001): line ceiling for production files** (web screens are already 500 by 027 OD-4). Today, production files over 500 / 800 / 1,000 lines: Go 34 / 14 / 9, web 17 / 10 / 6.
  Options: A 500 for everything (matches web screens, most work); B 800; C 1,000 (just the worst files).
  Recommendation: B 800, generated files and CRD `*_types.go` exempt.
- **OD-4 (FR-007, FR-019): does `make lint` enforce the ceiling, and do test files count?** Test files over 500 / 1,000 lines: 111 / 26.
  Options: A lint enforces it for production files only, tests exempt; B lint enforces it for tests too, and big test files are split by moving whole test functions verbatim; C no line check, review only (folder-shape check still added per 027 OD-5).
  Recommendation: A.
- **OD-5 (FR-020): `web/src/types.ts` vs the generated `web/src/api/types.ts`.** Premise: the generated file is imported by no file and nothing regenerates it (no script, `openapi-typescript` is not a dependency). `types.ts` mirrors CRD objects; the generated file mirrors REST payloads, so they don't map one to one. Merging changes types, so it is not a move.
  Options: A keep both, split `types.ts` by domain only if it is over the ceiling; B also delete the unused generated file; C merge `types.ts` into generated types (type changes, needs its own spec).
  Recommendation: A.
- **OD-6 (FR-005): sign-off for mechanical test edits forced by moves.** Moving Go code into sub-packages forces edits in moved test files: the `package` line, imports, qualified names (`foo` → `pkg.Foo`), and a new `TestMain` per sub-package that copies today's envtest setup. Web moves may need relative import / `vi.mock` path strings changed (most web imports use `@/`, so few).
  Options: A approve exactly these edit kinds for the whole spec; each PR lists every touched test file and a script proves no other line changed; B sign-off per PR; C no test edits (forces OD-2 B).
  Recommendation: A.
