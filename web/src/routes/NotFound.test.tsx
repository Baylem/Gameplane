import { describe, expect, it, vi } from "vitest";
import type { ReactNode } from "react";
import { render, screen } from "@testing-library/react";

vi.mock("@tanstack/react-router", () => ({
  Link: ({ children, to, ...rest }: { children: ReactNode; to: string } & Record<string, unknown>) => (
    <a href={to} {...rest}>{children}</a>
  ),
}));

import { NotFoundPage } from "./NotFound";

describe("NotFoundPage", () => {
  it("explains the problem and links back to the dashboard", () => {
    render(<NotFoundPage />);
    expect(screen.getByRole("heading", { name: "Page not found" })).toBeInTheDocument();
    expect(screen.getByText("The page you're looking for doesn't exist or has moved.")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Go to dashboard" })).toHaveAttribute("href", "/");
  });

  it("shows no cluster, server or version information", () => {
    const { container } = render(<NotFoundPage />);
    expect(container.textContent).toBe(
      "Page not foundThe page you're looking for doesn't exist or has moved.Go to dashboard",
    );
  });
});
