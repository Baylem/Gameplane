# Remote agent access

The optional agent gateway extends registered clusters with console, game-file
logs, file management, player operations, live status, and agent-based mods. The
central API still connects directly to each Kubernetes API for resource changes,
pod logs, PTY attach, and stdin module actions. The gateway does not provide a
Kubernetes tunnel, replicate storage, or run a second user database.

## Register the gateway in the central cluster

Install the updated operator, UID-aware agents and [optional gateway](gateway-install.md) in each game
cluster. The gateway must have an HTTPS certificate valid for its configured URL
and trust the central API's dedicated client certificate identity. Use separate
trust material from the local agent CA. Restrict network access to the central API
through private networking or an explicit firewall rule.

Create a Secret in the central API namespace (normally `gameplane-system`):

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: chicago-1-gateway-client
  namespace: gameplane-system
  labels:
    gameplane.local/agent-gateway-credentials: "true"
type: Opaque
stringData:
  ca.crt: <PEM CA bundle validating the gateway server>
  tls.crt: <PEM central API client certificate with the gateway's expected URI SAN>
  tls.key: <PEM central API client private key>
```

These placeholders describe the Secret keys; provide real credentials through
your normal Secret-management workflow. Do not commit private keys to Git.

Add the optional gateway reference to the existing `Cluster` resource:

```yaml
apiVersion: gameplane.local/v1alpha1
kind: Cluster
metadata:
  name: chicago-1
spec:
  kubeconfigSecret:
    name: chicago-1-kubeconfig
  agentGateway:
    url: https://chicago-1-gateway.internal:8443
    tlsSecretRef:
      name: chicago-1-gateway-client
```

The registration name must match the gateway's configured cluster ID. The URL is
an HTTPS origin with no userinfo, path, query or fragment. Credentials are read
only from the central API namespace and must carry the label above. Registration
through the existing dashboard remains unchanged; configure the optional gateway
field through Kubernetes until an explicit UI workflow is added.

## Routing and identity

Each request keeps the selected Kubernetes client and gateway transport separate
from other requests. The central API authorizes the user, reads that cluster's
GameServer UID, and sends an allowlisted operation to:

```text
/v1/clusters/{cluster}/namespaces/{namespace}/servers/{name}/uids/{uid}/{operation}
```

The gateway authenticates the central API, verifies its own cluster ID and the
live GameServer/agent Service ownership, and connects to the local agent's UID
route. The agent rejects a different UID. Older agents do not implement that
versioned route and fail closed; they must be upgraded before gateway access is
available. User cookies, bearer tokens and CSRF headers are never forwarded.

The central API is the user-authorization authority. A trusted gateway client
certificate therefore grants delegated access to the gateway's allowlisted agent
operations; it is not a user credential. The gateway does not independently
reconstruct central role bindings. Keep that client private key restricted to the
central API workload.

Internal mod-update reads use the same selected cluster for agent data and game
template lookup. RCON module actions use the gateway. Stdin module actions use the
selected Kubernetes client and verify workload ownership before attach; Kubernetes
attach does not support atomic UID preconditions, so this is a preflight check.

## Failure and rotation behavior

The central API reads the Cluster and credential Secret for every new gateway
operation. Missing, deleted, unlabeled or malformed credentials fail closed;
there is no fallback to a local namesake. Removing the optional gateway reference
removes new interactive access without removing Kubernetes management. Redirects
are not followed and writes are not automatically retried after an ambiguous
failure. Existing streams remain bounded by the gateway's session lifetime and
peer certificate expiration; reconnect performs fresh authorization and lookup.

Gateway health and Kubernetes reachability are separate. The existing Cluster
health field describes Kubernetes connectivity; it does not claim that the
gateway is healthy. The authenticated gateway capability endpoint is
`/v1/capabilities`. A gateway outage does not stop games or their local operator.

Capture-file downloads use a distinct sidecar transport and remain outside this
agent protocol. ID-list mod configuration also retains its existing local-only
REST path; the gateway supports the agent-based mods surface.
