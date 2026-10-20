# Open Decisions: 026-modular-dev-containers

Unsettled items. Do not treat any of the below as decided until valgul rules on it here.

## Open

No open decisions remain.

## Settled

- valgul, 2026-10-20: each component lists its own required system dependencies (US6, FR-004a, FR-004b).
- valgul, 2026-10-20, OD-1 (FR-002): main components + 3 parts. Components: web, api, operator, agent, website, modules, "Go services and libraries". Parts: dashboard + API, control plane, modules + gp-module. Plus full.
- valgul, 2026-10-20, OD-2 (US2, FR-005): the dashboard + API part includes in-container Docker and a kind cluster.
- valgul, 2026-10-20, OD-3 (FR-017): website and module profiles live in the main repo and also in the website and module repos.
