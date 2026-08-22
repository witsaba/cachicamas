/**
 * RegisterCell — one archetype's line on the board.
 *
 * This is the launcher tile, and the thing it must never do is imply that a
 * specialist works. Every assertion below is a version of that claim: the
 * state is in words, the evidence is cited, an invented figure is marked, and
 * an unplanned specialist does not dress up as a planned one.
 */
import { createDOM } from "@builder.io/qwik/testing";
import { describe, it, expect } from "vitest";
import { RegisterCell, lampToneFor } from "./register-cell";
import {
  ARCHETYPES,
  archetypeByCode,
  archetypeHref,
} from "~/lib/mock/registry";

const CHAT = archetypeByCode("CHAT")!;
const DBA = archetypeByCode("DBA")!;
const FINANCE = archetypeByCode("FINANCE")!;

describe("lampToneFor", () => {
  it("maps every state to a tone, and keeps the four apart", () => {
    const tones = (
      ["on-duty", "in-build", "planned", "unplanned"] as const
    ).map(lampToneFor);
    expect(new Set(tones).size).toBe(4);
  });

  it("never lights an unplanned specialist as if it were running", () => {
    expect(lampToneFor("unplanned")).toBe("idle");
    expect(lampToneFor("on-duty")).toBe("live");
  });
});

describe("components/os/register-cell", () => {
  it("links to the archetype's own destination", async () => {
    const { screen, render } = await createDOM();
    await render(<RegisterCell archetype={CHAT} />);
    const link = screen.querySelector('[data-testid="register-cell-chat"]');
    expect(link?.getAttribute("href")).toBe(archetypeHref(CHAT));
  });

  it("says the state in words, beside the lamp", async () => {
    const { screen, render } = await createDOM();
    await render(<RegisterCell archetype={DBA} />);
    const lamp = screen.querySelector('[data-testid="register-lamp-dba"]');
    expect(lamp?.textContent).toContain(DBA.stateWord);
  });

  it("exposes the state as data so a board can be asserted on it", async () => {
    const { screen, render } = await createDOM();
    await render(<RegisterCell archetype={FINANCE} />);
    expect(
      screen
        .querySelector('[data-testid="register-cell-finance"]')
        ?.getAttribute("data-state"),
    ).toBe("unplanned");
  });

  it("shows the function key, so the dock and the board agree", async () => {
    const { screen, render } = await createDOM();
    await render(<RegisterCell archetype={CHAT} />);
    expect(
      screen.querySelector('[data-testid="register-cell-chat"]')?.textContent,
    ).toContain(CHAT.fkey);
  });

  it("cites the decision record that put the specialist on the board", async () => {
    for (const a of ARCHETYPES) {
      const { screen, render } = await createDOM();
      await render(<RegisterCell archetype={a} />);
      const text =
        screen.querySelector(`[data-testid="register-cell-${a.slug}"]`)
          ?.textContent ?? "";
      expect(text, a.code).toContain(a.authority);
    }
  });

  it("labels an invented activity figure as demonstration data", async () => {
    const { screen, render } = await createDOM();
    await render(<RegisterCell archetype={CHAT} />);
    const text =
      screen.querySelector('[data-testid="register-cell-chat"]')?.textContent ??
      "";
    expect(text).toContain(String(CHAT.demoTurnsToday));
    expect(text).toContain("demo");
  });

  it("shows no activity row at all for a specialist that has none", async () => {
    const { screen, render } = await createDOM();
    await render(<RegisterCell archetype={FINANCE} />);
    const text =
      screen.querySelector('[data-testid="register-cell-finance"]')
        ?.textContent ?? "";
    expect(text).not.toContain("Turns today");
    expect(text).not.toContain("demo");
  });

  it("says 'none yet' rather than drawing an empty plan gauge", async () => {
    // An empty gauge next to a specialist with no plan would read as
    // "0 of something"; there is no something.
    const { screen, render } = await createDOM();
    await render(<RegisterCell archetype={FINANCE} />);
    const text =
      screen.querySelector('[data-testid="register-cell-finance"]')
        ?.textContent ?? "";
    expect(text).toContain("None yet");
  });

  it("shows a plan with its real, unshipped count", async () => {
    const { screen, render } = await createDOM();
    await render(<RegisterCell archetype={CHAT} />);
    const text =
      screen.querySelector('[data-testid="register-cell-chat"]')?.textContent ??
      "";
    expect(text).toContain("doc 0005");
    expect(text).toContain("0/12");
  });
});
