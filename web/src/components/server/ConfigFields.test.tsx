import { describe, expect, it, vi } from "vitest";
import { fireEvent, render, screen } from "@testing-library/react";

import { ConfigFieldInput, ConfigFields, STORED_PASSWORD_PLACEHOLDER } from "./ConfigFields";
import { CONFIG_REDACTED_MARKER, type ConfigField } from "@/lib/validation";

const enumField: ConfigField = { name: "DIFFICULTY", displayName: "Difficulty", type: "enum", enum: ["easy", "hard"], default: "easy" };
const boolField: ConfigField = { name: "PVP", displayName: "PvP", type: "bool" };
const intField: ConfigField = { name: "MAX_PLAYERS", displayName: "Max players", type: "int", default: "16", required: true, description: "Slots" };
const strField: ConfigField = { name: "MAX_MEMORY", type: "string", autoFromMemoryLimit: { percent: 75 } };
const passField: ConfigField = { name: "PASS", displayName: "Server password", type: "password" };
const fileField: ConfigField = { name: "MOTD", displayName: "MOTD", type: "string", target: "file" };

describe("ConfigFieldInput (wizard variant)", () => {
  it("renders an enum as a select with its options and default", () => {
    const onChange = vi.fn();
    render(<ConfigFieldInput field={enumField} value={undefined} onChange={onChange} />);
    const select = screen.getByRole("combobox");
    expect(select).toHaveValue("easy");
    expect(screen.getByRole("option", { name: "hard" })).toBeInTheDocument();
    fireEvent.change(select, { target: { value: "hard" } });
    expect(onChange).toHaveBeenCalledWith("hard");
  });

  it("renders a bool as a true/false select defaulting to false", () => {
    const onChange = vi.fn();
    render(<ConfigFieldInput field={boolField} value={undefined} onChange={onChange} />);
    const select = screen.getByRole("combobox");
    expect(select).toHaveValue("false");
    fireEvent.change(select, { target: { value: "true" } });
    expect(onChange).toHaveBeenCalledWith("true");
  });

  it("uses the schema default as the value, a numeric keyboard for ints, and shows the description", () => {
    const onChange = vi.fn();
    render(<ConfigFieldInput field={intField} value={undefined} onChange={onChange} />);
    const input = screen.getByRole("textbox");
    expect(input).toHaveValue("16");
    expect(input).toHaveAttribute("inputmode", "numeric");
    expect(screen.getByText("Slots")).toBeInTheDocument();
    // wizard variant has no required marker, notes or inline errors
    expect(screen.queryByText("*", { exact: false })).toBeNull();
    fireEvent.change(input, { target: { value: "20" } });
    expect(onChange).toHaveBeenCalledWith("20");
  });

  it("prefers the explicit value over the default", () => {
    render(<ConfigFieldInput field={intField} value="8" onChange={() => {}} />);
    expect(screen.getByRole("textbox")).toHaveValue("8");
  });

  it("shows the autoFromMemoryLimit placeholder", () => {
    render(<ConfigFieldInput field={strField} value={undefined} onChange={() => {}} />);
    expect(screen.getByPlaceholderText("Auto: 75% of the memory limit")).toBeInTheDocument();
  });

  it("renders a password as type=password and shows what was typed", () => {
    const { container } = render(<ConfigFieldInput field={passField} value="hunter2" onChange={() => {}} />);
    const input = container.querySelector('input[type="password"]');
    expect(input).not.toBeNull();
    expect(input).toHaveValue("hunter2");
    expect(screen.queryByText("Write-only")).toBeNull();
  });

  it("does not render the file note or errors", () => {
    render(<ConfigFieldInput field={fileField} value="x" onChange={() => {}} error="boom" />);
    expect(screen.queryByText("Written to file")).toBeNull();
    expect(screen.queryByRole("alert")).toBeNull();
  });
});

describe("ConfigFieldInput (settings variant)", () => {
  it("labels the control, marks required fields and notes file targets", () => {
    render(<ConfigFieldInput field={{ ...fileField, required: true }} value="hi" onChange={() => {}} variant="settings" />);
    expect(screen.getByLabelText(/^MOTD/)).toHaveValue("hi");
    expect(screen.getByText("*", { exact: false })).toBeInTheDocument();
    expect(screen.getByText("Written to file")).toBeInTheDocument();
  });

  it("shows a stored password as an empty write-only input and never renders the marker", () => {
    const onChange = vi.fn();
    const { container } = render(
      <ConfigFieldInput field={passField} value={CONFIG_REDACTED_MARKER} onChange={onChange} variant="settings" />,
    );
    const input = screen.getByLabelText(/Server password/);
    expect(input).toHaveValue("");
    expect(input).toHaveAttribute("placeholder", STORED_PASSWORD_PLACEHOLDER);
    expect(screen.getByText("Write-only")).toBeInTheDocument();
    expect(container.innerHTML).not.toContain(CONFIG_REDACTED_MARKER);
    fireEvent.change(input, { target: { value: "s3cret" } });
    expect(onChange).toHaveBeenCalledWith("s3cret");
  });

  it("shows a typed (non-marker) password and an unset password as empty with no stored placeholder", () => {
    const { rerender } = render(<ConfigFieldInput field={passField} value="typed" onChange={() => {}} variant="settings" />);
    expect(screen.getByLabelText(/Server password/)).toHaveValue("typed");
    rerender(<ConfigFieldInput field={passField} value={undefined} onChange={() => {}} variant="settings" />);
    expect(screen.getByLabelText(/Server password/)).toHaveValue("");
    expect(screen.getByLabelText(/Server password/)).not.toHaveAttribute("placeholder");
  });

  it("renders an inline error and marks the control invalid", () => {
    render(<ConfigFieldInput field={intField} value="900" onChange={() => {}} variant="settings" error="Must be between 1 and 255." />);
    expect(screen.getByRole("alert")).toHaveTextContent("Must be between 1 and 255.");
    expect(screen.getByLabelText(/Max players/)).toHaveAttribute("aria-invalid", "true");
  });

  it("marks selects invalid too and honours disabled", () => {
    render(<ConfigFieldInput field={enumField} value="easy" onChange={() => {}} variant="settings" disabled error="bad" />);
    const select = screen.getByLabelText(/Difficulty/);
    expect(select).toBeDisabled();
    expect(select).toHaveAttribute("aria-invalid", "true");
  });
});

describe("ConfigFields", () => {
  it("renders one row per field with values and per-field errors, and reports the field name on change", () => {
    const onChange = vi.fn();
    render(
      <ConfigFields
        schema={[intField, strField]}
        values={{ MAX_PLAYERS: "8" }}
        onChange={onChange}
        variant="settings"
        errors={{ MAX_PLAYERS: "nope" }}
      />,
    );
    expect(screen.getByLabelText(/Max players/)).toHaveValue("8");
    expect(screen.getByRole("alert")).toHaveTextContent("nope");
    fireEvent.change(screen.getByLabelText(/Max players/), { target: { value: "9" } });
    expect(onChange).toHaveBeenCalledWith("MAX_PLAYERS", "9");
  });

  it("defaults to the wizard variant and renders nothing for an empty schema", () => {
    const { container } = render(<ConfigFields schema={[]} values={{}} onChange={() => {}} />);
    expect(container.innerHTML).toBe("");
  });
});
