# Open Decisions

**Status**: 1 open (OD-1), 3 ruled (OD-2, OD-3, OD-4) on 2026-10-06.

Per CLAUDE.md rule 10, an open value MUST NOT be committed as a settled contract in code, chart defaults or docs until it is ruled here.

---

### OD-1: Default telemetry domain

**Status**: OPEN. **Blocks merge of this feature's PR.**

**Question**: Which domain (and ingest path) is the project's default telemetry destination?

**Why it matters**: FR-001 needs a concrete default. FR-002 forbids shipping a placeholder. Until this is ruled, the default stays unset, stock installs keep today's behavior, and the opt-out default (FR-003) has no effect.

**Context**: the project website has no custom domain yet. It is served from GitHub Pages at `valgulnecron.github.io/gameplane-website` (`website/src/config.ts`: `CUSTOM_DOMAIN` is undefined). GitHub Pages is static hosting, so it can't receive reports.

**Options**:
1. Register one project domain and use `telemetry.<domain>`. The website can move to the same domain later.
2. A dedicated domain used only for telemetry.

**Recommended default**: (1). A host users already know from the docs looks more trustworthy in the first-login notice and the Admin Settings destination line (FR-004, FR-017), and there is only one domain to register and renew.

**Interim ruling (2026-10-06, user)**: The domain is not decided yet.
- Implementation proceeds with `telemetry.DefaultEndpoint` left **blank**, as FR-002 allows.
- The feature's PR **MUST NOT merge until a domain is chosen**, recorded here as RULED, and set in `DefaultEndpoint`.

**How the block is enforced** (plan.md, Merge gate):
- The PR stays a **draft** until this decision is ruled. GitHub does not allow merging drafts, and the master ruleset has no required status checks that could block the merge instead.
- A dedicated CI job, `telemetry-default-gate`, fails while `DefaultEndpoint` is empty or isn't `https`.

**Blocks**:
- merge of the feature PR
- FR-001's default value
- the FR-008 upgrade notes
- the destination copy in the first-login notice and Admin Settings designs

---

### OD-2: Provider hosting, ownership, and data-handling statement

**Status**: RULED (2026-10-06, user)

**Question**: Who runs the default provider, and where is its data-handling statement published?

**Ruling**: The project maintainers run the default provider. The data-handling statement is a page on the project website, in the `website` submodule. It is linked from:
- the first-login notice (FR-004)
- Admin Settings → Telemetry (FR-017)
- the upgrade notes (FR-008)

**Consequences**:
- The statement page is new website work. It goes through the submodule's own PR flow, and the website's design-first rule applies only if it needs a new screen layout.
- The dashboard links to the statement through one constant, because the website URL changes once `CUSTOM_DOMAIN` is set (see OD-1).
- The statement states the OD-3 and OD-4 periods.

---

### OD-3: Aggregate retention ceiling

**Status**: RULED (2026-10-06, user)

**Question**: How long does the provider keep daily aggregates? FR-020 sets a floor of 12 months.

**Ruling**: **24 months**. `RETENTION_DAYS` defaults to `730`, the Helm value `api.telemetry.receiver.retentionDays` defaults to `730`, and the enforced minimum is 365. Two years covers year-over-year comparison, and the stated limit appears in the OD-2 statement.

---

### OD-4: Activity-record inactivity expiry

**Status**: RULED (2026-10-06, user)

**Question**: How long after an install's last report does the provider delete its activity record? An activity record holds the transformed install ID, the first-seen date, the last-seen date and the last version (FR-015).

**Ruling**: **90 days**. `ACTIVITY_EXPIRY_DAYS` defaults to `90`, the Helm value `api.telemetry.receiver.activityExpiryDays` defaults to `90`, and the enforced minimum is 31, because the expiry must stay longer than the 30-day lapsed window (US5). This keeps new and lapsed counts accurate for installs that go quiet for a season, and the period appears in the OD-2 statement.
