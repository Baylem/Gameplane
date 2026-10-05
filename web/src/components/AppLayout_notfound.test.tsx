import { describe, expect, it, vi } from "vitest";
import type { ReactNode } from "react";
import { screen } from "@testing-library/react";
import { renderWithQuery } from "@/test/render";

vi.mock("@tanstack/react-router", () => ({
  Link: ({ children, to, ...rest }: { children: ReactNode; to: string } & Record<string, unknown>) => (
    <a href={to} {...rest}>{children}</a>
  ),
  Outlet: () => <div data-testid="outlet">outlet</div>,
  useLocation: () => ({ pathname: "/definitely/not/here" }),
  useMatches: () => [{ routeId: "/app-layout/$" }],
  useNavigate: () => vi.fn(),
}));

import { AppLayout } from "./AppLayout";

describe("AppLayout on the 404 route", () => {
  it("shows a fixed 'Page not found' breadcrumb instead of the unknown URL segments", async () => {
    renderWithQuery(<AppLayout />);
    const crumbs = await screen.findByRole("navigation", { name: "Breadcrumb" });
    expect(crumbs).toHaveTextContent("gameplane");
    expect(crumbs).toHaveTextContent("Page not found");
    expect(crumbs).not.toHaveTextContent("definitely");
  });
});
