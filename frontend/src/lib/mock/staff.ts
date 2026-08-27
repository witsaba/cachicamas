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
    // statusWord and statusDetail for the Assistant are overridden
    // at render time by AgentDirectory from the live
    // /api/chat/assistant/config response (REQ-FADR-001/002).
    // The values here are a sensible fallback used only when the GET
    // fails (offline / anon / server) — see AgentDirectory for the
    // override path. Until the next archetype ships, the assistant
    // is the only colleague on the roster, so its statusWord is the
    // single source the workspace shows.
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
      // Until a specialist archetype ships, the hand-off target is
      // described in plain language — see AgentProfile's hands-off
      // rendering for how the missing colleague is phrased.
      to: "the specialist who owns it",
    },
    conversationsThisWeek: 64,
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
  // The previous mock carried three teams (revenue, platform, front-desk)
  // each pointing at the fake specialists. Until a second archetype ships
  // there is no real team to seed; TEAMS is kept exported as the empty
  // list so the route surface and downstream lookups (`teamsForAgent`,
  // `teamBySlug`) stay callable without crashing. The "no team references
  // a slug that does not exist in AGENTS" guard in staff.spec.ts keeps
  // future entries honest.
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
