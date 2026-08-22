/**
 * The form field vocabulary.
 *
 * One assertion here matters more than the rest: an input's border must clear
 * 3:1 against every ground it can land on, because on a white surface the
 * border IS how a person finds the field (WCAG 1.4.11). The softer decorative
 * line value is a real and easy mistake to make here, so it is pinned out.
 */
import { describe, it, expect } from "vitest";
import {
  FORM_ERROR,
  FORM_FIELDSET,
  FORM_HELP,
  FORM_INPUT,
  FORM_LABEL,
  FORM_LEGEND,
} from "./classes";

const ALL = [
  FORM_LABEL,
  FORM_INPUT,
  FORM_HELP,
  FORM_ERROR,
  FORM_FIELDSET,
  FORM_LEGEND,
].join(" ");

describe("components/ui/form — className table", () => {
  it("finds an input by its border, not by a decorative hairline", () => {
    expect(FORM_INPUT).toContain("border-line-control");
    expect(FORM_INPUT).not.toMatch(/\bborder-line(?!-control)\b/);
    expect(FORM_INPUT).not.toContain("border-transparent");
  });

  it("gives every field the product's radius and surface", () => {
    expect(FORM_INPUT).toContain("rounded-md");
    expect(FORM_INPUT).toContain("bg-surface");
    expect(FORM_INPUT).toContain("w-full");
  });

  it("states the disabled affordance rather than leaving it to the browser", () => {
    expect(FORM_INPUT).toContain("disabled:cursor-not-allowed");
    expect(FORM_INPUT).toContain("disabled:bg-sunken");
  });

  it("restyles no focus — global.css owns the one treatment", () => {
    expect(ALL).not.toMatch(/focus:/);
    expect(ALL).not.toMatch(/\bring-/);
  });

  it("separates help from error by colour AND weight", () => {
    // An error that differs from help text only in hue is an error somebody
    // will read as a hint.
    expect(FORM_HELP).toContain("text-ink-soft");
    expect(FORM_ERROR).toContain("text-stop");
    expect(FORM_ERROR).toContain("font-medium");
    expect(FORM_HELP).not.toContain("font-medium");
  });

  it("names no colour outside the product's tokens", () => {
    expect(ALL).not.toMatch(
      /\b(slate|zinc|gray|neutral|stone|indigo|red|green|blue)-\d/,
    );
    expect(ALL).not.toMatch(/gradient/);
  });
});
