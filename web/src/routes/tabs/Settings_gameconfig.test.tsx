import type { ReactElement } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { fireEvent, screen, waitFor } from "@testing-library/react";

import { renderWithQuery as baseRenderWithQuery } from "@/test/render";
import { ResourceTargetProvider } from "@/lib/resourceTarget";
import { CONFIG_REDACTED_MARKER } from "@/lib/validation";
import { SettingsTab } from "./Settings";
import type { GameServer } from "@/types";

type FetchInit = Parameters<typeof fetch>[1];

vi.mock("@tanstack/react-router", () => ({
  useNavigate: () => vi.fn(),
}));

const fetchMock = vi.fn();

beforeEach(() => {
  vi.stubGlobal("fetch", fetchMock);
});
afterEach(() => {
  fetchMock.mockReset();
  vi.unstubAllGlobals();
});

function jsonRes(body: unknown, status = 200): Response {
  return new Response(JSON.stringify(body), { status, headers: { "Content-Type": "application/json" } });
}

const SCHEMA = [
  { name: "MAX_PLAYERS", displayName: "Max players", type: "int", default: "16", min: 1, max: 255 },
  { name: "SERVER_PASSWORD", displayName: "Server password", type: "password" },
];

function gs(config?: Record<string, string>): GameServer {
  return {
    metadata: { name: "mc-survival", resourceVersion: "100" },
    spec: { templateRef: { name: "minecraft-java" }, config },
    status: { phase: "Running" },
  };
}

function stubTemplate(opts: { schema?: unknown[]; versions?: boolean } = {}, putBody?: { current: unknown }, latest?: GameServer) {
  fetchMock.mockImplementation(async (url: string, init?: FetchInit) => {
    if (url.startsWith("/templates/")) {
      return jsonRes({
        metadata: { name: "minecraft-java" },
        spec: {
          displayName: "Minecraft",
          game: "minecraft-java",
          version: "1.0",
          image: "x",
          configSchema: opts.schema,
          versions: opts.versions ? [{ id: "1.21", displayName: "1.21", default: true }] : undefined,
        },
      });
    }
    if (url === "/servers/mc-survival" && (!init || init.method === "GET")) return jsonRes(latest ?? gs());
    if (url === "/servers/mc-survival" && init?.method === "PUT") {
      if (putBody) putBody.current = JSON.parse(init.body as string);
      return jsonRes(JSON.parse(init.body as string));
    }
    throw new Error(`unexpected fetch: ${url} ${init?.method ?? "GET"}`);
  });
}

function renderTab(ui: ReactElement, canWrite = true) {
  return baseRenderWithQuery(
    <ResourceTargetProvider
      target={{ cluster: "local", name: "mc-survival" }}
      access={{ canWrite, canControl: canWrite, canConsole: canWrite, canDelete: canWrite, isOwner: canWrite, isCollaborator: false, permissions: canWrite ? ["*"] : ["servers:read"] }}
    >
      {ui}
    </ResourceTargetProvider>,
  );
}

describe("SettingsTab game configuration section", () => {
  it("appears after Version and before Resources when the template has a configSchema", async () => {
    stubTemplate({ schema: SCHEMA, versions: true });
    renderTab(<SettingsTab gs={gs()} name="mc-survival" />);
    await screen.findByRole("tab", { name: /Game configuration/i });
    const names = screen.getAllByRole("tab").map((t) => t.textContent ?? "");
    const v = names.findIndex((n) => n.includes("Version"));
    const c = names.findIndex((n) => n.includes("Game configuration"));
    const r = names.findIndex((n) => n.includes("Resources"));
    expect(v).toBeGreaterThanOrEqual(0);
    expect(c).toBe(v + 1);
    expect(r).toBe(c + 1);
  });

  it("is hidden when the template has no schema and the server has no config keys", async () => {
    stubTemplate({ schema: [] });
    renderTab(<SettingsTab gs={gs()} name="mc-survival" />);
    await waitFor(() => expect(fetchMock).toHaveBeenCalled());
    await screen.findByRole("tab", { name: /Resources/i });
    expect(screen.queryByRole("tab", { name: /Game configuration/i })).toBeNull();
  });

  it("is shown for a schemaless template when the server still has stray config keys", async () => {
    stubTemplate({ schema: [] });
    renderTab(<SettingsTab gs={gs({ OLD_SETTING: "legacy" })} name="mc-survival" />);
    fireEvent.click(await screen.findByRole("tab", { name: /Game configuration/i }));
    expect(await screen.findByText("OLD_SETTING")).toBeInTheDocument();
  });

  it("saves edited config and echoes the stored-password marker so the API keeps it", async () => {
    const put: { current: unknown } = { current: null };
    stubTemplate({ schema: SCHEMA }, put, gs({ SERVER_PASSWORD: CONFIG_REDACTED_MARKER }));
    renderTab(<SettingsTab gs={gs({ SERVER_PASSWORD: CONFIG_REDACTED_MARKER })} name="mc-survival" />);
    fireEvent.click(await screen.findByRole("tab", { name: /Game configuration/i }));
    fireEvent.change(await screen.findByLabelText(/Max players/), { target: { value: "8" } });
    fireEvent.click(screen.getByRole("button", { name: /Save changes/i }));
    await waitFor(() => expect(put.current).not.toBeNull());
    expect((put.current as GameServer).spec.config).toEqual({ MAX_PLAYERS: "8", SERVER_PASSWORD: CONFIG_REDACTED_MARKER });
  });

  it("disables Save while a value is invalid", async () => {
    stubTemplate({ schema: SCHEMA });
    renderTab(<SettingsTab gs={gs()} name="mc-survival" />);
    fireEvent.click(await screen.findByRole("tab", { name: /Game configuration/i }));
    fireEvent.change(await screen.findByLabelText(/Max players/), { target: { value: "900" } });
    expect(await screen.findByText("Must be between 1 and 255.")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /Save changes/i })).toBeDisabled();
  });

  it("is read-only without write access: note instead of Discard/Save", async () => {
    stubTemplate({ schema: SCHEMA });
    renderTab(<SettingsTab gs={gs()} name="mc-survival" />, false);
    fireEvent.click(await screen.findByRole("tab", { name: /Game configuration/i }));
    expect(await screen.findByText("You need permission to change this server's settings.")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /Save changes/i })).toBeNull();
    expect(screen.queryByRole("button", { name: /Discard/i })).toBeNull();
    expect(screen.getByLabelText(/Max players/)).toBeDisabled();
  });
});
