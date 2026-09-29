# Unified dashboard and server management

## Design decision — September 27, 2026

The user requested one dashboard and server list across independent backend clusters, with optional filters and infrastructure configured by administrators. This supersedes the selected-cluster default described in [the first multicluster UI slice](multicluster-ui.md). The existing-components design exception remains in effect: use the established Gameplane/HeroUI components and document the layout here; leave `design.pen` and its exports unchanged.

One central login/API presents the resources each user can access. Clusters remain independent execution and storage boundaries. Their Kubernetes APIs, operators and authenticated agent gateways continue to route operations; combining their presentation does not introduce cross-site scheduling, shared storage, game migration or a second user database.

## User experience

- **Dashboard:** combined server and player counts across authorized sites by default. An optional location/cluster filter narrows the same data. Inventory cards combine only sites where the user has inventory permission. CPU and memory ratios use summed usage and corresponding capacity; unavailable metrics never become zero. Display which authorized scopes are unavailable or truncated alongside partial totals.
- **Servers:** one list with existing game, namespace, status and text filters plus an optional cluster filter. Same-named servers remain distinct. Site and namespace metadata disambiguate results, while routine actions act on the selected row without a global infrastructure selection.
- **Server detail:** canonical links carry the server's cluster and namespace. Console, files, players, settings, backups and related actions inherit that immutable resource target. Refreshing or opening a link in another tab does not depend on a previous localStorage selection. Legacy links without a cluster mean local; errors never select another site automatically.
- **Backups:** snapshots, schedules and restores combine authorized scopes with optional filters. Detail, restore, create, edit and delete operations retain the originating resource's cluster and namespace. A server picker identifies duplicate names by full target.
- **Search:** one authorized server search across sites. Results and navigation retain cluster, namespace and name; identical names cannot overwrite one another.
- **Infrastructure:** cluster registrations and node inventory belong to administration. A local selector can filter that administrative view. The ordinary shell no longer requires a global cluster switcher. Accounts, roles, module catalog, audit and installation settings retain their central ownership.
- **Creation:** deployment targeting is a property of a new server, captured before creation and retained for any credential follow-up and success link. The API supplies eligible cluster/namespace placements and their templates using existing target-scoped permissions. A single eligible placement is automatic; several placements expose a location choice, initially preferring an eligible local placement and otherwise the first stable eligible choice. Changing placement revalidates the selected template and form. This UI does not introduce an automatic cross-cluster scheduler or retry creation on another site.

Use existing page headers, cards, tables, menus, filters, alerts and responsive layouts. The single-site experience should remain familiar. A cluster filter changes the list being viewed, not an already-open server's identity or an outstanding operation.

## Filter layout — September 29, 2026

Servers keeps Location inside its existing right-hand Filter popover, alongside Game and Namespace. There is no separate location selector above the statistics. Backups uses the same popover shell on the right of its search row: Location, Server and the current tab's Phase choices are grouped together. Schedules offers Location and Server without a phase field. Server choices retain their full target and narrow to the draft location; switching location clears an incompatible server choice.

All popovers stage selections until Apply. Clear resets the draft, Apply commits it, and closing without applying discards changes. The trigger counts applied filters, including Location. Search stays visible, and the layout wraps at mobile widths. Route-driven location state remains the authority so sidebar navigation and Back/Forward restore the correct applied location.

The existing design exception remains in effect. Pencil MCP review used the shared Filter Popover (`FyV6E`), Servers controls (`gciHD`) and existing Backups filter row (`wKe4F`); `design.pen` and exports are unchanged.

## Data and authorization contract

Authenticated fleet reads return items with explicit `{cluster, namespace, name, uid}` targets, a resource payload, applicable target-scoped action capabilities, and separate completeness information. Namespace/cluster permissions and existing owner/collaborator rules remain the authorization authority; a global union of permission names cannot authorize a row in another cluster. Backup permissions remain independent of server ownership.

Collection reads use bounded concurrency, timeouts and response limits. They return explicit partial/error/truncation indicators and stable identity ordering. Unauthorized registrations must not leak through counts or error messages. An authoritative empty result, a denied scope and an unavailable scope are different states. No failed remote query falls back to local data.

Filters issue a narrower backend read rather than only hiding rows from a capped combined response. Scope metadata exposes explicitly permitted scopes and scopes containing returned owned/collaborating servers, allowing stable location choices without revealing unrelated registrations.

Existing per-target mutation and stream APIs remain in use. Every client operation captures its target before asynchronous work, including chained calls and raw file/socket paths. Query keys and invalidation include the resource's cluster and namespace; instance-specific data additionally includes UID. Route changes discard old forms, results and reconnect attempts. UID is also a cache/row identity; it does not claim a new universal server-side mutation precondition beyond existing endpoint and gateway safeguards.

## Acceptance

1. The default dashboard, server list, backups and search include both permitted demo sites without changing a global selector; filters narrow their results consistently.
2. Same names in different clusters and namespaces remain independently addressable. Direct links and fresh browser tabs ignore unrelated stored cluster selections.
3. Row actions, files, streams and delayed follow-up mutations reach the originating target. Switching filters or opening another server cannot retarget them.
4. Per-row actions use target-scoped capabilities. Namespace-only and owner/collaborator users see only their permitted resources; they do not gain inventory or backup permissions.
5. A disconnected site or bounded-list limit produces a visible partial-data state. Aggregates remain mathematically meaningful, with unknown metrics displayed as unknown.
6. Desktop and narrow mobile layouts retain usable filters, distinct identity labels and reachable actions without horizontal clipping.
7. The Docker Desktop preview preserves accounts and GameServer/PVC identities. Meaningful real two-site checks and CI validate the change; no Home/Chicago deployment or merge is part of this work.

Compile checks may run locally. Repository test and lint suites run in GitHub Actions. Visual references may be reviewed under the existing design exception with original-capture provenance; comparison thresholds and Pencil exports remain unchanged.
