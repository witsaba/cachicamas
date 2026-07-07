/**
 * Component tests for `<Button>`.
 *
 * Reference: `openspec/changes/cachicamas-button-design-system/specs/frontend-ui-button/spec.md`
 *
 * The full (variant × size × state) matrix is exercised here. Class
 * tokens are asserted on the rendered DOM's `className` attribute —
 * not computed styles — so the test runs without a real CSS engine.
 *
 * Pure-function className coverage lives in `classes.spec.ts`; this
 * file focuses on the Qwik component layer (props, polymorphism,
 * default type, aria-busy, Slot children).
 */
import { createDOM } from "@builder.io/qwik/testing";
import { describe, it, expect } from "vitest";
import { Button } from "./button";

describe("components/ui/button", () => {
  describe("R-UB-001 — variant rendering", () => {
    it("primary (default) renders bg-slate-900 + text-white", async () => {
      const { screen, render } = await createDOM();
      await render(<Button>Save</Button>);
      const btn = screen.querySelector("button");
      expect(btn).toBeTruthy();
      expect(btn?.className).toContain("bg-slate-900");
      expect(btn?.className).toContain("text-white");
      // Drift regression guard.
      expect(btn?.className).not.toContain("bg-indigo-600");
    });

    it("secondary renders bg-white + border-slate-300 + text-slate-900", async () => {
      const { screen, render } = await createDOM();
      await render(<Button variant="secondary">Cancel</Button>);
      const btn = screen.querySelector("button");
      expect(btn?.className).toContain("bg-white");
      expect(btn?.className).toContain("border-slate-300");
      expect(btn?.className).toContain("text-slate-900");
      expect(btn?.className).not.toContain("bg-slate-900");
    });

    it("destructive renders bg-red-700 + text-white + no active:translate-y-px", async () => {
      const { screen, render } = await createDOM();
      await render(<Button variant="destructive">Delete</Button>);
      const btn = screen.querySelector("button");
      expect(btn?.className).toContain("bg-red-700");
      expect(btn?.className).toContain("text-white");
      expect(btn?.className).not.toContain("active:translate-y-px");
    });

    it("sizes: md uses text-sm + px-4 py-2", async () => {
      const { screen, render } = await createDOM();
      await render(<Button size="md">md</Button>);
      const md = screen.querySelector("button");
      expect(md?.className).toContain("text-sm");
      expect(md?.className).toContain("px-4 py-2");
    });

    it("sizes: lg uses text-base + px-5 py-3", async () => {
      const { screen, render } = await createDOM();
      await render(<Button size="lg">lg</Button>);
      const lg = screen.querySelector("button");
      expect(lg?.className).toContain("text-base");
      expect(lg?.className).toContain("px-5 py-3");
    });
  });

  describe("R-UB-002 — cursor affordance", () => {
    it("enabled button has cursor-pointer", async () => {
      const { screen, render } = await createDOM();
      await render(<Button>Click</Button>);
      expect(screen.querySelector("button")?.className).toContain(
        "cursor-pointer",
      );
    });

    it("disabled button carries disabled attribute + disabled:opacity-50", async () => {
      const { screen, render } = await createDOM();
      await render(<Button disabled={true}>Disabled</Button>);
      const btn = screen.querySelector("button");
      expect(btn?.hasAttribute("disabled")).toBe(true);
      expect(btn?.className).toContain("disabled:cursor-not-allowed");
      expect(btn?.className).toContain("disabled:opacity-50");
    });
  });

  describe("R-UB-003 — hover / transition / active", () => {
    it("primary has transition + duration-150 + active:translate-y-px", async () => {
      const { screen, render } = await createDOM();
      await render(<Button variant="primary">Save</Button>);
      const cls = screen.querySelector("button")?.className ?? "";
      expect(cls).toMatch(/transition-\[background-color/);
      expect(cls).toContain("duration-150");
      expect(cls).toContain("active:translate-y-px");
      expect(cls).toContain("not-disabled:hover:bg-slate-700");
    });

    it("secondary has transition + not-disabled:hover:bg-slate-50", async () => {
      const { screen, render } = await createDOM();
      await render(<Button variant="secondary">Cancel</Button>);
      const cls = screen.querySelector("button")?.className ?? "";
      expect(cls).toMatch(/transition-\[background-color/);
      expect(cls).toContain("duration-150");
      expect(cls).toContain("not-disabled:hover:bg-slate-50");
    });

    it("destructive has transition + not-disabled:hover:bg-red-800 + no active", async () => {
      const { screen, render } = await createDOM();
      await render(<Button variant="destructive">Delete</Button>);
      const cls = screen.querySelector("button")?.className ?? "";
      expect(cls).toMatch(/transition-\[background-color/);
      expect(cls).toContain("duration-150");
      expect(cls).toContain("not-disabled:hover:bg-red-800");
      expect(cls).not.toContain("active:translate-y-px");
    });
  });

  describe("R-UB-004 — focus ring", () => {
    it("primary has indigo focus ring", async () => {
      const { screen, render } = await createDOM();
      await render(<Button variant="primary">Save</Button>);
      const cls = screen.querySelector("button")?.className ?? "";
      expect(cls).toContain("focus:outline-none");
      expect(cls).toContain("focus-visible:ring-2");
      expect(cls).toContain("focus-visible:ring-indigo-500");
    });

    it("secondary has indigo focus ring", async () => {
      const { screen, render } = await createDOM();
      await render(<Button variant="secondary">Cancel</Button>);
      const cls = screen.querySelector("button")?.className ?? "";
      expect(cls).toContain("focus-visible:ring-indigo-500");
    });

    it("destructive has red focus ring", async () => {
      const { screen, render } = await createDOM();
      await render(<Button variant="destructive">Delete</Button>);
      const cls = screen.querySelector("button")?.className ?? "";
      expect(cls).toContain("focus-visible:ring-red-500");
    });
  });

  describe("R-UB-007 — loading state", () => {
    it("loading=true renders disabled + aria-busy=true", async () => {
      const { screen, render } = await createDOM();
      await render(<Button loading={true}>Save</Button>);
      const btn = screen.querySelector("button");
      expect(btn?.hasAttribute("disabled")).toBe(true);
      expect(btn?.getAttribute("aria-busy")).toBe("true");
    });

    it("loading does not render an <img> or animated <svg>", async () => {
      const { screen, render } = await createDOM();
      await render(<Button loading={true}>Save</Button>);
      const btn = screen.querySelector("button");
      expect(btn?.querySelector("img")).toBeFalsy();
      const svgs = btn?.querySelectorAll("svg") ?? [];
      for (const svg of Array.from(svgs)) {
        expect(svg.getAttribute("class") ?? "").not.toMatch(/animate-/);
      }
    });

    it("loading composes with consumer-provided children", async () => {
      const { screen, render } = await createDOM();
      await render(
        <Button loading={true}>
          <span>Saving…</span>
        </Button>,
      );
      const btn = screen.querySelector("button");
      expect(btn?.textContent?.trim()).toBe("Saving…");
      expect(btn?.hasAttribute("disabled")).toBe(true);
    });
  });

  describe("R-UB-008 — link polymorphism", () => {
    it("as='a' renders an <a> with the same className as the equivalent button", async () => {
      const { screen, render } = await createDOM();
      await render(
        <Button as="a" href="/workspaces/new" variant="primary">
          Create
        </Button>,
      );
      const a = screen.querySelector("a");
      expect(a).toBeTruthy();
      expect(a?.getAttribute("href")).toBe("/workspaces/new");
      expect(a?.className).toContain("bg-slate-900");
      expect(a?.className).toContain("cursor-pointer");
      // The anchor must NOT carry a `disabled` attribute (links cannot
      // be disabled in HTML; consumers handle that via route guards).
      expect(a?.hasAttribute("disabled")).toBe(false);
    });
  });

  describe("R-UB-010 — default type=button", () => {
    it("default render produces type='button' (anti-implicit-submit)", async () => {
      const { screen, render } = await createDOM();
      await render(<Button>Save</Button>);
      expect(screen.querySelector("button")?.getAttribute("type")).toBe(
        "button",
      );
    });

    it("type='submit' override is honored", async () => {
      const { screen, render } = await createDOM();
      await render(<Button type="submit">Save</Button>);
      expect(screen.querySelector("button")?.getAttribute("type")).toBe(
        "submit",
      );
    });

    it("type='reset' override is honored", async () => {
      const { screen, render } = await createDOM();
      await render(<Button type="reset">Reset</Button>);
      expect(screen.querySelector("button")?.getAttribute("type")).toBe(
        "reset",
      );
    });
  });

  describe("misc", () => {
    it("renders Slot children verbatim", async () => {
      const { screen, render } = await createDOM();
      await render(
        <Button>
          <span data-testid="child-span">Hello</span>
        </Button>,
      );
      expect(screen.querySelector('[data-testid="child-span"]')?.textContent).toBe(
        "Hello",
      );
    });

    it("testId prop is rendered as data-testid", async () => {
      const { screen, render } = await createDOM();
      await render(<Button testId="my-button">x</Button>);
      expect(
        screen.querySelector('[data-testid="my-button"]'),
      ).toBeTruthy();
    });
  });
});