# RFC: Cluster-aware interactive access

Status: proposal for review. The gateway described here is not implemented by
this documentation change. Cluster-aware Kubernetes Pod logs and terminal
attach are an independent first implementation; they do not provide remote
agent access. The choices below describe the proposed implementation, not
upstream agreement. Unsettled contracts remain in
[OPEN-DECISIONS.md](OPEN-DECISIONS.md).

## Problem and scope

A registered cluster already supplies a Kubernetes client to the central API.
Resource operations use that client, while interactive handlers retain a local
Kubernetes client or local agent service DNS and mTLS credentials. Their remote
guards are necessary: removing a guard without changing dispatch could let a
request authorized for a remote server reach a same-named local server.

This proposal adds interactive access without requiring shared Pod networks,
cross-cluster service DNS, or a second user database. One central dashboard/API
can run in a management cluster with no game workloads. Each target cluster
continues to run its own operator and agents, with an optional private gateway
for agent connections.

Non-goals are game migration, shared storage, automatic failover, Git export,
generic TCP forwarding, and gateway-only Kubernetes access. The central API
still needs connectivity and scoped credentials for each target Kubernetes API.

## Current implementation

Source baseline: `13a859ff7961d8d1f63198c682374d5dd325cb81`.

| Component | Existing behavior relevant to this proposal |
| --- | --- |
| [`kube.Registry`](../../../api/internal/kube/registry.go) | Concurrent pool of Kubernetes clients keyed by cluster ID; missing IDs do not resolve to the default client. |
| [`WatchClusters`](../../../api/internal/kube/watch.go) | Watches `Cluster` resources and loads labelled kubeconfig Secrets. It does not watch Secret changes directly. |
| [`ClusterSpec`](../../../operator/api/v1alpha1/cluster_types.go) | Contains a display name and kubeconfig Secret reference; no gateway configuration. |
| [`ws.Mount`](../../../api/internal/ws/dialer.go) | Builds one local agent HTTP/mTLS client and guards agent routes against remote selection. |
| [`podlogs.go`](../../../api/internal/ws/podlogs.go), [`attach.go`](../../../api/internal/ws/attach.go) | Use the local Kubernetes client behind the same guard. |
| [`AgentClient`](../../../api/internal/ws/agent_client.go) | Separate local-only JSON reader used by server-side handlers. |
| [`actions.go`](../../../api/internal/ws/actions.go) | Loads templates and sends stdin through the local Kubernetes client; RCON uses the agent proxy. |

The refactor must cover internal callers as well as browser proxy routes.
Changing only the agent dialer would leave template lookups, stdin actions, and
mod-update inspection bound to the wrong cluster.

## Proposed architecture

```text
Browser
  | same-origin HTTPS / WebSocket, existing session
Central dashboard / API + application database
  | authenticate, authorize, resolve one target, audit
  |
  +-- selected Kubernetes client --> target Kubernetes API
  |                                  |
  |                                  +--> local operator --> game workloads
  |
  +-- selected agent transport
       +-- existing direct-local connection --> local game agent
       +-- private authenticated connection --> target cluster gateway
                                                  |
                                                  +--> local game agent
```

| Operation | Execution path |
| --- | --- |
| Resource configuration and lifecycle | Central API to selected Kubernetes API; the local operator reconciles desired state. |
| Pod stdout logs and terminal attach | Central API to selected Kubernetes API. |
| RCON console, game log files, files, players, mods, live status | Central API to selected agent transport; remote transport uses the target gateway. |
| Module actions | Template lookup uses the selected Kubernetes client; stdin uses that client, RCON uses the selected agent transport. |
| Player traffic | Direct game endpoint; never the management gateway. |

Implement the gateway as a restricted subcommand in the existing API image,
reusing Kubernetes and transport packages without starting browser/admin
routes or the application database. It resolves verified local servers and
connects to their agents using cluster-local agent credentials. Its Kubernetes
permissions cover only reads needed to verify GameServers and owned agent
Services; additional permissions require an implemented use case.

Private connectivity must reach both the gateway and the Kubernetes API. An
outbound reverse tunnel or moving Kubernetes operations into the gateway would
be a separate proposal with a different credential and failure model.

## Target resolution and transport boundary

Resolve the cluster selector consistently with existing scope and RBAC logic.
Carry one immutable request target through authorization, lookup, execution,
and audit. After applicable authorization checks, bind it to the server's
namespace, name, and UID. Unknown or unavailable clusters never select the
local client. A shared handler must not mutate its client to route a request.

For Pod streams, select both the typed Kubernetes client and the `rest.Config`
from the same registry entry. Verify workload association before opening the
stream. Preserve existing browser authentication, origin checks, route formats,
message formats, cancellation, and applicable limits.

For agent operations, introduce a narrow internal transport with separate
HTTP and WebSocket paths. Its inputs are the verified target and a supported
operation, not a caller-supplied upstream URL. Keep the existing direct-local
adapter for installations without gateway configuration. A remote adapter
selects an administrator-configured endpoint and never discovers a destination
from browser headers or query parameters.

Migrate `AgentClient` consumers, action template lookup, and REST-side guards
alongside the relevant operations. Unsupported routes remain explicitly denied
until their remote implementation passes isolation tests. Optional capture
sidecar downloads need their own transport review; the port-8090 agent gateway
does not automatically support that separate sidecar. A successful console
connection does not establish feature parity.

## Trust and security requirements

The proposed initial contract trusts operation metadata on a dedicated mTLS
connection from an enrolled central API. The central API remains responsible
for sessions and user permissions. The gateway verifies a narrowly allowed
central peer identity, checks the addressed cluster against its configured
cluster, and validates the live GameServer UID and owned agent Service before
forwarding an allowlisted operation. Cluster identity cannot come solely from
an untrusted header. Browser cookies, OIDC tokens, and arbitrary identity
headers are not downstream credentials.

Use a dedicated trust root/client identity for central-to-gateway connections,
separate from the local agent CA. Reusing agent trust would accept identities
issued for a different purpose. The gateway keeps local agent credentials in
its own cluster. A compromised gateway must not expose other clusters' private
keys or the central API's fleet kubeconfigs.

The central API is a privileged trust boundary. Adding a signed delegation
token would not protect against its compromise; mTLS authenticates the sole
authority making the authorization decision. A short-lived assertion remains
an alternative if the reviewed threat model requires independently verifiable
delegation. The trust contract and exact peer identity rules require maintainer
review before being treated as a supported protocol.

The implementation must:

- Use a shared allowlist of HTTP method/agent-path pairs and verified local
  targets; forbid arbitrary upstream hosts, redirects, and unbounded bodies.
- Preserve file path checks, upload limits, console action validation,
  WebSocket limits, and backpressure at each existing trust boundary.
- Give streams a bounded lifetime and an explicit policy for permission
  removal, certificate expiry, disconnect, and cancellation.
- Bind audit records to actor, cluster, server identity, and request identity,
  without recording credentials, file contents, or sensitive console payloads.
- Avoid automatic replay of mutations after ambiguous failure. Report an
  unknown outcome when execution may have succeeded and require read-back or
  deliberate user retry; transport retries cannot promise exactly-once writes.

UID/Service ownership checks alone do not close the race between checking a
GameServer and reaching its name-based agent Service. The agent currently does
not enforce the requested GameServer UID. Resolve final-agent identity or
explicitly constrain supported replacement behavior before claiming protection
against a concurrent delete/recreate race; see OD-003.

## Configuration, compatibility, and failure behavior

Gateway registration should be additive to the existing `Cluster` resource,
with explicit endpoint/peer identity and Secret references instead of embedded
credentials. Restrict credential references to labelled Secrets in the central
namespace. Exact fields and Secret ownership remain open. Existing
kubeconfig-only registrations continue supporting their current operations;
existing local installations retain direct agent access.

Report Kubernetes connectivity separately from agent transport capability and
health. A gateway outage must not disable healthy resource operations. The
dashboard must not advertise remote file or console support merely because the
Kubernetes API is reachable. Report supported protocol versions and operations
explicitly; reject incompatible peers without silently changing transport.

Registry replacement must be atomic per request. Specify how endpoint changes,
credential rotation, cluster removal, and revoked trust affect active streams.
Do not rely on the current Cluster-only watch for prompt Secret reload. Reading
and validating referenced credentials for each new request is a possible
initial approach, but its load, failure behavior, and open-stream policy need
review alongside a watched cache.

Games and local operators continue when central management or its network link
fails, subject to their own dependencies. Management requests fail visibly;
destructive work is not queued for surprise execution after reconnection.
This proposal does not make the central database or dashboard highly available.

## Delivery sequence

1. **Cluster-aware Pod streams.** Resolve Pod logs and terminal attach through
   the existing registry per request. Keep agent routes guarded. No gateway,
   Cluster API change, or remote agent credentials are needed. Document the
   actual minimum tested `pods/log` and `pods/attach` RBAC; do not grant
   `pods/exec` or cluster-admin as a shortcut.
2. **Local agent transport refactor.** Introduce the internal HTTP/WebSocket
   boundary and migrate internal callers with local behavior unchanged. Use
   tests to establish target propagation and unsupported-remote behavior.
3. **Optional gateway.** Resolve relevant open decisions, then add remote
   transport, gateway subcommand, configuration, enrollment/rotation, capability
   reporting, and opt-in Helm deployment. Preserve operator reconciliation
   authority. Enable operations only with corresponding tests.
4. **Parity and release evidence.** Complete advertised routes, supported
   version combinations, dashboard capability behavior, and deployment docs.
   Update limitations based on demonstrated operations, not intended scope.

These are review boundaries, not a requirement for four fixed-size PRs. A
local-only refactor may be smaller than a PR, while gateway trust and feature
coverage may require several independently reviewable changes.

## Acceptance evidence

Use the existing two-cluster CI fixture and `multicluster` bucket, extending
them where necessary. The central installation must have no network route to
remote Pod/Service ranges; only its remote Kubernetes API and private gateway
are reachable. Register every new E2E test in the bucket inventory.

| Evidence | Required result |
| --- | --- |
| Same namespace/server name in two clusters, distinct outputs | Remote-only permissions can reach only the remote resource; denied, missing, malformed, and unavailable targets never reach the local namesake. |
| Concurrent local and remote streams | Client, credentials, target, and audit identity cannot cross requests. |
| Pod stream regression | Local/remote logs, terminal input, resize, missing RBAC, cancellation, and disconnect behave correctly without agent connectivity. |
| Agent route coverage | Console, files/upload, players, actions, mods, and internal callers use the selected cluster; unsupported game features remain explicit. |
| Target replacement and forged routing | Stale UID, name reuse, wrong cluster/peer, and caller-supplied destinations cannot redirect an authorized operation. |
| Credential lifecycle | Rotation, expiry, revocation, invalid registration, and permission removal obey the documented policy for new and active connections. |
| Failure and limits | Gateway partition preserves Kubernetes controls, ambiguous writes are not replayed, and uploads/streams remain bounded. |
| Upgrade compatibility | Existing single-cluster installs work unchanged; incompatible protocol versions fail clearly. |

Local compilation checks and CI verification follow
[contributing.md](../../contributing.md#submitting-a-change). This documentation
PR does not claim that gateway acceptance has passed.
