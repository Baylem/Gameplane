# Contract: `hack/check_layout.py`

Python 3 stdlib. Run as `make check-layout`, part of `make lint` (last PR, FR-019).

## Rules

1. **Ceiling (OD-3, OD-4)**: every tracked production `.go`, `.ts`, `.tsx` file has at most 500 lines. Skips `_test.go`, `*.test.ts(x)`, `web/src/test/`, `web/e2e/`, generated files (`zz_generated.*`, `web/src/api/types.ts`, `operator/config/`), `operator/api/v1alpha1/*_types.go`, `test/e2e/`, `modules/`, `website/`.
2. **Service shape (027 OD-3)**: `tunnel`, `sentinel`, `telemetry-receiver`, `audit-syslog-bridge`, `mcp-server`, `api`, `operator`, `agent`, `capture-sidecar` have no `.go` file at the module root; `cmd/main.go` exists.
3. **Web shape (027 OD-4, OD-7)**: screens live in `web/src/pages/`; `web/src/routes/` holds only route definitions (per OD-7).

## Output

One line per violation: `path: rule N (<docs/layout.md section>): detail`. Exit 1 on any.

## Tests

`hack/test_check_layout.py` builds temporary trees for each rule (pass and fail cases);
run in CI with the other `hack/` unit tests, never locally.
