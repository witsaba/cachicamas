/**
 * The form vocabulary.
 *
 * Forms are rare here, which is exactly why they drift: the next one written
 * will be written months from now by someone who last saw a field in a
 * different design. These constants are the shared answer, and the assertions
 * are the anti-drift guard around them.
 */
import { describe, it, expect } from "vitest";
import {
  FORM_ERROR,
  FORM_FIELDSET,
  FORM_HELP,
  FORM_INPUT,
  FORM_LABEL,
  FORM_LEGEND,
} from "./form-classes";

const ALL = [
  FORM_LABEL,
  FORM_INPUT,
  FORM_HELP,
  FORM_ERROR,
  FORM_FIELDSET,
  FORM_LEGEND,
].join(" ");

describe("components/os/form-classes", () => {
  it("puts the label in the machine voice and the field in the human one", () => {
    // What a person types is language; what the system asks for is a label.
    expect(FORM_LABEL).toContain("uppercase");
    expect(FORM_INPUT).toContain("font-human");
    expect(FORM_INPUT).not.toContain("uppercase");
  });

  it("makes a field a well with a rule, not a raised card", () => {
    expect(FORM_INPUT).toContain("border");
    expect(FORM_INPUT).toContain("bg-raise");
    expect(ALL).not.toMatch(/rounded|shadow-/);
  });

  it("warms the rule to amber on focus, matching every other input", () => {
    expect(FORM_INPUT).toContain("focus:border-amber");
  });

  it("turns the rule to the fail colour when the field is invalid", () => {
    // Driven by `aria-invalid`, so the visual state and the announced state
    // cannot drift apart.
    expect(FORM_INPUT).toContain("aria-invalid:border-fail");
    expect(FORM_ERROR).toContain("text-fail");
  });

  it("keeps help and error visually distinct", () => {
    expect(FORM_HELP).toContain("text-fg-dim");
    expect(FORM_ERROR).not.toContain("text-fg-dim");
  });

  it("carries no slate-era colour", () => {
    expect(ALL).not.toMatch(/slate|zinc|indigo|red-\d|bg-white/);
  });
});
