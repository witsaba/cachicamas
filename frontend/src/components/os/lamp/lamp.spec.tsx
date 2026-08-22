/**
 * StateLamp — a state, said twice.
 *
 * The accessibility commitment in PRODUCT.md is that no meaning is ever
 * carried by colour or mark alone. This component is where that commitment
 * either holds or quietly stops holding, so the assertion that the word is
 * always present — and that the mark is always hidden from assistive
 * technology — is the reason the file exists.
 */
import { createDOM } from "@builder.io/qwik/testing";
import { describe, it, expect } from "vitest";
import { StateLamp, type LampTone } from "./lamp";

const TONES: LampTone[] = ["live", "build", "ready", "hold", "fail", "idle"];

describe("components/os/lamp", () => {
  it("always renders its word, for every tone", async () => {
    for (const tone of TONES) {
      const { screen, render } = await createDOM();
      await render(<StateLamp tone={tone} word="Planned" />);
      expect(screen.querySelector("span")?.textContent, tone).toContain(
        "Planned",
      );
    }
  });

  it("hides the mark from assistive technology, because the word carries it", async () => {
    const { screen, render } = await createDOM();
    await render(<StateLamp tone="live" word="Running" testId="l" />);
    const lamp = screen.querySelector('[data-testid="l"]');
    const mark = lamp?.querySelector('[aria-hidden="true"]');
    expect(mark).toBeTruthy();
    // The mark must contribute nothing to the accessible name; the word beside
    // it is the whole message.
    expect(mark?.textContent).toBe("");
  });

  it("keeps all six tones distinguishable, on two axes", async () => {
    // Colour separates what is HAPPENING; shape separates what is not. The
    // two inactive tones are both neutral, so a colour-only vocabulary left
    // them ~1.5:1 apart — on a board where five of six cells are inactive,
    // that put the whole state read back onto the word. Every tone must still
    // be distinguishable from every other by its mark alone.
    const marks = new Map<string, string>();
    for (const tone of TONES) {
      const { screen, render } = await createDOM();
      await render(<StateLamp tone={tone} word="x" testId="l" />);
      const mark = screen
        .querySelector('[data-testid="l"]')
        ?.querySelector('[aria-hidden="true"]');
      const cls = mark?.className ?? "";
      expect(cls, tone).not.toBe("");
      marks.set(tone, cls);
    }
    expect(new Set(marks.values()).size).toBe(TONES.length);
  });

  it("fills a mark only when the state has activity behind it", async () => {
    const active = ["live", "build", "hold", "fail"] as const;
    const inactive = ["ready", "idle"] as const;
    for (const tone of active) {
      const { screen, render } = await createDOM();
      await render(<StateLamp tone={tone} word="x" testId="l" />);
      const cls =
        screen
          .querySelector('[data-testid="l"]')
          ?.querySelector('[aria-hidden="true"]')?.className ?? "";
      expect(cls, tone).toMatch(/bg-(live|amber|hold|fail)/);
      expect(cls, tone).not.toContain("bg-transparent");
    }
    for (const tone of inactive) {
      const { screen, render } = await createDOM();
      await render(<StateLamp tone={tone} word="x" testId="l" />);
      const cls =
        screen
          .querySelector('[data-testid="l"]')
          ?.querySelector('[aria-hidden="true"]')?.className ?? "";
      expect(cls, tone).toContain("bg-transparent");
      expect(cls, tone).toContain("border");
      // An inactive state must never wear a working colour.
      expect(cls, tone).not.toMatch(/(live|amber|hold|fail)/);
    }
  });

  it("exposes its tone as data, so a screen can be asserted on state", async () => {
    const { screen, render } = await createDOM();
    await render(<StateLamp tone="hold" word="Suspended" testId="l" />);
    expect(
      screen.querySelector('[data-testid="l"]')?.getAttribute("data-tone"),
    ).toBe("hold");
  });

  it("pulses only when asked, and only the mark", async () => {
    const { screen, render } = await createDOM();
    await render(<StateLamp tone="live" word="Running" pulse testId="l" />);
    const lamp = screen.querySelector('[data-testid="l"]');
    expect(lamp?.querySelector('[aria-hidden="true"]')?.className).toContain(
      "term-lamp",
    );
    // The word must not move; a blinking label is unreadable.
    const word = Array.from(lamp?.querySelectorAll("span") ?? []).find(
      (s) => s.getAttribute("aria-hidden") !== "true",
    );
    expect(word?.className ?? "").not.toContain("term-lamp");

    const still = await createDOM();
    await still.render(<StateLamp tone="ready" word="Planned" testId="l" />);
    expect(
      still.screen
        .querySelector('[data-testid="l"]')
        ?.querySelector('[aria-hidden="true"]')?.className,
    ).not.toContain("term-lamp");
  });
});
