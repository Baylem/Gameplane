# Multicluster dashboard design and acceptance

## Design decision — September 27, 2026

The user authorized implementing this UI with the existing Gameplane/HeroUI components and documenting the design here because Pencil MCP is unavailable. This explicitly approved exception leaves `design.pen` and its exports untouched; it does not claim a new Pencil design or maintainer approval.

One central login manages independent registered clusters. A cluster contains nodes and game servers; registering another cluster is distinct from adding a node to the selected cluster. The dashboard continues to summarize one selected cluster, rather than inventing fleet totals from incomplete inventory.

## Desktop and mobile screens

- **Clusters** is a discoverable authenticated navigation item and `/clusters` overview. Existing cards show each authorized registration's display name, stable ID, connection health, Kubernetes version and last check when available. Local is identified as the central cluster. A selected badge and **View servers** action make selection explicit; **View nodes** appears only when the API grants that registration's inventory capability. Namespace-only server readers can discover and select their sites without receiving node inventory access. This is a registry overview: connection health does not assert gateway readiness or game health. Registration remains an operator-managed prerequisite; there is no nonfunctional Add cluster wizard.
- **Cluster** at the existing `/cluster` URL remains node inventory. Its heading/subtitle identifies the selected cluster and node count. Loading, failed access and genuinely empty inventory are different states. Remote selection never shows central installation storage settings or enables central credential/node-join operations.
- **Dashboard** remains the selected cluster's game/inventory summary. The selected ID/name is visible and a Clusters link offers a clear way to find other sites. Errors do not become a successful empty node response.
- **Header:** the existing cluster dropdown stays visible at every viewport width, including the 375 px mobile layout. Its label truncates without removing the accessible name or hiding the switcher; the mobile page title can truncate, while navigation and the user menu remain reachable. The menu's existing misleading Add cluster action becomes **View all clusters**.

Use the existing PageHeader, Card, Button, Alert, LoadingCard, HeroUI dropdown and semantic health colors. No new visual design system, enrollment credential form, public status page or fleet-wide mutation surface is introduced.

## Context and safety behavior

Inventory and affected game queries include the selected cluster ID in their query keys and capture that ID when requests begin. Switching clusters cancels and clears outstanding cached queries before publishing the new selection. Route content remounts at the cluster boundary so a previous site's forms, confirmations and credential results cannot remain actionable. Existing streams close through their existing cluster subscription. Registration health is global, scoped by the API to authorized registrations, and never silently changes an unavailable selection to local.

Central Modules, Users/RBAC, Audit/System logs and Settings screens carry a central-management banner. Their API requests suppress automatic workload-cluster injection while preserving explicit binding-target query parameters. The central module catalog uses **Deploy locally**, which selects local before opening server creation. Direct server creation uses the selected cluster's template list and pins both the create request and any later tunnel-credential write to that same target. Authentication/profile cache remains central; it is not evidence of a selected cluster inventory grant.

The server remains the authorization authority. The registry overview does not grant cluster access. Remote node inventory uses the selected backend client; remote installation configuration and central kubeconfig/node-join buttons remain unavailable. A disconnected or removed remote produces an error and an explicit way to choose another cluster, never local fallback.

The Users edit dialog keeps existing primary-role controls and adds a cluster selector for supplemental grants. Each grant identifies cluster and namespace; remote all-namespace grants are independent of the managed local primary grant. All namespaces is available only for roles containing permitted cluster-workload capabilities such as `cluster:read`; global central-administration permissions remain restricted. Roles are created through the existing Roles tab, not automatically. This grant UI depends on the separately reviewed backend binding policy.

## Acceptance

CI component and mock-browser coverage must establish:

1. The overview lists both authorized registrations with clear cluster identity and health, distinguishing these entries from node cards; loading, error and empty registry states are explicit.
2. Selecting a registration opens its node inventory, with the selected ID threaded to API reads. Local and remote fixtures have distinct node names and same-named servers with distinct identities.
3. Switching during an outstanding local request cannot render its late response under a remote heading. Previous-site forms/results are discarded, and query caches do not share selected inventory or authorization data.
4. The selector and overview remain usable at desktop and 375 px mobile widths with keyboard-accessible controls and no horizontal page overflow.
5. Remote node-join/kubeconfig controls remain disabled before and after inventory loading or errors, and central storage configuration is absent remotely.
6. A remote inventory failure is displayed as an error rather than empty/healthy local inventory. Unauthenticated screens reveal no registry information.

Only compilation runs locally under the repository rules. Full component, lint and mock-browser suites run in GitHub Actions. A subsequent bounded disposable demo check should independently compare displayed local/remote nodes and file markers with each real cluster. This UI does not complete real-game/fault/rotation acceptance, gateway health enrollment, capture/ID-list mod routing, or UI-to-Git export.
