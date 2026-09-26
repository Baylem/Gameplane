import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { ReactNode } from "react";
import { render, screen } from "@testing-library/react";
import { RequirePermission } from "./RequireRole";
import { APIError } from "@/lib/api";

const useMeMock = vi.fn();
// Keep the real can(); only stub useMe.
vi.mock("@/lib/auth", async (orig) => ({
  ...(await orig<typeof import("@/lib/auth")>()),
  useMe: () => useMeMock(),
}));

vi.mock("@tanstack/react-router", () => ({
  Link: ({ children, ...rest }: { children: ReactNode } & Record<string, unknown>) => (
    <a {...rest}>{children}</a>
  ),
}));

const assignMock = vi.fn();
const refetchMock = vi.fn();
beforeEach(() => {
  vi.stubGlobal("location", { assign: assignMock });
});
afterEach(() => {
  useMeMock.mockReset();
  assignMock.mockReset();
  refetchMock.mockReset();
  vi.unstubAllGlobals();
});

describe("RequirePermission", () => {
  function renderPerm(perm: string) {
    return render(
      <RequirePermission perm={perm}>
        <div>secret</div>
      </RequirePermission>,
    );
  }

  it("renders children when the permission is held", () => {
    useMeMock.mockReturnValue({
      data: { id: 1, username: "a", role: "admin", permissions: { "*": ["*"] } },
      error: null,
      isLoading: false,
    });
    renderPerm("users:manage");
    expect(screen.getByText("secret")).toBeInTheDocument();
  });

  it("renders Forbidden when the permission is missing", () => {
    useMeMock.mockReturnValue({
      data: { id: 1, username: "v", role: "viewer", permissions: { "*": ["servers:read"] } },
      error: null,
      isLoading: false,
    });
    renderPerm("users:manage");
    expect(screen.getByText("Access denied")).toBeInTheDocument();
    expect(screen.queryByText("secret")).toBeNull();
  });

  // Regression: a 500 on /users/me used to fall through to Forbidden, so a
  // transient API restart told an admin they had lost their permissions.
  it("offers a retry — not Access denied — when the identity fetch fails", () => {
    useMeMock.mockReturnValue({
      data: undefined,
      error: new APIError(500, "boom"),
      isLoading: false,
      refetch: refetchMock,
    });
    renderPerm("users:manage");
    expect(screen.getByText("Can't reach the control plane")).toBeInTheDocument();
    expect(screen.queryByText("Access denied")).toBeNull();
  });

  it("shows HTTP status in error message", () => {
    useMeMock.mockReturnValue({
      data: undefined,
      error: new APIError(502, "bad gateway"),
      isLoading: false,
      refetch: refetchMock,
    });
    renderPerm("users:manage");
    expect(screen.getByText(/HTTP 502/)).toBeInTheDocument();
  });

  it("RequirePermission redirects 401 to login", () => {
    useMeMock.mockReturnValue({
      data: undefined,
      error: new APIError(401, "unauth"),
      isLoading: false,
    });
    render(
      <RequirePermission perm="users:manage">
        <div>secret</div>
      </RequirePermission>,
    );
    expect(assignMock).toHaveBeenCalledWith("/login");
  });
});
