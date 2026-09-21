import { describe, it, expect, vi } from "vitest";
// @ts-expect-error jsdom lacks type definitions in this project
import { JSDOM } from "jsdom";
import html from "../../index.html?raw";
const bootScript = /<script id="theme-boot">([\s\S]*?)<\/script>/.exec(html)?.[1];
if (!bootScript) throw new Error("theme boot script not found in index.html");

describe("theme boot script", () => {
  function runBootScript(dom: JSDOM, setupFn?: (window: Window) => void) {
    if (setupFn) {
      setupFn(dom.window as unknown as Window);
    }
    // Execute the script in the window realm by creating a real <script> element.
    // This is necessary because dom.window.eval() evaluates in the Node scope, not
    // the window scope, so localStorage and document would be undefined.
    const script = dom.window.document.createElement("script");
    script.textContent = bootScript;
    dom.window.document.head.appendChild(script);
  }

  describe("applies stored preference", () => {
    it("should apply stored 'dark' preference", () => {
      const dom = new JSDOM(
        `<!doctype html>
         <html lang="en" class="dark" data-theme="dark">
         <head></head>
         <body></body>
         </html>`,
        { url: "http://localhost", runScripts: "dangerously" },
      );

      const { window } = dom;
      window.localStorage.setItem("gameplane-theme", "dark");
      runBootScript(dom);

      expect(window.document.documentElement.classList.contains("dark")).toBe(true);
      expect(window.document.documentElement.classList.contains("light")).toBe(false);
      expect(window.document.documentElement.dataset.theme).toBe("dark");
    });

    it("should apply stored 'light' preference", () => {
      const dom = new JSDOM(
        `<!doctype html>
         <html lang="en" class="dark" data-theme="dark">
         <head></head>
         <body></body>
         </html>`,
        { url: "http://localhost", runScripts: "dangerously" },
      );

      const { window } = dom;
      window.localStorage.setItem("gameplane-theme", "light");
      runBootScript(dom);

      expect(window.document.documentElement.classList.contains("light")).toBe(true);
      expect(window.document.documentElement.classList.contains("dark")).toBe(false);
      expect(window.document.documentElement.dataset.theme).toBe("light");
    });
  });

  describe("respects 'system' preference with matchMedia", () => {
    it("should resolve to 'dark' when system prefers dark", () => {
      const dom = new JSDOM(
        `<!doctype html>
         <html lang="en" class="dark" data-theme="dark">
         <head></head>
         <body></body>
         </html>`,
        { url: "http://localhost", runScripts: "dangerously" },
      );

      // No stored preference → defaults to "system"
      // Mock matchMedia to return dark
      runBootScript(dom, (window) => {
        window.matchMedia = vi.fn((query: string) => ({
          matches: query === "(prefers-color-scheme: dark)",
          media: query,
          onchange: null,
          addListener: vi.fn(),
          removeListener: vi.fn(),
          addEventListener: vi.fn(),
          removeEventListener: vi.fn(),
          dispatchEvent: vi.fn(),
        })) as unknown as typeof window.matchMedia;
      });

      const { window } = dom;
      expect(window.document.documentElement.classList.contains("dark")).toBe(true);
      expect(window.document.documentElement.classList.contains("light")).toBe(false);
      expect(window.document.documentElement.dataset.theme).toBe("dark");
    });

    it("should resolve to 'light' when system prefers light", () => {
      const dom = new JSDOM(
        `<!doctype html>
         <html lang="en" class="dark" data-theme="dark">
         <head></head>
         <body></body>
         </html>`,
        { url: "http://localhost", runScripts: "dangerously" },
      );

      // No stored preference → defaults to "system"
      // Mock matchMedia to return light
      runBootScript(dom, (window) => {
        window.matchMedia = vi.fn((query: string) => ({
          matches: query === "(prefers-color-scheme: light)",
          media: query,
          onchange: null,
          addListener: vi.fn(),
          removeListener: vi.fn(),
          addEventListener: vi.fn(),
          removeEventListener: vi.fn(),
          dispatchEvent: vi.fn(),
        })) as unknown as typeof window.matchMedia;
      });

      const { window } = dom;
      expect(window.document.documentElement.classList.contains("light")).toBe(true);
      expect(window.document.documentElement.classList.contains("dark")).toBe(false);
      expect(window.document.documentElement.dataset.theme).toBe("light");
    });

    it("should default to 'dark' when no preference is stored and matchMedia is unavailable", () => {
      const dom = new JSDOM(
        `<!doctype html>
         <html lang="en" class="dark" data-theme="dark">
         <head></head>
         <body></body>
         </html>`,
        { url: "http://localhost", runScripts: "dangerously" },
      );

      // No stored preference
      // Remove matchMedia
      runBootScript(dom, (window) => {
        delete (window as unknown as { matchMedia?: typeof window.matchMedia }).matchMedia;
      });

      const { window } = dom;
      // Should keep the default dark
      expect(window.document.documentElement.classList.contains("dark")).toBe(true);
      expect(window.document.documentElement.dataset.theme).toBe("dark");
    });
  });

  describe("handles localStorage unavailability gracefully", () => {
    it("should keep dark default when localStorage throws", () => {
      const dom = new JSDOM(
        `<!doctype html>
         <html lang="en" class="dark" data-theme="dark">
         <head></head>
         <body></body>
         </html>`,
        { url: "http://localhost", runScripts: "dangerously" },
      );

      // Make localStorage throw
      runBootScript(dom, (window) => {
        Object.defineProperty(window, "localStorage", {
          get: () => {
            throw new Error("localStorage is not available");
          },
        });
      });

      const { window } = dom;
      // Should keep the default dark
      expect(window.document.documentElement.classList.contains("dark")).toBe(true);
      expect(window.document.documentElement.dataset.theme).toBe("dark");
    });
  });

  describe("initializes data-theme attribute correctly", () => {
    it("should set data-theme to match the resolved class", () => {
      const dom = new JSDOM(
        `<!doctype html>
         <html lang="en" class="dark" data-theme="dark">
         <head></head>
         <body></body>
         </html>`,
        { url: "http://localhost", runScripts: "dangerously" },
      );

      const { window } = dom;
      window.localStorage.setItem("gameplane-theme", "light");
      runBootScript(dom);

      expect(window.document.documentElement.dataset.theme).toBe("light");
    });

    it("should remove both dark and light classes before adding the resolved one", () => {
      const dom = new JSDOM(
        `<!doctype html>
         <html lang="en" class="dark light" data-theme="dark">
         <head></head>
         <body></body>
         </html>`,
        { url: "http://localhost", runScripts: "dangerously" },
      );

      const { window } = dom;
      window.localStorage.setItem("gameplane-theme", "light");
      runBootScript(dom);

      // Should have only light, not both
      expect(window.document.documentElement.classList.contains("light")).toBe(true);
      expect(window.document.documentElement.classList.contains("dark")).toBe(false);
    });
  });

  describe("reads the gameplane-theme-prefs cache (theme-tokens-v2.md §3, R-05)", () => {
    function setPrefs(window: Window, overrides: Record<string, unknown> = {}) {
      window.localStorage.setItem(
        "gameplane-theme-prefs",
        JSON.stringify({
          themeType: "preset",
          presetId: "pink",
          appearanceMode: "system",
          customColors: null,
          customCssEnabled: false,
          customCss: null,
          ...overrides,
        }),
      );
    }

    function makeDom(url = "http://localhost") {
      return new JSDOM(
        `<!doctype html>
         <html lang="en" class="dark" data-theme="dark">
         <head></head>
         <body></body>
         </html>`,
        { url, runScripts: "dangerously" },
      );
    }

    it("applies cached preset, base type, and mode before first paint", () => {
      const dom = makeDom();
      const { window } = dom;
      setPrefs(window, { presetId: "legacy", appearanceMode: "dark" });
      runBootScript(dom);

      const root = window.document.documentElement;
      expect(root.classList.contains("dark")).toBe(true);
      expect(root.classList.contains("light")).toBe(false);
      expect(root.dataset.theme).toBe("dark");
      expect(root.dataset.themePreset).toBe("legacy");
      expect(root.dataset.themeType).toBe("preset");
      expect(root.dataset.customCss).toBe("off");
      expect(window.document.getElementById("gameplane-custom-css")).toBeNull();
    });

    it("resolves appearanceMode 'system' via matchMedia when the cache exists", () => {
      const dom = makeDom();
      const { window } = dom;
      setPrefs(window, { appearanceMode: "system" });
      runBootScript(dom, (w) => {
        w.matchMedia = vi.fn((query: string) => ({
          matches: query === "(prefers-color-scheme: light)",
          media: query,
          onchange: null,
          addListener: vi.fn(),
          removeListener: vi.fn(),
          addEventListener: vi.fn(),
          removeEventListener: vi.fn(),
          dispatchEvent: vi.fn(),
        })) as unknown as typeof w.matchMedia;
      });

      const root = window.document.documentElement;
      expect(root.classList.contains("light")).toBe(true);
      expect(root.dataset.theme).toBe("light");
      expect(root.dataset.themePreset).toBe("pink");
      expect(root.dataset.themeType).toBe("preset");
      expect(root.dataset.customCss).toBe("off");
    });

    it("marks data-theme-type 'custom_colors' when the cache selects custom colors", () => {
      const dom = makeDom();
      const { window } = dom;
      setPrefs(window, {
        themeType: "custom_colors",
        customColors: { accent: "#3B82F6", surface: "#141318" },
      });
      runBootScript(dom);

      const root = window.document.documentElement;
      expect(root.dataset.themeType).toBe("custom_colors");
      expect(root.dataset.customCss).toBe("off");
      expect(window.document.getElementById("gameplane-custom-css")).toBeNull();
    });

    it("falls back to the legacy gameplane-theme key for mode when the prefs cache is absent", () => {
      const dom = makeDom();
      const { window } = dom;
      window.localStorage.setItem("gameplane-theme", "light");
      runBootScript(dom);

      const root = window.document.documentElement;
      expect(root.classList.contains("light")).toBe(true);
      expect(root.dataset.theme).toBe("light");
      // Mode-only fallback: preset/type/overlay fall back to defaults.
      expect(root.dataset.themePreset).toBe("pink");
      expect(root.dataset.themeType).toBe("preset");
      expect(root.dataset.customCss).toBe("off");
    });

    it("falls back cleanly to pink/preset/system defaults when the cache is malformed", () => {
      const dom = makeDom();
      const { window } = dom;
      window.localStorage.setItem("gameplane-theme-prefs", "{not valid json");
      expect(() => runBootScript(dom)).not.toThrow();

      const root = window.document.documentElement;
      expect(root.dataset.themePreset).toBe("pink");
      expect(root.dataset.themeType).toBe("preset");
      expect(root.dataset.customCss).toBe("off");
      // No matchMedia in jsdom and no legacy key: markup default dark is kept.
      expect(root.classList.contains("dark")).toBe(true);
      expect(root.dataset.theme).toBe("dark");
      expect(window.document.getElementById("gameplane-custom-css")).toBeNull();
    });

    it("falls back cleanly to pink/preset/system defaults when the cache is absent", () => {
      const dom = makeDom();
      const { window } = dom;
      expect(() => runBootScript(dom)).not.toThrow();

      const root = window.document.documentElement;
      expect(root.dataset.themePreset).toBe("pink");
      expect(root.dataset.themeType).toBe("preset");
      expect(root.dataset.customCss).toBe("off");
      expect(root.classList.contains("dark")).toBe(true);
    });
  });

  describe("custom CSS overlay injection (theme-tokens-v2.md §5)", () => {
    const customCss = ".gp-custom { color: red; }";

    function setPrefs(window: Window, overrides: Record<string, unknown> = {}) {
      window.localStorage.setItem(
        "gameplane-theme-prefs",
        JSON.stringify({
          themeType: "preset",
          presetId: "pink",
          appearanceMode: "system",
          customColors: null,
          customCssEnabled: false,
          customCss: null,
          ...overrides,
        }),
      );
    }

    function makeDom(url = "http://localhost") {
      return new JSDOM(
        `<!doctype html>
         <html lang="en" class="dark" data-theme="dark">
         <head></head>
         <body></body>
         </html>`,
        { url, runScripts: "dangerously" },
      );
    }

    it("injects <style id=\"gameplane-custom-css\"> as the last child of <head> when enabled", () => {
      const dom = makeDom();
      const { window } = dom;
      setPrefs(window, { customCssEnabled: true, customCss });
      runBootScript(dom);

      const styleEl = window.document.getElementById("gameplane-custom-css");
      expect(styleEl).not.toBeNull();
      expect(styleEl?.textContent).toBe(customCss);
      expect(window.document.head.lastElementChild?.id).toBe("gameplane-custom-css");
      expect(window.document.documentElement.dataset.customCss).toBe("on");
    });

    it("keeps the base theme applied while suspending the overlay in safe mode", () => {
      const dom = makeDom("http://localhost/?safe-mode=1");
      const { window } = dom;
      setPrefs(window, { presetId: "legacy", customCssEnabled: true, customCss });
      runBootScript(dom);

      const root = window.document.documentElement;
      expect(window.document.getElementById("gameplane-custom-css")).toBeNull();
      expect(root.dataset.customCss).toBe("off");
      expect(root.dataset.themePreset).toBe("legacy");
    });

    it("does not inject when customCssEnabled is false even if customCss is stored", () => {
      const dom = makeDom();
      const { window } = dom;
      setPrefs(window, { customCssEnabled: false, customCss });
      runBootScript(dom);

      expect(window.document.getElementById("gameplane-custom-css")).toBeNull();
      expect(window.document.documentElement.dataset.customCss).toBe("off");
    });

    it("does not inject when customCss is null even if customCssEnabled is true", () => {
      const dom = makeDom();
      const { window } = dom;
      setPrefs(window, { customCssEnabled: true, customCss: null });
      runBootScript(dom);

      expect(window.document.getElementById("gameplane-custom-css")).toBeNull();
      expect(window.document.documentElement.dataset.customCss).toBe("off");
    });

    it("never applies cached prefs or injects on /login (FR-011)", () => {
      const dom = makeDom("http://localhost/login");
      const { window } = dom;
      setPrefs(window, { presetId: "legacy", customCssEnabled: true, customCss });
      runBootScript(dom);

      const root = window.document.documentElement;
      expect(window.document.getElementById("gameplane-custom-css")).toBeNull();
      expect(root.dataset.customCss).not.toBe("on");
      expect(root.dataset.themePreset).not.toBe("legacy");
    });

    it("never applies cached prefs or injects on /share/:token (FR-011)", () => {
      const dom = makeDom("http://localhost/share/abc123");
      const { window } = dom;
      setPrefs(window, { presetId: "legacy", customCssEnabled: true, customCss });
      runBootScript(dom);

      const root = window.document.documentElement;
      expect(window.document.getElementById("gameplane-custom-css")).toBeNull();
      expect(root.dataset.customCss).not.toBe("on");
      expect(root.dataset.themePreset).not.toBe("legacy");
    });
  });
});
