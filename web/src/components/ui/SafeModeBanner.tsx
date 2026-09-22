import { TriangleAlert } from "lucide-react";

export interface SafeModeBannerProps {
  /** Navigates to the theme settings page (/settings/theme). */
  onOpenSettings: () => void;
  /** Hides the banner. Safe mode itself stays active for the session. */
  onDismiss: () => void;
}

/**
 * Safe-mode banner (contracts/theme-ui.md §4, design frame DAz77). Shown
 * while safe mode is active — via `?safe-mode=1`, the Ctrl+Shift+Alt+T
 * shortcut, or the login-page safe-mode link. Only the custom CSS overlay is
 * suspended; the stored stylesheet is kept, and "Open Appearance Settings"
 * leads to the theme settings page where it can be edited or cleared.
 */
export function SafeModeBanner({ onOpenSettings, onDismiss }: SafeModeBannerProps) {
  return (
    <div
      role="alert"
      className="flex w-full items-center gap-3 rounded-lg border border-warning-soft-foreground bg-warning-soft p-4"
    >
      <TriangleAlert aria-hidden="true" className="h-5 w-5 shrink-0 text-warning-soft-foreground" />
      <div className="flex min-w-0 flex-1 flex-col gap-2">
        <p className="text-sm font-medium text-warning-soft-foreground">
          Safe Mode Active: Custom CSS is suspended.
        </p>
        <div className="flex items-center gap-2.5 text-sm">
          <button
            type="button"
            onClick={onOpenSettings}
            className="font-medium text-warning-soft-foreground underline-offset-2 hover:underline"
          >
            Open Appearance Settings
          </button>
          <span aria-hidden="true" className="select-none text-warning-soft-foreground/60">
            |
          </span>
          <button
            type="button"
            onClick={onDismiss}
            className="font-medium text-warning-soft-foreground underline-offset-2 hover:underline"
          >
            Dismiss
          </button>
        </div>
      </div>
    </div>
  );
}
