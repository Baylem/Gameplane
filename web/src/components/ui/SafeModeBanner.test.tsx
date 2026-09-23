import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { userEvent } from "@testing-library/user-event";
import { SafeModeBanner } from "./SafeModeBanner";

describe("SafeModeBanner", () => {
  it("renders the safe-mode alert with the suspended-CSS message", () => {
    render(<SafeModeBanner onOpenSettings={() => {}} onDismiss={() => {}} />);
    expect(screen.getByRole("alert")).toHaveTextContent(
      "Safe Mode Active: Custom CSS is suspended.",
    );
  });

  it("calls onOpenSettings when 'Open Appearance Settings' is clicked", async () => {
    const onOpenSettings = vi.fn();
    const user = userEvent.setup();
    render(<SafeModeBanner onOpenSettings={onOpenSettings} onDismiss={() => {}} />);
    await user.click(screen.getByRole("button", { name: "Open Appearance Settings" }));
    expect(onOpenSettings).toHaveBeenCalledOnce();
  });

  it("calls onDismiss when 'Dismiss' is clicked", async () => {
    const onDismiss = vi.fn();
    const user = userEvent.setup();
    render(<SafeModeBanner onOpenSettings={() => {}} onDismiss={onDismiss} />);
    await user.click(screen.getByRole("button", { name: "Dismiss" }));
    expect(onDismiss).toHaveBeenCalledOnce();
  });
});
