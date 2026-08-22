/**
 * chat.ts — the demonstration conversations, and the scripts a typed message
 * plays out.
 *
 * DEMONSTRATION DATA. No message in this file ever reached a model. What is
 * real is the *shape* of a conversation with a colleague who can act: they
 * answer as they think, they use a tool and show you which one, they stop and
 * ask before doing anything that leaves the building, and when something is
 * genuinely broken they say so once instead of retrying forever.
 *
 * That shape is what the interface has to get right, because it is the part
 * that will not change when the conversations become real.
 */

/** One line of a conversation, as a person sees it. */
export type TranscriptEntry =
  /** The conversation narrating itself: started, stopped, finished. */
  | {
      readonly kind: "note";
      readonly id: string;
      readonly label: string;
      readonly detail?: string;
      readonly tone?: "plain" | "live" | "fail";
    }
  /** Language: what a person wrote, or what their colleague answered. */
  | {
      readonly kind: "said";
      readonly id: string;
      /** `chat` is the colleague's side. The screen renders their real name. */
      readonly who: "you" | "chat";
      readonly text: string;
      readonly state: "final" | "streaming";
    }
  /** A tool the colleague used, and what came back. */
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
   * Waiting on you. This is the moment the whole product is built around: a
   * colleague who is about to do something that leaves the building stops,
   * says exactly what it is, and does not move until a person answers.
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
  /** Something is genuinely wrong. Said once, with what to do about it. */
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
  /** Which colleague this conversation is with. Matches an `Agent.slug`. */
  readonly agentSlug: string;
  readonly turns: number;
  /** Relative, so the mock never looks stale. */
  readonly age: string;
  readonly entries: readonly TranscriptEntry[];
}

/**
 * Three seeded conversations, chosen so that every kind of line above appears
 * at least once before anybody types anything.
 */
export const CONVERSATIONS: readonly Conversation[] = [
  {
    id: "c-4471",
    title: "Anything unpaid past 30 days",
    agentSlug: "finance",
    turns: 4,
    age: "12 minutes ago",
    entries: [
      {
        kind: "note",
        id: "e1",
        label: "Conversation started",
        detail: "This morning",
      },
      {
        kind: "said",
        id: "e2",
        who: "you",
        text: "Is anything unpaid past 30 days? Just the amounts, I have a call in ten minutes.",
        state: "final",
      },
      {
        kind: "said",
        id: "e3",
        who: "chat",
        text: "Three, **£12,400** in total. Let me pull the exact ages rather than guess them.",
        state: "final",
      },
      {
        kind: "tool",
        id: "e4",
        tool: "Ledger",
        intent: "Read invoices marked unpaid",
        args: [
          ["older than", "30 days"],
          ["status", "unpaid"],
          ["access", "read only"],
        ],
        result: "3 invoices · £12,400 · oldest 61 days",
        state: "done",
      },
      {
        kind: "said",
        id: "e5",
        who: "chat",
        text: "Northgate at 61 days (£7,900), Halden at 44 (£3,100), and Pell & Co at 33 (£1,400). Northgate is the one worth a phone call — the other two have paid late before and always paid.\n\nI can draft the chasers, but I won't send anything without you.",
        state: "final",
      },
      {
        kind: "note",
        id: "e6",
        label: "Finished",
        detail: "One tool used · nothing left the building",
      },
    ],
  },
  {
    id: "c-4468",
    title: "Refund for order 4471",
    agentSlug: "support",
    turns: 3,
    age: "Yesterday",
    entries: [
      {
        kind: "note",
        id: "f1",
        label: "Conversation started",
        detail: "Yesterday, 16:20",
      },
      {
        kind: "said",
        id: "f2",
        who: "you",
        text: "Order 4471 arrived damaged. Refund it and tell them.",
        state: "final",
      },
      {
        kind: "said",
        id: "f3",
        who: "chat",
        text: "Within policy — under £200 and reported inside 14 days, so it doesn't need a second pair of eyes. I've written the email. Read it before it goes.",
        state: "final",
      },
      {
        kind: "hold",
        id: "f4",
        tool: "Helpdesk",
        intent: "Send a refund confirmation to the customer",
        args: [
          ["to", "j.moreau@northgate.example"],
          ["subject", "Your refund for order 4471"],
          ["amount", "£148.00"],
        ],
        risk: "This email leaves the building and cannot be recalled.",
        decision: "pending",
      },
    ],
  },
  {
    id: "c-4465",
    title: "Chasing Northgate",
    agentSlug: "finance",
    turns: 5,
    age: "Tuesday",
    entries: [
      {
        kind: "note",
        id: "h1",
        label: "Conversation started",
        detail: "Tuesday, 11:40",
      },
      {
        kind: "said",
        id: "h2",
        who: "you",
        text: "Northgate is 61 days late. Send them the reminder, but don't touch the payment plan we agreed.",
        state: "final",
      },
      {
        kind: "hold",
        id: "h3",
        tool: "Billing",
        intent: "Send a payment reminder",
        args: [
          ["to", "accounts@northgate.example"],
          ["about", "Invoice 7712 · £7,900"],
          ["tone", "firm, not final notice"],
        ],
        risk: "This email leaves the building and cannot be recalled.",
        decision: "granted",
      },
      {
        kind: "tool",
        id: "h4",
        tool: "Billing",
        intent: "Send a payment reminder",
        args: [["to", "accounts@northgate.example"]],
        result: "1 message sent · logged against your name",
        state: "done",
      },
      {
        kind: "hold",
        id: "h5",
        tool: "Billing",
        intent: "Cancel the agreed payment plan",
        args: [
          ["account", "Northgate"],
          ["effect", "Full balance due immediately"],
        ],
        risk: "This changes what the customer owes and cannot be undone here.",
        decision: "denied",
      },
      {
        kind: "said",
        id: "h6",
        who: "chat",
        text: "Understood — the plan stands. The reminder went out and I have noted that you refused the escalation, so nobody else here will suggest it again this week.",
        state: "final",
      },
      {
        kind: "note",
        id: "h7",
        label: "Finished",
        detail: "One approval given, one refused",
      },
    ],
  },
  {
    id: "c-4460",
    title: "Why did the calendar sync stop",
    agentSlug: "integrations",
    turns: 2,
    age: "Monday",
    entries: [
      {
        kind: "note",
        id: "g1",
        label: "Conversation started",
        detail: "Monday, 09:05",
      },
      {
        kind: "said",
        id: "g2",
        who: "you",
        text: "The calendar sync stopped some time last week and nobody noticed. Why?",
        state: "final",
      },
      {
        kind: "said",
        id: "g3",
        who: "chat",
        text: "I can see it stopped, and I can see when. I can't see why yet — I'm still being set up and don't have access to the calendar account.",
        state: "final",
      },
      {
        kind: "fault",
        id: "g4",
        code: "No access",
        message: "I have not been given access to the calendar account.",
        recovery:
          "Someone with admin rights can grant it from Organisation → Connected tools. I will not keep retrying in the meantime.",
      },
      {
        kind: "note",
        id: "g5",
        label: "Stopped",
        detail: "Nothing retried",
        tone: "fail",
      },
    ],
  },
];

/* ===========================================================================
 * The scripts.
 *
 * Typing into the composer plays one of the scripts below. They are chosen by
 * what the message contains, so a person can reach every state — a plain
 * answer, a tool used in the open, an approval the conversation genuinely
 * waits on, and an honest failure — without being told which words to type.
 * ======================================================================== */

export type Beat =
  | {
      readonly t: "note";
      readonly label: string;
      readonly detail?: string;
      readonly tone?: "plain" | "live" | "fail";
    }
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
      /** What they say once you allow it. */
      readonly onGranted: readonly string[];
      /** What they say once you refuse. */
      readonly onDenied: readonly string[];
      readonly result: string;
    }
  | {
      readonly t: "fault";
      readonly code: string;
      readonly message: string;
      readonly recovery: string;
    };

/** Split a sentence into the chunks an answer actually arrives in. */
function deltas(text: string): readonly string[] {
  return text.split(/(?<=\s)/);
}

// Plurals and gerunds count: people write `refunds` and `invoices`, and a
// branch table matching only the singular would fall through to the generic
// answer and quietly make the demonstration look thinner than it is.
const NEEDS_APPROVAL =
  /\b(send|sends?|email|refund|pay|paid|publish|post|delete|remove|cancel|charge|transfer|approve)\w*\b/i;
const MONEY =
  /\b(invoice|unpaid|owed|expense|budget|spend|cost|revenue|payment|supplier|close)\w*\b/i;
const BROKEN =
  /\b(fail|error|broke|broken|stopped|down|crash|incident|outage|missing)\w*\b/i;

/**
 * Pick the script for a message. Pure, so the branch table is a unit test
 * rather than something you discover by typing.
 */
export function scriptFor(prompt: string): readonly Beat[] {
  const opened: Beat = {
    t: "note",
    label: "Working",
    detail: "Just now",
    tone: "live",
  };

  if (BROKEN.test(prompt)) {
    return [
      opened,
      {
        t: "say",
        chunks: deltas(
          "Let me look. I can see what happened and when — whether I can see why depends on what I have been given access to.",
        ),
      },
      {
        t: "fault",
        code: "No access",
        message:
          "I have not been given access to that system, so I can only see that it stopped, not why.",
        recovery:
          "Someone with admin rights can grant it from Organisation → Connected tools. I will not keep retrying in the meantime.",
      },
      { t: "note", label: "Stopped", detail: "Nothing retried", tone: "fail" },
    ];
  }

  if (NEEDS_APPROVAL.test(prompt)) {
    return [
      opened,
      {
        t: "say",
        chunks: deltas(
          "I can do that, but it leaves the building — so you decide, not me. Here is exactly what I would send.",
        ),
      },
      {
        t: "hold",
        tool: "Helpdesk",
        intent: "Send a message on the company's behalf",
        args: [
          ["to", "the customer on this thread"],
          ["about", prompt.trim().slice(0, 64)],
          ["signed", "your company"],
        ],
        risk: "Once it is sent it cannot be recalled. Nothing happens until you answer.",
        onGranted: deltas(
          "Sent. The copy is on the thread, and I have logged who approved it.",
        ),
        onDenied: deltas(
          "Nothing was sent, and I will not ask again in this conversation.",
        ),
        result: "1 message sent · logged against your name",
      },
      { t: "note", label: "Finished", detail: "One approval asked" },
    ];
  }

  if (MONEY.test(prompt)) {
    return [
      opened,
      {
        t: "say",
        chunks: deltas(
          "I would rather read it than estimate it. One moment.",
        ),
      },
      {
        t: "tool",
        tool: "Ledger",
        intent: "Read the relevant entries",
        args: [
          ["about", prompt.trim().slice(0, 64)],
          ["period", "this quarter"],
          ["access", "read only"],
        ],
        result: "412 entries read · nothing changed",
      },
      {
        t: "say",
        chunks: deltas(
          "Read, not estimated. Two lines are drifting and one of them still has time to be fixed — say the word and I will write it up properly.",
        ),
      },
      {
        t: "note",
        label: "Finished",
        detail: "One tool used · nothing left the building",
      },
    ];
  }

  return [
    opened,
    {
      t: "say",
      chunks: deltas(
        "Here is where I would start. If part of this belongs to someone else here — money to Finance, customers to Support, anything touching the data to the Database Administrator — I will say so rather than guess at it.",
      ),
    },
    { t: "note", label: "Finished", detail: "No tools used" },
  ];
}
