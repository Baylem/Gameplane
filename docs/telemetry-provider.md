# Running a telemetry provider

This is the runbook for running a Gameplane telemetry receiver for other
installs: the project maintainers' own default receiver, or one you host
yourself. It covers deployment, TLS, secrets, retention, metrics and
backups. It is a runbook only; the repository ships no infrastructure
code for the project's receiver.

For what installs send, see [install.md](install.md#telemetry). For the
threat model, see [security.md](security.md#telemetry). For the receiver's
behaviour and metrics, see
[`telemetry-receiver/README.md`](../telemetry-receiver/README.md). The
project's data-handling statement is at
<https://valgulnecron.github.io/gameplane-website/telemetry/>.

## Listeners at a glance

| Listener | Variable (default) | Serves | Exposure |
|---|---|---|---|
| Public | `LISTEN_ADDR` (`:8080`) | `POST /ingest`, `GET /v1/summary` (only when `PUBLIC_SUMMARY=true`), `GET /v1/challenge` (only when `INGEST_POW=true`), `GET /healthz` | Public internet, behind TLS |
| Dashboard | `DASHBOARD_LISTEN_ADDR` (`:8081`) | login page, dashboard, `/api/v1/views`, `/metrics`, static assets | Private network only |

The dashboard listener starts only when `DASHBOARD_TOKEN` is set. The
public listener answers `404` for `/metrics`.

## Deploy the image

The receiver is a single static binary in a distroless, non-root image:

```sh
docker run -d --name telemetry-receiver \
  -p 127.0.0.1:8080:8080 -p 127.0.0.1:8081:8081 \
  -v telemetry-data:/data \
  -e DATA_DIR=/data \
  -e DASHBOARD_TOKEN="$(cat dashboard-token)" \
  -e ID_PEPPER="$(cat id-pepper)" \
  -e PUBLIC_SUMMARY=true \
  ghcr.io/valgulnecron/gameplane/telemetry-receiver:edge
```

Notes:

- Use a release tag instead of `edge` in production.
- The image runs as UID/GID `65532` with a read-only-friendly layout. The
  `/data` volume must be writable by that user.
- Without `DATA_DIR` the receiver keeps everything in memory, logs a
  warning, and loses it all on restart. Always set it for a real provider.
- Run **one replica** only. The database is a single SQLite file; two
  processes on it will conflict. Use a recreate rollout, not a rolling one.
- Kubernetes: the chart's receiver
  (`api.telemetry.receiver.enabled=true`) renders all of this, including a
  PVC at `/data`, a `Recreate` strategy and NetworkPolicies. A standalone
  provider can use the same Deployment shape or the chart.

### Configuration

| Variable | Default | Meaning |
|---|---|---|
| `LISTEN_ADDR` | `:8080` | Public listener. |
| `DASHBOARD_LISTEN_ADDR` | `:8081` | Dashboard listener. |
| `DASHBOARD_TOKEN` | *(empty)* | Dashboard credential, from a Secret. Empty means no dashboard listener. When set it must be at least 32 characters, or the receiver exits at startup. See [Secrets](#secrets). |
| `AUTH_TOKEN` | *(empty)* | Ingest token. Leave empty on a public provider: installs have no credential to present. |
| `DATA_DIR` | *(empty)* | SQLite directory. Empty means in-memory. |
| `PUBLIC_SUMMARY` | `false` | Enables `GET /v1/summary`. |
| `TRUSTED_PROXY_CIDRS` | *(empty)* | Comma-separated CIDRs whose `X-Forwarded-For` is trusted. |
| `INGEST_SOURCE_DAILY_LIMIT` | `20` | Accepted reports per source per UTC day. `0` means unlimited. |
| `RETENTION_DAYS` | `730` | Daily aggregate retention. Minimum `365`. |
| `ACTIVITY_EXPIRY_DAYS` | `90` | Activity record expiry. Minimum `31`. |
| `ID_PEPPER` | *(empty)* | HMAC pepper for install IDs, from a Secret. Empty means one is generated and kept in the database. |
| `INGEST_POW` | `false` | Requires proof-of-work on `/ingest` and serves `/v1/challenge`. See [Proof-of-work on ingest](#proof-of-work-on-ingest). |
| `INGEST_POW_TARGET_PER_MIN` | `60` | Normal challenge rate: no work is required at or below it. Must be at least `1`. |
| `INGEST_POW_MIN_BITS` | `0` | Lowest difficulty issued. Must not be negative. |
| `INGEST_POW_MAX_BITS` | `22` | Highest difficulty issued. At least `INGEST_POW_MIN_BITS` and at most `26`. |

A value below a minimum, a malformed `TRUSTED_PROXY_CIDRS` entry, an
unparsable number, a `DASHBOARD_TOKEN` shorter than 32 characters or an
`INGEST_POW_MAX_BITS` outside its range stops the receiver at start with an
error.

## TLS in front of `:8080`

The receiver speaks plain HTTP. Installs only send reports over `https`,
so terminate TLS in front of the public listener with a reverse proxy,
ingress controller or load balancer, and forward to `:8080`.

- Forward only `/ingest`, `/v1/summary`, `/healthz` and, when proof-of-work
  is on, `/v1/challenge`. Do not publish `:8081`.
- Keep the request body limit at or above 16 KiB (the receiver rejects
  anything larger with `413`).
- Pass the client address in `X-Forwarded-For` and set
  `TRUSTED_PROXY_CIDRS` (below).

## Keep `:8081` private

The dashboard shows every aggregate and `/metrics`. Reach it over a private
network, a VPN or `kubectl port-forward`, never a public route. In
Kubernetes, the chart's NetworkPolicy admits port 8081 only from peers in
`api.telemetry.receiver.dashboard.ingressFrom` (plus the Prometheus
namespace when `serviceMonitors.scrapeNamespaceSelector` is set); with no
peers, `kubectl port-forward` still works. The session cookie is `Secure`,
so a browser needs `https` (or `localhost`) to keep a login.

## Secrets

Create both from Secrets, not literals in a manifest.

```sh
# 32 random bytes: 44 characters in base64 (64 with `openssl rand -hex 32`).
kubectl -n gameplane-system create secret generic telemetry-dashboard \
  --from-literal=token="$(openssl rand -base64 32)"
kubectl -n gameplane-system create secret generic telemetry-pepper \
  --from-literal=pepper="$(openssl rand -base64 32)"
```

With the chart: `api.telemetry.receiver.dashboard.tokenSecretRef.name` and
`api.telemetry.receiver.pepperSecretRef.name` (key `token` and `pepper` by
default). They reach the pod as `DASHBOARD_TOKEN` and `ID_PEPPER`.

- `DASHBOARD_TOKEN` is the one credential for the dashboard, its JSON
  views and `/metrics`. Generate it from 32 random bytes, as above. The
  receiver refuses to start when the token is set but shorter than 32
  characters (it counts characters, not bytes), because the per-source
  login limit can be outrun by someone holding many IPv6 `/64`s and the
  token's length is what makes guessing infeasible. A rotated token must
  meet the same minimum, or the restart fails.
- `ID_PEPPER` keys the HMAC that turns an install ID into the stored
  record key. If you leave it unset, the receiver generates one on first
  start and stores it in the database. Set it explicitly when you want the
  pepper to survive a lost database or to stay out of database backups.
  Set it **before** the first report arrives: changing it later has the
  effect described below.

### Rotating secrets

<a id="rotating-secrets"></a>

| Secret | Rotate by | Effect |
|---|---|---|
| `DASHBOARD_TOKEN` | Update the Secret and restart the receiver. | Every browser session is invalidated (the cookie key derives from the token), scripts need the new token, and Prometheus needs the new Bearer token. No stored data is lost. |
| `ID_PEPPER` | Update the Secret and restart the receiver. | Stored activity records no longer match any install ID. For the next day, every install looks new (new-install counts spike and lapsed counts are wrong for the 30 days that follow), unique-install and by-install views lose their history, and ID claims are forgotten, so installs re-claim on their next report. No install is rejected and no daily aggregate changes. Rotate only after a compromise. |

Installs themselves are never affected by either rotation: they keep their
install IDs and signing keys. Neither rotation touches `AUTH_TOKEN`; rotate
that separately on the receiver and in `GAMEPLANE_TELEMETRY_AUTH` on each
install (only relevant for a private provider).

## Public summary

Set `PUBLIC_SUMMARY=true` (chart: `api.telemetry.receiver.publicSummary.enabled`)
to enable `GET /v1/summary` on the public listener. It needs no credentials
and returns exactly five keys:

```json
{
  "asOf": "2026-10-05",
  "reportsLatestDay": 412,
  "reports30d": 11873,
  "reportsTotal": 98211,
  "activeInstalls30d": 1290
}
```

It is served from a snapshot refreshed at most every 5 minutes, with
`Cache-Control: public, max-age=3600`, an `ETag` and
`Access-Control-Allow-Origin: *`, and is rate-limited per source. The
project's default receiver enables it; it is off by default elsewhere.

## Proof-of-work on ingest

<a id="proof-of-work-on-ingest"></a>

`/ingest` is open to the internet, so a flood of fabricated reports costs
the sender nothing but bandwidth. Proof-of-work makes the cost grow with the
request rate while normal traffic pays nothing. It is off by default. Turn it
on for a public provider; leave it off for a private or bundled receiver.

### Enable it

- Standalone: set `INGEST_POW=true`. The receiver then serves
  `GET /v1/challenge` and answers `428` to any `/ingest` without a valid
  solved challenge.
- Chart: `api.telemetry.receiver.ingestPow.enabled=true`, with
  `targetPerMinute`, `minBits` and `maxBits` for the three tuning
  variables below. The chart refuses to render when `targetPerMinute` is
  below `1`, `maxBits` is above `26` or `maxBits` is below `minBits`.
- Route `/v1/challenge` through your TLS proxy next to `/ingest`. It has its
  own per-source limit (10 per minute, burst 5).
- Installs that run an API without proof-of-work support cannot solve a
  challenge and are refused with `428 pow_required`. Installs that find no
  challenge endpoint (any answer other than `200`) send without it, which
  is why it is safe to leave off.

How it works: each install asks for a challenge, finds a number so that
`SHA-256(challenge ":" number)` starts with the requested count of zero
bits, and sends the answer in the `Gameplane-Telemetry-PoW` header. A
challenge is valid for 15 minutes and for one report. Verification happens
before the body is read, so a refused report costs the receiver one HMAC and
one hash. The challenge key and the set of used challenges are in memory: a
restart invalidates outstanding challenges, and installs recover with one
automatic retry.

### Tune it from `/metrics`

Difficulty depends on `r`, the challenges issued in the last 60 seconds, and
`INGEST_POW_TARGET_PER_MIN` (`T`). At or below `T` the difficulty is
`INGEST_POW_MIN_BITS` (default `0`: no work). Above it, it is
`ceil(2 * log2(r / T))`, capped at `INGEST_POW_MAX_BITS`; it rises at once and
falls one bit every five minutes. An install's expected work is about
`2^bits` hashes, and installs refuse anything above 26 bits.

Each daily report asks for one challenge, so challenges per minute are
about reports per minute (a retry after `428` asks for another). To choose
`INGEST_POW_TARGET_PER_MIN`:

1. Run for a few days with `INGEST_POW=true` and defaults.
2. Read the busiest normal minute from
   `gameplane_telemetry_pow_challenges_total`, for example
   `max_over_time(rate(gameplane_telemetry_pow_challenges_total[1m])[7d:]) * 60`.
3. Set the target a little above it, so ordinary peaks cost nothing and
   only a real surge raises the difficulty.

Read the result with the gauge `gameplane_telemetry_pow_bits`: it is the
difficulty a challenge issued now would carry. A gauge that sits at your
`INGEST_POW_MIN_BITS` means traffic is normal. A gauge that stays raised
means traffic is above the target (the gauge counts the next challenge too,
so it can read `1` when the last minute was exactly at the target). Watch
alongside it:

| Series | What it tells you |
|---|---|
| `gameplane_telemetry_pow_challenges_total` | Challenges issued; its rate is the `r` that drives difficulty. |
| `gameplane_telemetry_refused_total{reason="pow_required"}` | Reports with no solution: older installs, scripts, or a flood that skips the challenge. |
| `gameplane_telemetry_refused_total{reason="pow_invalid"}` | Bad, expired, reused or under-difficulty solutions. A burst right after a restart is normal. |
| `gameplane_telemetry_refused_total{reason="pow_busy"}` | The used-challenge set (1,000,000 entries) is full; reports get `503` with `Retry-After: 60`. Not seen in normal use. |
| `gameplane_telemetry_rate_limited_total{route="challenge"}` | One source requested challenges faster than 10 a minute. |

Under attack, respond from the provider side with no install upgrade: lower
`INGEST_POW_TARGET_PER_MIN` or raise `INGEST_POW_MIN_BITS` and restart the
receiver. Many sources can still push the difficulty to
`INGEST_POW_MAX_BITS`; that costs a legitimate install at most a few
seconds of work per daily report, and it decays when the flood stops.

## Source addresses and `TRUSTED_PROXY_CIDRS`

Per-source limits use the TCP peer address. Behind a reverse proxy, every
request would otherwise come from the proxy, and the limits
(`INGEST_SOURCE_DAILY_LIMIT`, default 20 accepted reports per source per
UTC day) would throttle all installs together. Set `TRUSTED_PROXY_CIDRS`
to the proxy's address range (comma-separated, for example
`10.0.0.0/8,192.168.0.0/16`, using your real range). The receiver then
reads `X-Forwarded-For`, but only when the peer is inside those ranges, so a
client can't forge its source. Leave it empty if the receiver is directly
exposed. Addresses are held in memory only, and never stored.

## Retention

| Setting | Default | Minimum | Covers |
|---|---|---|---|
| `RETENTION_DAYS` | `730` (24 months) | `365` | Daily aggregates. Older days are deleted by an hourly sweep. |
| `ACTIVITY_EXPIRY_DAYS` | `90` | `31` | Per-install activity records (transformed ID, key fingerprint, first and last seen, last send time, last version). An ID's claim expires with its record. |

The activity expiry must stay above 30 days because the dashboard's
"lapsed" figure looks back 30 days. The summary's "total reports" counter
is kept separately and survives retention sweeps. If you publish a
data-handling statement, state the values you configured.

## Scrape `/metrics` with the token

Metrics are on the dashboard listener only and need the dashboard token as
a Bearer token. Existing series keep their names; new ones:
`gameplane_telemetry_duplicates_total`,
`gameplane_telemetry_rate_limited_total{route}` (`ingest`, `login`,
`summary`, `challenge`),
`gameplane_telemetry_extended_reports_total`,
`gameplane_telemetry_refused_total{reason}` (`bad_signature`, `stale`,
`id_claimed`, `replay`, `pow_required`, `pow_invalid`, `pow_busy`),
`gameplane_telemetry_pow_challenges_total` and the gauge
`gameplane_telemetry_pow_bits`. The two proof-of-work series are always
exported; `pow_bits` is `0` while `INGEST_POW` is off.

Prometheus scrape config:

```yaml
scrape_configs:
  - job_name: telemetry-receiver
    metrics_path: /metrics
    authorization:
      type: Bearer
      credentials_file: /etc/prometheus/secrets/telemetry-dashboard-token
    static_configs:
      - targets: ["telemetry-receiver.internal:8081"]
```

With the chart and Prometheus Operator, `serviceMonitors.enabled=true`
renders a `ServiceMonitor` for the `dashboard` port with the token Secret
as `bearerTokenSecret`, but only when
`api.telemetry.receiver.dashboard.tokenSecretRef.name` is set. Without a
token, the chart renders no `ServiceMonitor` and prints a warning in the
install notes. Check manually with:

```sh
curl -H "Authorization: Bearer $DASHBOARD_TOKEN" http://localhost:8081/metrics
```

A request without the header gets `401 {"error":"unauthorized"}`.

## Back up and restore `telemetry.db`

All durable state is one SQLite database, `$DATA_DIR/telemetry.db`, in WAL
mode (with `telemetry.db-wal` and `telemetry.db-shm` beside it while the
receiver runs). It holds the daily aggregates, the activity records and,
unless you set `ID_PEPPER`, the pepper.

**Back up**

- Prefer a consistent snapshot: snapshot the volume, or stop the receiver
  (scale to 0) and copy the whole `DATA_DIR`, including the `-wal` and
  `-shm` files if present.
- Or, with the `sqlite3` CLI on a copy host and the receiver running:
  `sqlite3 /data/telemetry.db ".backup '/backup/telemetry-$(date +%F).db'"`.
  Do not `cp` the main file alone while the receiver is writing.
- Treat backups as sensitive: they contain the activity records and
  possibly the pepper. Encrypt them and keep them as short-lived as your
  retention allows.

**Restore**

1. Stop the receiver (scale to 0).
2. Replace `telemetry.db` in `DATA_DIR` with the backup, and delete any
   stale `telemetry.db-wal` and `telemetry.db-shm`.
3. Make sure `ID_PEPPER` is the same value as when the backup was taken (or
   unset, if the pepper is stored in the database). A different pepper
   behaves like a rotation: see [Rotating secrets](#rotating-secrets).
4. Start the receiver and check `/healthz` and the dashboard.

Reports received between the backup and the restore are lost; installs
send again within a day, and per-day deduplication keeps the counts right.
Losing the database with no backup restarts all history; installs just
re-claim their IDs on their next report.

## Checklist

- [ ] TLS terminates in front of `:8080`; only `/ingest`, `/v1/summary`,
  `/healthz` and (with proof-of-work on) `/v1/challenge` are routed.
- [ ] `:8081` is reachable only from a private network.
- [ ] `DATA_DIR` is a persistent volume; one replica.
- [ ] `DASHBOARD_TOKEN` (at least 32 characters, from 32 random bytes) and
  `ID_PEPPER` come from Secrets.
- [ ] A public provider has `INGEST_POW=true`, and `INGEST_POW_TARGET_PER_MIN`
  sits just above your busiest normal minute.
- [ ] `TRUSTED_PROXY_CIDRS` matches your proxy.
- [ ] Prometheus scrapes `:8081/metrics` with the Bearer token.
- [ ] Backups of `telemetry.db` are scheduled and a restore has been
  rehearsed.
- [ ] The data-handling statement states your retention values.
