import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { http, HttpResponse } from "msw";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { server } from "@/test/server";
import { renderWithQuery } from "@/test/render";
import { getCurrentCluster, setCurrentCluster } from "@/lib/cluster";

const navigate = vi.hoisted(() => vi.fn());
vi.mock("@tanstack/react-router", () => ({ useNavigate: () => navigate }));
import { ClustersPage } from "./Clusters";

const registrations = [
  { name: "local", displayName: "Central", phase: "Healthy", canViewInventory: true },
  { name: "east", displayName: "East", phase: "Unhealthy", canViewInventory: false },
];

describe("ClustersPage", () => {
  beforeEach(() => { setCurrentCluster("local"); navigate.mockReset(); });
  afterEach(() => setCurrentCluster("local"));

  it("lists API health and offers server access separately from inventory capability", async () => {
    server.use(http.get("/clusters", () => HttpResponse.json({ items: registrations })));
    renderWithQuery(<ClustersPage />);
    expect(await screen.findByRole("heading", { name: "East" })).toBeInTheDocument();
    expect(screen.getByText("API: Unhealthy")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "View servers in East" })).toBeEnabled();
    expect(screen.queryByRole("button", { name: "View nodes in East" })).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "View nodes in Central" })).toBeEnabled();
    expect(screen.getByText(/Kubernetes API connectivity/)).toBeInTheDocument();
  });

  it("changes context before opening the selected cluster's servers", async () => {
    server.use(http.get("/clusters", () => HttpResponse.json({ items: registrations })));
    const { client } = renderWithQuery(<ClustersPage />);
    client.setQueryData(["server", "same-name"], { uid: "old-local-uid" });
    await userEvent.click(await screen.findByRole("button", { name: "View servers in East" }));
    await waitFor(() => expect(navigate).toHaveBeenCalledWith({ to: "/servers" }));
    expect(getCurrentCluster()).toBe("east");
    expect(client.getQueryData(["server", "same-name"])).toBeUndefined();
  });

  it("opens inventory only for an explicitly capable registration", async () => {
    server.use(http.get("/clusters", () => HttpResponse.json({ items: [{ ...registrations[1], canViewInventory: true }] })));
    renderWithQuery(<ClustersPage />);
    await userEvent.click(await screen.findByRole("button", { name: "View nodes in East" }));
    await waitFor(() => expect(navigate).toHaveBeenCalledWith({ to: "/cluster" }));
    expect(getCurrentCluster()).toBe("east");
  });

  it("does not invent a local registration for empty or failed registry responses", async () => {
    server.use(http.get("/clusters", () => HttpResponse.json({ items: [] })));
    const { unmount } = renderWithQuery(<ClustersPage />);
    expect(await screen.findByText("No clusters available")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /View servers/ })).not.toBeInTheDocument();
    unmount();
    server.use(http.get("/clusters", () => HttpResponse.error()));
    renderWithQuery(<ClustersPage />);
    expect(await screen.findByText(/Couldn't load registered clusters/)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Retry" })).toBeInTheDocument();
  });
});
