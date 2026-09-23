import { Sun, Moon, Monitor } from "lucide-react";
import { useId, type ReactNode } from "react";
import type { AppearanceMode } from "@/types";

export type { AppearanceMode };

export interface AppearanceToggleProps {
  value: AppearanceMode;
  onChange: (mode: AppearanceMode) => void;
  /** D4: true while a custom-colors theme is active — mode then follows the surface color, not this toggle. */
  disabled?: boolean;
}

/** Three-state appearance toggle: light, dark, system. Controlled component — the caller owns persistence. */
export function AppearanceToggle({ value, onChange, disabled = false }: AppearanceToggleProps) {
  const noteId = useId();
  const modes: Array<{ mode: AppearanceMode; icon: ReactNode; label: string }> = [
    { mode: "light", icon: <Sun className="h-4 w-4" />, label: "Light" },
    { mode: "dark", icon: <Moon className="h-4 w-4" />, label: "Dark" },
    { mode: "system", icon: <Monitor className="h-4 w-4" />, label: "System" },
  ];

  return (
    <div
      role="group"
      aria-label="Appearance"
      aria-describedby={disabled ? noteId : undefined}
      className="flex items-center justify-center gap-1 rounded-lg bg-surface p-1"
    >
      {modes.map(({ mode, icon, label }) => (
        <button
          key={mode}
          type="button"
          aria-label={label}
          aria-pressed={value === mode}
          disabled={disabled}
          title={disabled ? "Set by your surface color" : label}
          onClick={() => onChange(mode)}
          className={`
            flex h-8 w-8 items-center justify-center rounded-md
            transition-colors disabled:cursor-not-allowed disabled:opacity-50
            ${
              value === mode
                ? "bg-primary/20 text-primary"
                : "text-muted hover:bg-border/60 hover:text-fg"
            }
          `}
        >
          {icon}
        </button>
      ))}
      {disabled && (
        <span id={noteId} className="sr-only">
          Set by your surface color
        </span>
      )}
    </div>
  );
}
