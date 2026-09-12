import { useEffect, useState } from "react";

/**
 * Debounces a loading boolean so that transient loading states don't flash a skeleton.
 *
 * @param loading - the current loading state
 * @param options.showAfterMs - delay before showing the skeleton (default 200ms)
 * @param options.minVisibleMs - minimum duration the skeleton stays visible once shown (default 300ms)
 * @returns the debounced loading state: true only if loading persists past showAfterMs,
 *          and once true stays true for at least minVisibleMs
 */
export function useDelayedLoading(
  loading: boolean,
  { showAfterMs = 200, minVisibleMs = 300 } = {},
): boolean {
  const [showSkeleton, setShowSkeleton] = useState(false);

  useEffect(() => {
    if (!loading) {
      // If we're not loading but the skeleton is showing, keep it visible for minVisibleMs
      if (showSkeleton) {
        const minVisibleTimer = setTimeout(() => {
          setShowSkeleton(false);
        }, minVisibleMs);

        return () => clearTimeout(minVisibleTimer);
      }
      return;
    }

    // loading is true
    const showTimer = setTimeout(() => {
      setShowSkeleton(true);
    }, showAfterMs);

    return () => clearTimeout(showTimer);
  }, [loading, showSkeleton, showAfterMs, minVisibleMs]);

  return showSkeleton;
}
