# Feature Specification: v0.3 Release Readiness Audit

**Feature Branch**: `018-v0-3-release-readiness`

**Created**: 2026-09-23

**Status**: Draft

**Input**: User description: "Do a full code review + live test on the kubelab cluster to move toward a v0.3 release (moving away from beta status to first 'release' but not yet v1) but during other run some bug where found and other may still exist. so every feature need a full live in cluster test. all part of the code will be tested and checked"

## Clarifications

### Session 2026-09-23

- Q: How should the release-candidate build be deployed onto kubelab for the live audit? → A: Publish candidate images under a public pre-release tag, then upgrade kubelab from there.
- Q: Which kinds of defects should block the v0.3 release until fixed? → A: Everything found; no deferrals of any severity.
- Q: What should "no longer beta" mean for the v0.3 release? → A: The version becomes v0.3.0 without the beta suffix, project status wording changes from Beta to a pre-v1 release, the upgrade from the last beta is tested live, and remaining pre-v1 caveats stay listed in the roadmap.
- Q: Where should the audit's findings, component coverage table and final report be kept? → A: As files inside this spec folder, versioned with the code.
- Q: If a feature cannot be live-tested on kubelab for lack of a prerequisite, can the release still go ahead? → A: Yes; the feature is listed as not live-verified in the report and release notes.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Every shipped feature is proven on a real cluster (Priority: P1)

A maintainer preparing the first non-beta release needs evidence that each user-facing capability of Gameplane actually works on a real multi-node cluster, not only in CI kind clusters or unit tests. They work through a feature inventory, exercise each item end to end on the kubelab cluster (as an operator/admin would through the dashboard and API), and record a pass/fail result with evidence for each.

**Why this priority**: Earlier live runs already surfaced bugs that automated suites missed. Without a complete live pass, the release claim "no longer beta" is unsupported. This is the core purpose of the release gate.

**Independent Test**: Can be fully tested by taking the feature inventory, executing each item's live procedure on kubelab, and checking that every inventory row has a recorded outcome and evidence. Delivers a trustworthy statement of what works today.

**Acceptance Scenarios**:

1. **Given** the feature inventory covering every dashboard screen, API capability, operator-managed resource kind, in-pod agent capability, and auxiliary component, **When** the live pass finishes, **Then** every inventory row has an outcome (pass, fail, blocked, or not-applicable with a stated reason) and linked evidence.
2. **Given** a feature that passes in CI, **When** it is exercised on kubelab through its normal user path, **Then** the result is recorded independently of the CI result, and any disagreement between the two is itself logged as a finding.
3. **Given** a feature that depends on a game module, **When** it is tested live, **Then** at least one real game server of a representative module is created, started, used, and removed as part of the test.

---

### User Story 2 - Every part of the code base receives a documented review (Priority: P1)

A maintainer needs assurance that all components (operator, API, agent, web dashboard, shared libraries, auxiliary services, Helm chart, module tooling, documentation) have been reviewed for correctness, security, and consistency with their specifications, so that known and latent defects are found before the release rather than after.

**Why this priority**: The request states that all parts of the code are to be tested and checked. Review coverage is the second half of the gate and finds defects a live run cannot reach (unreachable branches, security boundaries, dead code, spec drift).

**Independent Test**: Can be fully tested by checking a coverage table that lists every component and shows a completed review with findings recorded; a component with no review entry fails the check.

**Acceptance Scenarios**:

1. **Given** the list of all components in the repository, **When** the review is complete, **Then** each component has a review record listing what was examined, the findings, and each finding's severity.
2. **Given** a finding, **When** it is recorded, **Then** it states the affected component, how to reproduce or observe it, its severity, and its disposition (fixed, deferred with justification, or rejected as not a defect).
3. **Given** the security-relevant boundaries (login privacy, role-based access, network-guard rules, console-input guarding, audit integrity, secret handling), **When** the review runs, **Then** each boundary is checked both by reading the code and by an attempted live violation on the cluster.

---

### User Story 3 - Discovered bugs are fixed and re-verified live (Priority: P1)

A maintainer needs every defect found during the audit, including those already known from earlier runs, to be fixed and then shown to be fixed by repeating the failing live procedure on kubelab.

**Why this priority**: A findings list alone does not make a release. The gate is only passable if blockers are closed with proof.

**Independent Test**: Can be tested by taking any finding and confirming a linked fix plus a passing repeat of its original live reproduction.

**Acceptance Scenarios**:

1. **Given** a finding, **When** a fix is delivered, **Then** the original live reproduction is repeated on kubelab and passes, and the result is attached to the finding.
2. **Given** a previously known bug from earlier live runs, **When** the audit starts, **Then** it is entered into the findings list with its reproduction, so it is not lost or silently assumed fixed.
3. **Given** any finding of any severity, **When** the release decision is made, **Then** it is either fixed and re-verified, or closed as not a defect, or (only if it concerns a capability the roadmap explicitly places after v0.3) recorded as out of scope with a citation; no other deferral is allowed.

---

### User Story 4 - Upgrade and lifecycle safety on a long-lived cluster (Priority: P2)

An existing user running a beta release upgrades to the release candidate. A maintainer needs to confirm that upgrade, restart, node loss, and rollback do not destroy running servers, stored data, accounts, or configuration.

**Why this priority**: The first non-beta release implies that users can upgrade without data loss. It builds on P1 coverage but is a distinct risk area (the previous release's upgrade path is exercised in CI on kind only).

**Independent Test**: Can be tested by seeding kubelab with a running server, persistent data, and an admin account, performing the upgrade and a component restart, and verifying nothing was lost.

**Acceptance Scenarios**:

1. **Given** a running game server with saved world data and an existing admin account, **When** the release candidate is installed over the current release, **Then** the server keeps running or recovers automatically, its data is intact, and the admin can still sign in.
2. **Given** a worker node is drained or becomes unavailable, **When** a game server was scheduled on it, **Then** the outcome (rescheduling, waiting, or a clear error state) is observed and matches documented behavior.
3. **Given** a failed upgrade, **When** rollback is performed as documented, **Then** the previous release is functional again.

---

### User Story 5 - A release decision backed by a clear report (Priority: P2)

The project owner needs a single report that states, per component and per feature, what was reviewed and tested, what was found, what was fixed, what remains, and whether the release criteria are met, so they can decide to cut the first non-beta release and know what the documentation must say about status.

**Why this priority**: The audit is only actionable if it culminates in a decision-ready summary and status changes to project documentation.

**Independent Test**: Can be tested by reading the report cold and determining, without other context, whether the release gate is met and why.

**Acceptance Scenarios**:

1. **Given** the audit is complete, **When** the report is produced, **Then** it lists totals by outcome and severity, all open findings, and an explicit go/no-go statement against the release criteria.
2. **Given** a go decision, **When** release preparation starts, **Then** the places that currently describe the project as beta (version markers, roadmap, README, docs) are identified so status wording is updated consistently.

---

### Edge Cases

- A feature cannot be exercised on kubelab because the cluster lacks a prerequisite (e.g., a specific network add-on, external identity provider, or object storage). It is marked blocked with the missing prerequisite named, and the closest achievable alternative is tested and noted. A blocked item does not prevent release, but it MUST be listed as "not live-verified" in the audit report and the release notes.
- A live test would disturb existing real workloads or data on kubelab (a long-lived cluster). Tests run in isolated, clearly named test resources and are cleaned up; pre-existing servers, accounts, and data are never modified or removed.
- A test fails intermittently. It is repeated enough times to classify it as flaky versus deterministic, and the observed rate is recorded.
- Live behavior differs from documentation or specification. Which one is wrong is decided per case and either the code or the document is corrected; the mismatch is never silently ignored.
- Rate limits (login attempts) throttle the tester mid-pass. The plan accounts for this budget rather than treating throttling as a failure.
- A fix for one finding regresses an already-verified feature. Affected inventory rows are re-run before the release decision.
- Finding volume is large. Every finding still blocks the release; severity only sets the order in which fixes are made (data loss, security, and core paths first).

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The audit MUST produce a feature inventory enumerating every user-facing capability: each dashboard screen and action, each API capability, each operator-managed resource kind and its lifecycle, each in-pod agent capability, each auxiliary component, each installation option, and each bundled game module category.
- **FR-002**: Every inventory item MUST be exercised live on the kubelab cluster through its normal user path and receive a recorded outcome (pass, fail, blocked, or not-applicable with reason) with evidence. Only items blocked by a missing cluster prerequisite may remain unexercised at release, and they MUST be disclosed as not live-verified.
- **FR-003**: The audit MUST produce a component coverage table for all repository components, and every component MUST have a completed review record.
- **FR-004**: Each finding MUST be recorded with component, reproduction or observation steps, severity, and disposition.
- **FR-005**: Findings MUST be classified into severity levels used to order the fixing work. Every finding, regardless of severity (including cosmetic ones), blocks the release until fixed and re-verified, closed as not a defect, or recorded as out of scope because the roadmap places that capability after v0.3.
- **FR-006**: Previously known bugs from earlier live runs MUST be imported into the findings list before the pass begins.
- **FR-007**: Every finding that is neither closed as not a defect nor out of scope MUST be fixed and re-verified by repeating its live reproduction on kubelab.
- **FR-008**: Security-sensitive behavior (unauthenticated exposure of internal data, role-based access enforcement, outbound-connection guarding, console input guarding, audit-log tamper detection, secret handling) MUST be verified by an active attempt to violate it, not only by inspection.
- **FR-009**: Live testing MUST NOT modify or delete pre-existing workloads, accounts, or data on kubelab; all test resources MUST be clearly identifiable and removed after testing.
- **FR-010**: The audit MUST include an upgrade-path test from the current beta release to the release candidate on a cluster holding real data, including restart and rollback behavior.
- **FR-011**: The audit MUST include multi-node behavior checks (scheduling across nodes, node drain, node loss) for game server workloads.
- **FR-012**: Live test procedures MUST be written down so that they are repeatable by another maintainer, and any procedure that proves valuable and can be automated MUST be proposed for the automated end-to-end suite.
- **FR-013**: Documentation-versus-behavior mismatches found during the audit MUST be recorded and resolved by correcting whichever side is wrong.
- **FR-014**: The audit MUST end with a decision-ready report giving results by component and feature, open findings, and a go/no-go against the release criteria.
- **FR-015**: The release criteria MUST be defined before testing begins. "No longer beta" means: the released version is v0.3.0 with no beta suffix; project status wording changes from Beta to a pre-v1 release (not a production-support or v1 claim); the upgrade from the last beta is verified live; and remaining pre-v1 caveats stay listed in the roadmap.
- **FR-016**: On a go decision, all locations that describe the project's beta status or carry a beta version marker MUST be identified so they can be updated together to the v0.3.0 wording.
- **FR-017**: Automated test and lint verification remains the responsibility of the project's continuous integration; live results complement, and do not replace, that verification.
- **FR-018**: Release candidates under test MUST be deployed to kubelab from images published under a public pre-release tag, so the audited artifacts are the same ones users would pull. The deploy MUST change only the image version and any explicitly intended settings of the existing installation, and MUST leave its registry, module source, and other site-specific settings as they were unless the change is itself under test.
- **FR-019**: Publishing a pre-release tag is an outward-facing action; the maintainer MUST explicitly approve each publication before it happens.
- **FR-020**: The feature inventory, findings list, component coverage table, and final report MUST be kept as files inside this feature's spec folder, so they are versioned with the code and reviewable in the pull request.

### Key Entities

- **Feature Inventory Item**: A single user-facing capability; has a name, owning component, live test procedure, outcome, and evidence.
- **Component Review Record**: A per-component record of what was reviewed, when, by what method, and which findings resulted.
- **Finding**: A defect or discrepancy; has component, reproduction, severity, disposition, fix reference, and re-verification result.
- **Release Criteria**: The pre-agreed conditions (severity thresholds, inventory pass rate, upgrade safety) that decide go/no-go.
- **Audit Report**: The final consolidated summary of inventory outcomes, component coverage, findings, and the go/no-go decision.
- **Test Resource**: An identifiable, disposable object created on kubelab for a live test and removed afterward.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 100% of feature inventory items have a recorded live outcome with evidence; 0 items are left unassessed. Items blocked by a missing cluster prerequisite are named in the report and release notes as not live-verified; 0 failing items exist at release.
- **SC-002**: 100% of repository components have a completed review record.
- **SC-003**: 0 open findings of any severity at the time of the go/no-go decision.
- **SC-004**: 100% of fixed findings have a passing live re-verification recorded.
- **SC-005**: An existing beta installation with a running game server and saved data upgrades to the release candidate with zero data loss and zero lost accounts.
- **SC-006**: 100% of pre-existing workloads and data on kubelab are unchanged after the audit, and 0 test resources remain after cleanup.
- **SC-007**: Every security-sensitive behavior listed in FR-008 has at least one recorded active violation attempt and its result.
- **SC-008**: A maintainer who did not run the audit can determine the go/no-go outcome and its reasons from the report alone in under 15 minutes.
- **SC-009**: Every finding closed as not a defect or as out of scope has a written justification (with a roadmap citation for out-of-scope); 0 findings are deferred any other way.

## Assumptions

- "Release" means the first non-beta release, v0.3.0; it is explicitly not v1, and features deferred to v1 in the roadmap stay out of scope for the release criteria.
- Pre-release candidate versions (e.g. release-candidate tags) are used for the audit builds published per FR-018; the final v0.3.0 tag is cut only after a clean pass.
- kubelab (the maintainer's existing three-node cluster) is the live-test target. It is a long-lived, real cluster, so tests are isolated and non-destructive to existing data.
- Scope covers all repository components, including auxiliary services, the Helm chart, module tooling, and documentation; the public website and modules content are reviewed for consistency with product behavior but are separate repositories.
- Bugs found in earlier runs are recorded in the maintainers' notes, prior PRs, and issues; the audit collects them into the findings list.
- All bugs found are fixed as part of this effort, whatever their severity; the audit may therefore take several fix-and-retest rounds.
- Fixes follow the project's normal process (branches, pull requests, human review, CI verification); this specification does not authorize bypassing them.
- Some features may need prerequisites kubelab lacks (external identity provider, object storage, specific network add-ons); those are recorded as blocked rather than failed.
- kubelab currently runs side-loaded images from a private tag; moving it to public pre-release images is a deliberate change to its configuration, so it is recorded and the original values are noted so they can be restored after the audit.
