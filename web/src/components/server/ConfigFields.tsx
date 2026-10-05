import { useId, type ReactNode } from "react";
import { Input } from "@heroui/react";

import { cn } from "@/lib/utils";
import { CONFIG_REDACTED_MARKER, type ConfigField } from "@/lib/validation";

// ConfigFields renders a template's configSchema as form fields. It is the
// single implementation shared by the Create wizard (variant "wizard", the
// original stacked layout) and Settings > Game configuration (variant
// "settings": label-left rows, required marker, write-only passwords, file
// note, inline errors). No per-game branching: everything comes from the
// schema.

export type ConfigFieldVariant = "wizard" | "settings";

export const STORED_PASSWORD_PLACEHOLDER = "Unchanged — type to replace";

const SELECT_BASE = "h-9 w-full rounded-md border border-border bg-surface px-3 text-sm";

export interface ConfigFieldInputProps {
  field: ConfigField;
  /** Raw spec.config value; undefined when the key is unset. */
  value: string | undefined;
  onChange: (value: string) => void;
  variant?: ConfigFieldVariant;
  disabled?: boolean;
  /** Inline error text (rendered by the settings variant only). */
  error?: string;
}

export function ConfigFieldInput({
  field,
  value,
  onChange,
  variant = "wizard",
  disabled = false,
  error,
}: ConfigFieldInputProps) {
  const id = useId();
  const errorId = `${id}-error`;
  const label = field.displayName ?? field.name;
  const settings = variant === "settings";
  const invalid = settings && error !== undefined && error !== "";
  // Write-only: while the API reports a stored password (the redaction
  // marker) the input is empty and the marker is never rendered.
  const stored = settings && field.type === "password" && value === CONFIG_REDACTED_MARKER;

  const common = {
    id,
    disabled,
    "aria-invalid": invalid ? true : undefined,
    "aria-describedby": invalid ? errorId : undefined,
  } as const;

  let control: ReactNode;
  if (field.type === "enum") {
    control = (
      <select
        {...common}
        className={cn(SELECT_BASE, invalid && "border-danger", disabled && "opacity-60")}
        value={value ?? field.default ?? ""}
        onChange={(e) => onChange(e.target.value)}
      >
        {field.enum?.map((v) => (
          <option key={v} value={v}>
            {v}
          </option>
        ))}
      </select>
    );
  } else if (field.type === "bool") {
    control = (
      <select
        {...common}
        className={cn(SELECT_BASE, invalid && "border-danger", disabled && "opacity-60")}
        value={value ?? field.default ?? "false"}
        onChange={(e) => onChange(e.target.value)}
      >
        <option value="true">true</option>
        <option value="false">false</option>
      </select>
    );
  } else {
    const shown = stored ? "" : (value ?? field.default ?? "");
    const placeholder = stored
      ? STORED_PASSWORD_PLACEHOLDER
      : field.autoFromMemoryLimit
        ? `Auto: ${field.autoFromMemoryLimit.percent}% of the memory limit`
        : undefined;
    control = (
      <Input
        {...common}
        type={field.type === "password" ? "password" : "text"}
        inputMode={field.type === "int" ? "numeric" : undefined}
        className={invalid ? "border-danger" : undefined}
        value={shown}
        placeholder={placeholder}
        onChange={(e) => onChange(e.target.value)}
      />
    );
  }

  if (!settings) {
    return (
      <label className="space-y-1.5 block">
        <div className="text-xs text-muted">{label}</div>
        {control}
        {field.description && <span className="text-[11px] text-muted">{field.description}</span>}
      </label>
    );
  }

  return (
    <div className="grid grid-cols-1 items-start gap-1.5 sm:grid-cols-[200px_1fr] sm:gap-4">
      <label htmlFor={id} className="text-sm text-fg sm:pt-2">
        {label}
        {field.required && (
          <span aria-hidden="true" className="text-danger">
            {" *"}
          </span>
        )}
      </label>
      <div className="space-y-1">
        {control}
        {field.description && <p className="text-xs text-muted">{field.description}</p>}
        {field.type === "password" && <p className="text-xs text-muted">Write-only</p>}
        {field.target === "file" && <p className="text-xs text-muted">Written to file</p>}
        {invalid && (
          <p id={errorId} role="alert" className="text-xs text-danger">
            {error}
          </p>
        )}
      </div>
    </div>
  );
}

export interface ConfigFieldsProps {
  schema: ConfigField[];
  values: Record<string, string>;
  onChange: (name: string, value: string) => void;
  variant?: ConfigFieldVariant;
  disabled?: boolean;
  errors?: Record<string, string>;
}

export function ConfigFields({
  schema,
  values,
  onChange,
  variant = "wizard",
  disabled = false,
  errors,
}: ConfigFieldsProps) {
  return (
    <>
      {schema.map((f) => (
        <ConfigFieldInput
          key={f.name}
          field={f}
          value={values[f.name]}
          onChange={(v) => onChange(f.name, v)}
          variant={variant}
          disabled={disabled}
          error={errors?.[f.name]}
        />
      ))}
    </>
  );
}
