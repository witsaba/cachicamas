/**
 * Gauge — a count drawn as segments, with the figure beside it.
 *
 * `litSegments` is pure and is the whole behaviour: the rounding rule is the
 * only place this component can lie. 41 of 42 must not read as complete, and
 * 1 of 42 must not read as nothing — both are the difference between a gauge
 * and a decoration.
 */
import { createDOM } from "@builder.io/qwik/testing";
import { describe, it, expect } from "vitest";
import { Gauge, GAUGE_SEGMENTS, litSegments } from "./gauge";

describe("litSegments", () => {
  it("lights nothing at zero", () => {
    expect(litSegments(0, 12)).toBe(0);
  });

  it("lights everything only at completion", () => {
    expect(litSegments(42, 42)).toBe(GAUGE_SEGMENTS);
    expect(litSegments(43, 42)).toBe(GAUGE_SEGMENTS);
  });

  it("never rounds a near-miss up to full", () => {
    // 41 of 42 rounds to 8/8 arithmetically. Showing that as complete is the
    // one reading this component must never produce.
    expect(litSegments(41, 42)).toBe(GAUGE_SEGMENTS - 1);
    expect(litSegments(99, 100)).toBe(GAUGE_SEGMENTS - 1);
  });

  it("never rounds a real start down to nothing", () => {
    expect(litSegments(1, 42)).toBe(1);
    expect(litSegments(1, 1000)).toBe(1);
  });

  it("survives a nonsense total without dividing by zero", () => {
    expect(litSegments(3, 0)).toBe(0);
    expect(litSegments(0, 0)).toBe(0);
    expect(litSegments(-1, 10)).toBe(0);
  });
});

describe("components/os/gauge", () => {
  it("prints the literal figure beside the segments", async () => {
    const { screen, render } = await createDOM();
    await render(<Gauge done={24} total={24} testId="g" />);
    expect(screen.querySelector('[data-testid="g"]')?.textContent).toContain(
      "24/24",
    );
  });

  it("draws exactly the fixed number of segments, always", async () => {
    for (const [done, total] of [
      [0, 12],
      [7, 12],
      [42, 42],
    ] as const) {
      const { screen, render } = await createDOM();
      await render(<Gauge done={done} total={total} testId="g" />);
      const track = screen
        .querySelector('[data-testid="g"]')
        ?.querySelector('[aria-hidden="true"]');
      expect(track?.children.length, `${done}/${total}`).toBe(GAUGE_SEGMENTS);
    }
  });

  it("reads a genuine zero as a zero rather than as a broken component", async () => {
    const { screen, render } = await createDOM();
    await render(<Gauge done={0} total={12} testId="g" />);
    const gauge = screen.querySelector('[data-testid="g"]');
    expect(gauge?.textContent).toContain("0/12");
    const track = gauge?.querySelector('[aria-hidden="true"]');
    const lit = Array.from(track?.children ?? []).filter((c) =>
      /bg-(cyan|amber)/.test(c.className),
    );
    expect(lit.length).toBe(0);
  });

  it("carries no state colour at all — a quantity is not a state", async () => {
    // The lamp beside a gauge already reports whether the thing is running.
    // Colouring the bar too is how amber ended up on twenty marks per screen;
    // the fill is neutral at two brightnesses instead.
    for (const [done, total] of [
      [42, 42],
      [5, 42],
      [0, 12],
    ] as const) {
      const { screen, render } = await createDOM();
      await render(<Gauge done={done} total={total} testId="g" />);
      const html = screen.querySelector('[data-testid="g"]')?.innerHTML ?? "";
      expect(html, `${done}/${total}`).not.toMatch(
        /bg-(amber|cyan|live|hold|fail)/,
      );
    }
  });

  it("distinguishes complete from in-progress by brightness, not by hue", async () => {
    const complete = await createDOM();
    await complete.render(<Gauge done={42} total={42} testId="g" />);
    const completeHtml =
      complete.screen.querySelector('[data-testid="g"]')?.innerHTML ?? "";
    expect(completeHtml).toContain("bg-fg");

    const partial = await createDOM();
    await partial.render(<Gauge done={5} total={42} testId="g" />);
    const partialHtml =
      partial.screen.querySelector('[data-testid="g"]')?.innerHTML ?? "";
    expect(partialHtml).toContain("bg-fg-mid");
  });
});
