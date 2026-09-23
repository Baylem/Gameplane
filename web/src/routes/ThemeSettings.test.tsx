import { afterEach, beforeEach, describe, it, expect, vi } from "vitest";
import type { ReactNode } from "react";
import { http, HttpResponse } from "msw";
import { fireEvent, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { server } from "@/test/server";
import { renderWithQuery } from "@/test/render";
import { makeUser } from "@/test/factories";
import {
  THEME_PREFS_STORAGE_KEY,
  DEFAULT_THEME_PREFERENCES,
  THEME_VARS_CSS_STORAGE_KEY,
  CUSTOM_MODE_STORAGE_KEY,
} from "@/lib/useThemePreferences";
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
    // Other entries deep-link the matching admin settings section.
    // Admin sections appear once /users/me confirms config:manage.
    await userEvent.click(await screen.findByRole("button", { name: /General/i }));
    expect(navigate).toHaveBeenCalledWith({ to: "/admin", search: { section: "general" } });
  });

  it("shows only the Theme entry to users without config:manage", async () => {
    server.use(http.get("/users/me", () => HttpResponse.json(makeUser({ role: "viewer" }))));
    seedPrefs();
    const { client } = renderWithQuery(<ThemeSettingsPage />);
    await screen.findByRole("heading", { name: "Theme & Appearance" });
    await waitFor(() => expect(client.getQueryData(["me"])).toBeDefined());
    expect(screen.getByRole("link", { name: /Theme/i })).toHaveAttribute("aria-current", "page");
    for (const label of [/General/i, /Authentication/i, /Backup destinations/i, /Module sources/i]) {
      expect(screen.queryByRole("button", { name: label })).not.toBeInTheDocument();
    }
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

  it("disables Appearance mode with a note while Custom colors is active, and re-enables it on a preset (D4)", async () => {
    seedPrefs({ customColors: { accent: "#10B981", surface: "#1E293B" } });
    renderWithQuery(<ThemeSettingsPage />);
    await screen.findByRole("heading", { name: "Theme & Appearance" });

    const modeGroup = screen.getByRole("group", { name: "Appearance mode" });
    expect(within(modeGroup).getByRole("button", { name: "Light" })).toBeEnabled();
    expect(screen.queryByText("Set by your surface color")).not.toBeInTheDocument();

    await userEvent.click(screen.getByRole("radio", { name: /Custom colors/ }));
    await waitFor(() => {
      expect(within(modeGroup).getByRole("button", { name: "Light" })).toBeDisabled();
    });
    expect(within(modeGroup).getByRole("button", { name: "Dark" })).toBeDisabled();
    expect(within(modeGroup).getByRole("button", { name: "System" })).toBeDisabled();
    expect(screen.getByText("Set by your surface color")).toBeInTheDocument();

    await userEvent.click(screen.getByRole("radio", { name: /Modern Pink/i }));
    await waitFor(() => {
      expect(within(modeGroup).getByRole("button", { name: "Light" })).toBeEnabled();
    });
    expect(screen.queryByText("Set by your surface color")).not.toBeInTheDocument();
  });

  it("live preview follows the surface tone's brightness while Custom colors is active (D4)", async () => {
    seedPrefs({
      themeType: "custom_colors",
      customColors: { accent: "#10B981", surface: "#1E293B" }, // Dark Slate
    });
    renderWithQuery(<ThemeSettingsPage />);
    await screen.findByRole("heading", { name: "Theme & Appearance" });
    await waitFor(() => expect(document.documentElement.dataset.theme).toBe("dark"));

    await userEvent.click(screen.getByText("Crisp Light", { exact: true }));
    await waitFor(() => expect(document.documentElement.dataset.theme).toBe("light"));
  });

  it("entering valid hex in accent hex input updates the color picker", async () => {
    seedPrefs({ themeType: "custom_colors", customColors: { accent: "#3B82F6", surface: "#1E293B" } });
    renderWithQuery(<ThemeSettingsPage />);
    await screen.findByRole("heading", { name: "Theme & Appearance" });

    // Custom colors are active, so the hex input is enabled.
    const accentHexInput = screen.getByLabelText("Primary accent hex value");
    expect(accentHexInput).not.toBeDisabled();

    // Type a valid hex value and press Enter.
    await userEvent.clear(accentHexInput);
    await userEvent.type(accentHexInput, "#123456");
    await userEvent.keyboard("{Enter}");

    // The color picker should now reflect the new value.
    const accentPicker = screen.getByLabelText("Primary accent color picker") as HTMLInputElement;
    expect(accentPicker.value).toBe("#123456");
  });

  it("entering valid hex in surface tone hex input updates the color picker", async () => {
    seedPrefs({ themeType: "custom_colors", customColors: { accent: "#3B82F6", surface: "#1E293B" } });
    renderWithQuery(<ThemeSettingsPage />);
    await screen.findByRole("heading", { name: "Theme & Appearance" });

    // Custom colors are active, so the hex input is enabled.
    const surfaceHexInput = screen.getByLabelText("Surface tone hex value");
    expect(surfaceHexInput).not.toBeDisabled();

    // Type a valid hex value and press Enter.
    await userEvent.clear(surfaceHexInput);
    await userEvent.type(surfaceHexInput, "#abcdef");
    await userEvent.keyboard("{Enter}");

    // The color picker should now reflect the new value.
    const surfacePicker = screen.getByLabelText("Surface tone color picker") as HTMLInputElement;
    expect(surfacePicker.value).toBe("#abcdef");
  });

  it("reverts an invalid hex typed into the free accent field", async () => {
    seedPrefs({ themeType: "custom_colors", customColors: { accent: "#10B981", surface: "#1E293B" } });
    renderWithQuery(<ThemeSettingsPage />);
    await screen.findByRole("heading", { name: "Theme & Appearance" });
    const accentHex = (await screen.findByLabelText("Primary accent hex value")) as HTMLInputElement;
    await userEvent.clear(accentHex);
    await userEvent.type(accentHex, "not-a-color");
    await userEvent.tab();
    expect(accentHex.value).toBe("#10B981");
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

  // --- Custom CSS editor error branches (editorErrorMessage) -----------------

  it("surfaces an inline editor error for an external url() reference", async () => {
    seedPrefs({ customCss: "" });
    renderWithQuery(<ThemeSettingsPage />);
    const editor = await screen.findByLabelText("Custom CSS");
    fireEvent.change(editor, {
      target: { value: ".a { background: url(https://evil.com/x.png); }" },
    });
    expect(
      await screen.findByText(
        /External resource loads are not allowed \(line 1: `.*url\(https:\/\/evil\.com\/x\.png\).*`\)\. Paste the content inline instead\./,
      ),
    ).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /^Save$/i })).toBeDisabled();
  });

  it("surfaces an inline editor error for unbalanced braces", async () => {
    seedPrefs({ customCss: "" });
    renderWithQuery(<ThemeSettingsPage />);
    const editor = await screen.findByLabelText("Custom CSS");
    fireEvent.change(editor, { target: { value: ".a { color: red;" } });
    expect(
      await screen.findByText("Syntax error: unbalanced braces — every '{' must have a matching '}'."),
    ).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /^Save$/i })).toBeDisabled();
  });

  it("surfaces an inline editor error when custom CSS exceeds the max length", async () => {
    seedPrefs({ customCss: "" });
    renderWithQuery(<ThemeSettingsPage />);
    const editor = await screen.findByLabelText("Custom CSS");
    fireEvent.change(editor, { target: { value: "a".repeat(33000) } });
    expect(
      await screen.findByText("Custom CSS exceeds the 32,768-byte limit."),
    ).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /^Save$/i })).toBeDisabled();
  });

  it("surfaces an inline editor error for HTML <style>/<script> delimiters", async () => {
    seedPrefs({ customCss: "" });
    renderWithQuery(<ThemeSettingsPage />);
    const editor = await screen.findByLabelText("Custom CSS");
    fireEvent.change(editor, { target: { value: "<style>body{color:red}</style>" } });
    expect(
      await screen.findByText("HTML <style> and <script> tags are not allowed in custom CSS."),
    ).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /^Save$/i })).toBeDisabled();
  });

  it("surfaces an error message for external url() when findLine returns null (multiline url)", async () => {
    seedPrefs({ customCss: "" });
    renderWithQuery(<ThemeSettingsPage />);
    const editor = await screen.findByLabelText("Custom CSS");
    fireEvent.change(editor, {
      target: { value: ".a { background: url(\nhttps://evil.com/x.png); }" },
    });
    expect(
      await screen.findByText("External resource loads are not allowed (external url()). Paste the content inline instead."),
    ).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /^Save$/i })).toBeDisabled();
  });

  // --- Save / import error paths (onError) ------------------------------------

  it("save surfaces a server error inline", async () => {
    server.use(
      http.put("/users/me/preferences", () => new HttpResponse("save failed", { status: 500 })),
    );
    seedPrefs();
    renderWithQuery(<ThemeSettingsPage />);
    await screen.findByRole("heading", { name: "Theme & Appearance" });
    await userEvent.click(screen.getByRole("radio", { name: /Legacy Orange/i }));
    await userEvent.click(screen.getByRole("button", { name: /^Save$/i }));
    expect(await screen.findByText("save failed")).toBeInTheDocument();
  });

  it("apply import surfaces a server error inline", async () => {
    server.use(
      http.put("/users/me/preferences", () => new HttpResponse("import failed", { status: 500 })),
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
    await userEvent.click(await screen.findByRole("button", { name: /Apply import/i }));
    expect(await screen.findByText("import failed")).toBeInTheDocument();
  });

  // --- Reset to Defaults: error path and cancel -------------------------------

  it("Reset to Defaults surfaces a server error", async () => {
    server.use(
      http.post("/users/me/preferences/reset", () => new HttpResponse("reset failed", { status: 500 })),
    );
    seedPrefs();
    renderWithQuery(<ThemeSettingsPage />);
    await userEvent.click(await screen.findByRole("button", { name: /Reset to Defaults/i }));
    const dialog = await screen.findByRole("alertdialog");
    await userEvent.click(within(dialog).getByRole("button", { name: /^Reset$/i }));
    expect(await screen.findByText("reset failed")).toBeInTheDocument();
  });

  it("Reset to Defaults dialog can be cancelled without resetting", async () => {
    let resetCalled = false;
    server.use(
      http.post("/users/me/preferences/reset", () => {
        resetCalled = true;
        return HttpResponse.json({ ...DEFAULT_THEME_PREFERENCES });
      }),
    );
    seedPrefs();
    renderWithQuery(<ThemeSettingsPage />);
    await userEvent.click(await screen.findByRole("button", { name: /Reset to Defaults/i }));
    const dialog = await screen.findByRole("alertdialog");
    await userEvent.click(within(dialog).getByRole("button", { name: /^Cancel$/i }));
    await waitFor(() => expect(screen.queryByRole("alertdialog")).not.toBeInTheDocument());
    expect(resetCalled).toBe(false);
  });

  // --- Export: copy to clipboard and download ---------------------------------

  it("copies the export JSON to the clipboard", async () => {
    seedPrefs();
    const user = userEvent.setup();
    const writeText = vi.spyOn(navigator.clipboard, "writeText").mockResolvedValueOnce(undefined);
    renderWithQuery(<ThemeSettingsPage />);
    await user.click(await screen.findByRole("button", { name: /Copy to clipboard/i }));
    expect(await screen.findByText("Copied to clipboard")).toBeInTheDocument();
    expect(writeText).toHaveBeenCalled();
    writeText.mockRestore();
  });

  it("falls back to a manual-copy message when clipboard access fails", async () => {
    seedPrefs();
    const user = userEvent.setup();
    const writeText = vi
      .spyOn(navigator.clipboard, "writeText")
      .mockRejectedValueOnce(new Error("denied"));
    renderWithQuery(<ThemeSettingsPage />);
    await user.click(await screen.findByRole("button", { name: /Copy to clipboard/i }));
    expect(
      await screen.findByText("Clipboard unavailable — select the text and copy it manually."),
    ).toBeInTheDocument();
    writeText.mockRestore();
  });

  it("downloads the export as gameplane-theme.json", async () => {
    seedPrefs();
    const createURL = vi.fn(() => "blob:mock-theme");
    const revokeURL = vi.fn();
    const origCreate = URL.createObjectURL;
    const origRevoke = URL.revokeObjectURL;
    Object.defineProperty(URL, "createObjectURL", { configurable: true, writable: true, value: createURL });
    Object.defineProperty(URL, "revokeObjectURL", { configurable: true, writable: true, value: revokeURL });
    const clickSpy = vi.spyOn(HTMLAnchorElement.prototype, "click").mockImplementation(() => undefined);
    try {
      renderWithQuery(<ThemeSettingsPage />);
      await userEvent.click(
        await screen.findByRole("button", { name: /Download gameplane-theme\.json/i }),
      );
      expect(createURL).toHaveBeenCalledOnce();
      expect(clickSpy).toHaveBeenCalledOnce();
      expect(revokeURL).toHaveBeenCalledWith("blob:mock-theme");
    } finally {
      Object.defineProperty(URL, "createObjectURL", { configurable: true, writable: true, value: origCreate });
      Object.defineProperty(URL, "revokeObjectURL", { configurable: true, writable: true, value: origRevoke });
      clickSpy.mockRestore();
    }
  });

  // --- Import via the file picker ---------------------------------------------

  it("imports a theme via the file picker", async () => {
    seedPrefs();
    renderWithQuery(<ThemeSettingsPage />);
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
    const file = new File([JSON.stringify(doc)], "gameplane-theme.json", { type: "application/json" });
    const input = screen.getByLabelText("Theme export file") as HTMLInputElement;
    fireEvent.change(input, { target: { files: [file] } });
    expect(await screen.findByText("Import preview")).toBeInTheDocument();
    expect(screen.getByText("Preset: Legacy Orange")).toBeInTheDocument();
  });

  it("clicking the Choose file… button triggers the file picker input click", async () => {
    seedPrefs();
    renderWithQuery(<ThemeSettingsPage />);
    const click = vi.spyOn(HTMLInputElement.prototype, "click").mockImplementation(() => {});
    try {
      await userEvent.click(await screen.findByRole("button", { name: /Choose file/i }));
      expect(click).toHaveBeenCalled();
    } finally {
      click.mockRestore();
    }
  });

  // --- Appearance mode / custom color control interactions --------------------

  it("clicking an appearance mode button updates the draft and DOM", async () => {
    seedPrefs();
    renderWithQuery(<ThemeSettingsPage />);
    await screen.findByRole("heading", { name: "Theme & Appearance" });
    const darkBtn = screen.getByRole("button", { name: "Dark" });
    await userEvent.click(darkBtn);
    await waitFor(() => expect(darkBtn).toHaveAttribute("aria-pressed", "true"));
    expect(document.documentElement.dataset.theme).toBe("dark");
  });

  it("clicking an accent swatch updates the selected accent while custom colors is active", async () => {
    seedPrefs({ themeType: "custom_colors", customColors: { accent: "#3B82F6", surface: "#1E293B" } });
    renderWithQuery(<ThemeSettingsPage />);
    await screen.findByRole("heading", { name: "Theme & Appearance" });
    const emerald = screen.getByRole("button", { name: "Emerald" });
    await userEvent.click(emerald);
    await waitFor(() => expect(emerald).toHaveAttribute("aria-pressed", "true"));
  });

  it("changing the accent color picker input updates the draft accent", async () => {
    seedPrefs({ themeType: "custom_colors", customColors: { accent: "#3B82F6", surface: "#1E293B" } });
    renderWithQuery(<ThemeSettingsPage />);
    await screen.findByRole("heading", { name: "Theme & Appearance" });
    const accentPicker = screen.getByLabelText("Primary accent color picker") as HTMLInputElement;
    // A yellow-green hue also exercises the hslToHex 60°–120° branch used by
    // the derived-token math (theme-derivation.ts).
    fireEvent.change(accentPicker, { target: { value: "#80ff00" } });
    await waitFor(() => expect(accentPicker.value).toBe("#80ff00"));
  });

  it("changing the surface color picker input updates the draft surface", async () => {
    seedPrefs({ themeType: "custom_colors", customColors: { accent: "#3B82F6", surface: "#1E293B" } });
    renderWithQuery(<ThemeSettingsPage />);
    await screen.findByRole("heading", { name: "Theme & Appearance" });
    const surfacePicker = screen.getByLabelText("Surface tone color picker") as HTMLInputElement;
    fireEvent.change(surfacePicker, { target: { value: "#123123" } });
    await waitFor(() => expect(surfacePicker.value).toBe("#123123"));
  });

  it("reverts an invalid hex typed into the free surface field", async () => {
    seedPrefs({ themeType: "custom_colors", customColors: { accent: "#10B981", surface: "#1E293B" } });
    renderWithQuery(<ThemeSettingsPage />);
    await screen.findByRole("heading", { name: "Theme & Appearance" });
    const surfaceHex = (await screen.findByLabelText("Surface tone hex value")) as HTMLInputElement;
    await userEvent.clear(surfaceHex);
    await userEvent.type(surfaceHex, "not-a-color");
    await userEvent.tab();
    expect(surfaceHex.value).toBe("#1E293B");
  });

  it("applies a valid hex typed into the free surface field on blur", async () => {
    seedPrefs({ themeType: "custom_colors", customColors: { accent: "#10B981", surface: "#1E293B" } });
    renderWithQuery(<ThemeSettingsPage />);
    await screen.findByRole("heading", { name: "Theme & Appearance" });
    const surfaceHex = (await screen.findByLabelText("Surface tone hex value")) as HTMLInputElement;
    await userEvent.clear(surfaceHex);
    await userEvent.type(surfaceHex, "#334455");
    await userEvent.tab();
    const surfacePicker = screen.getByLabelText("Surface tone color picker") as HTMLInputElement;
    await waitFor(() => expect(surfacePicker.value).toBe("#334455"));
  });

  // --- useThemePreferences: reconciliation, persistence, offline retry --------

  it("adopts the server-reconciled preferences from /users/me over a stale localStorage cache", async () => {
    seedPrefs({ presetId: "pink", themeType: "preset" });
    server.use(
      http.get("/users/me", () =>
        HttpResponse.json(
          makeUser({
            preferences: {
              themeType: "preset",
              presetId: "legacy",
              appearanceMode: "system",
              customColors: null,
              customCssEnabled: false,
              customCss: null,
            },
          }),
        ),
      ),
    );
    renderWithQuery(<ThemeSettingsPage />);
    await waitFor(() => {
      expect(JSON.parse(window.localStorage.getItem(THEME_PREFS_STORAGE_KEY) ?? "{}")).toMatchObject({
        presetId: "legacy",
      });
    });
    expect(document.documentElement.dataset.themePreset).toBe("legacy");
  });

  it("save persists derived custom-color tokens to localStorage", async () => {
    server.use(
      http.put("/users/me/preferences", async ({ request }) => {
        const body = (await request.json()) as Record<string, unknown>;
        return HttpResponse.json({ ...DEFAULT_THEME_PREFERENCES, ...body });
      }),
    );
    seedPrefs();
    renderWithQuery(<ThemeSettingsPage />);
    await screen.findByRole("heading", { name: "Theme & Appearance" });
    await userEvent.click(screen.getByRole("radio", { name: /Custom colors/ }));
    await userEvent.click(screen.getByRole("button", { name: /^Save$/i }));
    await waitFor(() => {
      expect(window.localStorage.getItem(THEME_VARS_CSS_STORAGE_KEY)).toBeTruthy();
    });
    expect(window.localStorage.getItem(CUSTOM_MODE_STORAGE_KEY)).toBeTruthy();
  });

  it("queues a network failure and retries the save once connectivity returns", async () => {
    let calls = 0;
    server.use(
      http.put("/users/me/preferences", () => {
        calls += 1;
        if (calls === 1) return HttpResponse.error();
        return HttpResponse.json({ ...DEFAULT_THEME_PREFERENCES, presetId: "legacy" });
      }),
    );
    seedPrefs();
    renderWithQuery(<ThemeSettingsPage />);
    await screen.findByRole("heading", { name: "Theme & Appearance" });
    await userEvent.click(screen.getByRole("radio", { name: /Legacy Orange/i }));
    await userEvent.click(screen.getByRole("button", { name: /^Save$/i }));
    await waitFor(() => expect(calls).toBe(1));
    // A network-level failure reverts the optimistic DOM change.
    await waitFor(() => {
      expect(document.documentElement.dataset.themePreset).toBe("pink");
    });
    window.dispatchEvent(new Event("online"));
    await waitFor(() => expect(calls).toBe(2));
    await waitFor(() => {
      expect(document.documentElement.dataset.themePreset).toBe("legacy");
    });
  });

  it("treats a broken localStorage as no cached preferences", async () => {
    const originalLocalStorage = window.localStorage;
    Object.defineProperty(window, "localStorage", {
      configurable: true,
      value: {
        getItem: () => {
          throw new Error("blocked");
        },
        setItem: () => {},
        removeItem: () => {},
        clear: () => {},
        length: 0,
        key: () => null,
      },
    });
    try {
      renderWithQuery(<ThemeSettingsPage />);
      expect(await screen.findByRole("radio", { name: /Modern Pink/i })).toBeChecked();
    } finally {
      Object.defineProperty(window, "localStorage", { configurable: true, value: originalLocalStorage });
    }
  });

  it("keeps the custom CSS overlay applying when sessionStorage access throws (safe-mode check fails closed)", async () => {
    seedPrefs({ customCss: ":root { --radius: 14px; }", customCssEnabled: true });
    const originalSessionStorage = window.sessionStorage;
    Object.defineProperty(window, "sessionStorage", {
      configurable: true,
      value: {
        getItem: () => {
          throw new Error("blocked");
        },
        setItem: () => {},
        removeItem: () => {},
        clear: () => {},
        length: 0,
        key: () => null,
      },
    });
    try {
      renderWithQuery(<ThemeSettingsPage />);
      await waitFor(() => {
        expect(document.documentElement.dataset.customCss).toBe("on");
      });
    } finally {
      Object.defineProperty(window, "sessionStorage", { configurable: true, value: originalSessionStorage });
    }
  });

  it("falls back to dark appearance when matchMedia is unavailable", async () => {
    seedPrefs({ appearanceMode: "system" });
    const originalMatchMedia = window.matchMedia;
    delete (window as unknown as Record<string, unknown>).matchMedia;
    try {
      renderWithQuery(<ThemeSettingsPage />);
      await waitFor(() => {
        expect(document.documentElement.dataset.theme).toBe("dark");
      });
    } finally {
      window.matchMedia = originalMatchMedia;
    }
  });
});
