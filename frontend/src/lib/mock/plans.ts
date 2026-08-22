/**
 * plans.ts — the subscription mockup.
 *
 * PREVIEW PRICING. These figures are placeholders authored for the mockup, not
 * a commercial offer, and the pricing section says so in one line beneath it.
 * Replace every `price` in this file before this page is published anywhere a
 * customer could act on it.
 *
 * The shape is the part worth reviewing: four levels, one axis (how many
 * specialists you have on staff), and one thing only the top level gets — the
 * ability to build a specialist of your own and to put two of them on a team
 * as a permanent pair.
 */

export type Billing = "monthly" | "annual";

export interface Plan {
  readonly slug: string;
  readonly name: string;
  /** One line: who this level is for. */
  readonly forWhom: string;
  /** Whole currency units per person per month, at each billing period. */
  readonly price: { readonly monthly: number; readonly annual: number };
  /** How many specialists, as a phrase — the axis every level moves along. */
  readonly staffing: string;
  readonly includes: readonly string[];
  /** The one line that distinguishes this level from the one below it. */
  readonly step: string | null;
  readonly cta: string;
  /** Exactly one level may be marked. */
  readonly recommended: boolean;
}

export const PLANS: readonly Plan[] = [
  {
    slug: "starter",
    name: "Starter",
    forWhom: "A small company trying this with one colleague.",
    price: { monthly: 0, annual: 0 },
    staffing: "The Assistant",
    includes: [
      "The Assistant, on staff from day one",
      "Unlimited conversations",
      "Up to 5 people in the company",
      "Your staff directory and profiles",
    ],
    step: null,
    cta: "Start free",
    recommended: false,
  },
  {
    slug: "team",
    name: "Team",
    forWhom: "A team that already knows which two jobs it is short on.",
    price: { monthly: 24, annual: 19 },
    staffing: "The Assistant, plus any 2 specialists",
    includes: [
      "Everything in Starter",
      "Any 2 specialists of your choosing",
      "Up to 20 people in the company",
      "Teams, so a specialist can belong somewhere",
      "Approval before an agent acts on anything",
    ],
    step: "Pick the two specialists you are missing.",
    cta: "Choose Team",
    recommended: false,
  },
  {
    slug: "company",
    name: "Company",
    forWhom: "A company running its whole operation this way.",
    price: { monthly: 48, annual: 39 },
    staffing: "The Assistant, plus all 5 specialists",
    includes: [
      "Everything in Team",
      "Finance, Support, Integrations, Database Administrator and Coding",
      "Unlimited people",
      "Your organisation chart and officer roles",
      "Every conversation kept and searchable",
    ],
    step: "All five specialists, with no seat to choose between.",
    cta: "Choose Company",
    recommended: true,
  },
  {
    slug: "workforce",
    name: "Workforce",
    forWhom: "A company that needs colleagues nobody else sells.",
    price: { monthly: 96, annual: 79 },
    staffing: "More than ten agents on staff",
    includes: [
      "Everything in Company",
      "More than ten agents on staff at once",
      "3 open desks — build a specialist of your own",
      "A paired duo: two agents assigned to one team permanently",
      "A named person to set it up with you",
    ],
    step: "Build the specialist your company needs and nobody offers.",
    cta: "Talk to us",
    recommended: false,
  },
];

/** What annual billing saves, expressed the way a buyer reads it. */
export const ANNUAL_SAVING = "2 months free";

export const priceFor = (plan: Plan, billing: Billing): number =>
  billing === "annual" ? plan.price.annual : plan.price.monthly;

/**
 * The comparison rows. Deliberately short: a table with thirty rows is a table
 * nobody reads, and the axis here is genuinely one-dimensional.
 */
export interface ComparisonRow {
  readonly label: string;
  /** Keyed by plan slug. `true` renders a check, a string renders the string. */
  readonly values: Readonly<Record<string, boolean | string>>;
}

export const COMPARISON: readonly ComparisonRow[] = [
  {
    label: "Conversations",
    values: {
      starter: "Unlimited",
      team: "Unlimited",
      company: "Unlimited",
      workforce: "Unlimited",
    },
  },
  {
    label: "Specialists on staff",
    values: { starter: "0", team: "2", company: "5", workforce: "10+" },
  },
  {
    label: "People in the company",
    values: {
      starter: "5",
      team: "20",
      company: "Unlimited",
      workforce: "Unlimited",
    },
  },
  {
    label: "Teams",
    values: { starter: false, team: true, company: true, workforce: true },
  },
  {
    label: "Organisation chart",
    values: { starter: false, team: false, company: true, workforce: true },
  },
  {
    label: "Build your own specialist",
    values: { starter: false, team: false, company: false, workforce: "3 desks" },
  },
  {
    label: "A paired duo on one team",
    values: { starter: false, team: false, company: false, workforce: true },
  },
];
