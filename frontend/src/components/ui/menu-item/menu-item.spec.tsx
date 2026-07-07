/**
 * Component tests for `<MenuItem>`.
 *
 * Reference: `openspec/changes/cachicamas-button-design-system/specs/frontend-ui-button/spec.md`
 *   R-UB-006 — MenuItem primitive renders panel-row affordances.
 *
 * Class token coverage is in `classes.spec.ts`; this file focuses on
 * the Qwik component layer (props, Slot, default type, disabled).
 */
import { createDOM } from "@builder.io/qwik/testing";
import { describe, it, expect } from "vitest";
import { MenuItem } from "./menu-item";

describe("components/ui/menu-item", () => {
  it("renders a <button> with the panel-row className", async () => {
    const { screen, render } = await createDOM();
    await render(<MenuItem>Profile</MenuItem>);
    const btn = screen.querySelector("button");
    expect(btn).toBeTruthy();
    expect(btn?.className).toContain("block");
    expect(btn?.className).toContain("w-full");
    expect(btn?.className).toContain("text-left");
    expect(btn?.className).toContain("px-2 py-1.5");
    expect(btn?.className).toContain("text-sm");
  });

  it("has cursor-pointer (R-UB-002)", async () => {
    const { screen, render } = await createDOM();
    await render(<MenuItem>Profile</MenuItem>);
    expect(screen.querySelector("button")?.className).toContain(
      "cursor-pointer",
    );
  });

  it("hover uses bg-slate-100 (panel-row tint, not button surface)", async () => {
    const { screen, render } = await createDOM();
    await render(<MenuItem>Profile</MenuItem>);
    const cls = screen.querySelector("button")?.className ?? "";
    expect(cls).toContain("hover:bg-slate-100");
    expect(cls).not.toContain("hover:bg-slate-50");
    expect(cls).not.toContain("hover:bg-white");
  });

  it("has the transition set + duration-150 + active:translate-y-px", async () => {
    const { screen, render } = await createDOM();
    await render(<MenuItem>Profile</MenuItem>);
    const cls = screen.querySelector("button")?.className ?? "";
    expect(cls).toMatch(/transition-\[background-color/);
    expect(cls).toContain("duration-150");
    expect(cls).toContain("active:translate-y-px");
  });

  it("has indigo focus ring (R-UB-004)", async () => {
    const { screen, render } = await createDOM();
    await render(<MenuItem>Profile</MenuItem>);
    const cls = screen.querySelector("button")?.className ?? "";
    expect(cls).toContain("focus:outline-none");
    expect(cls).toContain("focus-visible:ring-2");
    expect(cls).toContain("focus-visible:ring-indigo-500");
  });

  it("default type is 'button' (R-UB-010)", async () => {
    const { screen, render } = await createDOM();
    await render(<MenuItem>Profile</MenuItem>);
    expect(screen.querySelector("button")?.getAttribute("type")).toBe(
      "button",
    );
  });

  it("type='submit' override is honored", async () => {
    const { screen, render } = await createDOM();
    await render(<MenuItem type="submit">Sign out</MenuItem>);
    expect(screen.querySelector("button")?.getAttribute("type")).toBe(
      "submit",
    );
  });

  it("disabled=true renders the disabled attribute + disabled tokens", async () => {
    const { screen, render } = await createDOM();
    await render(<MenuItem disabled={true}>Settings (soon)</MenuItem>);
    const btn = screen.querySelector("button");
    expect(btn?.hasAttribute("disabled")).toBe(true);
    expect(btn?.className).toContain("disabled:cursor-not-allowed");
    expect(btn?.className).toContain("disabled:opacity-50");
  });

  it("renders Slot children verbatim", async () => {
    const { screen, render } = await createDOM();
    await render(
      <MenuItem>
        <span data-testid="child-span">Hello</span>
      </MenuItem>,
    );
    expect(screen.querySelector('[data-testid="child-span"]')?.textContent).toBe(
      "Hello",
    );
  });

  it("testId prop is rendered as data-testid", async () => {
    const { screen, render } = await createDOM();
    await render(<MenuItem testId="my-row">x</MenuItem>);
    expect(screen.querySelector('[data-testid="my-row"]')).toBeTruthy();
  });
});