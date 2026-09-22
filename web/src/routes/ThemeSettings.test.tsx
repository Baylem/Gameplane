import { afterEach, beforeEach, describe, it, expect, vi } from "vitest";
import type { ReactNode } from "react";
import { http, HttpResponse } from "msw";
import { fireEvent, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { server } from "@/test/server";
import { renderWithQuery } from "@/test/render";
import { THEME_PREFS_STORAGE_KEY, DEFAULT_THEME_PREFERENCES } from "@/lib/useThemePreferences";
import type { UserThemePreferences } from "@/types";

const navigate = vi.fn();
vi.mock("@tanstack/react-router", () => ({
  useNavigate: () => navigate,
  Link: ({ children, to, ...rest }: { children: ReactNode; to: string } & Record<string, unknown>) => (
    <a href={to} {...rest}>{children}</a>
  ),
}));

import { ThemeSettingsPage } from "./ThemeSettings";

function seedPrefs(prefs: Partial<UserThemePreferences> = {}) {
  window.localStorage.setItem(
    THEME_PREFS_STORAGE_KEY,
    JSON.stringify({ ...DEFAULT_THEME_PREFERENCES, ...prefs }),
  );
}

beforeEach(() => {
  window.localStorage.clear();
  document.getElementById("gameplane-custom-theme-vars")?.remove();
  document.getElementById("gameplane-custom-css")?.remove();
  const html = document.documentElement;
  html.removeAttribute("data-theme-preset");
  html.removeAttribute("data-theme-type");
  html.removeAttribute("data-custom-css");
});

afterEach(() => {
  server.resetHandlers();
});

describe("ThemeSettingsPage", () => {
  it("renders the five section cards", async () => {
    seedPrefs();
    renderWithQuery(<ThemeSettingsPage />);
    expect(await screen.findByRole("heading", { name: "Theme & Appearance" })).toBeInTheDocument();
    for (const title of [
      "Preset theme",
      "Appearance mode",
      // "Custom colors" appears both as the card title and as the third
      // preset radio card's label — getAllByText covers both.
      "Custom colors",
      "Custom CSS overlay",
      "Export / Import",
    ]) {
      expect(screen.getAllByText(title).length).toBeGreaterThan(0);
    }
  });

  it("renders the settings nav with Theme active", async () => {
    seedPrefs();
    renderWithQuery(<ThemeSettingsPage />);
    await screen.findByRole("heading", { name: "Theme & Appearance" });
    const themeLink = screen.getByRole("link", { name: /Theme/i });
    expect(themeLink).toHaveAttribute("href", "/settings/theme");
    expect(themeLink).toHaveAttribute("aria-current", "page");
    // Other entries route back to the admin settings page.
    await userEvent.click(screen.getByRole("button", { name: /General/i }));
    expect(navigate).toHaveBeenCalledWith({ to: "/admin" });
  });

  it("clicking Legacy sets data-theme-preset=\"legacy\" (live preview)", async () => {
    seedPrefs();
    renderWithQuery(<ThemeSettingsPage />);
    await screen.findByRole("heading", { name: "Theme & Appearance" });
    await userEvent.click(screen.getByRole("radio", { name: /Legacy Orange/i }));
    await waitFor(() => {
      expect(document.documentElement.dataset.themePreset).toBe("legacy");
    });
    // Re-selecting Modern Pink restores the pink preset.
    await userEvent.click(screen.getByRole("radio", { name: /Modern Pink/i }));
    await waitFor(() => {
      expect(document.documentElement.dataset.themePreset).toBe("pink");
    });
  });

  it("Custom colors radio activates the card; presets disable it again", async () => {
    seedPrefs({ customColors: { accent: "#10B981", surface: "#1E293B" } });
    renderWithQuery(<ThemeSettingsPage />);
    await screen.findByRole("heading", { name: "Theme & Appearance" });

    // Preset base active: stored values stay visible but the controls are disabled.
    const blue = screen.getByRole("button", { name: "Blue" });
    expect(blue).toBeDisabled();
    expect(screen.getByRole("radio", { name: /Custom colors/ })).not.toBeChecked();
    expect(document.documentElement.dataset.themeType).toBe("preset");
    expect(document.getElementById("gameplane-custom-theme-vars")).not.toBeInTheDocument();

    // Selecting the Custom colors card activates the base and the controls.
    await userEvent.click(screen.getByRole("radio", { name: /Custom colors/ }));
    await waitFor(() => {
      expect(document.documentElement.dataset.themeType).toBe("custom_colors");
    });
    expect(blue).toBeEnabled();
    expect(document.getElementById("gameplane-custom-theme-vars")).toBeInTheDocument();

    // Back to a preset: base flips back, controls grey out, stored values remain.
    await userEvent.click(screen.getByRole("radio", { name: /Modern Pink/i }));
    await waitFor(() => {
      expect(document.documentElement.dataset.themeType).toBe("preset");
    });
    expect(document.documentElement.dataset.themePreset).toBe("pink");
    expect(blue).toBeDisabled();
    expect(document.getElementById("gameplane-custom-theme-vars")).not.toBeInTheDocument();
    // Stored custom colors survived the excursion (FR-012)…
    expect(screen.getByRole("button", { name: "Emerald" })).toHaveAttribute("aria-pressed", "true");
  });

  it("save calls PUT /users/me/preferences with the draft", async () => {
    let captured: unknown = null;
    server.use(
      http.put("/users/me/preferences", async ({ request }) => {
        captured = await request.json();
        return HttpResponse.json({ ...DEFAULT_THEME_PREFERENCES, presetId: "legacy" });
      }),
    );
    seedPrefs();
    renderWithQuery(<ThemeSettingsPage />);
    await screen.findByRole("heading", { name: "Theme & Appearance" });
    await userEvent.click(screen.getByRole("radio", { name: /Legacy Orange/i }));
    await userEvent.click(screen.getByRole("button", { name: /^Save$/i }));
    await waitFor(() => expect(captured).not.toBeNull());
    expect(captured).toMatchObject({
      themeType: "preset",
      presetId: "legacy",
      appearanceMode: "system",
      customCssEnabled: false,
    });
  });

  it("switching presets keeps stored custom colors and CSS (FR-012)", async () => {
    let captured: unknown = null;
    server.use(
      http.put("/users/me/preferences", async ({ request }) => {
        captured = await request.json();
        return HttpResponse.json({ ...DEFAULT_THEME_PREFERENCES, presetId: "legacy" });
      }),
    );
    seedPrefs({
      customColors: { accent: "#10B981", surface: "#1C1A20" },
      customCss: ".dashboard-card { border-radius: 12px; }",
      customCssEnabled: true,
    });
    renderWithQuery(<ThemeSettingsPage />);
    await screen.findByRole("heading", { name: "Theme & Appearance" });
    await userEvent.click(screen.getByRole("radio", { name: /Legacy Orange/i }));
    await userEvent.click(screen.getByRole("button", { name: /^Save$/i }));
    await waitFor(() => expect(captured).not.toBeNull());
    expect(captured).toMatchObject({
      themeType: "preset",
      presetId: "legacy",
      customColors: { accent: "#10B981", surface: "#1C1A20" },
      customCss: ".dashboard-card { border-radius: 12px; }",
      customCssEnabled: true,
    });
  });

  it("overlay toggle flips data-custom-css and mounts/unmounts #gameplane-custom-css", async () => {
    seedPrefs({ customCss: ":root { --radius: 14px; }" });
    renderWithQuery(<ThemeSettingsPage />);
    await screen.findByRole("heading", { name: "Theme & Appearance" });
    expect(document.documentElement.dataset.customCss).toBe("off");
    expect(document.getElementById("gameplane-custom-css")).not.toBeInTheDocument();

    const toggle = screen.getByRole("switch", { name: "Enable custom CSS overlay" });
    await userEvent.click(toggle);
    await waitFor(() => {
      expect(document.documentElement.dataset.customCss).toBe("on");
    });
    const overlay = document.getElementById("gameplane-custom-css");
    expect(overlay).toBeInTheDocument();
    expect(overlay?.textContent).toContain("--radius: 14px");

    await userEvent.click(toggle);
    await waitFor(() => {
      expect(document.documentElement.dataset.customCss).toBe("off");
    });
    expect(document.getElementById("gameplane-custom-css")).not.toBeInTheDocument();
    // Disabling keeps the stylesheet in the editor (FR-012).
    expect(screen.getByLabelText("Custom CSS")).toHaveValue(":root { --radius: 14px; }");
  });

  it("surfaces an inline editor error naming the offending @import line", async () => {
    seedPrefs({ customCss: "" });
    renderWithQuery(<ThemeSettingsPage />);
    const editor = await screen.findByLabelText("Custom CSS");
    await userEvent.type(editor, "@import url('https://example.com/theme.css');{enter}");
    expect(
      await screen.findByText(
        /External resource loads are not allowed \(line 1: `@import url\('https:\/\/example\.com\/theme\.css'\);`\)\. Paste the content inline instead\./,
      ),
    ).toBeInTheDocument();
    // A failing validation blocks the save action.
    expect(screen.getByRole("button", { name: /^Save$/i })).toBeDisabled();
  });

  it("warns when the accent/surface pair fails the WCAG contrast guard", async () => {
    // Amber accent (#F59E0B) on Crisp Light (#F8FAFC) is below the 3:1 floor.
    seedPrefs({ themeType: "custom_colors", customColors: { accent: "#F59E0B", surface: "#F8FAFC" } });
    renderWithQuery(<ThemeSettingsPage />);
    await screen.findByRole("heading", { name: "Theme & Appearance" });
    expect(await screen.findByText("Contrast guard")).toBeInTheDocument();
    expect(
      screen.getByText(/fails WCAG AA \([0-9.]+:1\)\. Choose a darker accent or a darker surface tone\./),
    ).toBeInTheDocument();
  });

  it("imports a valid document via preview and Apply import", async () => {
    let captured: unknown = null;
    server.use(
      http.put("/users/me/preferences", async ({ request }) => {
        captured = await request.json();
        return HttpResponse.json({ ...DEFAULT_THEME_PREFERENCES, presetId: "legacy" });
      }),
    );
    seedPrefs();
    renderWithQuery(<ThemeSettingsPage />);
    const pasteArea = await screen.findByLabelText("Paste theme JSON");
    const doc = {
      format: "gameplane-theme",
      version: 1,
      preferences: {
        themeType: "preset",
        presetId: "legacy",
        appearanceMode: "dark",
        customColors: null,
        customCssEnabled: false,
        customCss: null,
      },
    };
    fireEvent.change(pasteArea, { target: { value: JSON.stringify(doc) } });
    expect(await screen.findByText("Import preview")).toBeInTheDocument();
    expect(screen.getByText("Preset: Legacy Orange")).toBeInTheDocument();
    await userEvent.click(screen.getByRole("button", { name: /Apply import/i }));
    await waitFor(() => expect(captured).not.toBeNull());
    expect(captured).toMatchObject({ presetId: "legacy", appearanceMode: "dark" });
  });

  it("surfaces client-side import validation errors inline", async () => {
    seedPrefs();
    renderWithQuery(<ThemeSettingsPage />);
    const pasteArea = await screen.findByLabelText("Paste theme JSON");
    fireEvent.change(pasteArea, { target: { value: "{ not json" } });
    expect(await screen.findByText(/invalid theme export: not valid JSON/i)).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /Apply import/i })).not.toBeInTheDocument();
  });

  it("Reset to Defaults confirms, calls the reset endpoint, and clears customs", async () => {
    let captured: unknown = null;
    server.use(
      http.post("/users/me/preferences/reset", async ({ request }) => {
        captured = await request.json();
        return HttpResponse.json({ ...DEFAULT_THEME_PREFERENCES });
      }),
    );
    seedPrefs({
      themeType: "custom_colors",
      customColors: { accent: "#3B82F6", surface: "#171717" },
      customCss: ":root { --radius: 14px; }",
      customCssEnabled: true,
    });
    renderWithQuery(<ThemeSettingsPage />);
    await screen.findByRole("heading", { name: "Theme & Appearance" });
    expect(document.getElementById("gameplane-custom-css")).toBeInTheDocument();

    await userEvent.click(screen.getByRole("button", { name: /Reset to Defaults/i }));
    expect(
      await screen.findByText(
        "This deletes your custom colors and custom CSS and restores the selected preset. Continue?",
      ),
    ).toBeInTheDocument();
    const dialog = screen.getByRole("alertdialog");
    await userEvent.click(within(dialog).getByRole("button", { name: /^Reset$/i }));

    await waitFor(() => expect(captured).not.toBeNull());
    expect(captured).toMatchObject({ presetId: "pink", appearanceMode: "system" });
    // Customs are gone: overlay unmounted, editor cleared, base back to preset.
    await waitFor(() => {
      expect(document.getElementById("gameplane-custom-css")).not.toBeInTheDocument();
    });
    expect(document.documentElement.dataset.themeType).toBe("preset");
    expect(document.documentElement.dataset.customCss).toBe("off");
    expect(screen.getByLabelText("Custom CSS")).toHaveValue("");
  });
});
