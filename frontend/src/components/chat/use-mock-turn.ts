import {
  $,
  useSignal,
  useStore,
  useVisibleTask$,
  type QRL,
} from "@builder.io/qwik";
import { scriptFor, type Beat, type TranscriptEntry } from "~/lib/mock/chat";

/**
 * useMockTurn — a scripted agent turn, driven by one clock.
 *
 * `cachicamas_chat` is planned at 0 of 12 (doc 0005), so there is no turn to
 * subscribe to. This hook plays a script instead, and it is built so the shape
 * of the real thing survives the substitution:
 *
 *   - Text arrives in DELTAS, not in one block, because the real wire does and
 *     because everything downstream of a streaming surface — scroll anchoring,
 *     the caret, the disabled composer, cancel — only shows its defects when
 *     the text actually arrives over time.
 *   - A permission request SUSPENDS THE LOOP. `status` becomes `held` and the
 *     clock genuinely stops until someone decides. Approving out of band would
 *     make the event stream stop being a complete description of the run
 *     (doc 0005 R-15), so the mock refuses to model it that way.
 *   - Cancel is a discrete act with a visible consequence: the streaming line
 *     is closed where it stopped and the turn is marked cancelled. It is not
 *     a `close()` that silently leaves a half-sentence on screen.
 *
 * One `setInterval` drives everything. Every field of the store is JSON —
 * beats included — so Qwik can serialise the whole machine across resumption
 * with no special handling.
 */

/** Milliseconds per tick. One text delta lands per tick. */
export const TICK_MS = 26;

/** How many ticks a tool call appears to be running for. */
const TOOL_TICKS = 14;

export type TurnStatus = "idle" | "running" | "held";

export interface MockTurnStore {
  entries: TranscriptEntry[];
  status: TurnStatus;
  /** The remaining beats of the turn in flight. */
  script: Beat[];
  /** Index of the beat being played. */
  beat: number;
  /** Progress within the current beat (delta index, or tool tick count). */
  step: number;
  /** Monotonic id source, so entry keys never collide across turns. */
  seq: number;
}

export interface MockTurn {
  readonly state: MockTurnStore;
  /** Send a prompt. Ignored while a turn is running or held. */
  readonly submit: QRL<(prompt: string) => void>;
  /** Cancel the turn in flight. */
  readonly cancel: QRL<() => void>;
  /** Answer the permission request the run is suspended on. */
  readonly decide: QRL<(granted: boolean) => void>;
}

/** Advance the machine by exactly one tick. Pure over the store. */
export function advance(s: MockTurnStore): void {
  if (s.status !== "running") return;
  const beat = s.script[s.beat];
  if (!beat) {
    s.status = "idle";
    return;
  }

  switch (beat.t) {
    case "note": {
      s.entries.push({
        kind: "note",
        id: `n${s.seq++}`,
        label: beat.label,
        ...(beat.detail ? { detail: beat.detail } : {}),
        ...(beat.tone ? { tone: beat.tone } : {}),
      });
      s.beat += 1;
      s.step = 0;
      return;
    }

    case "say": {
      if (s.step === 0) {
        s.entries.push({
          kind: "said",
          id: `s${s.seq++}`,
          who: "chat",
          text: "",
          state: "streaming",
        });
      }
      const last = s.entries[s.entries.length - 1];
      if (last && last.kind === "said") {
        const next = beat.chunks[s.step] ?? "";
        s.entries[s.entries.length - 1] = {
          ...last,
          text: last.text + next,
          state: s.step + 1 >= beat.chunks.length ? "final" : "streaming",
        };
      }
      s.step += 1;
      if (s.step >= beat.chunks.length) {
        s.beat += 1;
        s.step = 0;
      }
      return;
    }

    case "tool": {
      if (s.step === 0) {
        s.entries.push({
          kind: "tool",
          id: `t${s.seq++}`,
          tool: beat.tool,
          intent: beat.intent,
          args: beat.args,
          state: "running",
        });
        s.step = 1;
        return;
      }
      if (s.step < TOOL_TICKS) {
        s.step += 1;
        return;
      }
      const idx = s.entries.length - 1;
      const running = s.entries[idx];
      if (running && running.kind === "tool") {
        s.entries[idx] = { ...running, state: "done", result: beat.result };
      }
      s.beat += 1;
      s.step = 0;
      return;
    }

    case "hold": {
      s.entries.push({
        kind: "hold",
        id: `h${s.seq++}`,
        tool: beat.tool,
        intent: beat.intent,
        args: beat.args,
        risk: beat.risk,
        decision: "pending",
      });
      // The loop is suspended here, not around here. Nothing advances until
      // `decide` is called.
      s.status = "held";
      return;
    }

    case "fault": {
      s.entries.push({
        kind: "fault",
        id: `x${s.seq++}`,
        code: beat.code,
        message: beat.message,
        recovery: beat.recovery,
      });
      s.beat += 1;
      s.step = 0;
      return;
    }
  }
}

/** Resolve a suspended permission request. Pure over the store. */
export function resolveHold(s: MockTurnStore, granted: boolean): void {
  if (s.status !== "held") return;
  const beat = s.script[s.beat];
  if (!beat || beat.t !== "hold") return;

  const idx = s.entries.length - 1;
  const pending = s.entries[idx];
  if (pending && pending.kind === "hold") {
    s.entries[idx] = { ...pending, decision: granted ? "granted" : "denied" };
  }

  if (granted) {
    s.entries.push({
      kind: "tool",
      id: `t${s.seq++}`,
      tool: beat.tool,
      intent: beat.intent,
      args: beat.args,
      result: beat.result,
      state: "done",
    });
  } else {
    s.entries.push({
      kind: "tool",
      id: `t${s.seq++}`,
      tool: beat.tool,
      intent: beat.intent,
      args: beat.args,
      state: "denied",
    });
  }

  // The turn continues with whatever the model says about the decision.
  s.script[s.beat] = {
    t: "say",
    chunks: granted ? beat.onGranted : beat.onDenied,
  };
  s.step = 0;
  s.status = "running";
}

/** Close a streaming line where it stopped and mark the turn cancelled. */
export function cancelTurn(s: MockTurnStore): void {
  if (s.status === "idle") return;
  const idx = s.entries.length - 1;
  const last = s.entries[idx];
  if (last && last.kind === "said" && last.state === "streaming") {
    s.entries[idx] = { ...last, state: "final" };
  }
  if (last && last.kind === "hold" && last.decision === "pending") {
    s.entries[idx] = { ...last, decision: "denied" };
  }
  s.entries.push({
    kind: "note",
    id: `n${s.seq++}`,
    label: "TURN CANCELLED",
    detail: "stopped by you · nothing further ran",
    tone: "fail",
  });
  s.script = [];
  s.beat = 0;
  s.step = 0;
  s.status = "idle";
}

export function useMockTurn(seed: readonly TranscriptEntry[] = []): MockTurn {
  const state = useStore<MockTurnStore>(
    {
      entries: [...seed],
      status: "idle",
      script: [],
      beat: 0,
      step: 0,
      seq: 1,
    },
    { deep: true },
  );
  const timer = useSignal(0);

  // The clock only exists in a browser, and this task holds nothing but its
  // own teardown — it does no work on mount.
  // eslint-disable-next-line qwik/no-use-visible-task
  useVisibleTask$(({ cleanup }) => {
    if (typeof window === "undefined") return;
    cleanup(() => {
      if (timer.value) clearInterval(timer.value);
    });
  });

  const start = $(() => {
    if (timer.value) clearInterval(timer.value);
    timer.value = setInterval(() => {
      advance(state);
      if (state.status !== "running" && timer.value) {
        clearInterval(timer.value);
        timer.value = 0;
      }
    }, TICK_MS) as unknown as number;
  });

  const submit = $(async (prompt: string) => {
    const text = prompt.trim();
    if (text.length === 0 || state.status !== "idle") return;
    state.entries.push({
      kind: "said",
      id: `u${state.seq++}`,
      who: "you",
      text,
      state: "final",
    });
    state.script = [...scriptFor(text)];
    state.beat = 0;
    state.step = 0;
    state.status = "running";
    await start();
  });

  const cancel = $(() => {
    if (timer.value) {
      clearInterval(timer.value);
      timer.value = 0;
    }
    cancelTurn(state);
  });

  const decide = $(async (granted: boolean) => {
    resolveHold(state, granted);
    if (state.status === "running") await start();
  });

  return { state, submit, cancel, decide };
}
