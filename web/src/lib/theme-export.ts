// Theme export/import (FR-014, contracts/theme-export.md). Export is a
// client-side re-serialization of the loaded UserThemePreferences into a
// versioned "gameplane-theme" v1 JSON document; import validates the
// document shape client-side, then maps to the PUT payload (the server
// re-validates and applies FR-013 sanitization as the authoritative gate).

import type { UserPreferencesUpdate } from "@/lib/endpoints";
import type {
  AppearanceMode,
  CustomColorConfig,
  ThemeExport,
  ThemePresetId,
  ThemeType,
  UserThemePreferences,
} from "@/types";

import { MAX_CUSTOM_CSS_LEN } from "@/lib/theme-sanitize";

export const THEME_EXPORT_FORMAT = "gameplane-theme";
export const THEME_EXPORT_VERSION = 1;

const HEX_RE = /^#([0-9a-fA-F]{6})$/;

const THEME_TYPES: readonly ThemeType[] = ["preset", "custom_colors"];
const PRESET_IDS: readonly ThemePresetId[] = ["pink", "legacy"];
const APPEARANCE_MODES: readonly AppearanceMode[] = ["light", "dark", "system"];

function isRecord(v: unknown): v is Record<string, unknown> {
  return typeof v === "object" && v !== null && !Array.isArray(v);
}

function enumValue<T extends string>(allowed: readonly T[], v: unknown): T | null {
  return typeof v === "string" && (allowed as readonly string[]).includes(v) ? (v as T) : null;
}

// utf8Len mirrors the server's byte-length accounting for the CSS cap.
function utf8Len(s: string): number {
  return new TextEncoder().encode(s).length;
}

// buildThemeExport builds the portable document from the loaded
// preferences. updatedAt is intentionally excluded (instance-local
// metadata); retained-but-inactive customs (customColors while on a preset,
// customCss while the overlay is disabled) are included so a full setup
// survives the transfer.
export function buildThemeExport(prefs: UserThemePreferences): ThemeExport {
  return {
    format: THEME_EXPORT_FORMAT,
    version: THEME_EXPORT_VERSION,
    preferences: {
      themeType: prefs.themeType,
      presetId: prefs.presetId,
      appearanceMode: prefs.appearanceMode,
      customColors: prefs.customColors ?? null,
      customCssEnabled: prefs.customCssEnabled,
      customCss: prefs.customCss ?? null,
    },
  };
}

export type ValidateThemeExportResult =
  | { ok: true; export: ThemeExport }
  | { ok: false; error: string };

// validateThemeExport parses and validates a theme export document. Returns
// a specific, field-naming error for every contract §2/§4 rejection reason;
// documents from a newer exporter (version > 1) get an upgrade prompt.
export function validateThemeExport(raw: string): ValidateThemeExportResult {
  let doc: unknown;
  try {
    doc = JSON.parse(raw);
  } catch {
    return { ok: false, error: "invalid theme export: not valid JSON" };
  }
  if (!isRecord(doc)) {
    return { ok: false, error: "invalid theme export: expected a JSON object" };
  }
  if (doc.format !== THEME_EXPORT_FORMAT) {
    return {
      ok: false,
      error: `invalid theme export: format must be "${THEME_EXPORT_FORMAT}"`,
    };
  }
  if (doc.version === undefined) {
    return { ok: false, error: 'invalid theme export: missing required field "version"' };
  }
  if (typeof doc.version !== "number") {
    return { ok: false, error: "invalid theme export: version must be a number" };
  }
  if (doc.version > THEME_EXPORT_VERSION) {
    return {
      ok: false,
      error:
        `invalid theme export: unsupported version ${doc.version} — this theme was exported ` +
        "by a newer version of Gameplane; upgrade this instance before importing",
    };
  }
  if (doc.version !== THEME_EXPORT_VERSION) {
    return { ok: false, error: `invalid theme export: version must be ${THEME_EXPORT_VERSION}` };
  }
  if (doc.preferences === undefined || doc.preferences === null) {
    return {
      ok: false,
      error: 'invalid theme export: missing required field "preferences"',
    };
  }
  if (!isRecord(doc.preferences)) {
    return { ok: false, error: "invalid theme export: preferences must be an object" };
  }
  const prefs = doc.preferences;

  const themeType = enumValue(THEME_TYPES, prefs.themeType);
  if (themeType === null) {
    if (prefs.themeType === undefined) {
      return {
        ok: false,
        error: 'invalid theme export: missing required field "preferences.themeType"',
      };
    }
    return {
      ok: false,
      error: 'invalid theme export: preferences.themeType must be "preset" or "custom_colors"',
    };
  }
  const presetId = enumValue(PRESET_IDS, prefs.presetId);
  if (presetId === null) {
    if (prefs.presetId === undefined) {
      return {
        ok: false,
        error: 'invalid theme export: missing required field "preferences.presetId"',
      };
    }
    return {
      ok: false,
      error: 'invalid theme export: preferences.presetId must be "pink" or "legacy"',
    };
  }
  const appearanceMode = enumValue(APPEARANCE_MODES, prefs.appearanceMode);
  if (appearanceMode === null) {
    if (prefs.appearanceMode === undefined) {
      return {
        ok: false,
        error: 'invalid theme export: missing required field "preferences.appearanceMode"',
      };
    }
    return {
      ok: false,
      error: 'invalid theme export: preferences.appearanceMode must be "light", "dark", or "system"',
    };
  }
  if (prefs.customCssEnabled === undefined) {
    return {
      ok: false,
      error: 'invalid theme export: missing required field "preferences.customCssEnabled"',
    };
  }
  if (typeof prefs.customCssEnabled !== "boolean") {
    return {
      ok: false,
      error: "invalid theme export: preferences.customCssEnabled must be a boolean",
    };
  }

  let customColors: CustomColorConfig | null = null;
  if (prefs.customColors !== undefined && prefs.customColors !== null) {
    if (!isRecord(prefs.customColors)) {
      return {
        ok: false,
        error: "invalid theme export: preferences.customColors must be an object or null",
      };
    }
    const cc = prefs.customColors;
    if (typeof cc.accent !== "string" || !HEX_RE.test(cc.accent)) {
      return {
        ok: false,
        error: "invalid theme export: preferences.customColors.accent must be a #RRGGBB hex color",
      };
    }
    if (typeof cc.surface !== "string" || !HEX_RE.test(cc.surface)) {
      return {
        ok: false,
        error: "invalid theme export: preferences.customColors.surface must be a #RRGGBB hex color",
      };
    }
    customColors = { accent: cc.accent, surface: cc.surface };
  }

  let customCss: string | null = null;
  if (prefs.customCss !== undefined && prefs.customCss !== null) {
    if (typeof prefs.customCss !== "string") {
      return {
        ok: false,
        error: "invalid theme export: preferences.customCss must be a string or null",
      };
    }
    if (utf8Len(prefs.customCss) > MAX_CUSTOM_CSS_LEN) {
      return {
        ok: false,
        error: `invalid theme export: preferences.customCss exceeds the maximum length of ${MAX_CUSTOM_CSS_LEN} characters`,
      };
    }
    customCss = prefs.customCss;
  }

  return {
    ok: true,
    export: {
      format: THEME_EXPORT_FORMAT,
      version: THEME_EXPORT_VERSION,
      preferences: {
        themeType,
        presetId,
        appearanceMode,
        customColors,
        customCssEnabled: prefs.customCssEnabled,
        customCss,
      },
    },
  };
}

// themeExportToUpdate maps a validated export document to the PUT payload.
// Per the endpoints.ts retention convention, absent/null customs are
// omitted (never sent as null) so the server keeps stored values; FR-013
// sanitization of customCss remains the server's job on the resulting PUT.
export function themeExportToUpdate(exportDoc: ThemeExport): UserPreferencesUpdate {
  const prefs = exportDoc.preferences;
  const update: UserPreferencesUpdate = {
    themeType: prefs.themeType,
    presetId: prefs.presetId,
    appearanceMode: prefs.appearanceMode,
    customCssEnabled: prefs.customCssEnabled,
  };
  if (prefs.customColors != null) {
    update.customColors = prefs.customColors;
  }
  if (prefs.customCss != null) {
    update.customCss = prefs.customCss;
  }
  return update;
}
