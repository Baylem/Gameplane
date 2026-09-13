import { useEffect, useRef, useState } from "react";

/**
 * Debounces a loading boolean so that transient loading states don't flash a skeleton.
 *
 * @param loading - the current loading state
 * @param options.showAfterMs - delay before showing the skeleton (default 200ms)
 * @param options.minVisibleMs - minimum duration the skeleton stays visible once shown (default 300ms)
 * @param options.initialImmediate - show skeleton immediately on initial load without showAfterMs delay,
 *        but still honour minVisibleMs when loading ends (default true)
 * @returns the debounced loading state: true only if loading persists past showAfterMs
 *          (or immediately on first mount if initialImmediate=true), and once true stays true for at least minVisibleMs
 */
export function useDelayedLoading(
  loading: boolean,
  { showAfterMs = 200, minVisibleMs = 300, initialImmediate = true } = {},
): boolean {
  // Capture the loading state at mount to detect true initial mounts
  const mountedLoadingRef = useRef(loading);
  const hasFinishedLoadingRef = useRef(false);

  // Lazy-initialize state: show immediately on first render if initialImmediate
  // is true AND the very first render has loading === true
  const [showSkeleton, setShowSkeleton] = useState(() => initialImmediate && loading);

  useEffect(() => {
    if (!loading) {
      // If we showed the skeleton, mark that we've finished a load cycle
      if (showSkeleton) {
        hasFinishedLoadingRef.current = true;
      }

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

    // If skeleton is already showing, nothing to do
    if (showSkeleton) {
      return;
    }

    // If this is the initial load (first mount had loading === true) and initialImmediate is true,
    // the skeleton is already shown via lazy-init, so nothing to do. This only applies to that
    // first mount-in-loading — once a load cycle has finished, a refetch must go through the
    // normal debounce below, not bail out here.
    if (initialImmediate && mountedLoadingRef.current && !hasFinishedLoadingRef.current) {
      return;
    }

    // Otherwise (subsequent loads or initialImmediate=false) use the normal debounce
    const showTimer = setTimeout(() => {
      setShowSkeleton(true);
    }, showAfterMs);

    return () => clearTimeout(showTimer);
  }, [loading, showSkeleton, showAfterMs, minVisibleMs, initialImmediate]);

  return showSkeleton;
}
