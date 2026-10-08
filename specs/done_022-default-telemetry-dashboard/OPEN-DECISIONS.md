# Open Decisions

**Status**: 0 open; 6 ruled (OD-2, OD-3 and OD-4 on 2026-10-06, OD-1 and OD-5 on 2026-10-07, OD-6 on 2026-10-08).

Per CLAUDE.md rule 10, an open value MUST NOT be committed as a settled contract in code, chart defaults or docs until it is ruled here.

---

### OD-1: Default telemetry domain

**Status**: RULED (2026-10-07, user)

**Question**: Which domain (and ingest path) is the project's default telemetry destination?

**Why it matters**: FR-001 needs a concrete default. FR-002 forbids shipping a placeholder. Until this is ruled, the default stays unset, stock installs keep today's behavior, and the opt-out default (FR-003) has no effect.

**Context**: the project website has no custom domain yet. It is served from GitHub Pages at `valgulnecron.github.io/gameplane-website` (`website/src/config.ts`: `CUSTOM_DOMAIN` is undefined). GitHub Pages is static hosting, so it can't receive reports.

**Options**:
1. Register one project domain and use `telemetry.<domain>`. The website can move to the same domain later.
2. A dedicated domain used only for telemetry.

**Recommended default**: (1). A host users already know from the docs looks more trustworthy in the first-login notice and the Admin Settings destination line (FR-004, FR-017), and there is only one domain to register and renew.

**Ruling (2026-10-07, user)**: the project domain is `gameplane.net`. The default telemetry destination host is `telemetry.gameplane.net`, so `telemetry.DefaultEndpoint = "https://telemetry.gameplane.net/ingest"` (the receiver serves `POST /ingest`, `GET /v1/challenge` and `GET /v1/summary`). The website moves to `gameplane.net`, so the data-handling statement URL (OD-2) is `https://gameplane.net/telemetry/`. The website-side changes (`CUSTOM_DOMAIN`, the statement page) belong to T096 in the `website` submodule.

**Consequences**:
- `DefaultEndpoint` is set, so `telemetry-default-gate` passes and the feature PR can leave draft once the other gates are green.
- The destination copy in the designs and the docs name `telemetry.gameplane.net`; the bundled and custom sample hosts stay samples.

**Superseded interim ruling (2026-10-06, user)**: The domain is not decided yet.
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

---

### OD-5: Proof-of-work defaults

**Status**: RULED (2026-10-07, user)

**Question**: Which defaults does the receiver's proof-of-work use (FR-039, FR-040, research R21)? The user set its shape on 2026-10-07 (spec Q12): off by default and turned on by the runbook for the project's provider, with a difficulty of zero while traffic is normal or absent, rising fast with the request rate and falling slowly.

**Ruling**: the "gentler ceiling" option.

| Setting | Value | Effect |
|---|---|---|
| `INGEST_POW` | `false` | Off in the binary and in the bundled receiver; the runbook turns it on for the project's provider. |
| `INGEST_POW_TARGET_PER_MIN` | `60` | The normal challenge rate. No work is required at or below it. |
| `INGEST_POW_MIN_BITS` | `0` | No work under normal load. |
| `INGEST_POW_MAX_BITS` | `22` | The ceiling during a flood: about 4.2 million hashes, under a second on a desktop and roughly 5–8 seconds on a Raspberry Pi 4 (estimates). |
| `MaxPoWBits` (install cap) | `26` | The install refuses harder challenges (about 67 million hashes, a minute or two on a Raspberry Pi 4), so a misconfigured provider can't burn an install's CPU. |
| Curve | `ceil(2 · log2(r / T))` | 10× the normal rate gives 7 bits, 100× gives 14, 1,000× gives 20. |
| Decay | 1 bit per 5 minutes | From 22 bits back to 0 in under 2 hours. |
| Challenge lifetime | 15 minutes | Longer than a Raspberry Pi 4 needs at the 26-bit cap. |
| Challenge requests per source | 10 per minute, burst 5 | One source can't raise the difficulty for everyone. |
| Used-challenge memory | 1,000,000 entries | Filling it needs that many solved challenges within 15 minutes. |

### OD-6: Dashboard as-of day

**Status**: RULED (2026-10-08, user)

**Question**: The receiver dashboard's ranges ended on the latest complete UTC day (yesterday), so a provider saw "No reports yet" for up to a day after its first accepted report, and today's reports never showed until the day closed.

**Ruling**: the dashboard shows all data. Every range (7, 30, 90 or 365 days) ends on today, the current UTC day, which is still in progress and labelled so. The empty state appears only when the range holds no report at all. Today's lapsed-installs value stays `0` until the lifecycle job finalises the day. The public `GET /v1/summary` is unchanged and keeps the latest complete day.
