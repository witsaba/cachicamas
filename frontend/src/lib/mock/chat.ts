/**
 * chat.ts — the chat archetype's demonstration transcript and its scripted turns.
 *
 * DEMONSTRATION DATA. `cachicamas_chat` is planned at 0 of 12 (doc 0005); no
 * turn in this file ever reached a model. What is real is the *shape*: the
 * entry kinds below are the browser-visible half of the frozen agent event
 * stream — a turn opens, text arrives in deltas, a tool call is proposed, a
 * permission decision suspends the run inside the loop rather than beside it,
 * a failure arrives as a typed envelope, and the turn closes with its cost.
 *
 * Keeping the shape honest is the point of the mock: when CH-03..CH-05 wire
 * the real stream, the component tree consuming these types should not have
 * to change, and anything that does change is a seam Layer 2 did not expose.
 *
 * The live wire itself is untouched and still lives in `lib/chat-api.ts` and
 * `lib/chat-types.ts`, unused by this mockup and waiting for CH-05.
 */

/** One line of a conversation, as the browser sees it. */
export type TranscriptEntry =
  /** The system narrating the run: turn boundaries, cancellation, cost. */
  | {
      readonly kind: "note";
      readonly id: string;
      readonly label: string;
      readonly detail?: string;
      readonly tone?: "plain" | "live" | "fail";
    }
  /** Language: what a person typed, or what the model answered. */
  | {
      readonly kind: "said";
      readonly id: string;
      readonly who: "you" | "chat";
      readonly text: string;
      readonly state: "final" | "streaming";
    }
  /** A tool the archetype ran, and what came back. */
  | {
      readonly kind: "tool";
      readonly id: string;
      readonly tool: string;
      readonly intent: string;
      readonly args: readonly (readonly [string, string])[];
      readonly result?: string;
      readonly state: "running" | "done" | "denied" | "failed";
    }
  /**
   * The run is suspended inside the loop, waiting on a person. This is the
   * product's defining moment: the agent cannot proceed and says exactly what
   * it wants to do before anyone decides (v2 § 6 seam 2, doc 0005 R-15).
   */
  | {
      readonly kind: "hold";
      readonly id: string;
      readonly tool: string;
      readonly intent: string;
      readonly args: readonly (readonly [string, string])[];
      readonly risk: string;
      readonly decision: "pending" | "granted" | "denied";
    }
  /** A typed error envelope. Never a spinner that never ends (doc 0005 R-12). */
  | {
      readonly kind: "fault";
      readonly id: string;
      readonly code: string;
      readonly message: string;
      readonly recovery: string;
    };

export interface Conversation {
  readonly id: string;
  readonly title: string;
  readonly turns: number;
  /** Relative, so the mock never looks stale. */
  readonly age: string;
  readonly entries: readonly TranscriptEntry[];
}

/**
 * Three seeded conversations, chosen to put every entry kind on screen at
 * least once without anyone having to type: a granted permission and its
 * tool result, a refused one, and a typed failure.
 */
export const CONVERSATIONS: readonly Conversation[] = [
  {
    id: "c-4471",
    title: "Who owns the invoices table",
    turns: 4,
    age: "2m",
    entries: [
      {
        kind: "note",
        id: "e1",
        label: "TURN OPENED",
        detail: "run 4471 · turn 1",
      },
      {
        kind: "said",
        id: "e2",
        who: "you",
        text: "Which archetype owns the invoices table, and can you read a row from it for me?",
        state: "final",
      },
      {
        kind: "said",
        id: "e3",
        who: "chat",
        text: "Finance owns `invoices` — every business system owns its own tables, and no archetype writes into another's schema. I can't read it directly. What I can do is ask the Database Administrator archetype over its MCP server; it brokers all database work. That needs your permission first.",
        state: "final",
      },
      {
        kind: "hold",
        id: "e4",
        tool: "dba.query",
        intent: "Ask the Database Administrator to read one row from finance.invoices",
        args: [
          ["system", "finance"],
          ["statement", "select * from invoices order by issued_at desc limit 1"],
          ["mode", "read-only"],
        ],
        risk: "Reads a table this archetype does not own. Nothing is written.",
        decision: "granted",
      },
      {
        kind: "tool",
        id: "e5",
        tool: "dba.query",
        intent: "Read one row from finance.invoices",
        args: [
          ["system", "finance"],
          ["rows", "1"],
        ],
        result: "1 row · invoice #2291 · ACME INDUSTRIAL · 4 820.00 · issued 2026-08-19 · unpaid",
        state: "done",
      },
      {
        kind: "said",
        id: "e6",
        who: "chat",
        text: "The most recent invoice is **#2291** to ACME INDUSTRIAL for 4 820.00, issued 2026-08-19 and still unpaid. If you want it chased, that is Finance's job rather than mine — I can hand it over.",
        state: "final",
      },
      {
        kind: "note",
        id: "e7",
        label: "TURN CLOSED",
        detail: "stop · 1 tool call · 2 140 in / 318 out",
      },
    ],
  },
  {
    id: "c-4468",
    title: "Drop the staging schema",
    turns: 2,
    age: "1h",
    entries: [
      { kind: "note", id: "f1", label: "TURN OPENED", detail: "run 4468 · turn 1" },
      {
        kind: "said",
        id: "f2",
        who: "you",
        text: "Drop the staging schema, we are rebuilding it tonight.",
        state: "final",
      },
      {
        kind: "said",
        id: "f3",
        who: "chat",
        text: "That is destructive and it is not mine to do — I would be asking the Database Administrator to run it. Here is exactly what I would send.",
        state: "final",
      },
      {
        kind: "hold",
        id: "f4",
        tool: "dba.execute",
        intent: "Ask the Database Administrator to drop the staging schema",
        args: [
          ["system", "staging"],
          ["statement", "drop schema staging cascade"],
          ["mode", "destructive"],
        ],
        risk: "Irreversible. Removes every table, view and row in the staging schema.",
        decision: "denied",
      },
      {
        kind: "note",
        id: "f5",
        label: "TURN CLOSED",
        detail: "permission refused · nothing ran · 980 in / 96 out",
        tone: "fail",
      },
    ],
  },
  {
    id: "c-4462",
    title: "Summarise yesterday's incidents",
    turns: 1,
    age: "yesterday",
    entries: [
      { kind: "note", id: "g1", label: "TURN OPENED", detail: "run 4462 · turn 1" },
      {
        kind: "said",
        id: "g2",
        who: "you",
        text: "Summarise yesterday's incidents from the ticket system.",
        state: "final",
      },
      {
        kind: "fault",
        id: "g3",
        code: "not_found",
        message: "No archetype owns the ticket system yet, so there is no MCP server to ask.",
        recovery:
          "Ticketing is registered as UNPLANNED (ADR 0009 § D4). Nothing to retry — the capability does not exist.",
      },
      {
        kind: "note",
        id: "g4",
        label: "TURN CLOSED",
        detail: "failed · no retry · 210 in / 0 out",
        tone: "fail",
      },
    ],
  },
] as const;

/* ===========================================================================
 * The scripted turn.
 *
 * Typing into the composer runs one of the scripts below. They are chosen by
 * what the prompt contains, so the demonstration can reach every state — a
 * plain answer, a brokered tool call, a permission the run genuinely waits on,
 * and a typed failure — without anyone being told which words to type.
 * ======================================================================== */

export type Beat =
  | { readonly t: "note"; readonly label: string; readonly detail?: string; readonly tone?: "plain" | "live" | "fail" }
  | { readonly t: "say"; readonly chunks: readonly string[] }
  | {
      readonly t: "tool";
      readonly tool: string;
      readonly intent: string;
      readonly args: readonly (readonly [string, string])[];
      readonly result: string;
    }
  | {
      readonly t: "hold";
      readonly tool: string;
      readonly intent: string;
      readonly args: readonly (readonly [string, string])[];
      readonly risk: string;
      /** What the model says once the person allows it. */
      readonly onGranted: readonly string[];
      /** What it says once the person refuses. */
      readonly onDenied: readonly string[];
      readonly result: string;
    }
  | { readonly t: "fault"; readonly code: string; readonly message: string; readonly recovery: string };

/** Split a sentence into delta-sized chunks, the way a real stream arrives. */
function deltas(text: string): readonly string[] {
  return text.split(/(?<=\s)/);
}

// Plurals and gerunds count: `incidents` and `tickets` are how people
// actually write, and a branch table that only matched the singular would
// silently fall through to the generic answer.
const DESTRUCTIVE =
  /\b(drop|delete|deletes?|truncate|deploy|migrate|revoke|purge|reset)\w*\b/i;
const DATABASE =
  /\b(table|schema|quer|row|column|index|sql|database|invoice)\w*\b/i;
const BROKEN = /\b(fail|error|break|broke|crash|incident|ticket)\w*\b/i;

/**
 * Pick the script for a prompt. Pure, so the branch table is a unit test
 * rather than something you discover by typing.
 */
export function scriptFor(prompt: string): readonly Beat[] {
  const opened: Beat = { t: "note", label: "TURN OPENED", detail: "run 4472 · turn 1", tone: "live" };

  if (BROKEN.test(prompt)) {
    return [
      opened,
      {
        t: "say",
        chunks: deltas(
          "Let me look. That would go through the ticket system, which is a business system rather than something I hold myself.",
        ),
      },
      {
        t: "fault",
        code: "not_found",
        message: "No archetype owns the ticket system yet, so there is no MCP server to ask.",
        recovery:
          "Ticketing is registered as UNPLANNED (ADR 0009 § D4). Nothing to retry — the capability does not exist.",
      },
      { t: "note", label: "TURN CLOSED", detail: "failed · no retry · 264 in / 41 out", tone: "fail" },
    ];
  }

  if (DESTRUCTIVE.test(prompt)) {
    return [
      opened,
      {
        t: "say",
        chunks: deltas(
          "That one is destructive, and it is not mine to run — the Database Administrator owns the tables. Here is exactly what I would ask it to do.",
        ),
      },
      {
        t: "hold",
        tool: "dba.execute",
        intent: "Ask the Database Administrator to run a destructive statement",
        args: [
          ["system", "staging"],
          ["statement", prompt.trim().slice(0, 72)],
          ["mode", "destructive"],
        ],
        risk: "Irreversible once it runs. The run is suspended here until you decide.",
        onGranted: deltas("Done. It ran once, and I have the receipt above."),
        onDenied: deltas("Understood — nothing ran, and I will not ask again this turn."),
        result: "statement accepted · 1 statement · 0 rows returned",
      },
      { t: "note", label: "TURN CLOSED", detail: "stop · 1 tool call · 1 402 in / 176 out" },
    ];
  }

  if (DATABASE.test(prompt)) {
    return [
      opened,
      {
        t: "say",
        chunks: deltas("I can ask the Database Administrator for that — it brokers every read."),
      },
      {
        t: "tool",
        tool: "dba.query",
        intent: "Read from the owning system, read-only",
        args: [
          ["mode", "read-only"],
          ["statement", prompt.trim().slice(0, 72)],
        ],
        result: "3 rows · 41 ms",
      },
      {
        t: "say",
        chunks: deltas(
          "Three rows came back in 41 ms. Ask me to narrow it and I will send a tighter statement rather than filtering here.",
        ),
      },
      { t: "note", label: "TURN CLOSED", detail: "stop · 1 tool call · 1 118 in / 204 out" },
    ];
  }

  return [
    opened,
    {
      t: "say",
      chunks: deltas(
        "I am the chat archetype — the thinnest specialist there is. I hold a conversation, and when something needs doing I hand it to whichever archetype owns it. Right now I am the only one with a plan in flight, so most of what you ask me I can only describe.",
      ),
    },
    { t: "note", label: "TURN CLOSED", detail: "stop · 0 tool calls · 842 in / 152 out" },
  ];
}
