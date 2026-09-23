/// <reference types="node" />
import { describe, it, expect, beforeAll } from "vitest";
import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { dirname, join } from "node:path";

// vitest's CSS plugin intercepts `?raw` imports of stylesheets and returns an
// empty string when `test.css` is unset, so read the file directly instead.
const cssPath = join(dirname(fileURLToPath(import.meta.url)), "..", "styles", "globals.css");
const cssText = readFileSync(cssPath, "utf8");

// Note: jsdom does not compute custom properties reliably, so this test
// parses the stylesheet source directly instead of using getComputedStyle().

/**
 * Parse `--name: value;` custom property declarations out of a declaration
 * block's source text, stripping trailing comments, e.g.
 * { "--accent": "oklch(69.11% 0.1944 44.01)" }.
 */
function parseDeclarations(declarationText: string): Record<string, string> {
  const tokens: Record<string, string> = {};

  // Match each property: --name: value; (dropping a trailing /* comment */)
  const propRegex = /--([\w-]+):\s*([^;]+);/g;
  let propMatch;
  while ((propMatch = propRegex.exec(declarationText)) !== null) {
    const name = `--${propMatch[1]}`;
    const value = propMatch[2].replace(/\/\*.*?\*\//g, "").trim();
    tokens[name] = value;
  }

  return tokens;
}

/**
 * Parse CSS custom property declarations out of the first `:root { ... }`
 * block, or the `.dark { ... }` block, in a stylesheet's source text.
 * Returns a map of property names to their raw declared values (trailing
 * comments stripped), e.g. { "--accent": "oklch(69.11% 0.1944 44.01)" }.
 */
function parseTokens(cssSource: string, selectorPrefix: ":root" | ".dark"): Record<string, string> {
  // Match the first block whose selector list contains the given prefix
  // (globals.css lists several selectors sharing one declaration block,
  // e.g. `:root,\n.light,\n.default, ... {`).
  const regex =
    selectorPrefix === ".dark"
      ? /\.dark[^{]*\{([^}]+)\}/
      : /:root[^{]*\{([^}]+)\}/;
  const match = cssSource.match(regex);

  if (!match) {
    const preview = cssSource.slice(0, 120);
    throw new Error(
      `Could not find ${selectorPrefix} block in CSS (cssSource.length=${cssSource.length}, first 120 chars: ${JSON.stringify(preview)})`,
    );
  }

  return parseDeclarations(match[1]);
}

/**
 * Parse CSS custom property declarations out of the block whose selector
 * list starts with the given selector, e.g.
 * `.dark[data-theme-preset="legacy"]` for the Legacy preset tokens.
 */
function parseBlock(cssSource: string, blockSelector: string): Record<string, string> {
  const escaped = blockSelector.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
  const match = cssSource.match(new RegExp(escaped + "[^{]*\\{([^}]+)\\}"));

  if (!match) {
    throw new Error(`Could not find block for selector ${blockSelector} in CSS`);
  }

  return parseDeclarations(match[1]);
}

describe("theme tokens", () => {
  let lightTokens: Record<string, string>;
  let darkTokens: Record<string, string>;

  beforeAll(() => {
    lightTokens = parseTokens(cssText, ":root");
    darkTokens = parseTokens(cssText, ".dark");
  });

  describe("light mode", () => {
    // Contract values from specs/done_014-heroui-web-rebuild/contracts/theme-tokens.md
    const expected: Record<string, string> = {
      "--accent": "oklch(59.16% 0.2180 0.58)",
      "--surface": "oklch(98.34% 0.0100 345.41)",
      "--background": "oklch(100.00% 0.0000 89.88)",
      "--foreground": "oklch(21.67% 0.0502 347.62)",
      "--border": "oklch(86.09% 0.0556 350.54)",
      "--muted": "oklch(47.33% 0.0441 343.67)",
      "--accent-soft": "oklch(100.00% 0.0000 89.88)",
      "--surface-secondary": "oklch(92.30% 0.0335 349.09)",
    };

    for (const [name, value] of Object.entries(expected)) {
      it(`${name} should be ${value}`, () => {
        expect(lightTokens[name]).toBe(value);
      });
    }
  });

  describe("dark mode", () => {
    // Contract values from specs/done_014-heroui-web-rebuild/contracts/theme-tokens.md
    const expected: Record<string, string> = {
      "--accent": "oklch(69.50% 0.2229 355.31)",
      "--surface": "oklch(22.27% 0.0119 300.63)",
      "--background": "oklch(18.02% 0.0063 300.93)",
      "--foreground": "oklch(96.68% 0.0057 308.40)",
      "--border": "oklch(28.79% 0.0167 300.56)",
      "--muted": "oklch(68.92% 0.0213 305.13)",
      "--accent-soft": "oklch(24.65% 0.0537 348.54)",
      "--surface-secondary": "oklch(20.03% 0.0103 303.61)",
    };

    for (const [name, value] of Object.entries(expected)) {
      it(`${name} should be ${value}`, () => {
        expect(darkTokens[name]).toBe(value);
      });
    }
  });

  describe("sleep is an explicit value for Asleep status (split from focus)", () => {
    it("light mode: --sleep has the explicit oklch value for violet Asleep color", () => {
      expect(lightTokens["--sleep"]).toBe("oklch(54.13% 0.2466 293.01)");
    });

    it("dark mode: --sleep has the explicit oklch value for violet Asleep color", () => {
      expect(darkTokens["--sleep"]).toBe("oklch(70.90% 0.1592 293.54)");
    });
  });

  describe("focus and link are explicit or aliased values", () => {
    it("light mode: --focus uses --accent (focus ring per OD-16), --link is explicit", () => {
      expect(lightTokens["--focus"]).toBe("var(--accent)");
      expect(lightTokens["--link"]).toBe("oklch(54.61% 0.2152 262.88)");
    });

    it("dark mode: --focus uses --accent (focus ring per OD-16), --link is explicit", () => {
      expect(darkTokens["--focus"]).toBe("var(--accent)");
      expect(darkTokens["--link"]).toBe("oklch(76.21% 0.1231 256.39)");
    });
  });

  describe("chart-stat is an explicit value (OD-9)", () => {
    it("light mode: --chart-stat has the explicit oklch value for #8B5CF6", () => {
      expect(lightTokens["--chart-stat"]).toBe("oklch(60.56% 0.2189 292.72)");
    });

    it("dark mode: --chart-stat has the explicit oklch value for #8B5CF6", () => {
      expect(darkTokens["--chart-stat"]).toBe("oklch(60.56% 0.2189 292.72)");
    });

    it("light and dark chart-stat values match (single-value token, no theme split)", () => {
      expect(lightTokens["--chart-stat"]).toBe(darkTokens["--chart-stat"]);
    });
  });

  describe("legacy preset tokens (contracts/theme-tokens-v2.md §2, research.md R-02)", () => {
    let legacyDarkTokens: Record<string, string>;
    let legacyLightTokens: Record<string, string>;

    beforeAll(() => {
      legacyDarkTokens = parseBlock(cssText, '.dark[data-theme-preset="legacy"]');
      legacyLightTokens = parseBlock(cssText, '.light[data-theme-preset="legacy"]');
    });

    it("binds the legacy tokens for both the class and data-theme selector forms", () => {
      expect(cssText).toContain('.dark[data-theme-preset="legacy"],');
      expect(cssText).toContain('[data-theme="dark"][data-theme-preset="legacy"]');
      expect(cssText).toContain('.light[data-theme-preset="legacy"],');
      expect(cssText).toContain('[data-theme="light"][data-theme-preset="legacy"]');
    });

    describe("dark mode", () => {
      // Contract values from specs/done_016-user-theme-customization/research.md R-02
      const expected: Record<string, string> = {
        "--accent": "oklch(69.11% 0.1944 44.01)",
        "--accent-foreground": "oklch(100% 0 0)",
        "--accent-soft": "oklch(25.0% 0.06 45.0)",
        "--accent-soft-foreground": "oklch(80.0% 0.15 45.0)",
        "--background": "oklch(14.5% 0 0)",
        "--foreground": "oklch(96.5% 0 0)",
        "--surface": "oklch(18.5% 0 0)",
        "--surface-secondary": "oklch(16.5% 0 0)",
        "--surface-tertiary": "oklch(15.0% 0 0)",
        "--overlay": "oklch(21.0% 0 0)",
        "--default": "oklch(21.0% 0 0)",
        "--default-foreground": "oklch(96.5% 0 0)",
        "--border": "oklch(26.0% 0 0)",
        "--separator": "oklch(26.0% 0 0)",
        "--muted": "oklch(65.0% 0 0)",
        "--field-background": "oklch(18.5% 0 0)",
        "--field-border": "oklch(26.0% 0 0)",
        "--field-placeholder": "oklch(50.0% 0 0)",
        "--field-foreground": "oklch(96.5% 0 0)",
        "--focus": "oklch(69.11% 0.1944 44.01)",
        "--link": "oklch(72.0% 0.16 45.0)",
        "--segment": "oklch(21.0% 0 0)",
        "--segment-foreground": "oklch(96.5% 0 0)",
      };

      for (const [name, value] of Object.entries(expected)) {
        it(`${name} should be ${value}`, () => {
          expect(legacyDarkTokens[name]).toBe(value);
        });
      }
    });

    describe("light mode", () => {
      // Contract values from specs/done_016-user-theme-customization/research.md R-02
      const expected: Record<string, string> = {
        "--accent": "oklch(62.0% 0.20 40.0)",
        "--accent-foreground": "oklch(100% 0 0)",
        "--accent-soft": "oklch(95.0% 0.04 45.0)",
        "--accent-soft-foreground": "oklch(55.0% 0.20 40.0)",
        "--background": "oklch(100% 0 0)",
        "--foreground": "oklch(18.0% 0.03 260)",
        "--surface": "oklch(98.5% 0.005 260)",
        "--surface-secondary": "oklch(96.0% 0.01 260)",
        "--surface-tertiary": "oklch(91.0% 0.02 260)",
        "--overlay": "oklch(98.5% 0.005 260)",
        "--border": "oklch(90.0% 0.01 260)",
        "--separator": "oklch(90.0% 0.01 260)",
        "--muted": "oklch(50.0% 0.02 260)",
        "--field-background": "oklch(100% 0 0)",
        "--field-border": "oklch(90.0% 0.01 260)",
        "--field-foreground": "oklch(18.0% 0.03 260)",
        "--focus": "oklch(62.0% 0.20 40.0)",
        "--link": "oklch(55.0% 0.20 40.0)",
      };

      for (const [name, value] of Object.entries(expected)) {
        it(`${name} should be ${value}`, () => {
          expect(legacyLightTokens[name]).toBe(value);
        });
      }
    });

    describe("accent differs from the pink preset (T014)", () => {
      it("light mode: legacy --accent is #EA580C, pink --accent is #DB2777", () => {
        expect(legacyLightTokens["--accent"]).toBe("oklch(62.0% 0.20 40.0)");
        expect(lightTokens["--accent"]).toBe("oklch(59.16% 0.2180 0.58)");
        expect(legacyLightTokens["--accent"]).not.toBe(lightTokens["--accent"]);
      });

      it("dark mode: legacy --accent is #F97316, pink --accent is #FF4FA3", () => {
        expect(legacyDarkTokens["--accent"]).toBe("oklch(69.11% 0.1944 44.01)");
        expect(darkTokens["--accent"]).toBe("oklch(69.50% 0.2229 355.31)");
        expect(legacyDarkTokens["--accent"]).not.toBe(darkTokens["--accent"]);
      });
    });

    describe("surface/border spot-checks per the token table", () => {
      it("dark mode: legacy background #0F0F0F is darker than pink #121114", () => {
        expect(legacyDarkTokens["--background"]).toBe("oklch(14.5% 0 0)");
        expect(darkTokens["--background"]).toBe("oklch(18.02% 0.0063 300.93)");
      });

      it("dark mode: legacy overlay #1C1C1C matches the historical card color", () => {
        expect(legacyDarkTokens["--overlay"]).toBe("oklch(21.0% 0 0)");
      });

      it("light mode: legacy surface stack is slate #F8FAFC/#F1F5F9/#E2E8F0, not pink", () => {
        expect(legacyLightTokens["--surface"]).toBe("oklch(98.5% 0.005 260)");
        expect(legacyLightTokens["--surface-secondary"]).toBe("oklch(96.0% 0.01 260)");
        expect(legacyLightTokens["--surface-tertiary"]).toBe("oklch(91.0% 0.02 260)");
        expect(legacyLightTokens["--surface"]).not.toBe(lightTokens["--surface"]);
      });

      it("dark mode: legacy border #292929 differs from pink #2C2932", () => {
        expect(legacyDarkTokens["--border"]).toBe("oklch(26.0% 0 0)");
        expect(legacyDarkTokens["--border"]).not.toBe(darkTokens["--border"]);
      });
    });
  });

  describe("accessibility guards (T017: High Contrast / Accessibility Modes)", () => {
    function mediaBlock(mediaQuery: string): string {
      const start = cssText.indexOf(`@media (${mediaQuery})`);
      if (start === -1) {
        throw new Error(`@media (${mediaQuery}) guard not found in globals.css`);
      }
      return cssText.slice(start);
    }

    it("declares a forced-colors guard that restores system colors for surfaces and text", () => {
      const block = mediaBlock("forced-colors: active");
      expect(block).toMatch(/--background:\s*Canvas/);
      expect(block).toMatch(/--foreground:\s*CanvasText/);
      expect(block).toMatch(/--surface:\s*Canvas/);
      expect(block).toMatch(/--overlay:\s*Canvas/);
    });

    it("forced-colors guard restores system colors for borders, fields, links, and focus", () => {
      const block = mediaBlock("forced-colors: active");
      expect(block).toMatch(/--border:\s*CanvasText/);
      expect(block).toMatch(/--separator:\s*CanvasText/);
      expect(block).toMatch(/--field-border:\s*CanvasText/);
      expect(block).toMatch(/--link:\s*LinkText/);
      expect(block).toMatch(/--focus:\s*Highlight/);
    });

    it("forced-colors guard keeps interactive controls and links on forced-color behavior", () => {
      const block = mediaBlock("forced-colors: active");
      expect(block).toMatch(/forced-color-adjust:\s*auto/);
      expect(block).toMatch(/a\s*\{[^}]*color:\s*LinkText/);
    });

    it("declares a prefers-contrast: more guard that strengthens border strokes", () => {
      const block = mediaBlock("prefers-contrast: more");
      expect(block).toMatch(/--border:\s*CanvasText/);
      expect(block).toMatch(/--field-border:\s*CanvasText/);
    });

    it("guards are declared after the legacy preset blocks so accessibility wins the cascade", () => {
      const legacyStart = cssText.indexOf('[data-theme-preset="legacy"]');
      expect(legacyStart).toBeGreaterThan(-1);
      expect(cssText.indexOf("@media (forced-colors: active)")).toBeGreaterThan(legacyStart);
      expect(cssText.indexOf("@media (prefers-contrast: more)")).toBeGreaterThan(legacyStart);
    });
  });
});
