/**
 * company.ts — the company a signed-in person is looking at.
 *
 * DEMONSTRATION DATA. The real organisation record already exists (it is what
 * the first-run setup step writes); this module stands in for it until the
 * workspace reads that record directly. Every field here has a counterpart
 * there, which is why the shape is deliberately small.
 */
export interface Company {
  readonly name: string;
  readonly domain: string;
  /** The letters shown in the company tile when there is no logo. */
  readonly initials: string;
  /** Which plan this company is on. Drives what the workspace offers to add. */
  readonly plan: string;
  readonly people: number;
}

export const COMPANY: Company = {
  name: "Witsaba",
  domain: "witsaba.com",
  initials: "W",
  plan: "Team",
  people: 5,
};
