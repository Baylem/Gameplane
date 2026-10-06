# Contract: telemetry-receiver HTTP surface

**Requirements**: FR-014, FR-016, FR-020–FR-033, FR-035–FR-038. **Research**: R3, R5–R9, R20.

The receiver runs two listeners. The dashboard listener is never started unless `DASHBOARD_TOKEN` is set.

## Public listener: `LISTEN_ADDR` (default `:8080`)

| Route | Method | Auth | Success | Failures |
|---|---|---|---|---|
| `/ingest` | POST | `AUTH_TOKEN` when set (unchanged). Reports with `ext` also need a valid `Gameplane-Telemetry-Signature`. | `204`, including for same-day duplicates (counted only in `duplicates`) | `400` structural error ([report-schema.md](report-schema.md)); `401` bad or missing token; `403` `bad_signature`, `stale` or `replay`; `409` `id_claimed`; `413` body over 16 KiB; `429` per-source daily limit reached |
| `/v1/summary` | GET | none | `200` JSON (below) | `404` when `PUBLIC_SUMMARY` is not `true`; `429` per-source rate limit |
| `/healthz` | GET | none | `200 ok` | — |

### `GET /v1/summary` response (FR-029, FR-030)

```json
{
  "asOf": "2026-10-05",
  "reportsLatestDay": 412,
  "reports30d": 11873,
  "reportsTotal": 98211,
  "activeInstalls30d": 1290
}
```

- The response has exactly these five keys and nothing else.
- `asOf` is the latest complete UTC day.
- Response headers:
  - `Content-Type: application/json`
  - `Cache-Control: public, max-age=3600`
  - `ETag`
  - `Access-Control-Allow-Origin: *`
- The response is served from a snapshot recomputed at most every 5 minutes.
- With no data at all, every count is `0` and `asOf` is the previous UTC day. The summary is a counter, not a chart, so zeros here are accurate.

The public listener answers `404` for `/metrics`. Operational metrics live only on the dashboard listener, behind the token (FR-030).

### New `/metrics` series (operations only)

- `gameplane_telemetry_duplicates_total`
- `gameplane_telemetry_rate_limited_total{route}`
- `gameplane_telemetry_extended_reports_total`
- `gameplane_telemetry_refused_total{reason="bad_signature"|"stale"|"id_claimed"|"replay"}`

The existing series keep their names and in-memory semantics.

## Dashboard listener: `DASHBOARD_LISTEN_ADDR` (default `:8081`, only when `DASHBOARD_TOKEN` is set)

| Route | Method | Auth | Success | When unauthenticated |
|---|---|---|---|---|
| `/login` | GET | none | Login form, which contains no data | — |
| `/login` | POST | form field `token`, same-origin `Origin` required | `303` to `/`, sets the session cookie | Login form re-rendered with "Invalid credentials". `429` after 5 attempts per minute per source. |
| `/logout` | POST | session, same-origin | `303` to `/login`, clears the cookie | `303` to `/login` |
| `/` | GET | session | Overview HTML for `?range=7\|30\|90\|365` (default 30; invalid values become 30) | `303` to `/login` |
| `/api/v1/views` | GET | session cookie, or `Authorization: Bearer <DASHBOARD_TOKEN>` | `200` JSON with the same view model the HTML renders | `401 {"error":"unauthorized"}` |
| `/metrics` | GET | `Authorization: Bearer <DASHBOARD_TOKEN>` only | Prometheus text (existing series, plus the new counters below) | `401 {"error":"unauthorized"}` |
| `/static/*` | GET | none | Embedded CSS and icons only | — |

**Session cookie**: `gp_telemetry_session`, attributes `HttpOnly; Secure; SameSite=Strict; Path=/; Max-Age=43200`. The value is `expiry || HMAC(K, expiry)`, where `K` is derived from `DASHBOARD_TOKEN` (research R8). Changing the token invalidates every session.

**Headers on every dashboard response**:

- `Content-Security-Policy: default-src 'none'; style-src 'self'; img-src 'self' data:; form-action 'self'; frame-ancestors 'none'; base-uri 'none'`
- `X-Content-Type-Options: nosniff`
- `Referrer-Policy: no-referrer`
- `Cache-Control: no-store`

**Refusal invariant (FR-023, SC-011)**: an unauthenticated response never contains:

- any figure
- any date other than the page chrome
- any version, category or module name
- anything that varies with whether data exists

An E2E check (R17) compares the unauthenticated response against the login page from an empty receiver.

### `GET /api/v1/views` response shape

```json
{
  "range": 30,
  "asOf": "2026-10-05",
  "empty": false,
  "basic": {
    "reportsPerDay": [{"day": "2026-09-06", "reports": 401}],
    "versions": [{"label": "0.3.0", "reports": 8123}, {"label": "Other", "reports": 52}, {"label": "Invalid", "reports": 3}],
    "fleet": {
      "servers":   {"bands": [{"le": "0", "reports": 120}], "median": 2},
      "templates": {"bands": [{"le": "0", "reports": 15}],  "median": 4}
    },
    "latestDay": {"serversTotal": 1840, "templatesTotal": 2611}
  },
  "extended": {
    "coverage": 0.71,
    "installs": {"active1d": 290, "active7d": 812, "active30d": 1290},
    "newPerDay": [{"day": "2026-09-06", "count": 12}],
    "lapsedPerDay": [{"day": "2026-09-06", "count": 4}],
    "versionsByInstall": {"windowDays": 30, "items": [{"label": "0.3.0", "installs": 1011}]},
    "env": {"k8s": [{"label": "1.31", "share": 0.42}], "distro": [], "arch": [], "nodes": []},
    "games": [{"module": "minecraft-java", "installsShare": 0.63, "serversPerDay": 2140.5}, {"module": "custom", "installsShare": 0.08, "serversPerDay": 77.1}],
    "features": [{"feature": "wakeOnConnect", "share": 0.18}],
    "tunnels": [{"label": "playit", "share": 0.05}],
    "clusters": [], "db": [], "language": []
  }
}
```

- When the range has no data: `empty: true`, with `basic` and `extended` set to `null`. The HTML renders the empty state (FR-027).
- When there is basic data but no extended reports: `extended: null`, and the HTML explains that no reports in the range included extended data.

## Configuration

| Variable | Default | Meaning |
|---|---|---|
| `LISTEN_ADDR` | `:8080` | Public listener. |
| `DASHBOARD_LISTEN_ADDR` | `:8081` | Dashboard listener. |
| `DASHBOARD_TOKEN` | *(empty)* | Dashboard credential, from a Secret. Empty means no dashboard listener. |
| `AUTH_TOKEN` | *(empty)* | Ingest token (unchanged). |
| `DATA_DIR` | *(empty)* | SQLite directory. Empty means in-memory, and a startup warning is logged. |
| `PUBLIC_SUMMARY` | `false` | Enables `/v1/summary`. |
| `TRUSTED_PROXY_CIDRS` | *(empty)* | Comma-separated CIDRs whose `X-Forwarded-For` is trusted. |
| `INGEST_SOURCE_DAILY_LIMIT` | `20` | Accepted reports per source per UTC day. `0` means unlimited. |
| `RETENTION_DAYS` | `730` (OD-3) | Daily aggregate retention. Minimum 365. |
| `ACTIVITY_EXPIRY_DAYS` | `90` (OD-4) | Activity record expiry. Minimum 31. |
| `ID_PEPPER` | *(empty)* | HMAC pepper for install IDs, from a Secret. Empty means one is generated and kept in `meta`. |
