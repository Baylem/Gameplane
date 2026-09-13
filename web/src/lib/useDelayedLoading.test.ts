import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { useDelayedLoading } from "./useDelayedLoading";

describe("useDelayedLoading", () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.clearAllTimers();
    vi.useRealTimers();
  });

  it("false stays false", () => {
    const { result } = renderHook(() => useDelayedLoading(false));
    expect(result.current).toBe(false);

    vi.advanceTimersByTime(500);
    expect(result.current).toBe(false);
  });

  it("true for 100ms then false → never shows", () => {
    const { result, rerender } = renderHook(({ loading }) => useDelayedLoading(loading), {
      initialProps: { loading: true },
    });

    expect(result.current).toBe(false);

    // Advance 100ms — not enough to trigger showAfterMs (default 200ms)
    vi.advanceTimersByTime(100);
    expect(result.current).toBe(false);

    // Now stop loading
    rerender({ loading: false });

    // The skeleton should never show since we cancelled before showAfterMs
    expect(result.current).toBe(false);

    vi.advanceTimersByTime(500);
    expect(result.current).toBe(false);
  });

  it("true past 200ms → shows", () => {
    const { result } = renderHook(() => useDelayedLoading(true));

    expect(result.current).toBe(false);

    // Advance past showAfterMs (default 200ms)
    act(() => {
      vi.advanceTimersByTime(200);
    });
    expect(result.current).toBe(true);

    act(() => {
      vi.advanceTimersByTime(100);
    });
    expect(result.current).toBe(true);
  });

  it("true → shows → false → stays visible until 300ms elapsed", () => {
    const { result, rerender } = renderHook(({ loading }) => useDelayedLoading(loading), {
      initialProps: { loading: true },
    });

    expect(result.current).toBe(false);

    // Advance past showAfterMs to trigger skeleton
    act(() => {
      vi.advanceTimersByTime(200);
    });
    expect(result.current).toBe(true);

    // Stop loading
    rerender({ loading: false });

    // Should still be visible immediately after loading ends (minVisibleMs not elapsed yet)
    expect(result.current).toBe(true);

    // Advance 100ms — still within minVisibleMs (default 300ms)
    act(() => {
      vi.advanceTimersByTime(100);
    });
    expect(result.current).toBe(true);

    // Advance to the minVisibleMs boundary (total 300ms)
    act(() => {
      vi.advanceTimersByTime(200);
    });
    expect(result.current).toBe(false);
  });

  it("respects custom showAfterMs", () => {
    const { result } = renderHook(() => useDelayedLoading(true, { showAfterMs: 100 }));

    expect(result.current).toBe(false);

    act(() => {
      vi.advanceTimersByTime(100);
    });
    expect(result.current).toBe(true);
  });

  it("respects custom minVisibleMs", () => {
    const { result, rerender } = renderHook(({ loading }) => useDelayedLoading(loading, { minVisibleMs: 100 }), {
      initialProps: { loading: true },
    });

    // Trigger skeleton
    act(() => {
      vi.advanceTimersByTime(200);
    });
    expect(result.current).toBe(true);

    // Stop loading
    rerender({ loading: false });

    // Should still be visible
    expect(result.current).toBe(true);

    // Advance exactly 100ms (custom minVisibleMs)
    act(() => {
      vi.advanceTimersByTime(100);
    });
    expect(result.current).toBe(false);
  });

  it("clears timers on unmount", () => {
    const { unmount } = renderHook(() => useDelayedLoading(true));

    expect(() => {
      unmount();
    }).not.toThrow();

    // No pending timers should remain
    expect(vi.getTimerCount()).toBe(0);
  });
});
