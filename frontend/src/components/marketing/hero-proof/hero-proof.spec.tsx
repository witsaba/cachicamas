/**
 * The public page's proof.
 *
 * The page mounts a component that PLAYS, so a render test at first paint can
 * only see the opening line. What matters is where the play ENDS: on a
 * permission that has stopped the work and is waiting for a person. That is
 * driven here a tick at a time, over the same pure machine the workspace
 * conversation runs on — no timers, no flake.
 */
import { createDOM } from "@builder.io/qwik/testing";
import { describe, it, expect } from "vitest";
import { HeroProof } from "./hero-proof";
import {
  runToEnd,
  type MockTurnStore,
} from "~/components/chat/use-mock-turn";
import { HERO_OPENING, HERO_SCRIPT } from "~/lib/mock/chat";
import { agentBySlug } from "~/lib/mock/staff";

const ASSISTANT = agentBySlug("assistant")!;

function played(): MockTurnStore {
  const s: MockTurnStore = {
    entries: [HERO_OPENING],
    status: "running",
    script: [...HERO_SCRIPT],
    beat: 0,
    step: 0,
    seq: 1,
  };
  runToEnd(s);
  return s;
}

describe("the hero proof", () => {
  it("ends suspended on a permission, not finished", async () => {
    // If this ever ends `idle`, the page is demonstrating a colleague that
    // went ahead — the opposite of the product's argument.
    const s = played();
    expect(s.status).toBe("held");
    const last = s.entries[s.entries.length - 1];
    expect(last.kind).toBe("hold");
    if (last.kind === "hold") expect(last.decision).toBe("pending");
  });

  it("shows the exact thing it is about to do before anyone can answer", async () => {
    const s = played();
    const hold = s.entries.find((e) => e.kind === "hold");
    expect(hold).toBeTruthy();
    if (hold?.kind !== "hold") return;
    const args = Object.fromEntries(hold.args.map(([k, v]) => [k, v]));
    expect(args.to).toContain("@");
    expect(args.subject).toBeTruthy();
    expect(args.amount).toBeTruthy();
    expect(hold.risk).toMatch(/cannot be recalled/i);
  });

  it("answers as it thinks, in more than one piece", async () => {
    // A block of text appearing at once is a screenshot with extra steps.
    const say = HERO_SCRIPT.find((b) => b.t === "say");
    expect(say?.t).toBe("say");
    if (say?.t === "say") expect(say.chunks.length).toBeGreaterThan(8);
  });

  it("renders the opening line and the colleague at first paint", async () => {
    const { screen, render } = await createDOM();
    await render(<HeroProof agent={ASSISTANT} />);
    const proof = screen.querySelector('[data-testid="hero-proof"]');
    expect(proof).toBeTruthy();
    const text = proof?.textContent ?? "";
    expect(text).toContain(ASSISTANT.name);
    expect(text).toContain("Agent");
    expect(text).toContain("Order 4471 arrived damaged");
    expect(text).toContain("Nothing is sent until a person answers");
  });

  it("reserves the height it will need, so the card does not grow", async () => {
    // Growth under the reader is motion nobody asked for.
    const { screen, render } = await createDOM();
    await render(<HeroProof agent={ASSISTANT} />);
    const list = screen.querySelector('[data-testid="hero-proof"] ol');
    expect(list?.className ?? "").toMatch(/min-h-\[/);
    expect(list?.getAttribute("aria-live")).toBe("polite");
  });
});
