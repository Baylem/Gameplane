# Open Decisions: 028-split-big-packages

Unsettled items. Do not treat any of the below as decided until a maintainer rules on it here.

## Settled elsewhere (binding, not reopened here)

- Spec 027 OD-3 (Go layout), OD-4 (web `pages/` layout, 500-line screens), OD-5 (this spec adds the folder-shape check to `make lint`), OD-2 (`specs.md` stays per module). valgul, 2026-10-20.

## Settled (valgul, 2026-10-20)

- **OD-1 (FR-013): split order.** Web first, then api, then operator; then the small services; tunnel and sentinel wait for their audit fixes (#721–#723, #729) to merge. Chosen on a card.
- **OD-2 (US2, US3, FR-002): sub-package boundaries.** By resource: `api/internal/handlers/` and `operator/internal/controller/` each become sub-packages per resource group, with helpers shared by several groups in one internal helper package per module, one group per PR. (Premise corrected before asking: a by-layer split does not exist in today's code and would need logic rewrites, which this spec forbids.) Chosen on a card.
- **OD-3 (FR-018, SC-001): line ceiling.** 500 lines for every production source file (Go and web), the same as web screens (027 OD-4). Generated files and CRD `*_types.go` are exempt. Today 34 Go and 17 web production files are over 500. Chosen on a card.
- **OD-4 (FR-007, FR-019): enforcement.** `make lint` enforces the 500-line ceiling for production files only. Test files are exempt and are not split. Chosen on a card.
- **OD-5 (FR-020): web types.** No merge; keep both files. The generated `web/src/api/types.ts` stays untouched. The hand-written `web/src/types.ts` (1,132 lines) is over the 500-line ceiling (OD-3), so it is split by domain into files that re-export through `web/src/types.ts`, keeping every `@/types` import working unchanged. Chosen on a card.
- **OD-6 (FR-005): mechanical test edits.** valgul signs off, for the whole spec, on exactly these edits inside moved test files: the `package` clause, import lines, qualified names of identifiers that moved package (`foo` → `pkg.Foo`), relative import and `vi.mock` path strings, and one new `TestMain`/suite file per new sub-package that copies today's envtest setup unchanged. Each PR lists every touched test file, and a script proves no other kind of line changed. Nothing else in a test may change (no assertion, test name, table entry, timeout, skip or fixture). Chosen on a card.

## Open (asked as cards in the "Spec 028 split big packages" thread, 2026-10-20)

Raised while planning: spec 027 OD-4 does not say these two details, and its thread is closed.

- **OD-7 (US4): what stays in `web/src/routes/`.** 027 says `routes/` "only exposes pages (route definitions, no screen code)" and `router/` holds the route tree. Today all route definitions are in `router/tree.tsx` and `routes/` holds screen code.
  Options: A one `routes/<Screen>.ts` per page holding its route definition moved out of `router/tree.tsx`, which only assembles the tree; B delete `routes/`, `router/tree.tsx` imports from `pages/`; C thin re-export files.
  Recommendation: A.
- **OD-8 (US4): where the server-detail tabs go.** `routes/tabs/` (11 tabs) and `routes/tabs/settings/` (13 sub-tabs) belong to the ServerDetail screen.
  Options: A sections of ServerDetail: `pages/ServerDetail/<Tab>.tsx`, `pages/ServerDetail/Settings/<Section>.tsx`, split again over 500 lines; B keep a `tabs/` level: `pages/ServerDetail/tabs/...`; C each tab its own top-level page.
  Recommendation: A.
- **OD-9 (FR-005, OD-6 follow-up): relative CRD paths in a copied envtest setup.** `operator/internal/controller/suite_envtest_test.go:45,48` and `api/internal/handlers/suite_envtest_test.go:55` find CRDs with `filepath.Join("..", …)`. A copy one folder deeper needs one more `".."`, a path-string edit OD-6 did not name.
  Options: A allow exactly one extra `".."` per extra folder level in these paths, checked by the edit-check script; B ask per PR.
  Recommendation: A.
- **OD-10 (constitution I): new e2e test.** Nothing user-facing changes; every move PR runs the full existing e2e suite with `ci: full-run` plus the test-name parity check. 023 OD-4 required an e2e test for a CI-only feature.
  Options: A no new e2e test (recorded exemption for a behaviour-preserving refactor); B add one (e.g. layout check run from `test/e2e/`).
  Recommendation: A.
