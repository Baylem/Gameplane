# Per-cluster gateway installation

Run one central Gameplane API/dashboard and an operator plus optional gateway in
each remote cluster. The gateway proxies approved agent operations over mTLS. The
central API still connects directly to each registered Kubernetes API for CRD
management, Pod logs and PTY attach; a gateway does not replace those credentials.

## Prepare the remote cluster

Use matching API, operator and agent versions. The gateway uses the API image's
`gateway` subcommand, a dedicated ServiceAccount, and read-only `get` access to
GameServers and Services in its configured namespaces. It has no database and no
user login. Users authenticate to the central API, which retains authorization.

Create these Secrets in the chart release namespace before enabling the gateway:

| Secret value | Required keys | Purpose |
|---|---|---|
| `gateway.serverTLSSecret` | `tls.crt`, `tls.key` | Gateway server identity, with DNS SAN matching the configured endpoint |
| `gateway.centralClientCASecret` | `ca.crt` | CA trusted to issue central API client certificates |
| `gateway.agentClientSecret` | `tls.crt`, `tls.key` | Existing local agent client credentials; defaults to `gameplane-agent-client` |
| `gateway.agentCASecret` | `ca.crt` | Existing local agent trust; defaults to `gameplane-agent-ca` |

The gateway additionally requires an exact `gateway.peerURI` URI SAN in the central
client certificate. Use a separate management trust root from the local agent CA.
The chart mounts only `ca.crt` from the agent CA Secret, never its signing key.
The existing chart continues provisioning the local agent CA/client Secrets when
the central API is disabled. Credential rotation requires a rolling restart of
the gateway so new material is loaded; coordinate client/server trust overlap
before retiring old certificates.

## Helm configuration for a fresh remote installation

Adapt the [example values](../charts/gameplane/examples/gateway-values.yaml):

```yaml
api:
  enabled: false
gateway:
  enabled: true
  clusterID: remote-1
  peerURI: spiffe://gameplane.example/central-api
  namespaces: [gameplane-games]
  serverTLSSecret: gameplane-gateway-server
  centralClientCASecret: gameplane-central-client-ca
  networkPolicy:
    peerCIDRs: [10.20.30.40/32]
    apiServerCIDRs: [10.96.0.1/32, 10.0.0.10/32]
```

`clusterID` must equal the `Cluster` resource name in the central installation.
An empty namespace list selects only `gamesNamespace`; additional namespaces must
already exist. The chart creates a Role/RoleBinding in each configured namespace. When
`networkPolicies.enabled` is true, it also creates matching agent ingress rules.
When game network policies are disabled, no new game Pod ingress isolation is
introduced; any externally managed default-deny policies must allow gateway
traffic on TCP 8090. It does not grant cluster-wide gateway permissions.

```sh
helm upgrade --install gameplane ./charts/gameplane \
  --namespace gameplane-system --create-namespace \
  --values gateway-values.yaml
```

`api.enabled: false` omits the API Deployment, Service, ServiceAccount/RBAC and
SQLite PVC, plus the dashboard, dashboard ingress, cluster-operations grants,
API ServiceMonitor and API-only audit/telemetry receivers. The operator, CRDs,
agent mTLS and operator/agent monitoring remain available. Default installations
retain `api.enabled: true` and `gateway.enabled: false`.

Use this profile for a fresh remote installation. For an existing API/database,
preserve its PVC and data before disabling the API. Current chart-created SQLite
PVCs carry `helm.sh/resource-policy: keep`; Helm retains those during removal.
A live Helm render also refuses to disable an older unannotated PVC. That lookup
cannot inspect the cluster during offline `helm template`/GitOps rendering, and
non-Helm pruning controllers may not honor Helm's retention annotation. Configure
the actual deployment controller's retention mechanism before changing an
existing installation. The profile does not migrate the database or its users.

## Private networking

The gateway Service is **ClusterIP on TCP 8443**. Provide private routing or a
private TLS-passthrough relay to reach it; this chart creates no public ingress,
NodePort or LoadBalancer. TLS must reach the gateway unchanged, because the
gateway verifies the central client's certificate itself.

A dedicated NetworkPolicy is installed whenever the gateway is enabled, including
when the chart's game network policies are disabled. It requires explicit peer
and API endpoint allowlists. It permits only:

- Incoming TCP 8443 from configured `peerCIDRs` or `peerSelectors`.
- DNS on TCP/UDP 53 to configured DNS pods (or exact `dnsCIDRs` for NodeLocal DNS).
- TCP 443/6443 to `apiServerCIDRs` in the local cluster.
- TCP 8090 to game agents in the explicitly allowed namespaces.

Use the source IPs the target cluster actually observes, accounting for private
relay/SNAT behavior. `peerSelectors` identify pods **inside this cluster**, such
as a relay; they cannot select pods by labels across clusters. Supply both a
namespace selector and pod selector to narrow a relay allowance. Include the
Kubernetes Service and post-DNAT API endpoint addresses as required by the CNI.
NetworkPolicy enforcement requires a supporting CNI; other overlapping policies
can broaden access because Kubernetes policies are additive.

The chart uses TCP liveness/readiness probes so no unauthenticated HTTP endpoint
is exposed. These indicate an open listener, not an end-to-end authenticated
agent health check. Confirm the central API can authenticate and reach a test
server before relying on interactive management.

## Upgrade behavior

The operator now injects `GAMEPLANE_SERVER_UID` into agents. Updating the operator
can change existing StatefulSet templates and roll game Pods during reconciliation;
schedule this rollout for a suitable maintenance window. Remote requests use only
`/v1/targets/{uid}/...`. Old agents or an operator that has not populated this UID
fail closed with 404; there is no fallback to legacy agent paths.
