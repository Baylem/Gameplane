# Open Decisions: 026-modular-dev-containers

Unsettled items. Do not treat any of the below as decided until valgul rules on it here.

## Open

- **OD-4 (Principle I, plan Constitution Check): E2E coverage for dev containers.** The real path is the `devcontainers.yaml` CI job that builds each profile and runs its compile checks.
  - Recommended: also add a Go e2e test `test/e2e/devcontainers_e2e_test.go` that runs the generator on fixture trees (union rule, unaccounted tool, missing list, drift, `need.sh` message), registered in the `operator` bucket. This follows the spec 023 OD-4 precedent.
  - Alternative: treat the CI build job as this feature's E2E path and record a Principle I exemption here; no `test/e2e/` test.
- **OD-5 (FR-003, research R10): fix today's version drift in the full profile?**
  - Recommended: align to the Makefile/CI values (kubectl 1.36 to match envtest/kind K8s 1.36.2; setup-envtest `release-0.24`; pin helm instead of `latest`), and amend FR-003 to allow exactly these three changes.
  - Alternative: keep today's values exactly (kubectl 1.31, `setup-envtest@latest`, `helm: latest`), and drop the generator rule that the Makefile pins must match `versions.toml`.

## Settled

- valgul, 2026-10-20: each component lists its own required system dependencies (US6, FR-004a, FR-004b).
- valgul, 2026-10-20, OD-1 (FR-002): main components + 3 parts. Components: web, api, operator, agent, website, modules, "Go services and libraries". Parts: dashboard + API, control plane, modules + gp-module. Plus full.
- valgul, 2026-10-20, OD-2 (US2, FR-005): the dashboard + API part includes in-container Docker and a kind cluster.
- valgul, 2026-10-20, OD-3 (FR-017): website and module profiles live in the main repo and also in the website and module repos.
