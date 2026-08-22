/**
 * Component tests for `<MenuItem>`.
 *
 * Reference: `openspec/changes/cachicamas-button-design-system/specs/frontend-ui-button/spec.md`
 *   R-UB-006 — MenuItem primitive renders panel-row affordances.
 *
 * Class token coverage is in `classes.spec.ts`; this file focuses on
 * the Qwik component layer (props, Slot, default type, disabled,
 * polymorphism, class override).
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
    expect(btn?.className).toContain("px-3 py-2");
    expect(btn?.className).toContain("text-label");
  });

  it("has cursor-pointer (R-UB-002)", async () => {
    const { screen, render } = await createDOM();
    await render(<MenuItem>Profile</MenuItem>);
    expect(screen.querySelector("button")?.className).toContain(
      "cursor-pointer",
    );
  });

  it("hover tints the row (bg-raise), it does not reverse a cell", async () => {
    const { screen, render } = await createDOM();
    await render(<MenuItem>Profile</MenuItem>);
    const cls = screen.querySelector("button")?.className ?? "";
    expect(cls).toContain("hover:bg-raise");
  });

  it("transitions colour at the product duration", async () => {
    const { screen, render } = await createDOM();
    await render(<MenuItem>Profile</MenuItem>);
    const cls = screen.querySelector("button")?.className ?? "";
    expect(cls).toMatch(/transition-\[background-color/);
    expect(cls).toContain("duration-150");
  });

  it("does not restyle focus — global.css owns the one treatment", async () => {
    const { screen, render } = await createDOM();
    await render(<MenuItem>Profile</MenuItem>);
    const cls = screen.querySelector("button")?.className ?? "";
    expect(cls).not.toMatch(/ring-/);
    expect(cls).not.toMatch(/focus-visible:outline-/);
  });

  it("default type is 'button' (R-UB-010)", async () => {
    const { screen, render } = await createDOM();
    await render(<MenuItem>Profile</MenuItem>);
    expect(screen.querySelector("button")?.getAttribute("type")).toBe("button");
  });

  it("type='submit' override is honored", async () => {
    const { screen, render } = await createDOM();
    await render(<MenuItem type="submit">Sign out</MenuItem>);
    expect(screen.querySelector("button")?.getAttribute("type")).toBe("submit");
  });

  it("disabled=true renders the disabled attribute + disabled tokens", async () => {
    const { screen, render } = await createDOM();
    await render(<MenuItem disabled={true}>Settings (soon)</MenuItem>);
    const btn = screen.querySelector("button");
    expect(btn?.hasAttribute("disabled")).toBe(true);
    expect(btn?.className).toContain("disabled:cursor-not-allowed");
    expect(btn?.className).toContain("disabled:opacity-40");
  });

  it("renders Slot children verbatim", async () => {
    const { screen, render } = await createDOM();
    await render(
      <MenuItem>
        <span data-testid="child-span">Hello</span>
      </MenuItem>,
    );
    expect(
      screen.querySelector('[data-testid="child-span"]')?.textContent,
    ).toBe("Hello");
  });

  it("testId prop is rendered as data-testid", async () => {
    const { screen, render } = await createDOM();
    await render(<MenuItem testId="my-row">x</MenuItem>);
    expect(screen.querySelector('[data-testid="my-row"]')).toBeTruthy();
  });

  describe("class override (consumer personalization)", () => {
    it("consumer class is appended to MENU_ITEM_BASE", async () => {
      const { screen, render } = await createDOM();
      await render(<MenuItem class="hover:bg-panel">Clear</MenuItem>);
      const cls = screen.querySelector("button")?.className ?? "";
      // Base tokens preserved
      expect(cls).toContain("block w-full text-left");
      expect(cls).toContain("hover:bg-raise");
      // Override appended
      expect(cls).toContain("hover:bg-panel");
    });
  });

  describe("polymorphism", () => {
    it("as='a' renders an <a> with the panel-row className + href", async () => {
      const { screen, render } = await createDOM();
      await render(
        <MenuItem as="a" href="/profile/">
          Profile
        </MenuItem>,
      );
      const a = screen.querySelector("a");
      expect(a).toBeTruthy();
      expect(a?.getAttribute("href")).toBe("/profile/");
      expect(a?.className).toContain("block w-full text-left");
      expect(a?.className).toContain("hover:bg-raise");
    });
  });
});
