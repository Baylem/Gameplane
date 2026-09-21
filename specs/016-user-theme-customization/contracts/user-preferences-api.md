# Contract: User Preferences API

**Feature**: `016-user-theme-customization`  
**Binding Modules**: `api/internal/handlers/users.go`, `api/internal/db/`, `web/src/lib/endpoints.ts`  
**Status**: Binding (revised 2026-09-21 after clarification session 2026-09-21)  

---

## 1. Endpoints

### 1.1 `GET /api/v1/users/me/preferences`

Retrieves the authenticated user's current styling and theme preferences.

- **Authentication**: Required (valid session cookie).
- **Permissions**: None (any authenticated user can read their own preferences).

#### Response: `200 OK`

```json
{
  "themeType": "preset",
  "presetId": "legacy",
  "appearanceMode": "system",
  "customColors": {
    "accent": "#3B82F6",
    "surface": "#18181B"
  },
  "customCssEnabled": true,
  "customCss": "/* optional user styles */",
  "updatedAt": "2026-09-19T14:32:00Z"
}
```

*Note: If no record exists in `user_preferences` for this user (e.g. newly created user), the API returns the default configuration:*

```json
{
  "themeType": "preset",
  "presetId": "pink",
  "appearanceMode": "system",
  "customColors": null,
  "customCssEnabled": false,
  "customCss": null,
  "updatedAt": "2026-09-19T14:32:00Z"
}
```

#### Error Responses

- `401 Unauthorized`: When session cookie is missing or invalid.

---

### 1.2 `PUT /api/v1/users/me/preferences`

Updates the authenticated user's styling preferences.

- **Authentication**: Required (valid session cookie + CSRF header).
- **Permissions**: None (users manage their own styling preferences).

#### Request Body

```json
{
  "themeType": "custom_colors",
  "presetId": "pink",
  "appearanceMode": "dark",
  "customColors": {
    "accent": "#10B981",
    "surface": "#121114"
  },
  "customCssEnabled": true,
  "customCss": ".dashboard-card { border-radius: 12px; }"
}
```

#### Validation Rules

- `themeType`: Required. Must be one of `"preset"`, `"custom_colors"` (base mode only; custom CSS is controlled by `customCssEnabled`).
- `presetId`: Required. Must be one of `"pink"`, `"legacy"`.
- `appearanceMode`: Required. Must be one of `"light"`, `"dark"`, `"system"`.
- `customColors`: Optional object. If provided, `accent` and `surface` must match `^#([0-9a-fA-F]{6})$`.
- `customCssEnabled`: Required boolean. Toggling it never deletes `customCss` (see Retention).
- `customCss`: Optional string, sanitized per FR-013. Rejected (`400`) when it:
  - exceeds 32,768 characters (32 KB);
  - contains an `@import` rule;
  - contains an external `url()` reference (absolute `http://`/`https://` or protocol-relative `//host/...`), including remote font loads — inline `data:` URIs are permitted;
  - has unparseable syntax (unbalanced braces / invalid declarations);
  - contains `<style`, `</style`, `<script`, or `</script`.

  The `400` response message identifies the offending rule so the editor can surface it.

#### Retention Semantics (FR-012)

A `PUT` that changes `themeType`/`presetId` or toggles `customCssEnabled` MUST NOT null `customColors` or `customCss` on the server. Stored custom settings are deleted only via the reset endpoint (§1.3). Clients therefore send ordinary updates without re-submitting unchanged custom values; omitted optional fields keep their stored values.

#### Response: `200 OK`

Returns the updated `UserThemePreferences` object.

#### Error Responses

- `400 Bad Request`: Validation failure (invalid color format, invalid enum, sanitization rejection, or payload exceeding size limit).
  ```json
  { "error": "customCss rejected: external url() reference 'https://fonts.example.com/x.css' is not allowed" }
  ```
- `401 Unauthorized`: Missing or invalid authentication session.
- `403 Forbidden`: CSRF token mismatch or expired.

---

### 1.3 `POST /api/v1/users/me/preferences/reset`

Implements the single explicit "Reset to Defaults" action (FR-012) — the **only** operation that deletes stored custom settings.

- **Authentication**: Required (valid session cookie + CSRF header).
- **Effect**: Sets `custom_accent = NULL`, `custom_surface = NULL`, `custom_css = NULL`, `custom_css_enabled = 0`, `theme_type = 'preset'`. `preset_id` and `appearance_mode` are set to the values supplied in the request body (the preset the user confirmed), defaulting to the current `preset_id` / `appearance_mode` when omitted.

#### Request Body (optional)

```json
{ "presetId": "pink", "appearanceMode": "system" }
```

#### Response: `200 OK`

Returns the resulting `UserThemePreferences` object (customs all `null`, `customCssEnabled: false`, `themeType: "preset"`).

#### Error Responses

- `400 Bad Request`: Invalid `presetId` or `appearanceMode` in the body.
- `401 Unauthorized` / `403 Forbidden`: As in §1.2.

---

### 1.4 Extension to `GET /api/v1/users/me`

To avoid an extra network round-trip on dashboard boot, `GET /api/v1/users/me` is extended to include the preferences object directly in the response payload:

```json
{
  "id": 42,
  "username": "alex",
  "displayName": "Alex Rivera",
  "email": "alex@example.com",
  "role": "operator",
  "provider": "local",
  "createdAt": "2026-08-15T10:00:00Z",
  "permissions": { "...": "..." },
  "preferences": {
    "themeType": "preset",
    "presetId": "legacy",
    "appearanceMode": "system",
    "customColors": null,
    "customCssEnabled": false,
    "customCss": null,
    "updatedAt": "2026-09-19T14:32:00Z"
  }
}
```

---

## 2. Export / Import Note (FR-014)

Theme export and import require **no dedicated endpoints**:
- **Export** is generated client-side from this payload into the versioned `ThemeExport` document defined in `contracts/theme-export.md`.
- **Import** is a client-validated `PUT /users/me/preferences` call; the validation and FR-013 sanitization in §1.2 are the authoritative enforcement gate, so an import cannot smuggle in rejected CSS.
