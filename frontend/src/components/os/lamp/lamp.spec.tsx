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

  it("gives each tone a distinct colour, so the vocabulary stays six-valued", async () => {
    const seen = new Set<string>();
    for (const tone of TONES) {
      const { screen, render } = await createDOM();
      await render(<StateLamp tone={tone} word="x" testId="l" />);
      const mark = screen
        .querySelector('[data-testid="l"]')
        ?.querySelector('[aria-hidden="true"]');
      const bg = (mark?.className ?? "").match(/bg-[a-z-]+/)?.[0] ?? "";
      expect(bg, tone).not.toBe("");
      seen.add(bg);
    }
    expect(seen.size).toBe(TONES.length);
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
