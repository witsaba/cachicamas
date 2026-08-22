/**
 * The scripted conversation's contract.
 *
 * `scriptFor` is the branch table of the whole demonstration: which words a
 * person types decides which of a colleague's states they get to see. It is
 * pure so that table is a test rather than something you find out by typing
 * for five minutes.
 *
 * The seeded conversations are checked for the property that makes them worth
 * seeding at all: between them they put every kind of line on screen before
 * anyone interacts with anything.
 */
import { describe, it, expect } from "vitest";
import { CONVERSATIONS, scriptFor, type Beat } from "./chat";
import { AGENTS } from "./staff";

const kinds = (beats: readonly Beat[]) => beats.map((b) => b.t);

describe("scriptFor", () => {
  it("always opens and closes, whatever the message", () => {
    for (const prompt of [
      "hello",
      "send the invoice",
      "what broke?",
      "how did we do this quarter",
    ]) {
      const beats = scriptFor(prompt);
      expect(beats[0], prompt).toMatchObject({ t: "note", label: "Working" });
      const last = beats[beats.length - 1];
      expect(last.t, prompt).toBe("note");
    }
  });

  it("never opens with a tool call before it has said anything", () => {
    // A colleague that acts before it speaks is one nobody can follow.
    for (const prompt of ["send it", "why did it break", "unpaid invoices"]) {
      const beats = scriptFor(prompt);
      const firstAction = beats.findIndex(
        (b) => b.t === "tool" || b.t === "hold",
      );
      const firstWord = beats.findIndex((b) => b.t === "say");
      if (firstAction >= 0) {
        expect(firstWord, prompt).toBeGreaterThanOrEqual(0);
        expect(firstWord, prompt).toBeLessThan(firstAction);
      }
    }
  });

  describe("anything that leaves the building waits for a person", () => {
    for (const prompt of [
      "send the reminder",
      "email them the quote",
      "refund order 4471",
      "delete the duplicate contact",
      "cancel their plan",
      "approve these expenses",
    ]) {
      it(`"${prompt}" suspends rather than acting`, () => {
        const beats = scriptFor(prompt);
        expect(kinds(beats), prompt).toContain("hold");
        const hold = beats.find((b) => b.t === "hold");
        if (hold?.t === "hold") {
          // The person is told what it wants to do, with what, and what the
          // consequence is — before there is anything to answer.
          expect(hold.intent.length, prompt).toBeGreaterThan(8);
          expect(hold.args.length, prompt).toBeGreaterThan(0);
          expect(hold.risk.length, prompt).toBeGreaterThan(8);
          // And both answers exist. A permission with only one branch is a
          // confirmation dialog wearing a permission's clothes.
          expect(hold.onGranted.length, prompt).toBeGreaterThan(0);
          expect(hold.onDenied.length, prompt).toBeGreaterThan(0);
        }
      });
    }
  });

  it("reads before it answers a question about money", () => {
    const beats = scriptFor("which invoices are unpaid");
    expect(kinds(beats)).toContain("tool");
    const tool = beats.find((b) => b.t === "tool");
    if (tool?.t === "tool") {
      expect(tool.args.length).toBeGreaterThan(0);
      expect(tool.result.length).toBeGreaterThan(0);
    }
  });

  it("fails once, names the recovery, and never retries", () => {
    const beats = scriptFor("the calendar sync stopped");
    const fault = beats.find((b) => b.t === "fault");
    expect(fault).toBeTruthy();
    if (fault?.t === "fault") {
      // A failure names the problem AND the way out, or it is a dead end with
      // better manners.
      expect(fault.message.length).toBeGreaterThan(12);
      expect(fault.recovery.length).toBeGreaterThan(12);
    }
    expect(kinds(beats)).not.toContain("tool");
    // Nothing retries: exactly one failure line per failed conversation.
    expect(beats.filter((b) => b.t === "fault")).toHaveLength(1);
  });

  it("plurals and gerunds count — people do not write in the singular", () => {
    // `refunds` and `incidents` are how the branch table gets missed.
    expect(kinds(scriptFor("process these refunds"))).toContain("hold");
    expect(kinds(scriptFor("two syncs are broken"))).toContain("fault");
  });

  it("asking beats acting when a message matches two branches", () => {
    // "delete the duplicate invoices" is both money and an irreversible act.
    // Suspending must win; the opposite ordering would do it.
    expect(kinds(scriptFor("delete the duplicate invoices"))).toContain("hold");
  });
});

describe("the seeded conversations", () => {
  it("put every kind of line on screen before anyone types", () => {
    const seen = new Set(
      CONVERSATIONS.flatMap((c) => c.entries.map((e) => e.kind)),
    );
    for (const kind of ["note", "said", "tool", "hold", "fault"]) {
      expect(seen, kind).toContain(kind);
    }
  });

  it("show a permission answered both ways, and one still waiting", () => {
    const decisions = CONVERSATIONS.flatMap((c) =>
      c.entries.flatMap((e) => (e.kind === "hold" ? [e.decision] : [])),
    );
    expect(decisions).toContain("granted");
    expect(decisions).toContain("denied");
    expect(decisions).toContain("pending");
  });

  it("belong to colleagues who actually work here", () => {
    const slugs = new Set(AGENTS.map((a) => a.slug));
    for (const c of CONVERSATIONS) {
      expect(slugs, c.id).toContain(c.agentSlug);
    }
  });

  it("mention no part of how the product is built", () => {
    // The people using this hire colleagues; they do not run a runtime. A word
    // from the engineering vocabulary on a customer surface is a defect.
    const forbidden =
      /\b(archetype|runtime|MCP|schema|layer 1|layer 2|SSE|stream|token|endpoint|milestone)\b/i;
    for (const c of CONVERSATIONS) {
      for (const e of c.entries) {
        const text = JSON.stringify(e);
        expect(text, `${c.id}/${e.id}`).not.toMatch(forbidden);
      }
    }
  });
});
