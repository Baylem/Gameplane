# Open decisions: Cluster-aware interactive access

These questions are unsettled design choices, not committed API or security
contracts. They apply to gateway follow-ups; cluster-aware Kubernetes Pod logs
and attach can proceed independently. Proposed starting choices are recorded
for review rather than represented as upstream agreement.

## OD-001: Gateway packaging and transport boundary

The proposed implementation uses a restricted gateway subcommand in the API
image, reusing packages without browser/admin routes or a user database. Review
that boundary before committing to it as a supported interface. Keep transport
internal until a remote implementation proves the abstraction. An outbound
tunnel and gateway-only Kubernetes access are out of initial scope.

## OD-002: Delegated authorization and stream lifetime

The proposed initial contract trusts structured operation metadata from an
explicitly enrolled central identity over dedicated mTLS. Review whether a
signed, short-lived assertion adds a necessary independently verifiable
boundary. Define actor/operation/target binding, request integrity, expiry, and
replay behavior without a second user directory. Set a maximum stream lifetime
and decide whether access revocation uses periodic renewal or bounded
reconnect. Define unknown write outcomes without assuming exactly-once delivery.

## OD-003: Target identity across the final agent connection

The central API and gateway can check GameServer UID and agent Service
ownership, but Service DNS is name-based and the agent currently does not
verify the requested GameServer UID. Choose how the connected agent proves the
same server instance or how the gateway prevents a delete/recreate race.
Include workload ownership, Service endpoint replacement, stale certificates,
and active streams. Do not claim UID/Service checks alone prevent all races.

## OD-004: Enrollment, rotation, and revocation

Use dedicated central-to-gateway trust, separate from agent trust. Resolve
certificate issuance, exact allowed peer identity, minimum key/certificate
validation, and the relationship to existing certificate hooks. Define
Secret scopes, reload triggers, rotation overlap, and removal behavior. Compare
per-request credential validation with watched caches. The current registry
watches Cluster changes, not Secret changes directly; reuse alone does not
provide prompt rotation or revocation. Active streams need an explicit policy.

## OD-005: Additive registration and capability contract

Review exact `Cluster` fields for endpoint, peer identity, labelled credential
references, and status, plus Helm values and installation steps. Specify which
component writes each health condition, freshness semantics, and whether
capability reporting is persisted or returned by the API. Define protocol
version negotiation and supported central/gateway/agent version skew. Preserve
kubeconfig-only registrations and direct-local access.

## OD-006: Initial operation coverage and verification scope

Choose the smallest gateway slice with its corresponding permissions and tests.
Inventory proxy routes, internal agent callers, mixed Kubernetes/agent actions,
and optional capture sidecar paths before advertising parity. Resolve which
cases belong in unit tests versus the existing two-cluster CI fixture. UI
capability changes follow the repository's design-first workflow.
