/**
 * Field — one labelled reading.
 *
 * The leader between label and value is decoration in the strict sense, so it
 * must be hidden from assistive technology; the label and value must not be.
 */
import { createDOM } from "@builder.io/qwik/testing";
import { describe, it, expect } from "vitest";
import { Field } from "./field";

describe("components/os/field", () => {
  it("renders its label and its value", async () => {
    const { screen, render } = await createDOM();
    await render(
      <Field label="Milestones" testId="f">
        24/24
      </Field>,
    );
    const text = screen.querySelector('[data-testid="f"]')?.textContent ?? "";
    expect(text).toContain("Milestones");
    expect(text).toContain("24/24");
  });

  it("hides the dotted leader from assistive technology", async () => {
    const { screen, render } = await createDOM();
    await render(
      <Field label="Plan" testId="f">
        doc 0005
      </Field>,
    );
    const leader = screen
      .querySelector('[data-testid="f"]')
      ?.querySelector('[aria-hidden="true"]');
    expect(leader).toBeTruthy();
    expect(leader?.textContent).toBe("");
  });

  it("lets the value be markup, not just a string", async () => {
    const { screen, render } = await createDOM();
    await render(
      <Field label="Plan">
        <span data-testid="rich">doc 0005</span>
      </Field>,
    );
    expect(screen.querySelector('[data-testid="rich"]')).toBeTruthy();
  });
});
