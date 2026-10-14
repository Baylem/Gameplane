# Open Decisions: App Localization & Community Language Packs

All four decisions below were settled by valgul on 2026-10-10 and are recorded in spec.md (Clarifications, Session 2026-10-10).

## OD-001: Which pilot language ships at launch — RESOLVED: French (valgul, 2026-10-10)

At launch, Gameplane must ship English plus one pilot language that is 100% complete for dashboard text, server messages, and the three reference modules (Minecraft Java, Terraria, Valheim). The user confirmed the 2-language strategy on 2026-09-23 and chose French on 2026-10-10.

Options and constraints:

- The pilot language must have a native-speaking reviewer available to approve translations and community contributions.
- Ideally a language with high open-source community activity to enable ongoing contribution.
- Candidates must be evaluated for translation site availability (OD-002) and completeness by launch.

Affects: FR-037, SC-002

## OD-002: Which community translation site — RESOLVED: Hosted Weblate (valgul, 2026-10-10)

The system must support two contribution paths: file-based pull requests and a web-based community translation site where translators work in the browser. The chosen service is Hosted Weblate (libre plan); setup is described in research.md R10.

The site must:

- Be free for open-source projects, owned or managed by the maintainers.
- Support human review and approval workflows; only approved translations ship.
- Send approved work back into the repository automatically as pull requests subject to automated checks (FR-015).
- Show per-text context (where the text appears, placeholder meaning, length hints) to translators.
- Handle languages with multiple plural forms and placeholder reordering.
- Cover both the app repository and module repositories for consistent translation across the project.
- Automatically surface new or changed English text within 24 hours without manual steps.

Affects: FR-020, US3

## OD-003: Minimum completeness threshold for language listing — RESOLVED: 80% (valgul, 2026-10-10)

Languages below a certain completeness threshold must not be offered to end users, though they remain available to translators for continued work. The threshold is 80% completeness of the app text source (see data-model.md, Language).

Options and constraints:

- Threshold must be specified as a completeness percentage.
- Threshold is adjustable by maintainers after launch.
- Incomplete packs below threshold fall back to English for all their text; users are not offered the incomplete language.
- Completeness is computed and reported automatically on pull requests and the translation site.

Affects: FR-017

## OD-004: Notification destination language — RESOLVED: per-destination (valgul, 2026-10-10)

Notifications are sent to multiple destinations (Discord, email, webhooks, and others), each of which could have its own language preference, or all could use a single instance-wide language.

Options:

- Per-destination language setting: each notification destination can be configured in a specific language (default: the instance default language). This allows a team to receive Discord alerts in one language and email notifications in another.
- Instance-wide only: all notification destinations use the instance default language, and no per-destination language configuration is offered.

Decision: per-destination language setting, defaulting to the instance default language.

Affects: FR-028
