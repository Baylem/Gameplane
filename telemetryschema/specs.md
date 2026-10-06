# telemetryschema — Specification

**Status:** skeleton (spec 022, filled in by T016)
**Module / package:** `github.com/ValgulNecron/gameplane/telemetryschema`
**Dependencies:** stdlib only (Go 1.26+)

## Purpose

Shared telemetry report contract used by `api` (reporter and preview) and `telemetry-receiver` (ingest), so both sides decode, validate and sign the same report shape.

## Responsibilities

1. Define the report types and enumerations.
2. Decode and validate report bodies, mapping out-of-set values to `other`.
3. Hold the official module catalog.
4. Sign and verify reports.

(Skeleton: the full list is written in task T016.)

## External interface

See [`specs/022-default-telemetry-dashboard/contracts/report-schema.md`](../specs/022-default-telemetry-dashboard/contracts/report-schema.md).

## Key invariants

- No free text in a report; every category is bounded.
- Stdlib only; no network or filesystem access.

## Testing & coverage

Unit tests only. Coverage gate: 90% total, set in `.testcoverage.yml` and enforced by `make cover-go-check`.
