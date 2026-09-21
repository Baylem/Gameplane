import { describe, it, expect, vi } from "vitest";
import { http, HttpResponse } from "msw";
import { screen, fireEvent, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { server } from "@/test/server";
import { renderWithQuery } from "@/test/render";
import { BuildModuleDialog } from "./BuildModuleDialog";

const mockScaffoldResponse = {
  moduleYaml: "apiVersion: gameplane.local/module/v1\nname: test-module\nversion: 1.0.0\n",
  templateYaml: "apiVersion: gameplane.local/v1alpha1\nkind: GameTemplate\nspec:\n  game: test-game\n",
  readmeMd: "# Test Module\n",
  iconBase64: "",
};

const mockValidateResponse = {
  clean: true,
  errorCount: 0,
  warningCount: 0,
  findings: [],
};

const mockPreviewResponse = {
  resolvedImage: "ghcr.io/valgul/test:v1@sha256:1111111111111111111111111111111111111111111111111111111111111111",
  effectiveEnv: [
    { name: "MAX_MEMORY", value: "6144M", source: "configSchema" },
  ],
  computedConfig: {
    MAX_MEMORY: "6144M",
  },
  ports: [
    { name: "game", containerPort: 27015, protocol: "UDP", advertise: true },
  ],
  storage: {
    size: "20Gi",
    mountPath: "/data",
  },
};

describe("BuildModuleDialog", () => {
  it("renders wizard step 1 with archetypes and validates DNS-1123 name", async () => {
    const user = userEvent.setup();
    renderWithQuery(
      <BuildModuleDialog open onOpenChange={() => undefined} sources={["uploads"]} />
    );

    expect(screen.getByText("Create game module")).toBeInTheDocument();
    expect(await screen.findByText("SteamCMD Dedicated Server")).toBeInTheDocument();
    expect(await screen.findByText("Java Application Server")).toBeInTheDocument();
    expect(await screen.findByText("Generic Container Server")).toBeInTheDocument();

    const nameInput = screen.getByPlaceholderText("e.g. cs2-match");
    expect(nameInput).toHaveValue("");
    expect(screen.getByRole("button", { name: /Continue to Container & Ports/i })).toBeDisabled();

    // Invalid DNS name
    await user.clear(nameInput);
    await user.type(nameInput, "INVALID_UPPERCASE");
    expect(screen.getByText("Must be lowercase alphanumeric with hyphens")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /Continue to Container & Ports/i })).toBeDisabled();

    // Valid DNS name
    await user.clear(nameInput);
    await user.type(nameInput, "cs2-server");
    expect(screen.getByText("✓ Valid DNS-1123 label")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /Continue to Container & Ports/i })).not.toBeDisabled();
  });

  it("navigates through step 1, 2, and 3 with live scaffold and validation", async () => {
    const user = userEvent.setup();
    server.use(
      http.post("/modules/builder/scaffold", () => HttpResponse.json(mockScaffoldResponse)),
      http.post("/modules/builder/validate", () => HttpResponse.json(mockValidateResponse)),
      http.post("/modules/builder/preview", () => HttpResponse.json(mockPreviewResponse)),
    );

    renderWithQuery(
      <BuildModuleDialog open onOpenChange={() => undefined} sources={["uploads"]} />
    );

    // Provide valid module name
    await user.type(screen.getByPlaceholderText("e.g. cs2-match"), "my-game");

    // Step 1 -> Step 2
    await user.click(screen.getByRole("button", { name: /Continue to Container & Ports/i }));
    expect(await screen.findByText("Container Image")).toBeInTheDocument();
    expect(screen.getByText("Port Mappings")).toBeInTheDocument();
    expect(screen.getByText("Persistent Storage")).toBeInTheDocument();

    // Step 2 -> Step 3
    await user.click(screen.getByRole("button", { name: /Continue to Review & Export/i }));

    // Step 3 Review & Export
    expect(await screen.findByText("Module Validated Cleanly")).toBeInTheDocument();
    expect(screen.getByText("Memory Simulation")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /Download .tar.gz/i })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /Install to Cluster/i })).toBeInTheDocument();
  });

  it("installs module directly to cluster from step 3", async () => {
    const user = userEvent.setup();
    let exportCalled = false;
    server.use(
      http.post("/modules/builder/scaffold", () => HttpResponse.json(mockScaffoldResponse)),
      http.post("/modules/builder/validate", () => HttpResponse.json(mockValidateResponse)),
      http.post("/modules/builder/preview", () => HttpResponse.json(mockPreviewResponse)),
      http.post("/modules/builder/export", () => {
        exportCalled = true;
        return HttpResponse.json({ installed: true, moduleName: "my-game" }, { status: 201 });
      }),
    );

    const onInstalled = vi.fn();
    const onOpenChange = vi.fn();

    renderWithQuery(
      <BuildModuleDialog open onOpenChange={onOpenChange} sources={["uploads"]} onInstalled={onInstalled} />
    );

    // Provide valid module name
    await user.type(screen.getByPlaceholderText("e.g. cs2-match"), "my-game");

    // Navigate to step 3
    await user.click(screen.getByRole("button", { name: /Continue to Container & Ports/i }));
    expect(await screen.findByText("Container Image")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: /Continue to Review & Export/i }));
    expect(await screen.findByText("Module Validated Cleanly")).toBeInTheDocument();

    // Install to cluster
    const installBtn = screen.getByRole("button", { name: /Install to Cluster/i });
    await user.click(installBtn);

    await waitFor(() => {
      expect(exportCalled).toBe(true);
      expect(onInstalled).toHaveBeenCalled();
      expect(onOpenChange).toHaveBeenCalledWith(false);
    });
  });

  it("handles download archive in step 3", async () => {
    const user = userEvent.setup();
    let downloadCalled = false;
    server.use(
      http.post("/modules/builder/scaffold", () => HttpResponse.json(mockScaffoldResponse)),
      http.post("/modules/builder/validate", () => HttpResponse.json(mockValidateResponse)),
      http.post("/modules/builder/preview", () => HttpResponse.json(mockPreviewResponse)),
      http.post("/modules/builder/export", () => {
        downloadCalled = true;
        return new HttpResponse(new Blob(["fake tar"]), {
          status: 200,
          headers: { "Content-Type": "application/gzip" },
        });
      }),
    );

    renderWithQuery(
      <BuildModuleDialog open onOpenChange={() => undefined} sources={["uploads"]} />
    );

    await user.type(screen.getByPlaceholderText("e.g. cs2-match"), "my-game");
    await user.click(screen.getByRole("button", { name: /Continue to Container & Ports/i }));
    expect(await screen.findByText("Container Image")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: /Continue to Review & Export/i }));
    expect(await screen.findByText("Module Validated Cleanly")).toBeInTheDocument();

    const downloadBtn = screen.getByRole("button", { name: /Download .tar.gz/i });
    await user.click(downloadBtn);

    await waitFor(() => {
      expect(downloadCalled).toBe(true);
    });
  });

  it("handles revalidation error gracefully in step 3", async () => {
    const user = userEvent.setup();
    server.use(
      http.post("/modules/builder/scaffold", () => HttpResponse.json(mockScaffoldResponse)),
      http.post("/modules/builder/validate", () => HttpResponse.json(mockValidateResponse)),
      http.post("/modules/builder/preview", () => HttpResponse.json(mockPreviewResponse)),
    );

    renderWithQuery(
      <BuildModuleDialog open onOpenChange={() => undefined} sources={["uploads"]} />
    );

    await user.type(screen.getByPlaceholderText("e.g. cs2-match"), "my-game");
    await user.click(screen.getByRole("button", { name: /Continue to Container & Ports/i }));
    expect(await screen.findByText("Container Image")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: /Continue to Review & Export/i }));
    expect(await screen.findByText("Module Validated Cleanly")).toBeInTheDocument();

    // Now mock validation failure
    server.use(
      http.post("/modules/builder/validate", () => {
        return HttpResponse.json({ error: "Invalid syntax" }, { status: 400 });
      }),
    );

    // Trigger tab change or editor change
    const textareas = screen.getAllByRole("textbox");
    if (textareas.length > 0) {
      fireEvent.change(textareas[0], { target: { value: "invalid: yaml: [" } });
    }

    await waitFor(() => {
      expect(screen.getByText(/1 errors/i)).toBeInTheDocument();
    });
  });
});

