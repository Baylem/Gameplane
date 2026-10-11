# Contract: Localizable Server Messages

**Feature**: `019-localization-language-packs` | **Binding modules**: `api/internal/httperr`, `api/internal/messages`, `web/src/lib/api.ts` | **Status**: Binding

## 1. Negotiation

| Request `Accept` | Error response |
|---|---|
| contains `application/problem+json` | `Content-Type: application/problem+json`, body §2 |
| anything else (including absent, `*/*`, `application/json`) | unchanged: `Content-Type: text/plain; charset=utf-8`, body = English text + `\n` exactly as today |

Success responses are unaffected. The dashboard always sends `Accept: application/json, application/problem+json`.

## 2. Problem body (RFC 9457 + extensions)

```json
{
  "type": "about:blank",
  "status": 403,
  "title": "Forbidden",
  "detail": "namespace not permitted",
  "messageId": "server.namespace_forbidden",
  "args": {}
}
```

- `detail` MUST equal the text the plain-text response would carry (SC-010).
- `messageId` is present for every message in the registry; responses from code paths not yet migrated carry `"messageId": "server.generic.<status>"` (e.g. `server.generic.400`) so the dashboard still shows a translated generic sentence followed by `detail`.
- `args` values are strings or numbers; names match the English entry's placeholders.
- 5xx keep today's behaviour: `detail` is the status text, never internal error text.
- Unauthenticated endpoints (`/auth/*`, public shares) only ever use generic ids (`server.auth.invalid_credentials`, `server.generic.*`) — never ids or args that reveal account existence (FR-030, CLAUDE.md rule 3).

## 3. Go API

```go
// api/internal/messages
type ID string
func New(id ID, args map[string]any) *httperr.Message  // English rendered from the registry

// api/internal/httperr
type Message struct { ID string; Args map[string]any; Status int; English string }
func (m *Message) Error() string                        // English
func WriteMessage(w http.ResponseWriter, r *http.Request, m *Message)
// Write/WriteCode/classify: sentinel errors map to registry ids; errors.As(*Message) honoured.
```

Registry entries are `{ID, English ICU, Description}`; `i18n-check --sync-server` writes them into `i18n/locales/en/app.json`; CI fails on drift.

## 4. Dashboard

`APIError` gains `messageId?: string` and `args?: Record<string, string | number>`. Display: `intl.formatMessage({id: messageId}, args)` when the id exists in the active catalog, else `detail`. Plain-text responses (older API) keep today's behaviour.

## 5. Operator condition reasons

Translation id `condition.<type>.<reason>` (lower-cased, e.g. `condition.ready.imagepullfailed`), args from `status.conditionArgs[<Type>]`. Fallback: the condition's English `message`.
