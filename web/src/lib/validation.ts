import type { GameTemplate } from "@/types";

export type ConfigField = NonNullable<GameTemplate["spec"]["configSchema"]>[number];

const DNS_LABEL = /^[a-z0-9]([-a-z0-9]*[a-z0-9])?$/;
const QUANTITY = /^(\d+)(\.\d+)?(Ki|Mi|Gi|Ti|Pi|Ei|m|k|M|G|T|P|E)?$/;

export function isValidK8sName(s: string): boolean {
  return s.length > 0 && s.length <= 63 && DNS_LABEL.test(s);
}

export function isValidQuantity(s: string): boolean {
  const trimmed = s.trim();
  if (trimmed === "" || trimmed !== s) return false;
  return QUANTITY.test(trimmed);
}

// isValidVersion checks a chosen version id against a template's catalog.
// When the template declares no versions, the version is irrelevant (true).
// When it does, a value must be supplied and must match a catalog id.
export function isValidVersion(
  template: GameTemplate | undefined,
  version: string | undefined,
): boolean {
  const versions = template?.spec.versions;
  if (!versions || versions.length === 0) return true;
  if (!version) return false;
  return versions.some((v) => v.id === version);
}

// defaultVersionId returns the template's pre-selected version: the entry
// marked default, else the first, else undefined when no catalog exists.
export function defaultVersionId(template: GameTemplate | undefined): string | undefined {
  const versions = template?.spec.versions;
  if (!versions || versions.length === 0) return undefined;
  return (versions.find((v) => v.default) ?? versions[0]).id;
}

export interface ConfigError {
  name: string;
  // Full sentence for step-level summaries (the wizard shows the first one).
  message: string;
  // Text for the inline message under the field (no field name prefix).
  text: string;
}

// CONFIG_REDACTED_MARKER is what the API sends instead of a stored
// password-type spec.config value. Sending it back in a PUT keeps the stored
// value. It must never be rendered.
export const CONFIG_REDACTED_MARKER = "__gameplane_redacted__";

const GO_BOOLS = ["1", "t", "T", "TRUE", "true", "True", "0", "f", "F", "FALSE", "false", "False"];

// fieldValueProblem mirrors the operator's materializeConfig checks for one
// non-empty value. The operator stays authoritative; this only gives early
// feedback.
function fieldValueProblem(field: ConfigField, value: string): string | null {
  switch (field.type) {
    case "int": {
      if (!/^[+-]?\d+$/.test(value)) return "Must be a whole number.";
      const n = Number(value);
      const { min, max } = field;
      if (min != null && max != null && (n < min || n > max)) return `Must be between ${min} and ${max}.`;
      if (min != null && n < min) return `Must be at least ${min}.`;
      if (max != null && n > max) return `Must be at most ${max}.`;
      return null;
    }
    case "bool":
      return GO_BOOLS.includes(value) ? null : "Must be true or false.";
    case "string":
    case "password": {
      const len = new TextEncoder().encode(value).length;
      const { minLength, maxLength } = field;
      if (minLength != null && len < minLength) return `Must be at least ${minLength} characters.`;
      if (maxLength != null && len > maxLength) return `Must be at most ${maxLength} characters.`;
      return null;
    }
    default:
      return null;
  }
}

export function validateConfig(
  schema: ConfigField[],
  values: Record<string, string>,
): ConfigError[] {
  const errors: ConfigError[] = [];
  for (const field of schema) {
    const raw = values[field.name];
    const provided = raw ?? field.default ?? "";
    const label = field.displayName ?? field.name;
    if (field.required && provided === "") {
      const message = `${label} is required`;
      errors.push({ name: field.name, message, text: message });
      continue;
    }
    if (provided === "") continue;
    // A stored password reported by the API is unchanged: nothing to check.
    if (field.type === "password" && provided === CONFIG_REDACTED_MARKER) continue;
    if (field.type === "enum" && field.enum && !field.enum.includes(provided)) {
      const message = `${label} must be one of: ${field.enum.join(", ")}`;
      errors.push({ name: field.name, message, text: message });
      continue;
    }
    const problem = fieldValueProblem(field, provided);
    if (problem) {
      errors.push({ name: field.name, message: `${label}: ${problem}`, text: problem });
    }
  }
  return errors;
}

// PASSWORD_MASK is shown in place of a password-type template config value
// anywhere config values are displayed. The API itself never returns stored
// passwords (it sends a fixed redaction marker), so the dashboard must not
// render either the value or the marker.
export const PASSWORD_MASK = "********";

// maskPasswordConfig returns a copy of values with every non-empty value of a
// password-type field replaced by PASSWORD_MASK. Empty values stay empty.
export function maskPasswordConfig(
  schema: ConfigField[],
  values: Record<string, string>,
): Record<string, string> {
  const passwords = new Set(schema.filter((f) => f.type === "password").map((f) => f.name));
  const out: Record<string, string> = {};
  for (const [name, value] of Object.entries(values)) {
    out[name] = passwords.has(name) && value !== "" ? PASSWORD_MASK : value;
  }
  return out;
}
