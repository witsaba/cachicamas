/**
 * Component tests for `<SettingCard>`.
 *
 * Reference: `sdd/settings-app-grid/{spec,design}.md` (engram).
 *   - REQ-1 (primitive exists, importable from the folder)
 *   - REQ-2 (icon via `icon: JSXNode` prop)
 *   - REQ-4 (polymorphism — default `as="a"`, supports `as="button"`)
 *   - REQ-5 (consumer `class` appended after system tokens)
 *   - REQ-6 (focus-visible ring present)
 *
 * Pure-function className coverage lives in `classes.spec.ts`; this
 * file focuses on the Qwik component layer (polymorphism, icon
 * container placement, label text, props forwarding, anchor attrs,
 * testId mapping).
 */
import { createDOM } from "@builder.io/qwik/testing";
import { describe, it, expect } from "vitest";
import { SettingCard } from "./setting-card";

/** A trivial icon used by tests — verifies the slot is rendered. */
const TestIcon = () => <svg data-testid="test-icon" />;

describe("components/ui/setting-card", () => {
  describe("REQ-1 — primitive is importable + renderable", () => {
    it("renders an <a href='...'> with the system className by default", async () => {
      const { screen, render } = await createDOM();
      await render(
        <SettingCard
          href="/settings/prompts"
          label="Prompts"
          icon={<TestIcon />}
        />,
      );
      const a = screen.querySelector("a");
      expect(a).toBeTruthy();
      expect(a?.getAttribute("href")).toBe("/settings/prompts");
      // System tokens present on the anchor (CARD_BASE begins the string).
      expect(a?.className).toContain("group");
      expect(a?.className).toContain("flex");
      expect(a?.className).toContain("flex-col");
      expect(a?.className).toContain("items-center");
    });
  });

  describe("REQ-4 — polymorphism", () => {
    it("as='button' renders a <button> (NOT an anchor)", async () => {
      const { screen, render } = await createDOM();
      await render(
        <SettingCard
          as="button"
          type="button"
          label="Toggle"
          icon={<TestIcon />}
        />,
      );
      const btn = screen.querySelector("button");
      expect(btn).toBeTruthy();
      // No anchor in the DOM (polymorphism switch worked).
      expect(screen.querySelector("a")).toBeNull();
    });

    it("as='button' defaults type to 'button' (anti-implicit-submit)", async () => {
      const { screen, render } = await createDOM();
      await render(
        <SettingCard as="button" label="Toggle" icon={<TestIcon />} />,
      );
      expect(screen.querySelector("button")?.getAttribute("type")).toBe(
        "button",
      );
    });

    it("as='button' forwards type='submit'", async () => {
      const { screen, render } = await createDOM();
      await render(
        <SettingCard as="button" type="submit" label="Save" icon={<TestIcon />} />,
      );
      expect(screen.querySelector("button")?.getAttribute("type")).toBe(
        "submit",
      );
    });

    it("as='button' forwards disabled (button case only)", async () => {
      const { screen, render } = await createDOM();
      await render(
        <SettingCard
          as="button"
          disabled={true}
          label="Off"
          icon={<TestIcon />}
        />,
      );
      const btn = screen.querySelector("button");
      expect(btn?.hasAttribute("disabled")).toBe(true);
      expect(btn?.className).toContain("disabled:cursor-not-allowed");
      expect(btn?.className).toContain("disabled:opacity-50");
    });
  });

  describe("REQ-2 — icon prop renders inside the icon container", () => {
    it("places the icon JSX INSIDE the icon container <span>", async () => {
      const { screen, render } = await createDOM();
      await render(
        <SettingCard
          href="/x"
          label="X"
          icon={<TestIcon />}
          testId="settings-card-test"
        />,
      );
      // Find the anchor, then assert the icon testid lives inside.
      const card = screen.querySelector('[data-testid="settings-card-test"]');
      expect(card).toBeTruthy();
      const icon = card?.querySelector('[data-testid="test-icon"]');
      expect(icon).toBeTruthy();
      // The icon's enclosing span carries the icon-container tokens.
      const container = icon?.parentElement;
      expect(container?.className).toContain("h-16");
      expect(container?.className).toContain("w-16");
      expect(container?.className).toContain("rounded-xl");
      expect(container?.className).toContain("bg-slate-100");
    });
  });

  describe("label rendering", () => {
    it("renders the label string inside the label <span>", async () => {
      const { screen, render } = await createDOM();
      await render(
        <SettingCard
          href="/x"
          label="Prompts"
          icon={<TestIcon />}
          testId="settings-card-test"
        />,
      );
      const card = screen.querySelector('[data-testid="settings-card-test"]');
      // Label text matches the prop (case-sensitive, exact match).
      expect(card?.textContent).toContain("Prompts");
      // Label <span> carries LABEL tokens.
      const labelSpan = Array.from(card?.querySelectorAll("span") ?? []).find(
        (s) => s.textContent === "Prompts",
      );
      expect(labelSpan?.className).toContain("mt-3");
      expect(labelSpan?.className).toContain("text-sm");
      expect(labelSpan?.className).toContain("font-medium");
      expect(labelSpan?.className).toContain("text-slate-900");
    });
  });

  describe("anchor attribute forwarding", () => {
    it("forwards target='_blank' + rel on the anchor", async () => {
      const { screen, render } = await createDOM();
      await render(
        <SettingCard
          href="https://example.com"
          target="_blank"
          rel="noopener noreferrer"
          label="External"
          icon={<TestIcon />}
        />,
      );
      const a = screen.querySelector("a");
      expect(a?.getAttribute("target")).toBe("_blank");
      expect(a?.getAttribute("rel")).toBe("noopener noreferrer");
    });
  });

  describe("REQ-5 — consumer class appended after system tokens", () => {
    it("places consumer class LAST in the rendered className", async () => {
      const { screen, render } = await createDOM();
      await render(
        <SettingCard
          href="/x"
          label="X"
          icon={<TestIcon />}
          class="!bg-pink-500"
          testId="settings-card-test"
        />,
      );
      const card = screen.querySelector('[data-testid="settings-card-test"]');
      const cls = card?.className ?? "";
      expect(cls).toContain("!bg-pink-500");
      // System tokens precede consumer tokens.
      const consumerIdx = cls.indexOf("!bg-pink-500");
      const systemIdx = cls.indexOf("group");
      expect(consumerIdx).toBeGreaterThan(systemIdx);
    });
  });

  describe("testId forwarding", () => {
    it("renders testId as data-testid on the outer element", async () => {
      const { screen, render } = await createDOM();
      await render(
        <SettingCard
          href="/x"
          label="X"
          icon={<TestIcon />}
          testId="my-tile"
        />,
      );
      expect(screen.querySelector('[data-testid="my-tile"]')).toBeTruthy();
    });
  });

  describe("REQ-6 — focus ring present", () => {
    it("the outer element carries the focus-visible tokens", async () => {
      const { screen, render } = await createDOM();
      await render(
        <SettingCard
          href="/x"
          label="X"
          icon={<TestIcon />}
          testId="settings-card-test"
        />,
      );
      const card = screen.querySelector('[data-testid="settings-card-test"]');
      const cls = card?.className ?? "";
      expect(cls).toContain("focus:outline-none");
      expect(cls).toContain("focus-visible:ring-2");
      expect(cls).toContain("focus-visible:ring-indigo-500");
      expect(cls).toContain("focus-visible:ring-offset-2");
    });
  });
});
