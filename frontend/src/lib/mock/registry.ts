/**
 * registry.ts — the archetype register, and the runtime beneath it.
 *
 * DEMONSTRATION DATA (PRODUCT.md § Evidence on Hand). Nothing here is read
 * from a running system, because no Layer 3 archetype exists on disk yet.
 * Two rules govern what may be written in this file:
 *
 *   1. Every *structural* fact is real and citable — the layer counts, the
 *      milestone counts, the plan documents, which archetype owns which
 *      business system. These come from the repository and are checked by
 *      `registry.spec.ts` against the numbers quoted in the docs.
 *   2. Every *activity* number (turns today, last-seen) is invented, is
 *      marked `demo: true`, and the interface labels it as demonstration
 *      data wherever it is shown. PRODUCT.md names overstating readiness as
 *      the one lie this interface cannot afford.
 *
 * When the chat archetype ships (doc 0005, CH-00..CH-11), this module is
 * replaced by a real read — not edited to look more impressive.
 */

/**
 * What an archetype is doing, as a machine value. Every one of these is
 * rendered with its own literal word beside its lamp — the lamp colour is
 * never the only carrier (PRODUCT.md § Accessibility).
 */
export type ArchetypeState =
  /** Shipped and answering. Nothing is in this state yet. */
  | "on-duty"
  /** Has a milestone plan and work has started or is scheduled next. */
  | "in-build"
  /** Named and planned in a decision record; no milestone document yet. */
  | "planned"
  /** Named in ADR 0009 § D1 as a job a company needs. Nothing else. */
  | "unplanned";

export interface Archetype {
  /** Function key that opens it. The dock legend and the real keydown agree. */
  readonly fkey: string;
  /** The code you type at the command line. Uppercase, no spaces. */
  readonly code: string;
  /** Route segment. */
  readonly slug: string;
  /** Display name. */
  readonly name: string;
  /**
   * The name the dock uses. "Database Administrator" across a dock cell pushes
   * every other destination off a laptop screen, so long names get a short
   * form and the full one lives on the archetype's own screen.
   */
  readonly dockName: string;
  /** What this specialist is for, in the product's own plain language. */
  readonly role: string;
  readonly state: ArchetypeState;
  /** The literal word rendered next to the lamp. Never derived from colour. */
  readonly stateWord: string;
  /** The business system it owns, per ADR 0009 § D4. `null` when it owns none. */
  readonly system: string | null;
  /** Milestone plan, when one exists. */
  readonly plan: { readonly label: string; readonly done: number; readonly total: number } | null;
  /** Where the decision that created it lives. */
  readonly authority: string;
  /** Invented activity figure. Always rendered under a DEMO marker. */
  readonly demoTurnsToday: number | null;
  /** What this specialist would be responsible for, once it exists. */
  readonly wouldDo: readonly string[];
  /** The honest answer to "why can I not use this yet". */
  readonly blockedBy: string;
}

/**
 * The register, in dock order. `chat` leads because it is the one being
 * built (doc 0005 § Outcome first), not because it is the most valuable.
 */
export const ARCHETYPES: readonly Archetype[] = [
  {
    fkey: "F1",
    code: "CHAT",
    slug: "chat",
    name: "Chat",
    dockName: "Chat",
    role: "Talk to a model, one turn at a time",
    state: "in-build",
    stateWord: "IN BUILD",
    system: null,
    plan: { label: "doc 0005", done: 0, total: 12 },
    authority: "ADR 0009 § D2",
    demoTurnsToday: 12,
    wouldDo: [
      "Hold a conversation with a model, one turn at a time, and stream the answer back as it arrives",
      "Suspend itself mid-run when a tool needs a person's permission, and show exactly what it is about to do",
      "Remember its conversations in tables it owns, so a reload brings the thread back",
      "Hand work it does not own to the specialist that does",
    ],
    blockedBy: "Its milestone plan exists and is scheduled first, because it is the cheapest complete proof that the architecture works. Nothing is on disk yet; CH-00 is the first milestone.",
  },
  {
    fkey: "F2",
    code: "CODING",
    slug: "coding",
    name: "Coding",
    dockName: "Coding",
    role: "Read, write and ship the company's software",
    state: "planned",
    stateWord: "PLANNED",
    system: "the source repository",
    plan: { label: "doc 0004", done: 0, total: 25 },
    authority: "ADR 0004 · ADR 0006",
    demoTurnsToday: null,
    wouldDo: [
      "Read, write and ship this company's software against a real repository",
      "Carry its own skills, prompts and slash commands, resolved from two stores",
      "Persist a session as an append-only record it can resume from",
      "Never write back to Postgres — that boundary is this archetype's own answer, not a layer rule",
    ],
    blockedBy: "Planned in full at doc 0004, 25 milestones, and deliberately scheduled behind the chat archetype so the first end-to-end proof is the cheap one.",
  },
  {
    fkey: "F3",
    code: "DBA",
    slug: "dba",
    name: "Database Administrator",
    dockName: "Database",
    role: "Owns every table. Every other archetype asks it for schema, queries and capacity",
    state: "planned",
    stateWord: "PLANNED",
    system: "database_administrator",
    plan: null,
    authority: "ADR 0009 § D5",
    demoTurnsToday: null,
    wouldDo: [
      "Front the existing database_administrator service through an MCP client",
      "Broker every other archetype's schema, query, migration and capacity work",
      "Own the tables it administers, and refuse writes into systems it does not own",
      "Be the first business system integrated under the MCP pattern",
    ],
    blockedBy: "Named and its destination fixed by ADR 0009 § D5, which says plainly that planned means planned: it gets a milestone document when its planning starts, and that has not happened.",
  },
  {
    fkey: "F4",
    code: "FINANCE",
    slug: "finance",
    name: "Finance",
    dockName: "Finance",
    role: "Books, invoices, payroll and what they cost",
    state: "unplanned",
    stateWord: "UNPLANNED",
    system: null,
    plan: null,
    authority: "ADR 0009 § D1",
    demoTurnsToday: null,
    wouldDo: [
      "Own the books, the invoices, the payroll and what all of it costs",
      "Run its own MCP server, with exactly one owning archetype — itself",
      "Own the finance tables, and ask the Database Administrator for any schema work",
    ],
    blockedBy: "Named in ADR 0009 § D1 as one of the jobs a company needs. There is no decision record beyond that, and inventing one here would be the interface lying about its own maturity.",
  },
  {
    fkey: "F5",
    code: "MARKETING",
    slug: "marketing",
    name: "Marketing",
    dockName: "Marketing",
    role: "Campaigns, copy and the company's public voice",
    state: "unplanned",
    stateWord: "UNPLANNED",
    system: null,
    plan: null,
    authority: "ADR 0009 § D1",
    demoTurnsToday: null,
    wouldDo: [
      "Own campaigns, copy and the company's public voice",
      "Run its own MCP server against whatever marketing systems the company uses",
    ],
    blockedBy: "Named in ADR 0009 § D1 as one of the jobs a company needs. There is no decision record beyond that.",
  },
  {
    fkey: "F6",
    code: "TICKETS",
    slug: "tickets",
    name: "Ticketing",
    dockName: "Tickets",
    role: "What is broken, who asked, and what happened next",
    state: "unplanned",
    stateWord: "UNPLANNED",
    system: "a ticket system, over its own MCP server",
    plan: null,
    authority: "ADR 0009 § D4",
    demoTurnsToday: null,
    wouldDo: [
      "Own what is broken, who asked, and what happened next",
      "Run its own MCP server — the worked example ADR 0009 § D4 uses to explain the pattern",
    ],
    blockedBy: "Named in ADR 0009 § D4 as the worked example of a business system with one owning archetype. The pattern is decided; this occupant is not scheduled.",
  },
] as const;

/** The stack the register stands on. Every figure here is real. */
export interface RuntimeLayer {
  readonly code: string;
  readonly name: string;
  readonly detail: string;
  readonly done: number;
  readonly total: number;
  readonly stateWord: string;
  readonly state: "complete" | "frozen" | "open";
}

export const RUNTIME: readonly RuntimeLayer[] = [
  {
    code: "L1",
    name: "Model adapter",
    detail: "One wire, any vendor",
    done: 42,
    total: 42,
    stateWord: "COMPLETE",
    state: "complete",
  },
  {
    code: "L2",
    name: "Agent runtime",
    detail: "Pure mechanism. It cannot tell which archetype is standing on it",
    done: 24,
    total: 24,
    stateWord: "FROZEN",
    state: "frozen",
  },
  {
    code: "L3",
    name: "Archetype layer",
    detail: "Where a specialist's policy, tools, memory and screen live",
    done: 0,
    total: ARCHETYPES.length,
    stateWord: "OPEN",
    state: "open",
  },
] as const;

/** Look an archetype up by its command-line code. Case-insensitive. */
export function archetypeByCode(code: string): Archetype | undefined {
  const needle = code.trim().toUpperCase();
  return ARCHETYPES.find((a) => a.code === needle);
}

/** Look an archetype up by its route segment. */
export function archetypeBySlug(slug: string): Archetype | undefined {
  const needle = slug.trim().toLowerCase();
  return ARCHETYPES.find((a) => a.slug === needle);
}

/** Where an archetype's own screen lives. `chat` has one; the rest have a stub. */
export function archetypeHref(a: Archetype): string {
  return a.slug === "chat" ? "/chat/" : `/archetypes/${a.slug}/`;
}

/**
 * Resolve what the user typed at the command line to somewhere to go.
 *
 * The command line accepts an archetype code, a system code, or a bare
 * route. Anything else is refused with a message that names what it
 * accepts — never a silent no-op.
 */
export type CommandResolution =
  | { readonly ok: true; readonly href: string; readonly label: string }
  | { readonly ok: false; readonly message: string };

const SYSTEM_COMMANDS: ReadonlyArray<{ code: string; href: string; label: string }> = [
  { code: "DESK", href: "/home/", label: "Desk" },
  { code: "HOME", href: "/home/", label: "Desk" },
  { code: "SYSTEM", href: "/settings/", label: "System" },
  { code: "SETTINGS", href: "/settings/", label: "System" },
  { code: "PROFILE", href: "/profile/", label: "Profile" },
  { code: "SIGNOUT", href: "/auth/signout/", label: "Sign out" },
];

export function resolveCommand(input: string): CommandResolution {
  const raw = input.trim();
  if (raw.length === 0) {
    return { ok: false, message: "Type an archetype code — try CHAT — or SYSTEM." };
  }
  const upper = raw.toUpperCase();

  const archetype = archetypeByCode(upper);
  if (archetype) {
    return { ok: true, href: archetypeHref(archetype), label: archetype.name };
  }

  const system = SYSTEM_COMMANDS.find((c) => c.code === upper);
  if (system) {
    return { ok: true, href: system.href, label: system.label };
  }

  if (raw.startsWith("/")) {
    return { ok: true, href: raw, label: raw };
  }

  return {
    ok: false,
    message: `No archetype or system called "${raw}". Registered codes: ${ARCHETYPES.map(
      (a) => a.code,
    ).join(", ")}.`,
  };
}

/**
 * Prefix search over everything the command line can reach, for the
 * suggestion list under the input. Ordered: archetypes first, then system.
 */
export interface CommandSuggestion {
  readonly code: string;
  readonly label: string;
  readonly hint: string;
  readonly href: string;
}

export function suggestCommands(input: string): readonly CommandSuggestion[] {
  const needle = input.trim().toUpperCase();
  const all: CommandSuggestion[] = [
    ...ARCHETYPES.map((a) => ({
      code: a.code,
      label: a.name,
      hint: a.stateWord,
      href: archetypeHref(a),
    })),
    { code: "DESK", label: "Desk", hint: "THE REGISTER", href: "/home/" },
    { code: "SYSTEM", label: "System", hint: "SETTINGS", href: "/settings/" },
    { code: "PROFILE", label: "Profile", hint: "YOUR ACCOUNT", href: "/profile/" },
    { code: "SIGNOUT", label: "Sign out", hint: "END SESSION", href: "/auth/signout/" },
  ];
  if (needle.length === 0) return all;
  return all.filter((c) => c.code.startsWith(needle) || c.label.toUpperCase().startsWith(needle));
}
