# Open Decisions: App Localization & Community Language Packs

These values are unsettled. They are not contracts; resolve them during `/speckit-clarify` or `/speckit-plan`.

## OD-001: Which pilot language ships at launch — OPEN

At launch, Gameplane must ship English plus one pilot language that is 100% complete for dashboard text, server messages, and the three reference modules (Minecraft Java, Terraria, Valheim). The user confirmed the 2-language strategy on 2026-09-23 but did not specify which language to choose.

Options and constraints:

- The pilot language must have a native-speaking reviewer available to approve translations and community contributions.
- Ideally a language with high open-source community activity to enable ongoing contribution.
- Candidates must be evaluated for translation site availability (OD-002) and completeness by launch.

Affects: FR-037, SC-002

## OD-002: Which community translation site — OPEN (planning)

The system must support two contribution paths: file-based pull requests and a web-based community translation site where translators work in the browser. The specific translation site service is not yet chosen.

The site must:

- Be free for open-source projects, owned or managed by the maintainers.
- Support human review and approval workflows; only approved translations ship.
- Send approved work back into the repository automatically as pull requests subject to automated checks (FR-015).
- Show per-text context (where the text appears, placeholder meaning, length hints) to translators.
- Handle languages with multiple plural forms and placeholder reordering.
- Cover both the app repository and module repositories for consistent translation across the project.
- Automatically surface new or changed English text within 24 hours without manual steps.

Affects: FR-020, US3

## OD-003: Minimum completeness threshold for language listing (default assumed 50%) — OPEN

Languages below a certain completeness threshold must not be offered to end users, though they remain available to translators for continued work. The default assumption is 50% completeness.

Options and constraints:

- Threshold must be specified as a completeness percentage.
- Threshold is adjustable by maintainers after launch.
- Incomplete packs below threshold fall back to English for all their text; users are not offered the incomplete language.
- Completeness is computed and reported automatically on pull requests and the translation site.

Affects: FR-017

## OD-004: Notification destination language — per-destination or instance-wide only? (default assumed per-destination) — OPEN

Notifications are sent to multiple destinations (Discord, email, webhooks, and others), each of which could have its own language preference, or all could use a single instance-wide language.

Options:

- Per-destination language setting: each notification destination can be configured in a specific language (default: the instance default language). This allows a team to receive Discord alerts in one language and email notifications in another.
- Instance-wide only: all notification destinations use the instance default language, and no per-destination language configuration is offered.

Default assumption: per-destination language setting.

Affects: FR-028
