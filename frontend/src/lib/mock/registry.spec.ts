/**
 * The register's contract.
 *
 * Two different kinds of claim live in `registry.ts` and they have different
 * standards of proof. The STRUCTURAL facts — layer counts, milestone counts,
 * which archetype owns which system — are read from this repository's own
 * documents and are checked here against those numbers. The ACTIVITY figures
 * are invented, and what is checked about them is that they are confined to
 * archetypes the interface labels as demonstration data.
 *
 * If a milestone document's count changes and this file does not, this suite
 * is where the drift surfaces — which is the whole reason the numbers are not
 * scattered through the JSX.
 */
import { describe, it, expect } from "vitest";
import {
  ARCHETYPES,
  RUNTIME,
  archetypeByCode,
  archetypeBySlug,
  archetypeHref,
  resolveCommand,
  suggestCommands,
} from "./registry";

describe("the archetype register", () => {
  it("gives every archetype a unique code, slug and function key", () => {
    const codes = ARCHETYPES.map((a) => a.code);
    const slugs = ARCHETYPES.map((a) => a.slug);
    const keys = ARCHETYPES.map((a) => a.fkey);
    expect(new Set(codes).size).toBe(codes.length);
    expect(new Set(slugs).size).toBe(slugs.length);
    expect(new Set(keys).size).toBe(keys.length);
  });

  it("gives the dock a name short enough to fit beside the others", () => {
    for (const a of ARCHETYPES) {
      expect(a.dockName.length, a.code).toBeLessThanOrEqual(10);
      expect(a.dockName, a.code).not.toContain(" ");
    }
  });

  it("keeps codes typeable: uppercase, no spaces", () => {
    for (const a of ARCHETYPES) {
      expect(a.code, a.code).toMatch(/^[A-Z]+$/);
    }
  });

  it("claims nothing is on duty, because nothing is", () => {
    // No Layer 3 archetype exists on disk. The moment one does, this
    // assertion should fail and be updated deliberately.
    expect(ARCHETYPES.filter((a) => a.state === "on-duty")).toEqual([]);
  });

  it("gives every archetype a literal status word and a cited authority", () => {
    for (const a of ARCHETYPES) {
      expect(a.stateWord.length, a.code).toBeGreaterThan(0);
      // Colour alone may never carry the state (PRODUCT.md § Accessibility).
      expect(a.stateWord, a.code).toBe(a.stateWord.toUpperCase());
      // Nothing appears on the board without a decision record behind it.
      expect(a.authority, a.code).toMatch(/ADR \d{4}/);
    }
  });

  it("says out loud why each unbuilt specialist is unbuilt", () => {
    for (const a of ARCHETYPES) {
      expect(a.blockedBy.length, a.code).toBeGreaterThan(40);
      expect(a.wouldDo.length, a.code).toBeGreaterThan(0);
    }
  });

  it("only shows an activity figure for the archetype under construction", () => {
    // An invented number beside a specialist nobody has started building would
    // read as a working system. Only `chat` carries one, and the interface
    // marks it.
    const withFigures = ARCHETYPES.filter((a) => a.demoTurnsToday !== null);
    expect(withFigures.map((a) => a.code)).toEqual(["CHAT"]);
    expect(withFigures[0].state).toBe("in-build");
  });

  it("carries a milestone plan only where a plan document exists", () => {
    const planned = ARCHETYPES.filter((a) => a.plan !== null);
    expect(planned.map((a) => a.code).sort()).toEqual(["CHAT", "CODING"]);
    // doc 0005 — the chat archetype, 12 milestones, none shipped.
    expect(archetypeByCode("CHAT")!.plan).toEqual({
      label: "doc 0005",
      done: 0,
      total: 12,
    });
    // doc 0004 — the coding archetype, 25 milestones, none shipped.
    expect(archetypeByCode("CODING")!.plan).toEqual({
      label: "doc 0004",
      done: 0,
      total: 25,
    });
  });
});

describe("the runtime beneath the register", () => {
  it("reports the real, shipped layer counts", () => {
    expect(RUNTIME.map((l) => [l.code, l.done, l.total])).toEqual([
      ["L1", 42, 42],
      ["L2", 24, 24],
      ["L3", 0, ARCHETYPES.length],
    ]);
  });

  it("counts Layer 3 against the register, so the two can never disagree", () => {
    expect(RUNTIME[2].total).toBe(ARCHETYPES.length);
    expect(RUNTIME[2].done).toBe(
      ARCHETYPES.filter((a) => a.state === "on-duty").length,
    );
  });
});

describe("archetypeHref", () => {
  it("sends chat to its own application and everything else to its record", () => {
    expect(archetypeHref(archetypeByCode("CHAT")!)).toBe("/chat/");
    expect(archetypeHref(archetypeByCode("DBA")!)).toBe("/archetypes/dba/");
  });
});

describe("lookups", () => {
  it("resolves a code case-insensitively, with surrounding space", () => {
    expect(archetypeByCode("  chat ")?.code).toBe("CHAT");
    expect(archetypeByCode("Chat")?.code).toBe("CHAT");
  });

  it("resolves a slug case-insensitively", () => {
    expect(archetypeBySlug("DBA")?.code).toBe("DBA");
  });

  it("returns undefined rather than guessing", () => {
    expect(archetypeByCode("CH")).toBeUndefined();
    expect(archetypeBySlug("cha")).toBeUndefined();
  });
});

describe("resolveCommand", () => {
  it("opens an archetype by its code", () => {
    expect(resolveCommand("chat")).toEqual({
      ok: true,
      href: "/chat/",
      label: "Chat",
    });
  });

  it("opens the system surfaces by name", () => {
    expect(resolveCommand("SYSTEM").ok).toBe(true);
    expect(resolveCommand("desk")).toMatchObject({ href: "/home/" });
    expect(resolveCommand("signout")).toMatchObject({ href: "/auth/signout/" });
  });

  it("passes a bare route straight through", () => {
    expect(resolveCommand("/profile/")).toMatchObject({ href: "/profile/" });
  });

  it("refuses out loud, naming what it does accept", () => {
    // A command bar that silently does nothing on an unknown input is the
    // single most common defect in the pattern.
    const result = resolveCommand("finanace");
    expect(result.ok).toBe(false);
    if (!result.ok) {
      expect(result.message).toContain("finanace");
      for (const a of ARCHETYPES) {
        expect(result.message).toContain(a.code);
      }
    }
  });

  it("treats an empty line as a prompt for help, not as an error about nothing", () => {
    const result = resolveCommand("   ");
    expect(result.ok).toBe(false);
    if (!result.ok) expect(result.message).toContain("CHAT");
  });
});

describe("suggestCommands", () => {
  it("offers everything reachable when the line is empty", () => {
    const all = suggestCommands("");
    for (const a of ARCHETYPES) {
      expect(all.map((s) => s.code)).toContain(a.code);
    }
    expect(all.map((s) => s.code)).toContain("SYSTEM");
  });

  it("narrows by prefix on the code and on the label", () => {
    expect(suggestCommands("ch").map((s) => s.code)).toEqual(["CHAT"]);
    expect(suggestCommands("data").map((s) => s.code)).toEqual(["DBA"]);
  });

  it("returns nothing rather than everything when nothing matches", () => {
    expect(suggestCommands("zzz")).toEqual([]);
  });

  it("every suggestion resolves to the destination it advertises", () => {
    for (const s of suggestCommands("")) {
      const resolved = resolveCommand(s.code);
      expect(resolved.ok, s.code).toBe(true);
      if (resolved.ok) expect(resolved.href, s.code).toBe(s.href);
    }
  });
});
