# Standalone panel

Run the dashboard and API on a host that has no Kubernetes installation. Remote
game clusters run their own operator and game server agents. The central host is
never registered as a `local` cluster and cannot host games itself.

The default Helm install remains a combined panel and operator installation.
Standalone mode is an explicit choice for a **fresh** central panel; this guide
does not migrate an existing installation's Kubernetes Secrets or Cluster CRs
into SQL storage.

## Docker Compose

Install Docker with the Compose plugin, then clone this repository and check out
the version containing standalone support. The Compose file builds the API and
web images from that checkout so both components use the same code:

```sh
docker compose -f deploy/standalone/compose.yaml up -d --build
docker compose -f deploy/standalone/compose.yaml logs gameplane-api
```

Only two services run: `gameplane-api` and `web`. Neither mounts a kubeconfig,
host Docker socket, or Kubernetes credentials. The API is reachable only through
the Compose network; nginx forwards requests to `http://gameplane-api:8000`.
The dashboard is bound to `127.0.0.1:8080` on the host. Set `GAMEPLANE_PORT` to
change that port. Use `http://localhost:8080` locally; for remote access, put an
HTTPS reverse proxy in front of the loopback port and forward WebSocket upgrades
and streaming responses. Login cookies require a secure browser context; serving
the dashboard on an ordinary remote HTTP origin will not work.

Create the first administrator using the same database volume. This Bash example
reads the password without including it in shell history or process arguments:

```sh
read -r -s -p 'Admin password: ' PANEL_PASSWORD
printf '\n'
printf '%s\n' "$PANEL_PASSWORD" | docker compose -f deploy/standalone/compose.yaml \
  exec -T gameplane-api /api bootstrap-admin --username admin --password-stdin
unset PANEL_PASSWORD
```

Use a password of at least 12 characters. Sign in at the dashboard. A new panel
has no clusters; register a remote cluster before using game-server operations.
Authentication, users, roles, settings, and registration remain available before
any remote cluster exists. Administrators can use **Register cluster** on the
Clusters page or the registration API below. The cluster picker selects already
registered clusters. Gateway credentials are configured through the API.

## Storage and recovery

The named `panel-data` volume contains `/data/gameplane.db` and
`/data/panel.key`. Standalone settings and remote credentials are stored in SQL;
credential values are encrypted using the persistent key. Both files are needed
for recovery. Stop the services before copying the entire volume so the SQLite
database and its WAL files are consistent. Restore the volume with ownership
`65532:65532`, then start the services. Normal `docker compose down` retains the
volume; `down --volumes` deletes the database and key.

The API image seeds a new named volume with a directory writable by its non-root
runtime user. If you replace it with a host bind mount, create that directory with
ownership `65532:65532` yourself. Keep the key private and back it up with the
database; creating a replacement key cannot decrypt existing credentials.

For custom deployments, pass `--standalone` or `GAMEPLANE_STANDALONE=true` to the
API. `--panel-key-file` / `GAMEPLANE_PANEL_KEY_FILE` selects the key path (default
`/data/panel.key`). A persistent key file is required even when using the
experimental PostgreSQL build. Run one API replica.

## Central panel on Kubernetes

The panel can also run in a Kubernetes cluster without using that cluster for
games or granting the API Kubernetes access. From this checkout:

```sh
helm upgrade --install gameplane charts/gameplane \
  --namespace gameplane-system --create-namespace --skip-crds \
  --values charts/gameplane/examples/standalone-panel-values.yaml
```

Use images built from the matching version (set `image.registry` and `image.tag`
for your build). The example enables `api.standalone`, disables `operator.enabled`
and the CRD upgrade hook, and disables ingress until you configure your hostname
and TLS. Expose the web Service through your HTTPS ingress or use a local port
forward. Bootstrap the admin with `/api bootstrap-admin` in the API pod as in the
[combined installation guide](install.md#first-time-setup).

**Keep `--skip-crds` on installation commands.** Helm installs files in `crds/`
before evaluating templates; setting values alone cannot suppress those CRDs.
Standalone mode skips the CRD apply hook automatically. It creates no operator,
agent certificates, game namespace, module source, game network policies,
operator monitors, or API Kubernetes RBAC. Its API pod disables service account
token mounting. Its PVC holds both SQLite and the panel key. Do not switch an
existing combined release to these values as a migration procedure.

## Install and register remote game clusters

Each target still needs Kubernetes, Gameplane CRDs, an operator, and storage for
games. Install the matching chart there with `api.enabled=false`; leave
`operator.enabled=true` and `api.standalone=false`. Agents are created alongside
game servers by that operator. Enable the optional
[private gateway](gateway-install.md) for agent operations such as console,
files, and player management. Without a gateway, Kubernetes resource operations
remain available, but agent-backed operations are unavailable.

The central API must reach each target's Kubernetes API directly. The gateway
does not tunnel Kubernetes traffic. Use a kubeconfig containing credentials and
CA data that are usable from the central host, with the
[documented remote permissions](install.md#prerequisites-1). Do not use kubeconfigs
whose credentials require local files or interactive credential plugins. Use
the same operator namespace on the targets as the API's `--namespace` setting
(default `gameplane-system`).

Register the target with an authenticated administrator session. In the examples
below, `cookies.txt` is a private cookie jar for that session and `CSRF_TOKEN`
is its `gameplane_csrf` cookie value. `PANEL_URL` is the HTTPS dashboard origin.
The kubeconfig is a raw YAML string in JSON, not base64:

```sh
jq -n --rawfile kubeconfig remote-kubeconfig.yaml \
  '{name:"remote-1",displayName:"Remote games",kubeconfig:$kubeconfig}' |
  curl --fail-with-body --cookie cookies.txt \
    --header "X-Gameplane-CSRF: $CSRF_TOKEN" \
    --header 'Content-Type: application/json' \
    --data-binary @- "$PANEL_URL/clusters"
```

The standalone panel saves the registration and credentials in its database;
there is no central `Cluster` CR or Kubernetes Secret to create. It monitors
remote Kubernetes connectivity itself. Selecting that registration routes game
operations to the target, and never falls back to the central host.

Registration does not grant workload access, including to the bootstrap admin.
Its initial role administers the central panel. In **Users**, use the **Roles**
tab to create a custom role containing the target permissions you need:

- `cluster:read` for node inventory.
- `modules:read` and `modules:manage` for the catalog, sources, and module installation.
- `templates:read` and `templates:write` for game templates.
- The server, backup, and other namespace permissions needed for your workloads.

Open your user in the Users tab, then under **Cluster &
namespace grants**, select the remote cluster and that role. Choose a game
namespace for a namespace grant, or **All namespaces** for an allowed role that
should apply across that remote cluster, and click **Add**. Inventory, module,
and template permissions need an **All namespaces** grant on that target; a grant
limited to one game namespace does not grant those cluster-wide operations.
Adding a grant ends
that user's sessions; sign in again to use it. Grant other users access the same
way. Do not remove the bootstrap user's primary role to add remote access.

Remote grants cannot include central administration permissions or the built-in
admin wildcard. Use a custom workload role rather than assigning the built-in
admin role to all namespaces. Wait for the registration to connect before adding
a target grant; registration metadata can appear before its client is loaded.

For a configured remote gateway, store the central client's dedicated mTLS
credentials through the standalone registration API:

```sh
jq -n --rawfile caCert gateway-ca.crt \
  --rawfile clientCert central-client.crt --rawfile clientKey central-client.key \
  '{url:"https://remote-1-gateway.internal:8443",caCert:$caCert,clientCert:$clientCert,clientKey:$clientKey}' |
  curl --fail-with-body --request PUT --cookie cookies.txt \
    --header "X-Gameplane-CSRF: $CSRF_TOKEN" \
    --header 'Content-Type: application/json' \
    --data-binary @- "$PANEL_URL/clusters/remote-1/gateway"
```

The gateway's `clusterID` must be `remote-1`, and its `peerURI` must match the
client certificate URI SAN. Its server certificate must match the endpoint
hostname. Use a separate trust root from the target's local agent CA. Repeat PUT
to rotate credentials; DELETE on the same endpoint removes gateway access while
retaining the cluster registration. Credential responses do not return private
keys. See [gateway routing and trust](multicluster-agent-gateway.md) for the
network and certificate requirements; that document's central Kubernetes Secret
instructions apply to combined mode, while standalone uses the API above.
