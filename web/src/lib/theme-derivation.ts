// Custom color derivation (specs/016-user-theme-customization research.md
// R-03): from a user-chosen accent + surface pair, compute the full set of
// semantic theme tokens. Pure module — no DOM access; the caller injects
// the serialized tokens into <style id="gameplane-custom-theme-vars">.

export const WCAG_AA_TEXT_RATIO = 4.5;
// Accent must be distinguishable from the surface it sits on (WCAG 3:1
// non-text contrast floor); text tones must hit the 4.5:1 AA text floor.
export const MIN_ACCENT_SURFACE_RATIO = 3;

export const CUSTOM_THEME_TOKEN_NAMES = [
  "background",
  "foreground",
  "surface",
  "surface-secondary",
  "surface-tertiary",
  "overlay",
  "accent",
  "accent-foreground",
  "accent-soft",
  "accent-soft-foreground",
  "muted",
  "border",
  "focus",
  "link",
] as const;

export type CustomThemeTokenName = (typeof CUSTOM_THEME_TOKEN_NAMES)[number];

// Semantic token map keyed by CSS variable name (without the "--" prefix);
// every value is a lowercase #rrggbb hex string.
export type CustomThemeTokens = Record<CustomThemeTokenName, string>;

const WHITE = "#ffffff";
const BLACK = "#000000";
const DARK_FOREGROUND = "#f5f5f5";
const LIGHT_FOREGROUND = "#0f172a";

function clamp(v: number, lo: number, hi: number): number {
  return Math.min(hi, Math.max(lo, v));
}

// parseHex accepts exactly #RRGGBB (the format validated by the API's
// ^#([0-9a-fA-F]{6})$ rule) and throws on anything else.
function parseHex(hex: string): { r: number; g: number; b: number } {
  const m = /^#([0-9a-fA-F]{6})$/.exec(hex.trim());
  if (!m || !m[1]) {
    throw new Error(`invalid hex color: ${hex}`);
  }
  const v = parseInt(m[1], 16);
  return { r: (v >> 16) & 0xff, g: (v >> 8) & 0xff, b: v & 0xff };
}

function rgbToHex(r: number, g: number, b: number): string {
  const c = (n: number) => clamp(Math.round(n), 0, 255).toString(16).padStart(2, "0");
  return `#${c(r)}${c(g)}${c(b)}`;
}

export interface Hsl {
  h: number;
  s: number;
  l: number;
}

export function hexToHsl(hex: string): Hsl {
  const { r, g, b } = parseHex(hex);
  const rn = r / 255;
  const gn = g / 255;
  const bn = b / 255;
  const max = Math.max(rn, gn, bn);
  const min = Math.min(rn, gn, bn);
  const l = (max + min) / 2;
  if (max === min) {
    return { h: 0, s: 0, l: Math.round(l * 100) };
  }
  const d = max - min;
  const s = l > 0.5 ? d / (2 - max - min) : d / (max + min);
  let h: number;
  switch (max) {
    case rn:
      h = ((gn - bn) / d + (gn < bn ? 6 : 0)) * 60;
      break;
    case gn:
      h = ((bn - rn) / d + 2) * 60;
      break;
    default:
      h = ((rn - gn) / d + 4) * 60;
      break;
  }
  return { h: Math.round(h), s: Math.round(s * 100), l: Math.round(l * 100) };
}

export function hslToHex(h: number, s: number, l: number): string {
  const hn = ((h % 360) + 360) % 360;
  const sn = clamp(s, 0, 100) / 100;
  const ln = clamp(l, 0, 100) / 100;
  const c = (1 - Math.abs(2 * ln - 1)) * sn;
  const x = c * (1 - Math.abs((hn / 60) % 2 - 1));
  const m = ln - c / 2;
  let r = 0;
  let g = 0;
  let b = 0;
  if (hn < 60) {
    r = c;
    g = x;
  } else if (hn < 120) {
    r = x;
    g = c;
  } else if (hn < 180) {
    g = c;
    b = x;
  } else if (hn < 240) {
    g = x;
    b = c;
  } else if (hn < 300) {
    r = x;
    b = c;
  } else {
    r = c;
    b = x;
  }
  return rgbToHex((r + m) * 255, (g + m) * 255, (b + m) * 255);
}

// WCAG relative luminance (sRGB, piecewise-linear transfer).
export function relativeLuminance(hex: string): number {
  const { r, g, b } = parseHex(hex);
  const f = (n: number) => {
    const c = n / 255;
    return c <= 0.04045 ? c / 12.92 : Math.pow((c + 0.055) / 1.055, 2.4);
  };
  return 0.2126 * f(r) + 0.7152 * f(g) + 0.0722 * f(b);
}

// WCAG contrast ratio between two sRGB hex colors.
export function contrastRatio(a: string, b: string): number {
  const la = relativeLuminance(a);
  const lb = relativeLuminance(b);
  const hi = Math.max(la, lb);
  const lo = Math.min(la, lb);
  return (hi + 0.05) / (lo + 0.05);
}

// Alpha-composite fg over bg in sRGB space (fgFraction of fg).
export function mixHex(fg: string, bg: string, fgFraction: number): string {
  const a = parseHex(fg);
  const b = parseHex(bg);
  const mix = (x: number, y: number) => x * fgFraction + y * (1 - fgFraction);
  return rgbToHex(mix(a.r, b.r), mix(a.g, b.g), mix(a.b, b.b));
}

// Walk the lightness of an (h, s) tone from baseHex toward the mode extreme
// until it reaches the WCAG AA text ratio against baseHex. Returns the
// best-effort extreme when no reachable lightness passes (mid-tone bases) —
// passesContrastGuard reports that case as illegible.
function legibleTone(baseHex: string, h: number, s: number, towardLight: boolean): string {
  const start = hexToHsl(baseHex).l;
  const extreme = towardLight ? 100 : 0;
  const step = towardLight ? 1 : -1;
  let candidate: string;
  for (let l = start; ; l += step) {
    candidate = hslToHex(h, s, l);
    if (contrastRatio(candidate, baseHex) >= WCAG_AA_TEXT_RATIO || l === extreme) {
      return candidate;
    }
  }
}

// deriveCustomThemeTokens computes the semantic token map for a custom
// color base theme from the two user inputs (both #RRGGBB). Dark vs light
// treatment is derived from the surface's HSL lightness (< 50% = dark),
// matching the surface/border conventions of the bundled presets.
export function deriveCustomThemeTokens(accent: string, surface: string): CustomThemeTokens {
  // Normalize inputs so passthrough tokens (accent, surface) are stable
  // lowercase #rrggbb regardless of how the picker formatted them.
  accent = accent.trim().toLowerCase();
  surface = surface.trim().toLowerCase();
  const accentHsl = hexToHsl(accent);
  const surfaceHsl = hexToHsl(surface);
  const dark = surfaceHsl.l < 50;
  const dir = dark ? 1 : -1;

  // Accent foreground: white or black, whichever yields the higher WCAG
  // contrast against the accent (>= 4.5:1 for any usable accent).
  const accentForeground =
    contrastRatio(accent, WHITE) >= contrastRatio(accent, BLACK) ? WHITE : BLACK;

  // Surface layers follow the preset convention: secondary/tertiary sit
  // slightly below the base surface; borders move toward the mode extreme
  // (+10% lightness in dark, -10% in light); overlay is a raised layer.
  const surfaceSecondary = hslToHex(surfaceHsl.h, surfaceHsl.s, surfaceHsl.l - 3);
  const surfaceTertiary = hslToHex(surfaceHsl.h, surfaceHsl.s, surfaceHsl.l - 5);
  const overlay = dark ? hslToHex(surfaceHsl.h, surfaceHsl.s, surfaceHsl.l + 2) : surface;
  const border = hslToHex(surfaceHsl.h, surfaceHsl.s, surfaceHsl.l + 10 * dir);

  // Accent soft: ~15% accent tint over the surface.
  const accentSoft = mixHex(accent, surface, 0.15);
  const accentSoftForeground = legibleTone(accentSoft, accentHsl.h, accentHsl.s, dark);

  // Muted secondary text: intermediate tone between surface and foreground
  // guaranteed >= 4.5:1 against the surface whenever a reachable tone
  // achieves that.
  const muted = legibleTone(surface, surfaceHsl.h, surfaceHsl.s, dark);

  // Focus ring reuses the accent (per the preset tokens); links shift the
  // accent lightness toward the mode extreme so they read as interactive.
  const focus = accent;
  const link = hslToHex(accentHsl.h, accentHsl.s, accentHsl.l + 8 * dir);
  const foreground = dark ? DARK_FOREGROUND : LIGHT_FOREGROUND;

  return {
    background: surface,
    foreground,
    surface,
    "surface-secondary": surfaceSecondary,
    "surface-tertiary": surfaceTertiary,
    overlay,
    accent,
    "accent-foreground": accentForeground,
    "accent-soft": accentSoft,
    "accent-soft-foreground": accentSoftForeground,
    muted,
    border,
    focus,
    link,
  };
}

// passesContrastGuard reports whether an accent + surface pair is legible
// enough to offer as a custom theme: the accent must be distinguishable
// from the surface (>= 3:1), the derived accent foreground must meet the
// 4.5:1 text floor on the accent, and the derived muted tone must meet the
// 4.5:1 text floor on the surface. The UI warns when this returns false
// (e.g. accent identical to the surface).
export function passesContrastGuard(accent: string, surface: string): boolean {
  const tokens = deriveCustomThemeTokens(accent, surface);
  return (
    contrastRatio(accent, surface) >= MIN_ACCENT_SURFACE_RATIO &&
    contrastRatio(tokens.accent, tokens["accent-foreground"]) >= WCAG_AA_TEXT_RATIO &&
    contrastRatio(tokens.muted, surface) >= WCAG_AA_TEXT_RATIO
  );
}

// Selector the modal uses when injecting the derived tokens into
// <style id="gameplane-custom-theme-vars">.
export const CUSTOM_THEME_VARS_SELECTOR = ":root, .dark, .light";

// customThemeTokensToCss serializes a token map into a CSS block for the
// given selector (defaults to the selector shared by both appearance
// modes, so the derived base wins over the active preset tokens).
export function customThemeTokensToCss(
  tokens: CustomThemeTokens,
  selector: string = CUSTOM_THEME_VARS_SELECTOR,
): string {
  const lines = CUSTOM_THEME_TOKEN_NAMES.map((name) => `  --${name}: ${tokens[name]};`);
  return `${selector} {\n${lines.join("\n")}\n}`;
}
