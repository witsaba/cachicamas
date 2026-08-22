/**
 * The mocked turn machine.
 *
 * `advance`, `resolveHold` and `cancelTurn` are pure over the store precisely
 * so the interesting behaviour can be driven here a tick at a time, instead of
 * being chased through a `setInterval` in a component test.
 *
 * The three properties worth defending, all of which come from the real
 * runtime contract rather than from UI taste:
 *
 *   1. A permission request SUSPENDS the machine. Nothing after it happens
 *      until a person answers (doc 0005 R-15).
 *   2. A refusal produces a visible, auditable non-event: the call is recorded
 *      as denied, and no result appears.
 *   3. Cancelling closes the streaming line where it stopped rather than
 *      leaving a half-sentence that reads as still arriving.
 */
import { describe, it, expect } from "vitest";
import {
  advance,
  cancelTurn,
  resolveHold,
  type MockTurnStore,
} from "./use-mock-turn";
import { scriptFor, type Beat } from "~/lib/mock/chat";

function store(script: readonly Beat[]): MockTurnStore {
  return {
    entries: [],
    status: "running",
    script: [...script],
    beat: 0,
    step: 0,
    seq: 1,
  };
}

/** Run the machine to a standstill, with a hard bound so a bug cannot hang. */
function run(s: MockTurnStore, maxTicks = 2000): number {
  let ticks = 0;
  while (s.status === "running" && ticks < maxTicks) {
    advance(s);
    ticks += 1;
  }
  return ticks;
}

describe("advance", () => {
  it("does nothing at all unless the machine is running", () => {
    const s = store(scriptFor("hello"));
    s.status = "idle";
    advance(s);
    expect(s.entries).toEqual([]);
  });

  it("plays a plain turn to completion and stops", () => {
    const s = store(scriptFor("who are you?"));
    const ticks = run(s);
    expect(s.status).toBe("idle");
    expect(ticks).toBeLessThan(2000);
    expect(s.entries[0]).toMatchObject({ kind: "note", label: "Working" });
    expect(s.entries[s.entries.length - 1]).toMatchObject({ kind: "note" });
  });

  it("accumulates text one delta at a time", () => {
    const s = store(scriptFor("who are you?"));
    advance(s); // TURN OPENED
    advance(s); // first delta
    const first = s.entries[1];
    expect(first.kind).toBe("said");
    if (first.kind === "said") {
      expect(first.state).toBe("streaming");
      const afterOne = first.text;
      advance(s);
      const second = s.entries[1];
      if (second.kind === "said") {
        expect(second.text.startsWith(afterOne)).toBe(true);
        expect(second.text.length).toBeGreaterThan(afterOne.length);
      }
    }
  });

  it("marks a line final on its last delta, not a tick later", () => {
    const s = store(scriptFor("who are you?"));
    run(s);
    const said = s.entries.filter((e) => e.kind === "said");
    expect(said.length).toBeGreaterThan(0);
    for (const e of said) {
      if (e.kind === "said") expect(e.state).toBe("final");
    }
  });

  it("shows a tool call running before it shows a result", () => {
    const s = store(scriptFor("show me the invoices table"));
    let sawRunning = false;
    while (s.status === "running") {
      advance(s);
      const running = s.entries.find(
        (e) => e.kind === "tool" && e.state === "running",
      );
      if (running) sawRunning = true;
    }
    expect(sawRunning).toBe(true);
    const tool = s.entries.find((e) => e.kind === "tool");
    expect(tool).toMatchObject({ state: "done" });
    if (tool?.kind === "tool") expect(tool.result).toBeTruthy();
  });
});

describe("the suspension", () => {
  it("stops the machine dead and waits", () => {
    const s = store(scriptFor("send the reminder"));
    run(s);
    expect(s.status).toBe("held");
    const last = s.entries[s.entries.length - 1];
    expect(last).toMatchObject({ kind: "hold", decision: "pending" });

    // Ticking a suspended machine must change absolutely nothing. If it
    // advanced, the run would be proceeding past a decision nobody made.
    const before = JSON.stringify(s.entries);
    for (let i = 0; i < 50; i += 1) advance(s);
    expect(JSON.stringify(s.entries)).toBe(before);
  });

  it("records the grant, runs the call once, and carries on", () => {
    const s = store(scriptFor("send the reminder"));
    run(s);
    resolveHold(s, true);
    expect(s.status).toBe("running");
    const hold = s.entries.find((e) => e.kind === "hold");
    expect(hold).toMatchObject({ decision: "granted" });
    const tools = s.entries.filter((e) => e.kind === "tool");
    expect(tools.length).toBe(1);
    expect(tools[0]).toMatchObject({ state: "done" });

    run(s);
    expect(s.status).toBe("idle");
    const prose = s.entries.filter((e) => e.kind === "said");
    expect(prose.length).toBeGreaterThan(1);
  });

  it("records the refusal as an auditable non-event", () => {
    const s = store(scriptFor("send the reminder"));
    run(s);
    resolveHold(s, false);
    const hold = s.entries.find((e) => e.kind === "hold");
    expect(hold).toMatchObject({ decision: "denied" });
    const tools = s.entries.filter((e) => e.kind === "tool");
    // The attempted call is still on the record — refusing must not erase the
    // fact that something wanted to run — but it carries no result.
    expect(tools.length).toBe(1);
    expect(tools[0]).toMatchObject({ state: "denied" });
    if (tools[0].kind === "tool") expect(tools[0].result).toBeUndefined();
  });

  it("ignores a decision when nothing is suspended", () => {
    const s = store(scriptFor("who are you?"));
    run(s);
    const before = JSON.stringify(s);
    resolveHold(s, true);
    expect(JSON.stringify(s)).toBe(before);
  });

  it("cannot be answered twice", () => {
    const s = store(scriptFor("send the reminder"));
    run(s);
    resolveHold(s, true);
    const afterFirst = s.entries.length;
    resolveHold(s, false);
    expect(s.entries.length).toBe(afterFirst);
    expect(s.entries.find((e) => e.kind === "hold")).toMatchObject({
      decision: "granted",
    });
  });
});

describe("cancelTurn", () => {
  it("closes a half-written line where it stopped", () => {
    const s = store(scriptFor("who are you?"));
    advance(s);
    advance(s);
    advance(s);
    const partial = s.entries[1];
    expect(partial).toMatchObject({ state: "streaming" });
    const textAtCancel = partial.kind === "said" ? partial.text : "";

    cancelTurn(s);
    const closed = s.entries[1];
    expect(closed).toMatchObject({ state: "final" });
    // Cancelling must not invent or discard text.
    if (closed.kind === "said") expect(closed.text).toBe(textAtCancel);
  });

  it("says the turn was cancelled, and by whom", () => {
    const s = store(scriptFor("who are you?"));
    advance(s);
    cancelTurn(s);
    const last = s.entries[s.entries.length - 1];
    expect(last).toMatchObject({ kind: "note", label: "TURN CANCELLED" });
    if (last.kind === "note") expect(last.detail).toContain("stopped by you");
  });

  it("treats cancelling a suspended run as a refusal, not as a limbo", () => {
    const s = store(scriptFor("send the reminder"));
    run(s);
    cancelTurn(s);
    expect(s.entries.find((e) => e.kind === "hold")).toMatchObject({
      decision: "denied",
    });
    expect(s.status).toBe("idle");
  });

  it("leaves nothing behind that a later turn could resume", () => {
    const s = store(scriptFor("send the reminder"));
    run(s);
    cancelTurn(s);
    expect(s.script).toEqual([]);
    expect(s.beat).toBe(0);
    expect(s.step).toBe(0);
    // Ticking after a cancel must be inert.
    const before = JSON.stringify(s.entries);
    for (let i = 0; i < 20; i += 1) advance(s);
    expect(JSON.stringify(s.entries)).toBe(before);
  });

  it("does nothing when there is nothing to cancel", () => {
    const s = store([]);
    s.status = "idle";
    cancelTurn(s);
    expect(s.entries).toEqual([]);
  });
});

describe("entry identity", () => {
  it("mints a unique id for every entry across a whole session", () => {
    const s = store(scriptFor("send the reminder"));
    run(s);
    resolveHold(s, true);
    run(s);
    s.script = [...scriptFor("show me the invoices table")];
    s.beat = 0;
    s.step = 0;
    s.status = "running";
    run(s);
    const ids = s.entries.map((e) => e.id);
    expect(new Set(ids).size).toBe(ids.length);
  });
});
