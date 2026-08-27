/**
 * The staff data's contract.
 *
 * This is demonstration data, so it cannot be checked against a running
 * system. What it CAN be checked against is the promises the interface makes
 * about it — and those are the promises that would be embarrassing to break:
 * that every colleague has a status word, that colour never identifies anyone
 * on its own, that no two colleagues claim the same job, and that not one line
 * of it mentions how the product is built.
 */
import { describe, it, expect } from "vitest";
import {
  AGENTS,
  ORG_ROLES,
  PEOPLE,
  TEAMS,
  agentBySlug,
  availableAgents,
  personById,
  teamsForAgent,
  workingAgents,
} from "./staff";

describe("the staff", () => {
  it("gives every colleague a slug, a name and two initials", () => {
    const slugs = new Set<string>();
    for (const a of AGENTS) {
      expect(a.slug, a.name).toMatch(/^[a-z][a-z-]*$/);
      expect(slugs.has(a.slug), `duplicate slug ${a.slug}`).toBe(false);
      slugs.add(a.slug);
      expect(a.name.length, a.slug).toBeGreaterThan(2);
      // Two letters, because three in a 32px tile stops being legible.
      expect(a.initials, a.slug).toMatch(/^[A-Z]{2}$/);
    }
  });

  it("gives every status a word and a sentence, never a colour alone", () => {
    for (const a of AGENTS) {
      expect(a.statusWord.length, a.slug).toBeGreaterThan(2);
      expect(a.statusDetail.length, a.slug).toBeGreaterThan(10);
    }
  });

  it("only claims a tenure for a colleague who actually joined", () => {
    // "On staff 5 months" beside someone the company has not hired is the
    // exact kind of small lie that costs a product its credibility.
    for (const a of AGENTS) {
      if (a.status === "available") {
        expect(a.joined, a.slug).toBeNull();
        expect(a.tenure, a.slug).toBeNull();
      } else {
        expect(a.joined, a.slug).toMatch(/^\d{4}-\d{2}-\d{2}$/);
        expect(a.tenure, a.slug).toBeTruthy();
      }
    }
  });

  it("only shows a workload for someone who is actually working", () => {
    for (const a of AGENTS) {
      if (a.status !== "working") {
        expect(a.conversationsThisWeek, a.slug).toBeNull();
      }
    }
  });

  it("gives each department its own hue, and never reuses one", () => {
    const departments = AGENTS.map((a) => a.department);
    expect(new Set(departments).size).toBe(departments.length);
  });

  it("keeps the roster to the one real archetype for now", () => {
    // Only the assistant is a real archetype; the other five used to be
    // mock placeholders shown on the FE. Until the next archetype ships,
    // the roster is exactly one colleague.
    expect(AGENTS.map((a) => a.slug)).toEqual(["assistant"]);
  });

  it("says what each colleague may use, and what for", () => {
    for (const a of AGENTS) {
      expect(a.tools.length, a.slug).toBeGreaterThan(0);
      for (const t of a.tools) {
        // A tool listed without a limit is an access grant nobody reviewed.
        expect(t.purpose.length, `${a.slug}/${t.name}`).toBeGreaterThan(8);
      }
      expect(a.skills.length, a.slug).toBeGreaterThan(2);
    }
  });

  it("hands work off only to colleagues who exist", () => {
    const names = new Set(AGENTS.map((a) => a.name));
    for (const a of AGENTS) {
      if (a.handsOff) {
        expect(
          names.has(a.handsOff.to) ||
            a.handsOff.to === "the specialist who owns it",
          `${a.slug} hands off to ${a.handsOff.to}`,
        ).toBe(true);
      }
    }
  });

  it("mentions nothing about how the product is built", () => {
    const forbidden =
      /\b(archetype|runtime|MCP|Layer [123]|SSE|endpoint|milestone|ADR|Qwik)\b/i;
    expect(JSON.stringify(AGENTS)).not.toMatch(forbidden);
  });

  it("splits the roster the way the interface splits it", () => {
    expect(workingAgents().every((a) => a.status === "working")).toBe(true);
    expect(availableAgents().every((a) => a.status === "available")).toBe(true);
    // Only the assistant is on staff right now.
    expect(workingAgents().length).toBe(1);
    // Nobody is on the "available" shelf until a second archetype ships.
    expect(availableAgents().length).toBe(0);
  });

  it("resolves a colleague by slug, and nothing by a slug that is not one", () => {
    expect(agentBySlug(AGENTS[0].slug)?.name).toBe(AGENTS[0].name);
    expect(agentBySlug("not-a-colleague")).toBeUndefined();
  });
});

describe("the teams", () => {
  it("no team references a slug that does not exist in AGENTS", () => {
    // The contract: TEAMS is exported for other modules that depend on
    // the shape, but its entries must stay grounded in the real roster.
    // If a future team ships with a slug no one has hired yet, this
    // catches it before the UI shows a ghost avatar.
    const known = new Set(AGENTS.map((a) => a.slug));
    for (const team of TEAMS) {
      for (const slug of team.agentSlugs) {
        expect(known.has(slug), `${team.slug}/${slug}`).toBe(true);
      }
    }
  });

  it("answer which teams a colleague belongs to", () => {
    for (const team of TEAMS) {
      for (const slug of team.agentSlugs) {
        expect(teamsForAgent(slug).map((t) => t.slug)).toContain(team.slug);
      }
    }
    expect(teamsForAgent("not-a-colleague")).toHaveLength(0);
  });
});

describe("the organisation", () => {
  it("names every seat and what it answers for", () => {
    const keys = new Set<string>();
    for (const role of ORG_ROLES) {
      expect(keys.has(role.key), `duplicate role ${role.key}`).toBe(false);
      keys.add(role.key);
      expect(role.title.length).toBeGreaterThan(2);
      // A seat with no stated responsibility is a title, not a role.
      expect(role.responsibility.length, role.key).toBeGreaterThan(10);
    }
  });

  it("leaves at least one seat open, because an empty seat is information", () => {
    expect(ORG_ROLES.some((r) => r.holder === null)).toBe(true);
  });

  it("fills seats only with people who exist, or with the signed-in person", () => {
    for (const role of ORG_ROLES) {
      if (role.holder === null || role.holder === "you") continue;
      expect(personById(role.holder), role.key).toBeTruthy();
    }
  });

  it("gives every person two initials", () => {
    for (const p of PEOPLE) {
      expect(p.initials, p.name).toMatch(/^[A-Z]{2}$/);
    }
  });
});
