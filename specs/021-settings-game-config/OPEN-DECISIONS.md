# Open Decisions: 021-settings-game-config

Unsettled items. Do not treat any of the below as decided until a maintainer rules on it here.

## Settled (user, 2026-10-05)
Q1 passwords redacted by the API (PR #565) and write-only in the UI; Q2 always restart; Q3 no API-side schema validation; Q4 separate section after Version; Q5 orphan keys with Remove; Q6 "Written to file" note, selects for bool; Q7 gate on `access.canWrite`.

## OD-1: Move password values out of the CR into a Secret — **Open**
Passwords still live in plaintext in `GameServer.spec.config` (readable with cluster access; the API now redacts them in responses). Options: (a) API strips password values into the `<gs>-config` Secret on PUT/POST and leaves a sentinel, operator reads the Secret when the sentinel is present; (b) a dedicated `PUT /servers/{name}:config-secrets` endpoint like `:tunnel-credentials`. Either changes operator/API/CRD semantics and needs migration of existing servers. Not decided here.

## OD-2: Clearing a stored optional password — **Open**
The design has no "clear" affordance; emptying the input returns to "unchanged". The API would clear an optional password if the key were absent or `""`. Needs a design (e.g. a "Remove password" action).

## OD-3: Show the template file path for `target: file` fields — **Open**
The design says "Written to serverconfig.txt"; the schema carries no path (configFiles own paths), so the UI says "Written to file". Needs a schema addition if the path should be shown.

## OD-4: Per-field "Reset to default" — **Open**
Not in the design; setting a value equal to the default removes the key instead.
