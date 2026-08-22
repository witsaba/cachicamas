/**
 * staff.ts — the company's staff, its people, its teams and its plans.
 *
 * DEMONSTRATION DATA. Every conversation, tenure, figure and colleague in this
 * file is an example authored for the mockup, and the workspace says so on
 * every screen (see `Workspace`'s demonstration strip). What is real is the
 * *shape*: an agent has a department, a job, a set of things it can do, a set
 * of tools it is allowed to use, a status, and a date it joined — and that is
 * what a company will actually see when the first specialists start work.
 *
 * Nothing in this module mentions how any of it is built. That is deliberate:
 * the people using this product hire colleagues, they do not run a runtime.
 */

/** Which department a member belongs to. Drives one hue, never a rank. */
export type Department =
  | "assistant"
  | "finance"
  | "support"
  | "integrations"
  | "data"
  | "engineering";

/**
 * Where an agent stands with *this* company. Each value ships with a literal
 * word and a sentence; the colour is never the only carrier.
 */
export type AgentStatus =
  /** On staff and answering now. */
  | "working"
  /** Hired, still being set up. */
  | "training"
  /** Included in a plan the company can move to; not on staff yet. */
  | "available";

export interface Skill {
  readonly name: string;
  readonly detail: string;
}

export interface Tool {
  readonly name: string;
  /** What the agent is allowed to do with it, in plain words. */
  readonly purpose: string;
}

export interface Agent {
  readonly slug: string;
  /** Two letters, shown inside the avatar tile. Never the only identifier. */
  readonly initials: string;
  readonly name: string;
  readonly department: Department;
  readonly departmentName: string;
  /** One line, in the agent's own voice-free plain register. */
  readonly tagline: string;
  /** Two or three sentences a manager would read before hiring. */
  readonly summary: string;
  readonly status: AgentStatus;
  /** The literal word rendered beside the status dot. */
  readonly statusWord: string;
  /** The honest sentence under the word. */
  readonly statusDetail: string;
  /** ISO date the agent joined this company, or null if it has not. */
  readonly joined: string | null;
  /** "5 months", precomputed so the mock never drifts against a clock. */
  readonly tenure: string | null;
  readonly skills: readonly Skill[];
  readonly tools: readonly Tool[];
  /** What this agent will always hand to someone else, and to whom. */
  readonly handsOff: { readonly what: string; readonly to: string } | null;
  /** Example workload figure. Rendered only inside the demonstration strip's scope. */
  readonly conversationsThisWeek: number | null;
}

export interface Person {
  readonly id: string;
  readonly name: string;
  readonly initials: string;
  /** The job title this person holds, or null when they hold no officer role. */
  readonly title: string | null;
}

export interface OrgRole {
  readonly key: string;
  readonly title: string;
  readonly responsibility: string;
  /** `Person.id`, or null when the seat is open. */
  readonly holder: string | null;
}

export interface Team {
  readonly slug: string;
  readonly name: string;
  readonly purpose: string;
  readonly agentSlugs: readonly string[];
  readonly personIds: readonly string[];
  /**
   * A paired duo: two agents assigned to work together on this team, which is
   * a Workforce-plan arrangement. `null` on every other team.
   */
  readonly pair: readonly [string, string] | null;
}

/* ---------------------------------------------------------------------------
 * The staff
 * ------------------------------------------------------------------------ */

export const AGENTS: readonly Agent[] = [
  {
    slug: "assistant",
    initials: "AS",
    name: "Assistant",
    department: "assistant",
    departmentName: "Front desk",
    tagline: "The colleague everyone talks to first.",
    summary:
      "Handles the work that has no other home: answering questions, drafting and rewriting, reading a long thread and telling you what it says. When a job belongs to a specialist, the Assistant says so and introduces you.",
    status: "working",
    statusWord: "Working",
    statusDetail: "On staff and answering now.",
    joined: "2026-01-12",
    tenure: "7 months",
    skills: [
      {
        name: "Answer questions in plain language",
        detail: "About the company, its documents, and how things are done here.",
      },
      {
        name: "Draft and rewrite",
        detail: "Emails, notes, announcements, job posts. In your own register.",
      },
      {
        name: "Summarise a long thread",
        detail: "What was decided, what is still open, who owes what.",
      },
      {
        name: "Find the right colleague",
        detail: "Points you at the specialist whose job it actually is.",
      },
    ],
    tools: [
      { name: "Company handbook", purpose: "Reads it. Never edits it." },
      { name: "Shared documents", purpose: "Reads and drafts, with your approval to save." },
      { name: "Staff directory", purpose: "Reads who does what." },
    ],
    handsOff: {
      what: "Anything about money, customers, data or code",
      to: "the specialist who owns it",
    },
    conversationsThisWeek: 64,
  },
  {
    slug: "finance",
    initials: "FN",
    name: "Finance",
    department: "finance",
    departmentName: "Operations",
    tagline: "Keeps the money legible.",
    summary:
      "Reconciles what came in against what was invoiced, reviews expenses before they become a surprise, and prepares the month-end close so it takes an afternoon instead of a week. Asks before it sends anything to a supplier or a customer.",
    status: "working",
    statusWord: "Working",
    statusDetail: "On staff and answering now.",
    joined: "2026-03-02",
    tenure: "5 months",
    skills: [
      { name: "Invoice reconciliation", detail: "Matches payments to invoices and flags what does not line up." },
      { name: "Expense review", detail: "Reads receipts, applies your policy, questions the outliers." },
      { name: "Budget against actuals", detail: "Tells you which lines are drifting while there is still time." },
      { name: "Month-end close", detail: "Assembles the pack and lists what is missing." },
      { name: "Chasing what is owed", detail: "Drafts the reminder. You decide whether it goes." },
    ],
    tools: [
      { name: "Ledger", purpose: "Reads entries. Proposes corrections for approval." },
      { name: "Billing", purpose: "Reads invoices. Sends nothing without a decision." },
      { name: "Bank feed", purpose: "Reads transactions only." },
    ],
    handsOff: {
      what: "Anything that needs a report built from the raw numbers",
      to: "Database Administrator",
    },
    conversationsThisWeek: 31,
  },
  {
    slug: "support",
    initials: "SP",
    name: "Support",
    department: "support",
    departmentName: "Customer",
    tagline: "Has read the ticket before you open it.",
    summary:
      "Triages what arrives overnight, drafts a first reply in your company's voice, and writes the escalation summary so whoever picks it up is not starting from zero. Nothing reaches a customer until a person says yes.",
    status: "working",
    statusWord: "Working",
    statusDetail: "On staff and answering now.",
    joined: "2026-04-14",
    tenure: "4 months",
    skills: [
      { name: "First reply drafting", detail: "In your voice, with the account's history already read." },
      { name: "Ticket triage", detail: "Sorts by what is actually urgent, not by what shouted loudest." },
      { name: "Escalation summaries", detail: "What happened, what was tried, what the customer wants." },
      { name: "Policy checks", detail: "Whether a refund or an exception fits the rules you wrote." },
      { name: "Knowledge base upkeep", detail: "Notices the answer that keeps being retyped." },
    ],
    tools: [
      { name: "Helpdesk", purpose: "Reads and drafts. Sends only on approval." },
      { name: "Knowledge base", purpose: "Reads, and proposes edits." },
      { name: "Customer records", purpose: "Reads only." },
    ],
    handsOff: { what: "Refunds and credit notes", to: "Finance" },
    conversationsThisWeek: 88,
  },
  {
    slug: "integrations",
    initials: "IN",
    name: "Integrations",
    department: "integrations",
    departmentName: "Operations",
    tagline: "Makes the tools you already pay for talk to each other.",
    summary:
      "Connects a new system, maps the fields between it and everything else, and watches the joins afterwards — because the expensive failure is not the one that errors, it is the one that quietly stops syncing.",
    status: "training",
    statusWord: "In training",
    statusDetail: "Hired. Learning your systems this week.",
    joined: "2026-08-17",
    tenure: "5 days",
    skills: [
      { name: "Connect a new tool", detail: "Walks the setup and tells you what it will be able to see." },
      { name: "Map fields between systems", detail: "Proposes the mapping and shows you the edge cases first." },
      { name: "Watch for broken joins", detail: "Notices a sync that stopped, not just one that failed." },
      { name: "Backfill what went missing", detail: "Fills the gap after an outage, with a dry run first." },
    ],
    tools: [
      { name: "Connectors", purpose: "Configures, with approval for anything that writes." },
      { name: "Sync monitor", purpose: "Reads and alerts." },
      { name: "Field mapper", purpose: "Proposes mappings for approval." },
    ],
    handsOff: { what: "Anything that changes a table", to: "Database Administrator" },
    conversationsThisWeek: null,
  },
  {
    slug: "database-administrator",
    initials: "DB",
    name: "Database Administrator",
    department: "data",
    departmentName: "Platform",
    tagline: "Owns the tables nobody else may touch.",
    summary:
      "Answers questions with real numbers instead of estimates, reviews every change to how your data is shaped, and keeps the backups honest by restoring them. The only colleague allowed to change your data's structure.",
    status: "available",
    statusWord: "Available",
    statusDetail: "Included on the Company plan. Not on your staff yet.",
    joined: null,
    tenure: null,
    skills: [
      { name: "Answer with real numbers", detail: "Turns a question into a query, and shows you the query." },
      { name: "Review structural changes", detail: "Nothing reshapes your data without a written plan." },
      { name: "Keep things fast", detail: "Finds the slow query before your customers do." },
      { name: "Prove the backups", detail: "Restores them on a schedule, because an untested backup is a rumour." },
    ],
    tools: [
      { name: "Company database", purpose: "Reads freely. Writes only with an approved plan." },
      { name: "Query console", purpose: "Runs read-only queries." },
      { name: "Backups", purpose: "Takes and restores, on approval." },
    ],
    handsOff: { what: "Application changes", to: "Coding" },
    conversationsThisWeek: null,
  },
  {
    slug: "coding",
    initials: "CO",
    name: "Coding",
    department: "engineering",
    departmentName: "Platform",
    tagline: "Ships the small changes that never reach the roadmap.",
    summary:
      "Picks up the fixes that are too small to schedule and too annoying to leave: the copy change, the broken link, the dependency that needs bumping, the test that was skipped. Opens the change for review; a person merges it.",
    status: "available",
    statusWord: "Available",
    statusDetail: "Included on the Company plan. Not on your staff yet.",
    joined: null,
    tenure: null,
    skills: [
      { name: "Fix small things", detail: "The bug that has an owner but never a free afternoon." },
      { name: "Review a change", detail: "Reads a colleague's work and says what it would break." },
      { name: "Keep dependencies current", detail: "One at a time, with the tests run each time." },
      { name: "Write the skipped tests", detail: "Starting with the code that changed most this quarter." },
    ],
    tools: [
      { name: "Code repository", purpose: "Reads, and opens changes for review. Never merges." },
      { name: "Test runner", purpose: "Runs and reports." },
      { name: "Review queue", purpose: "Comments only." },
    ],
    handsOff: { what: "Anything touching the shape of your data", to: "Database Administrator" },
    conversationsThisWeek: null,
  },
];

/* ---------------------------------------------------------------------------
 * The people
 * ------------------------------------------------------------------------ */

/**
 * Example colleagues. These names are invented for the mockup; the signed-in
 * person is added on top of this list at render time so a viewer always sees
 * themselves in their own company.
 */
export const PEOPLE: readonly Person[] = [
  { id: "ana", name: "Ana Rivas", initials: "AR", title: "Head of Finance" },
  { id: "marco", name: "Marco Silva", initials: "MS", title: "Head of Customer" },
  { id: "priya", name: "Priya Nair", initials: "PN", title: "Chief Technology Officer" },
  { id: "tom", name: "Tom Becker", initials: "TB", title: null },
];

/** The officer seats a company fills. Order is the order they are shown in. */
export const ORG_ROLES: readonly OrgRole[] = [
  {
    key: "founder",
    title: "Founder",
    responsibility: "Sets what the company is for.",
    holder: "you",
  },
  {
    key: "ceo",
    title: "Chief Executive",
    responsibility: "Answers for the whole of it.",
    holder: "you",
  },
  {
    key: "cto",
    title: "Chief Technology Officer",
    responsibility: "Owns how the product is built and kept running.",
    holder: "priya",
  },
  {
    key: "coo",
    title: "Chief Operating Officer",
    responsibility: "Owns how the work actually gets done.",
    holder: null,
  },
  {
    key: "finance",
    title: "Head of Finance",
    responsibility: "Owns the money and what it is spent on.",
    holder: "ana",
  },
  {
    key: "customer",
    title: "Head of Customer",
    responsibility: "Owns what customers experience after they buy.",
    holder: "marco",
  },
];

/* ---------------------------------------------------------------------------
 * The teams
 * ------------------------------------------------------------------------ */

export const TEAMS: readonly Team[] = [
  {
    slug: "revenue",
    name: "Revenue",
    purpose: "Get paid, and keep the customers who pay.",
    agentSlugs: ["finance", "support"],
    personIds: ["ana", "marco"],
    pair: ["finance", "support"],
  },
  {
    slug: "platform",
    name: "Platform",
    purpose: "Keep the data honest and the tools connected.",
    agentSlugs: ["integrations", "database-administrator", "coding"],
    personIds: ["priya"],
    pair: null,
  },
  {
    slug: "front-desk",
    name: "Front desk",
    purpose: "Everything that has no other home yet.",
    agentSlugs: ["assistant"],
    personIds: ["tom"],
    pair: null,
  },
];

/* ---------------------------------------------------------------------------
 * Lookups
 * ------------------------------------------------------------------------ */

export const agentBySlug = (slug: string): Agent | undefined =>
  AGENTS.find((a) => a.slug === slug);

export const personById = (id: string): Person | undefined =>
  PEOPLE.find((p) => p.id === id);

export const teamBySlug = (slug: string): Team | undefined =>
  TEAMS.find((t) => t.slug === slug);

/** The agents a person can start a conversation with today. */
export const workingAgents = (): readonly Agent[] =>
  AGENTS.filter((a) => a.status === "working");

/** Agents a company could add by moving up a plan. */
export const availableAgents = (): readonly Agent[] =>
  AGENTS.filter((a) => a.status === "available");

export const agentHref = (slug: string): string => `/agents/${slug}/`;

/**
 * The company's people, with the signed-in person folded in exactly once.
 *
 * This is one function rather than three because the first attempt at it was
 * three, and they disagreed: the shell said five people, the Organisation
 * panel said four, and a seat whose holder had been de-duplicated away
 * rendered "Nobody yet" beside an avatar that still showed a face. Whoever
 * counts the people and whoever fills the seats have to be the same code.
 *
 * When the signed-in person's name matches one of the examples they are the
 * same person, so the example is replaced rather than listed beside them — and
 * `holdersFor` below rewrites that person's officer seats to point at them.
 */
export interface Roster {
  readonly people: readonly Person[];
  /** The example person the signed-in one stands in for, if any. */
  readonly twinId: string | null;
}

export function rosterFor(name: string, email = ""): Roster {
  const key = (value: string) => value.trim().toLowerCase();
  const you: Person = {
    id: "you",
    name: name.trim() || email.trim() || "You",
    initials: "",
    title: null,
  };
  const twin = PEOPLE.find((p) => key(p.name) === key(you.name)) ?? null;
  return {
    people: [you, ...PEOPLE.filter((p) => p !== twin)],
    twinId: twin?.id ?? null,
  };
}

/**
 * Who holds each seat, once the signed-in person has absorbed their twin.
 * Keyed by `OrgRole.key`; an empty string means the seat is open.
 */
export function holdersFor(twinId: string | null): Record<string, string> {
  return Object.fromEntries(
    ORG_ROLES.map((role) => [
      role.key,
      role.holder === null
        ? ""
        : role.holder === twinId
          ? "you"
          : role.holder,
    ]),
  );
}

export const teamsForAgent = (slug: string): readonly Team[] =>
  TEAMS.filter((t) => t.agentSlugs.includes(slug));
