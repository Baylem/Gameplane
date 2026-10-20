# Open Decisions: 026-modular-dev-containers

Unsettled items. Do not treat any of the below as decided until valgul rules on it here.

## Open

- **OD-1 (FR-002): initial profile catalogue.** Which components get their own profile, and which parts exist?
  - Recommended: component profiles for web, api, operator, agent, website, modules, plus one "Go services and libraries" profile for the smaller Go modules; parts "dashboard + API", "control plane" (operator + api + agent + chart + local cluster + e2e) and "modules + gp-module"; plus full.
  - Alternatives: one profile per Go module (17+, very fine grained); or only parts + full, no single-component profiles.
- **OD-2 (US2, FR-005): does the "dashboard + API" part include a local Kubernetes cluster?**
  - Recommended: yes, include the in-container Docker and kind so the dashboard shows real servers (heavier, still lighter than full).
  - Alternatives: no cluster, run the API in standalone mode (spec 025; light, but cluster and server screens stay empty); or offer both as two parts.
- **OD-3 (FR-017): where do website and module profiles live?**
  - Recommended: main repo only (opened from the main clone with submodules).
  - Alternative: also add a profile inside the website and module repos so they work when cloned on their own (more files to keep in step across three repos).

## Settled

- valgul, 2026-10-20: each component lists its own required system dependencies (US6, FR-004a, FR-004b).
