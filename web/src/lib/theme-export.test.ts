import { describe, expect, it } from "vitest";
import type { ThemeExport, UserThemePreferences } from "@/types";
import {
  buildThemeExport,
  themeExportToUpdate,
  validateThemeExport,
  type ValidateThemeExportResult,
} from "./theme-export";

const UPDATED_AT = "2026-09-21T12:00:00Z";

const fullPrefs: UserThemePreferences = {
  themeType: "custom_colors",
  presetId: "legacy",
  appearanceMode: "dark",
  customColors: { accent: "#10B981", surface: "#121114" },
  customCssEnabled: true,
  customCss: ".dashboard-card { border-radius: 12px; }",
  updatedAt: UPDATED_AT,
};

function rawDoc(
  prefsOverrides: Record<string, unknown> = {},
  topOverrides: Record<string, unknown> = {},
): string {
  return JSON.stringify({
    format: "gameplane-theme",
    version: 1,
    preferences: {
      themeType: "preset",
      presetId: "pink",
      appearanceMode: "system",
      customCssEnabled: false,
      ...prefsOverrides,
    },
    ...topOverrides,
  });
}

function expectError(result: ValidateThemeExportResult, pattern: RegExp): string {
  if (result.ok) {
    throw new Error(`expected validation failure, got ok: ${JSON.stringify(result.export)}`);
  }
  expect(result.error).toMatch(pattern);
  return result.error;
}

describe("buildThemeExport", () => {
  it("builds a gameplane-theme v1 document from loaded preferences", () => {
    const doc = buildThemeExport(fullPrefs);
    expect(doc).toEqual({
      format: "gameplane-theme",
      version: 1,
      preferences: {
        themeType: "custom_colors",
        presetId: "legacy",
        appearanceMode: "dark",
        customColors: { accent: "#10B981", surface: "#121114" },
        customCssEnabled: true,
        customCss: ".dashboard-card { border-radius: 12px; }",
      },
    });
  });

  it("omits updatedAt (instance-local metadata)", () => {
    const doc = buildThemeExport(fullPrefs);
    expect(doc.preferences).not.toHaveProperty("updatedAt");
    expect(JSON.stringify(doc)).not.toContain(UPDATED_AT);
  });

  it("includes retained-but-inactive customs so a full setup survives transfer", () => {
    const prefs: UserThemePreferences = {
      themeType: "preset",
      presetId: "pink",
      appearanceMode: "system",
      customColors: { accent: "#3B82F6", surface: "#18181B" },
      customCssEnabled: false, // overlay disabled, stylesheet retained
      customCss: ".card { border-radius: 8px; }",
    };
    const doc = buildThemeExport(prefs);
    expect(doc.preferences.customColors).toEqual({ accent: "#3B82F6", surface: "#18181B" });
    expect(doc.preferences.customCssEnabled).toBe(false);
    expect(doc.preferences.customCss).toBe(".card { border-radius: 8px; }");
  });

  it("normalizes absent customs to explicit null", () => {
    const prefs: UserThemePreferences = {
      themeType: "preset",
      presetId: "pink",
      appearanceMode: "system",
      customCssEnabled: false,
    };
    const doc = buildThemeExport(prefs);
    expect(doc.preferences.customColors).toBeNull();
    expect(doc.preferences.customCss).toBeNull();
  });
});

describe("validateThemeExport — round-trip", () => {
  it("accepts a document built by buildThemeExport (serialize → validate)", () => {
    const doc = buildThemeExport(fullPrefs);
    const result = validateThemeExport(JSON.stringify(doc, null, 2));
    expect(result.ok).toBe(true);
    if (result.ok) {
      expect(result.export).toEqual(doc);
    }
  });

  it("accepts a minimal document and defaults customs to null", () => {
    const result = validateThemeExport(rawDoc());
    expect(result.ok).toBe(true);
    if (result.ok) {
      expect(result.export.preferences.customColors).toBeNull();
      expect(result.export.preferences.customCss).toBeNull();
    }
  });
});

describe("validateThemeExport — JSON and envelope", () => {
  it("rejects unparseable JSON", () => {
    expectError(validateThemeExport("{not json"), /not valid JSON/);
  });

  it("rejects non-object JSON", () => {
    expectError(validateThemeExport("[]"), /expected a JSON object/);
    expectError(validateThemeExport("null"), /expected a JSON object/);
  });

  it("rejects a wrong or missing format", () => {
    expectError(validateThemeExport(rawDoc({}, { format: "other-theme" })), /format must be "gameplane-theme"/);
    expectError(
      validateThemeExport(JSON.stringify({ version: 1, preferences: {} })),
      /format must be "gameplane-theme"/,
    );
  });
});

describe("validateThemeExport — version", () => {
  it("rejects higher versions with an upgrade prompt", () => {
    const message = expectError(validateThemeExport(rawDoc({}, { version: 2 })), /unsupported version 2/);
    expect(message).toMatch(/upgrade/i);
  });

  it("rejects a non-numeric version", () => {
    expectError(validateThemeExport(rawDoc({}, { version: "1" })), /version must be a number/);
  });

  it("rejects a version below 1", () => {
    expectError(validateThemeExport(rawDoc({}, { version: 0 })), /version must be 1/);
  });

  it("rejects a missing version", () => {
    const obj = JSON.parse(rawDoc()) as Record<string, unknown>;
    delete obj.version;
    expectError(validateThemeExport(JSON.stringify(obj)), /missing required field "version"/);
  });
});

describe("validateThemeExport — required fields and enums", () => {
  it("rejects missing preferences", () => {
    const obj = JSON.parse(rawDoc()) as Record<string, unknown>;
    delete obj.preferences;
    expectError(validateThemeExport(JSON.stringify(obj)), /missing required field "preferences"/);
    expectError(validateThemeExport(rawDoc({}, { preferences: "nope" })), /preferences must be an object/);
  });

  it("rejects a missing or invalid themeType", () => {
    expectError(validateThemeExport(rawDoc({ themeType: "gradient" })), /preferences\.themeType must be "preset" or "custom_colors"/);
    const obj = JSON.parse(rawDoc()) as { preferences: Record<string, unknown> };
    delete obj.preferences.themeType;
    expectError(validateThemeExport(JSON.stringify(obj)), /missing required field "preferences\.themeType"/);
  });

  it("rejects a missing or invalid presetId", () => {
    expectError(validateThemeExport(rawDoc({ presetId: "neon" })), /preferences\.presetId must be "pink" or "legacy"/);
    const obj = JSON.parse(rawDoc()) as { preferences: Record<string, unknown> };
    delete obj.preferences.presetId;
    expectError(validateThemeExport(JSON.stringify(obj)), /missing required field "preferences\.presetId"/);
  });

  it("rejects a missing or invalid appearanceMode", () => {
    expectError(
      validateThemeExport(rawDoc({ appearanceMode: "auto" })),
      /preferences\.appearanceMode must be "light", "dark", or "system"/,
    );
    const obj = JSON.parse(rawDoc()) as { preferences: Record<string, unknown> };
    delete obj.preferences.appearanceMode;
    expectError(validateThemeExport(JSON.stringify(obj)), /missing required field "preferences\.appearanceMode"/);
  });

  it("rejects a missing or non-boolean customCssEnabled", () => {
    expectError(validateThemeExport(rawDoc({ customCssEnabled: "yes" })), /preferences\.customCssEnabled must be a boolean/);
    const obj = JSON.parse(rawDoc()) as { preferences: Record<string, unknown> };
    delete obj.preferences.customCssEnabled;
    expectError(validateThemeExport(JSON.stringify(obj)), /missing required field "preferences\.customCssEnabled"/);
  });
});

describe("validateThemeExport — customColors and customCss", () => {
  it("accepts uppercase hex colors", () => {
    const result = validateThemeExport(
      rawDoc({ customColors: { accent: "#10B981", surface: "#121114" } }),
    );
    expect(result.ok).toBe(true);
    if (result.ok) {
      expect(result.export.preferences.customColors).toEqual({ accent: "#10B981", surface: "#121114" });
    }
  });

  it.each([
    ["non-hex", "#XYZ"],
    ["short", "#12345"],
    ["missing hash", "3B82F6"],
    ["long", "#10B9810"],
  ])("rejects a %s accent", (_label, accent) => {
    expectError(
      validateThemeExport(rawDoc({ customColors: { accent, surface: "#18181B" } })),
      /preferences\.customColors\.accent must be a #RRGGBB hex color/,
    );
  });

  it("rejects a bad surface hex, naming surface", () => {
    expectError(
      validateThemeExport(rawDoc({ customColors: { accent: "#3B82F6", surface: "#18181" } })),
      /preferences\.customColors\.surface must be a #RRGGBB hex color/,
    );
  });

  it("rejects a non-object customColors", () => {
    expectError(validateThemeExport(rawDoc({ customColors: "red" })), /preferences\.customColors must be an object or null/);
  });

  it("accepts customCss exactly at the 32,768 cap", () => {
    const result = validateThemeExport(rawDoc({ customCss: "a".repeat(32768) }));
    expect(result.ok).toBe(true);
  });

  it("rejects customCss over the cap", () => {
    expectError(
      validateThemeExport(rawDoc({ customCss: "a".repeat(32769) })),
      /preferences\.customCss exceeds the maximum length of 32768 characters/,
    );
  });

  it("rejects a non-string customCss", () => {
    expectError(validateThemeExport(rawDoc({ customCss: 42 })), /preferences\.customCss must be a string or null/);
  });
});

describe("themeExportToUpdate", () => {
  it("maps a full document to the PUT payload", () => {
    const doc = buildThemeExport(fullPrefs);
    expect(themeExportToUpdate(doc)).toEqual({
      themeType: "custom_colors",
      presetId: "legacy",
      appearanceMode: "dark",
      customColors: { accent: "#10B981", surface: "#121114" },
      customCssEnabled: true,
      customCss: ".dashboard-card { border-radius: 12px; }",
    });
  });

  it("carries customCssEnabled from the document even when false", () => {
    const doc: ThemeExport = {
      format: "gameplane-theme",
      version: 1,
      preferences: {
        themeType: "preset",
        presetId: "pink",
        appearanceMode: "system",
        customCssEnabled: false,
        customCss: ".card { border-radius: 8px; }",
      },
    };
    const update = themeExportToUpdate(doc);
    expect(update.customCssEnabled).toBe(false);
    // Retained stylesheet still transfers with the document.
    expect(update.customCss).toBe(".card { border-radius: 8px; }");
  });

  it("omits absent customs from the payload (PUT retention: omit, never send null)", () => {
    const result = validateThemeExport(rawDoc());
    expect(result.ok).toBe(true);
    if (result.ok) {
      const update = themeExportToUpdate(result.export);
      expect(update).toEqual({
        themeType: "preset",
        presetId: "pink",
        appearanceMode: "system",
        customCssEnabled: false,
      });
      expect(update).not.toHaveProperty("customColors");
      expect(update).not.toHaveProperty("customCss");
    }
  });
});
