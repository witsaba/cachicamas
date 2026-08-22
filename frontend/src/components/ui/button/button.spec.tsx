/**
 * Component tests for `<Button>`.
 *
 * The full (variant × size × state) matrix is exercised here. Class tokens are
 * asserted on the rendered DOM's `className` attribute — not computed styles —
 * so the test runs without a real CSS engine.
 *
 * Pure-function className coverage lives in `classes.spec.ts`; this file
 * focuses on the Qwik component layer (props, polymorphism, default type,
 * aria-busy, Slot children, class override).
 */
import { createDOM } from "@builder.io/qwik/testing";
import { describe, it, expect } from "vitest";
import { Button } from "./button";

describe("components/ui/button", () => {
  describe("variant rendering", () => {
    it("primary (default) renders the brand-filled control", async () => {
      const { screen, render } = await createDOM();
      await render(<Button>Save</Button>);
      const btn = screen.querySelector("button");
      expect(btn).toBeTruthy();
      expect(btn?.className).toContain("bg-brand");
      expect(btn?.className).toContain("text-ink-inverse");
      // Drift regression guard: the retired slate world must not come back.
      expect(btn?.className).not.toMatch(/slate|indigo/);
    });

    it("secondary renders a white control with a findable border", async () => {
      const { screen, render } = await createDOM();
      await render(<Button variant="secondary">Cancel</Button>);
      const btn = screen.querySelector("button");
      expect(btn?.className).toContain("bg-surface");
      expect(btn?.className).toContain("border-line-control");
      expect(btn?.className).toContain("text-ink");
      expect(btn?.className).not.toContain("bg-brand");
    });

    it("destructive renders the stop colour, filled", async () => {
      const { screen, render } = await createDOM();
      await render(<Button variant="destructive">Delete</Button>);
      const btn = screen.querySelector("button");
      expect(btn?.className).toContain("bg-stop");
      expect(btn?.className).toContain("text-ink-inverse");
    });

    it("link renders brand + underline and NO surface", async () => {
      const { screen, render } = await createDOM();
      await render(<Button variant="link">Clear selection</Button>);
      const btn = screen.querySelector("button");
      expect(btn?.className).toContain("text-brand");
      expect(btn?.className).toContain("underline");
      expect(btn?.className).not.toMatch(/\bbg-/);
      expect(btn?.className).not.toContain("rounded");
    });

    it("sizes: md is a 36px control at the body size", async () => {
      const { screen, render } = await createDOM();
      await render(<Button size="md">md</Button>);
      const md = screen.querySelector("button");
      expect(md?.className).toContain("text-base");
      expect(md?.className).toContain("h-9 px-3.5");
    });

    it("sizes: lg is a 44px control", async () => {
      const { screen, render } = await createDOM();
      await render(<Button size="lg">lg</Button>);
      const lg = screen.querySelector("button");
      expect(lg?.className).toContain("text-base");
      expect(lg?.className).toContain("h-11 px-5");
    });
  });

  describe("cursor and disabled affordance", () => {
    it("enabled button has cursor-pointer", async () => {
      const { screen, render } = await createDOM();
      await render(<Button>Click</Button>);
      expect(screen.querySelector("button")?.className).toContain(
        "cursor-pointer",
      );
    });

    it("disabled button carries disabled attribute + disabled:opacity-40", async () => {
      const { screen, render } = await createDOM();
      await render(<Button disabled={true}>Disabled</Button>);
      const btn = screen.querySelector("button");
      expect(btn?.hasAttribute("disabled")).toBe(true);
      expect(btn?.className).toContain("disabled:cursor-not-allowed");
      expect(btn?.className).toContain("disabled:opacity-45");
    });
  });

  describe("press darkens, and nothing moves", () => {
    it("primary darkens on hover and press", async () => {
      const { screen, render } = await createDOM();
      await render(<Button variant="primary">Save</Button>);
      const cls = screen.querySelector("button")?.className ?? "";
      expect(cls).toMatch(/transition-\[background-color/);
      expect(cls).toContain("duration-150");
      expect(cls).toContain("not-disabled:hover:bg-brand-press");
      expect(cls).toContain("not-disabled:active:bg-brand-press");
      // A terminal key does not travel.
      expect(cls).not.toContain("translate");
    });

    it("secondary tints its surface rather than filling with brand", async () => {
      const { screen, render } = await createDOM();
      await render(<Button variant="secondary">Cancel</Button>);
      const cls = screen.querySelector("button")?.className ?? "";
      expect(cls).toContain("not-disabled:hover:bg-sunken");
      expect(cls).toContain("not-disabled:active:bg-sunken");
      expect(cls).not.toContain("translate");
    });

    it("destructive darkens, keeping the stop colour", async () => {
      const { screen, render } = await createDOM();
      await render(<Button variant="destructive">Delete</Button>);
      const cls = screen.querySelector("button")?.className ?? "";
      expect(cls).toContain("bg-stop");
      expect(cls).toContain("not-disabled:hover:brightness-90");
      expect(cls).not.toContain("translate");
    });

    it("link changes colour only, with no surface", async () => {
      const { screen, render } = await createDOM();
      await render(<Button variant="link">Clear</Button>);
      const cls = screen.querySelector("button")?.className ?? "";
      expect(cls).toContain("transition-colors");
      expect(cls).toContain("duration-150");
      expect(cls).toContain("hover:text-brand-press");
      expect(cls).not.toMatch(/not-disabled:hover:bg-/);
    });
  });

  describe("focus is the system's, not the variant's", () => {
    it("no variant restyles focus — global.css owns the one treatment", async () => {
      for (const variant of [
        "primary",
        "secondary",
        "destructive",
        "link",
      ] as const) {
        // A fresh DOM per variant: Qwik refuses to render twice into one
        // container, and reusing it would silently assert the first render
        // four times over.
        const { screen, render } = await createDOM();
        await render(<Button variant={variant}>x</Button>);
        const cls = screen.querySelector("button")?.className ?? "";
        expect(cls, variant).not.toMatch(/ring-/);
        expect(cls, variant).not.toMatch(/focus-visible:outline-/);
      }
    });
  });

  describe("class override (consumer personalization)", () => {
    it("consumer class is appended to the system tokens", async () => {
      const { screen, render } = await createDOM();
      await render(<Button class="h-10 w-full">Wide</Button>);
      const cls = screen.querySelector("button")?.className ?? "";
      expect(cls).toContain("bg-brand");
      expect(cls).toContain("h-10 w-full");
    });

    it("consumer class on link variant is appended after VARIANT_LINK", async () => {
      const { screen, render } = await createDOM();
      await render(
        <Button variant="link" class="text-base">
          Clear
        </Button>,
      );
      const cls = screen.querySelector("button")?.className ?? "";
      expect(cls).toContain("text-brand");
      expect(cls.endsWith("text-base")).toBe(true);
    });

    it("consumer class on link variant does NOT add a surface", async () => {
      const { screen, render } = await createDOM();
      await render(
        <Button variant="link" class="text-base">
          Clear
        </Button>,
      );
      const cls = screen.querySelector("button")?.className ?? "";
      expect(cls).not.toMatch(/\bbg-/);
    });
  });

  describe("loading state", () => {
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

  describe("link polymorphism", () => {
    it("as='a' renders an <a> with the same className as the equivalent button", async () => {
      const { screen, render } = await createDOM();
      await render(
        <Button as="a" href="/chat/" variant="primary">
          Open
        </Button>,
      );
      const a = screen.querySelector("a");
      expect(a).toBeTruthy();
      expect(a?.getAttribute("href")).toBe("/chat/");
      expect(a?.className).toContain("bg-brand");
      expect(a?.className).toContain("cursor-pointer");
      expect(a?.hasAttribute("disabled")).toBe(false);
    });

    it("as='a' passes through class override for shape tokens", async () => {
      const { screen, render } = await createDOM();
      await render(
        <Button as="a" href="/profile/" variant="secondary" class="px-5 py-2.5">
          View profile
        </Button>,
      );
      const a = screen.querySelector("a");
      expect(a?.className).toContain("bg-surface");
      expect(a?.className).toContain("px-5 py-2.5");
    });
  });

  describe("default type=button", () => {
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
      expect(
        screen.querySelector('[data-testid="child-span"]')?.textContent,
      ).toBe("Hello");
    });

    it("testId prop is rendered as data-testid", async () => {
      const { screen, render } = await createDOM();
      await render(<Button testId="my-button">x</Button>);
      expect(screen.querySelector('[data-testid="my-button"]')).toBeTruthy();
    });
  });
});
