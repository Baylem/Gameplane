# Remote server feature parity

## Design

The same server pages and permissions apply to local and registered remote game
servers. Each operation stays bound to its cluster, namespace and resource
identity; an unavailable remote never falls back to a local namesake.

Registry browsing resolves the selected GameServer and GameTemplate before using
the central provider configuration. Modpack and mod-ID configuration updates go
to that selected cluster, with Kubernetes resource-version and UID protection.
Provider credentials remain centrally managed. The operator still applies the
desired game configuration.
Mod-ID and modpack writes retry bounded Kubernetes version conflicts only when
the selected server's UID, original spec and ownership grants are unchanged.
Concurrent configuration or ownership edits remain conflicts, and transport
failures are not retried.

Capture data uses a separate, explicit gateway operation rather than the agent
file protocol. The request identifies both the GameServer UID and NetworkCapture
UID. The gateway verifies live ownership, state and expiry, and derives the local
capture endpoint. The sidecar verifies a persisted identity binding before
serving or removing bytes. No client-supplied host, port or filesystem path is
accepted. Existing captures without the identity binding cannot use the remote
route. Downloads preserve Range handling, streaming and synchronous audit.
Remote deletion requires confirmation of bound file cleanup before deleting the
CR with a UID precondition. A failed cleanup retains the CR for retry. A retained
identity tombstone makes a repeated same-UID cleanup safe after a lost response;
it cannot authorize deletion of a replacement file. Local cleanup keeps its
existing behavior.

Capture availability comes from the selected site's configuration. Authenticated
gateway capabilities advertise protocol, capture support, retention limits and
start defaults; the central
installation's capture switch does not determine a remote site's availability.
The UI keeps the existing capture screen, download button and error components.
Loading, unavailable and older-gateway states disable unsupported operations with
an explanation; a capability response never grants authorization.

The September 30 Pencil reconciliation represents the selected site's capture
capabilities in the access/capability state board (`w1QYmf`), disabled capture
screen (`Bbnga`), site-limit example in the Start Capture modal (`O08uaD`), and
mobile Capture screen (`SUtGZ`). Loading, unavailable and older-gateway states
explain disabled operations; retained-download availability remains distinct
from permission to start new captures. Existing components and theme tokens
are retained. The Capture list (`m5kOm4`) inherits shared header changes but was
not directly updated or re-exported in this pass. The [design manifest](../design-export/MANIFEST.md)
records the exact updated nodes and export scope. Earlier browser-reference
provenance and upstream approval requirements remain unchanged.

This change does not move central users, provider credentials, module catalog,
installation administration or node enrollment into server-scoped operations.
It introduces no game migration, shared storage or cross-cluster scheduler.

## Acceptance

- Same-named local and remote servers and captures remain independent across all
  registry, modpack, mod-ID, capture download and cleanup operations.
- Wrong cluster, namespace, server/capture UID, ownership, permissions, peer
  identity or certificate cannot read or mutate another target.
- Capture files remain bound to identity after sidecar restart and in-memory
  history eviction; older unbound files fail closed on remote routes.
- Full and ranged downloads stream correctly; cleanup removes only the intended
  remote file; unavailable sites and ambiguous writes are not retried locally.
- Existing local operations and narrow custom-role permissions continue to work.
- CI deploys an actual gateway between two independent Kubernetes clusters and
  exercises real agent/game operations, capability compatibility, certificate
  rotation/revocation, endpoint changes and connection failures.
- The final tested revision is deployed to the disposable Docker Desktop demo
  with existing accounts and GameServer/PVC identities preserved, and reviewed
  through the app before the upstream PR head is advanced.

Repository test and lint suites run in GitHub Actions. Compilation and bounded
manual checks of the disposable preview may run locally. Passing evidence is
recorded for the exact submitted revision; this design is not a claim that the
acceptance checks have already passed.
