# telemetry-dashboard.pen export manifest

This is a snapshot of `telemetry-receiver/telemetry-dashboard.pen` for review in git. The `.pen` file is accessed only through Pencil MCP. It holds the receiver's private dashboard pages for spec 022 (T066, T075), built on a copy of the HeroUI design-system kit.

## Contents (2026-10-07, first export)

| Node | Frame | Spec |
|---|---|---|
| `HBYBE` | Receiver / Login | T066, FR-022 |
| `a7wUpP` | Receiver / Login — refused (generic "Invalid credentials") | T066, FR-023 |
| `i0xK1E` | Receiver / Overview: basic section (KPIs, reports per day, versions, servers per install) and extended section (coverage, install KPIs, new and lapsed small multiples, environment, games, feature adoption, privacy footnote) | T066, T075, FR-024–FR-027 |
| `Z079vK` | Receiver / Overview — empty (no zero-filled charts) | T066, FR-027 |
| `jy9Rn` | Receiver / Extended — none in range | T075, T076 |
| `y49Ie` | Telemetry components: `sDDfo` Telemetry/Stat tile, `wPbIp` Telemetry/Share row | T066 |

`variables.json` lists every colour and font token, light and dark, for the server-rendered CSS (T068).

## Theme

- **Palette.** This follows the pink theme the main app and website share: the `web/src/styles/globals.css` values under the same HeroUI token names, the same as `website/website-export`. The kit copy in this file is re-themed with it.
- **Typography.** Fonts are Geist (sans) and JetBrains Mono (headings, eyebrows, wordmark).
- **Modes.** The frames are rendered in dark mode, the website's default. Light values are defined for every token.
- **Chart tokens.**
  - `chart/series-1` is pink: `#DB2777` light, `#EC4899` dark. Brand `#FF4FA3` is too light for dark-surface marks.
  - `chart/series-2` is blue: `#2a78d6` light, `#3987e5` dark, used for lapsed installs.
  - Both pass the dataviz validator in both modes against the file's surfaces (`#FFF7FB`, `#1C1A20`).
  - `chart/grid`, `chart/baseline` and `chart/track` are recessive hairline and track tones.
- **Chart rules.** Columns start at zero, there is a single y-axis, and only the latest value is labelled. Small multiples share a scale. A "View as table" link gives a table alternative, because the CSP-locked pages have no JavaScript for hover.

## Method

- **JSON.** `Get(id, {depth: 40, includePathGeometry: true})` was printed from one `execute` call. A script split the output into `json/<id>.json` (2-space indent, UTF-8, trailing newline). The compact re-serialisation length of every file matches `JSON.stringify` in Pencil exactly, and no file contains `"..."` elisions.
- **PNG.** `Export(ids, "png", …, {scale: 2})`. Login, refused and empty are 2880×1800, the overview is 2880×5634, "none in range" is 2880×328 and the components frame is 1624×354.
- **Content checks.** Each string matched only its own file:
  - "Invalid credentials" → `a7wUpP`
  - "No reports yet" → `Z079vK`
  - "Lapsed installs per day" → `i0xK1E`
  - "No reports in this range included extended data" → `jy9Rn`
  - "DASHBOARD_TOKEN value" → `HBYBE`, `a7wUpP`

## Incremental export (2026-10-08): dashboard ends on today (OD-6)

| Node | Change |
|---|---|
| `i0xK1E` | "Data through 2026-10-06 (UTC, today in progress)"; tiles "Reports, today" and "Active installs, today" with "2026-10-06 (UTC, in progress)"; "Median servers per install" sub "Last 30 days"; captions "…, ending today (in progress)", "Reports in range by reported version…", "Reports by game-server count. Median 3." and the lapsed note "Today's bar is final after the UTC day closes." |
| `Z079vK` | Empty-state text: "Figures appear as soon as this receiver accepts a report; today's figures are in progress and update as reports arrive. …" |
| `y49Ie` | `sDDfo` Stat tile sample: "Reports, today", "2026-10-06 (UTC, in progress)" |

Method: 15 `Update` calls in one `execute`. The JSON files were produced by applying the same edits to the previous export, then checked equal to Pencil (`JSON.stringify(Get(id, {depth: 40, includePathGeometry: true}))` length and FNV-1a hash per frame: `i0xK1E` 69071, `Z079vK` 4684, `y49Ie` 1854). PNGs: `Export(ids, "png", …, {scale: 2})`, same sizes as before. Content check: "today in progress" → `i0xK1E`; "accepts a report" → `Z079vK`.
