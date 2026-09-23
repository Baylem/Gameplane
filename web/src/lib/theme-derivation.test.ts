import { describe, expect, it } from "vitest";
import {
  type CustomThemeTokens,
  CUSTOM_THEME_VARS_SELECTOR,
  contrastRatio,
  customThemeTokensToCss,
  deriveCustomThemeTokens,
  hexToHsl,
  hslToHex,
  mixHex,
  passesContrastGuard,
  surfaceAppearance,
} from "./theme-derivation";

const DARK_SURFACE = "#121114"; // research.md R-03 default dark surface
const LIGHT_SURFACE = "#F8FAFC"; // research.md R-03 default light surface

describe("deriveCustomThemeTokens — WCAG accent-foreground selection", () => {
  it("picks white for a dark accent that meets 4.5:1 with white", () => {
    const tokens = deriveCustomThemeTokens("#DB2777", DARK_SURFACE);
    expect(tokens["accent-foreground"]).toBe("#ffffff");
    expect(contrastRatio(tokens.accent, tokens["accent-foreground"])).toBeGreaterThanOrEqual(4.5);
  });

  it("picks black for a mid-light accent where white fails 4.5:1", () => {
    const tokens = deriveCustomThemeTokens("#10B981", DARK_SURFACE);
    expect(tokens["accent-foreground"]).toBe("#000000");
    expect(contrastRatio(tokens.accent, tokens["accent-foreground"])).toBeGreaterThanOrEqual(4.5);
  });

  it("picks black for a near-white accent", () => {
    const tokens = deriveCustomThemeTokens("#F8FAFC", DARK_SURFACE);
    expect(tokens["accent-foreground"]).toBe("#000000");
  });

  it("keeps the accent verbatim (normalized to lowercase)", () => {
    const tokens = deriveCustomThemeTokens("#10B981", DARK_SURFACE);
    expect(tokens.accent).toBe("#10b981");
    expect(tokens.focus).toBe("#10b981");
  });
});

describe("deriveCustomThemeTokens — accent-soft tint", () => {
  it("blends 15% accent over the surface", () => {
    expect(mixHex("#ff0000", "#000000", 0.15)).toBe("#260000");
    const tokens = deriveCustomThemeTokens("#10B981", DARK_SURFACE);
    expect(tokens["accent-soft"]).toBe("#122a24");
  });

  it("accent-soft-foreground is legible on the soft tint", () => {
    const tokens = deriveCustomThemeTokens("#10B981", DARK_SURFACE);
    expect(contrastRatio(tokens["accent-soft-foreground"], tokens["accent-soft"])).toBeGreaterThanOrEqual(4.5);
  });
});

describe("deriveCustomThemeTokens — surface layers and borders", () => {
  it("derives dark-mode layers below the surface and a lighter border", () => {
    const tokens = deriveCustomThemeTokens("#10B981", DARK_SURFACE);
    expect(hexToHsl(tokens["surface-secondary"]).l).toBe(hexToHsl(DARK_SURFACE).l - 3);
    expect(hexToHsl(tokens["surface-tertiary"]).l).toBe(hexToHsl(DARK_SURFACE).l - 5);
    expect(hexToHsl(tokens.border).l).toBe(hexToHsl(DARK_SURFACE).l + 10);
    // Overlay is a raised layer above the surface in dark mode.
    expect(hexToHsl(tokens.overlay).l).toBe(hexToHsl(DARK_SURFACE).l + 2);
    expect(tokens.background).toBe(DARK_SURFACE.toLowerCase());
    expect(tokens.surface).toBe(DARK_SURFACE.toLowerCase());
    expect(tokens.foreground).toBe("#f5f5f5");
  });

  it("derives light-mode layers below the surface and a darker border", () => {
    const tokens = deriveCustomThemeTokens("#DB2777", LIGHT_SURFACE);
    expect(hexToHsl(tokens["surface-secondary"]).l).toBe(hexToHsl(LIGHT_SURFACE).l - 3);
    expect(hexToHsl(tokens["surface-tertiary"]).l).toBe(hexToHsl(LIGHT_SURFACE).l - 5);
    expect(hexToHsl(tokens.border).l).toBe(hexToHsl(LIGHT_SURFACE).l - 10);
    // Light overlay reuses the surface, matching the preset convention.
    expect(tokens.overlay).toBe(LIGHT_SURFACE.toLowerCase());
    expect(tokens.foreground).toBe("#0f172a");
  });
});

describe("deriveCustomThemeTokens — muted text", () => {
  it("guarantees >= 4.5:1 against a dark surface", () => {
    const tokens = deriveCustomThemeTokens("#10B981", DARK_SURFACE);
    expect(contrastRatio(tokens.muted, DARK_SURFACE)).toBeGreaterThanOrEqual(4.5);
  });

  it("guarantees >= 4.5:1 against a light surface", () => {
    const tokens = deriveCustomThemeTokens("#DB2777", LIGHT_SURFACE);
    expect(contrastRatio(tokens.muted, LIGHT_SURFACE)).toBeGreaterThanOrEqual(4.5);
  });

  it("keeps the muted tone between the surface and the foreground extreme", () => {
    const tokens = deriveCustomThemeTokens("#10B981", DARK_SURFACE);
    const mutedL = hexToHsl(tokens.muted).l;
    expect(mutedL).toBeGreaterThan(hexToHsl(DARK_SURFACE).l);
    expect(mutedL).toBeLessThanOrEqual(100);
  });
});

describe("deriveCustomThemeTokens — focus and link", () => {
  it("reuses the accent as the focus ring", () => {
    const tokens = deriveCustomThemeTokens("#10B981", DARK_SURFACE);
    expect(tokens.focus).toBe(tokens.accent);
  });

  it("lightens the link in dark mode and darkens it in light mode", () => {
    const dark = deriveCustomThemeTokens("#10B981", DARK_SURFACE);
    expect(hexToHsl(dark.link).l).toBe(hexToHsl(dark.accent).l + 8);
    const light = deriveCustomThemeTokens("#DB2777", LIGHT_SURFACE);
    expect(hexToHsl(light.link).l).toBe(hexToHsl(light.accent).l - 8);
  });
});

describe("passesContrastGuard — extreme pairs", () => {
  it("fails when the accent is identical to the surface", () => {
    expect(passesContrastGuard("#121114", "#121114")).toBe(false);
    expect(passesContrastGuard("#F8FAFC", "#F8FAFC")).toBe(false);
  });

  it("still derives tokens for an illegible pair (best effort)", () => {
    const tokens = deriveCustomThemeTokens("#121114", "#121114");
    expect(tokens.accent).toBe("#121114");
    expect(tokens["accent-foreground"]).toBe("#ffffff");
  });

  it("passes for a well-separated pair", () => {
    expect(passesContrastGuard("#10B981", "#121114")).toBe(true);
    expect(passesContrastGuard("#DB2777", "#F8FAFC")).toBe(true);
  });

  it("fails for an accent too close to the surface even when distinct", () => {
    expect(passesContrastGuard("#151316", "#121114")).toBe(false);
  });
});

describe("deriveCustomThemeTokens — input validation", () => {
  it("throws on malformed hex", () => {
    expect(() => deriveCustomThemeTokens("red", DARK_SURFACE)).toThrow(/invalid hex color/);
    expect(() => deriveCustomThemeTokens("#10B981", "#12345")).toThrow(/invalid hex color/);
  });
});

describe("customThemeTokensToCss", () => {
  it("serializes every token into a :root, .dark, .light block", () => {
    const tokens: CustomThemeTokens = deriveCustomThemeTokens("#10B981", DARK_SURFACE);
    const css = customThemeTokensToCss(tokens);
    expect(css.startsWith(`${CUSTOM_THEME_VARS_SELECTOR} {\n`)).toBe(true);
    expect(css.endsWith("\n}")).toBe(true);
    expect(css).toContain("  --accent: #10b981;");
    expect(css).toContain("  --accent-foreground: #000000;");
    expect(css).toContain("  --surface: #121114;");
    expect(css).toContain("  --muted: ");
    expect(css).toContain("  --border: ");
    expect(css).toContain("  --focus: #10b981;");
    expect(css).toContain("  --link: ");
  });

  it("targets only the custom_colors root", () => {
    const html = document.documentElement;
    html.setAttribute("data-theme", "dark");
    html.setAttribute("data-theme-type", "custom_colors");
    expect(html.matches(CUSTOM_THEME_VARS_SELECTOR)).toBe(true);
    html.setAttribute("data-theme-type", "preset");
    expect(html.matches(CUSTOM_THEME_VARS_SELECTOR)).toBe(false);
    html.removeAttribute("data-theme-type");
    html.removeAttribute("data-theme");
  });

  it("accepts a custom selector", () => {
    const tokens = deriveCustomThemeTokens("#10B981", DARK_SURFACE);
    const css = customThemeTokensToCss(tokens, ".preview-scope");
    expect(css.startsWith(".preview-scope {\n")).toBe(true);
  });
});

describe("hslToHex — hue segment coverage", () => {
  it("computes the [60, 120) hue segment (r falling, g at max)", () => {
    // At hue 90 (yellow-green), R should fall from the c/x formula (x at
    // this hue) and G should sit at the chroma peak — the branch at
    // theme-derivation.ts:107-108, not exercised by any preset/custom
    // fixture hue elsewhere in this suite.
    expect(hslToHex(90, 100, 50)).toBe("#80ff00");
  });

  it("round-trips a [60, 120) hue through hexToHsl", () => {
    const hex = hslToHex(90, 100, 50);
    const hsl = hexToHsl(hex);
    expect(hsl.h).toBe(90);
  });
});

describe("surfaceAppearance — D4 mode resolution", () => {
  it("classifies the four SURFACE_TONES swatches correctly", () => {
    expect(surfaceAppearance("#1E293B")).toBe("dark");  // Dark Slate
    expect(surfaceAppearance("#0F172A")).toBe("dark");  // Midnight
    expect(surfaceAppearance("#171717")).toBe("dark");  // Charcoal
    expect(surfaceAppearance("#F8FAFC")).toBe("light"); // Crisp Light
  });

  it("classifies the #808080 edge case as light (just above the 0.179 luminance threshold)", () => {
    expect(surfaceAppearance("#808080")).toBe("light");
  });

  it("classifies pure black and pure white at the extremes", () => {
    expect(surfaceAppearance("#000000")).toBe("dark");
    expect(surfaceAppearance("#ffffff")).toBe("light");
  });
});
