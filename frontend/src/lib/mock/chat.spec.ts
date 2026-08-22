/**
 * The scripted turn's contract.
 *
 * `scriptFor` is the branch table of the whole demonstration: which words a
 * person types decides which of the archetype's states they get to see. It is
 * pure so that table is a test rather than something you find out by typing
 * for five minutes.
 *
 * The seeded conversations are checked for the property that makes them worth
 * seeding at all: between them they put every entry kind on screen without
 * anyone having to interact.
 */
import { describe, it, expect } from "vitest";
import { CONVERSATIONS, scriptFor, type Beat } from "./chat";

const kinds = (beats: readonly Beat[]) => beats.map((b) => b.t);

describe("scriptFor", () => {
  it("always opens and closes the turn, whatever the prompt", () => {
    for (const prompt of ["hello", "drop the table", "what broke?", "select 1"]) {
      const beats = scriptFor(prompt);
      expect(beats[0], prompt).toMatchObject({ t: "note", label: "TURN OPENED" });
      const last = beats[beats.length - 1];
      expect(last.t, prompt).toBe("note");
      if (last.t === "note") {
        expect(last.label, prompt).toMatch(/TURN CLOSED/);
      }
    }
  });

  it("streams text as deltas, never as one block", () => {
    // The whole point of the mock is that the surface sees text arrive over
    // time. A single-chunk `say` would hide every defect a stream exposes.
    const say = scriptFor("hello").find((b) => b.t === "say");
    expect(say).toBeTruthy();
    if (say?.t === "say") expect(say.chunks.length).toBeGreaterThan(5);
  });

  it("suspends on a destructive verb, and offers both answers", () => {
    for (const prompt of [
      "drop the staging schema",
      "please DELETE the old rows",
      "migrate prod tonight",
    ]) {
      const beats = scriptFor(prompt);
      expect(kinds(beats), prompt).toContain("hold");
      const hold = beats.find((b) => b.t === "hold");
      if (hold?.t === "hold") {
        expect(hold.onGranted.length, prompt).toBeGreaterThan(0);
        expect(hold.onDenied.length, prompt).toBeGreaterThan(0);
        // The person must be able to read the exact call before deciding.
        expect(hold.args.length, prompt).toBeGreaterThan(0);
        expect(hold.risk.length, prompt).toBeGreaterThan(0);
      }
    }
  });

  it("quotes the person's own words back inside the call it wants to run", () => {
    const hold = scriptFor("drop schema staging cascade").find(
      (b) => b.t === "hold",
    );
    if (hold?.t === "hold") {
      const statement = hold.args.find(([k]) => k === "statement")?.[1] ?? "";
      expect(statement).toContain("drop schema staging");
    }
  });

  it("brokers a read through a tool when the prompt is about data", () => {
    const beats = scriptFor("show me the invoices table");
    expect(kinds(beats)).toContain("tool");
    expect(kinds(beats)).not.toContain("hold");
  });

  it("fails typed when the prompt reaches a system nobody owns", () => {
    const beats = scriptFor("summarise yesterday's incidents");
    const fault = beats.find((b) => b.t === "fault");
    expect(fault).toBeTruthy();
    if (fault?.t === "fault") {
      expect(fault.code).toBe("not_found");
      // A typed error names the problem AND the recovery, or it is a spinner
      // with better manners.
      expect(fault.recovery.length).toBeGreaterThan(0);
    }
  });

  it("puts the destructive branch ahead of the data branch", () => {
    // "drop the invoices table" matches both patterns. Suspending is the safe
    // resolution and must win; the opposite ordering would run it.
    expect(kinds(scriptFor("drop the invoices table"))).toContain("hold");
  });

  it("answers an ordinary question without reaching for a tool", () => {
    const beats = scriptFor("who are you?");
    expect(kinds(beats)).toEqual(["note", "say", "note"]);
  });
});

describe("the seeded conversations", () => {
  it("show every entry kind without anyone typing", () => {
    const seen = new Set(
      CONVERSATIONS.flatMap((c) => c.entries.map((e) => e.kind)),
    );
    expect([...seen].sort()).toEqual(["fault", "hold", "note", "said", "tool"]);
  });

  it("show a permission both granted and refused", () => {
    const decisions = CONVERSATIONS.flatMap((c) =>
      c.entries.flatMap((e) => (e.kind === "hold" ? [e.decision] : [])),
    );
    expect(decisions).toContain("granted");
    expect(decisions).toContain("denied");
  });

  it("never leave a refused call looking like it ran", () => {
    for (const c of CONVERSATIONS) {
      const refused = c.entries.filter(
        (e) => e.kind === "hold" && e.decision === "denied",
      );
      if (refused.length === 0) continue;
      // Nothing after a refusal may carry a tool result in that conversation.
      const results = c.entries.filter(
        (e) => e.kind === "tool" && e.state === "done",
      );
      expect(results, c.id).toEqual([]);
    }
  });

  it("give every entry an id unique within its conversation", () => {
    for (const c of CONVERSATIONS) {
      const ids = c.entries.map((e) => e.id);
      expect(new Set(ids).size, c.id).toBe(ids.length);
    }
  });

  it("never leave a line mid-stream in stored history", () => {
    for (const c of CONVERSATIONS) {
      for (const e of c.entries) {
        if (e.kind === "said") expect(e.state, `${c.id}/${e.id}`).toBe("final");
      }
    }
  });
});
