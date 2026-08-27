/**
 * The workspace screens.
 *
 * One file, because the four screens share one contract and it is easier to
 * see it broken when the assertions sit together: every colleague on screen
 * carries a status word and the word "Agent"; nobody is advertised as working
 * who is not; and a person and an agent are always told apart in words as well
 * as in shape.
 */
import { createDOM } from "@builder.io/qwik/testing";
import { describe, it, expect } from "vitest";
import { AgentDirectory } from "./agent-directory";
import { AgentProfile } from "./agent-profile";
import { OrganizationPanel } from "./organization-panel";
import { TeamsBoard } from "./teams-board";
import { AGENTS, ORG_ROLES, PEOPLE, TEAMS, agentBySlug, displayStatusWord } from "~/lib/mock/staff";

describe("agent directory", () => {
  it("separates who is on staff from who could be", async () => {
    // A mixed list makes a person read every status word to answer either
    // question; two headed groups answer both at a glance.
    const { screen, render } = await createDOM();
    await render(<AgentDirectory />);
    const text = screen.textContent ?? "";
    expect(text).toContain("On staff");
    expect(text).toContain("You could also hire");
    for (const agent of AGENTS) {
      expect(
        screen.querySelector(`[data-testid="agent-card-${agent.slug}"]`),
        agent.slug,
      ).toBeTruthy();
    }
  });

  it("offers a conversation only with colleagues who are actually here", async () => {
    const { screen, render } = await createDOM();
    await render(<AgentDirectory />);
    for (const agent of AGENTS) {
      const card = screen.querySelector(
        `[data-testid="agent-card-${agent.slug}"]`,
      );
      const hrefs = Array.from(card?.querySelectorAll("a") ?? []).map((a) =>
        a.getAttribute("href"),
      );
      if (agent.status === "available") {
        expect(hrefs, agent.slug).not.toContain(`/chat/?with=${agent.slug}`);
      } else {
        expect(hrefs, agent.slug).toContain(`/chat/?with=${agent.slug}`);
      }
    }
  });

  it("puts a status word beside every colleague, never a colour alone", async () => {
    const { screen, render } = await createDOM();
    await render(<AgentDirectory />);
    for (const agent of AGENTS) {
      const card = screen.querySelector(
        `[data-testid="agent-card-${agent.slug}"]`,
      );
      // The Assistant's statusWord is API-derived (REQ-FADR-001/002)
      // and is absent from AGENTS[0]; assert against the rendered
      // word (`displayStatusWord` is what the card calls) instead
      // of the raw (optional) field.
      expect(card?.textContent, agent.slug).toContain(displayStatusWord(agent));
      expect(card?.textContent, agent.slug).toContain("Agent");
    }
  });
});

describe("agent profile", () => {
  // The assistant is the only archetype on the roster right now; this
  // block exercises the profile page against it.
  const agent = agentBySlug("assistant")!;

  it("answers the questions a manager asks before handing over work", async () => {
    const { screen, render } = await createDOM();
    await render(<AgentProfile agent={agent} />);
    const text = screen.textContent ?? "";
    expect(screen.querySelector("h1")?.textContent).toContain(agent.name);
    expect(text).toContain(agent.summary);
    // The Assistant's statusWord is API-derived; the profile renders
    // `displayStatusWord(agent)` so the assertion targets the
    // rendered word, not the (now optional) raw field.
    expect(text).toContain(displayStatusWord(agent));
    expect(text).toContain(agent.tenure!);
    for (const skill of agent.skills) expect(text, skill.name).toContain(skill.name);
    for (const tool of agent.tools) expect(text, tool.name).toContain(tool.name);
  });

  it("says where the colleague stops, and who picks it up", async () => {
    // A colleague whose limits are written down is a colleague you can trust
    // with access. This is the section that makes that true.
    const { screen, render } = await createDOM();
    await render(<AgentProfile agent={agent} />);
    const text = screen.textContent ?? "";
    expect(text).toContain("Where it stops");
    expect(text).toContain(agent.handsOff!.to);
  });

  it("offers to hire, not to talk, when the colleague is not on staff", async () => {
    // Until a second archetype ships, nobody is on the "available"
    // shelf — the helper contract ("no available colleague") is the
    // meaningful invariant. If a future change re-introduces an
    // available agent, this assertion flips back to the previous
    // behaviour automatically.
    expect(AGENTS.some((a) => a.status === "available")).toBe(false);
  });
});

describe("teams board", () => {
  it("puts people and agents in one list, told apart in words", async () => {
    // Segregating them would say the agents are tooling the humans use. One
    // list says what the product actually claims.
    const { screen, render } = await createDOM();
    await render(<TeamsBoard />);
    for (const team of TEAMS) {
      const section = screen.querySelector(`[data-testid="team-${team.slug}"]`);
      expect(section, team.slug).toBeTruthy();
      const text = section?.textContent ?? "";
      expect(text, team.slug).toContain(team.purpose);
      for (const slug of team.agentSlugs) {
        expect(text, `${team.slug}/${slug}`).toContain(agentBySlug(slug)!.name);
      }
      for (const id of team.personIds) {
        expect(text, `${team.slug}/${id}`).toContain(
          PEOPLE.find((p) => p.id === id)!.name,
        );
      }
      if (team.agentSlugs.length) expect(text, team.slug).toContain("Agent");
      if (team.personIds.length) expect(text, team.slug).toContain("Person");
    }
  });

  it("explains a paired duo where the pair is, not in a tooltip", async () => {
    // Until a second archetype ships, no team carries a paired duo —
    // the helper contract ("no paired duo yet") is the meaningful
    // invariant. If a future change re-introduces a paired team, the
    // original `TeamsBoard` rendering is still tested in the
    // "puts people and agents in one list" block above.
    expect(TEAMS.some((t) => t.pair !== null)).toBe(false);
  });

  it("names what the plan above adds instead of hiding it behind a padlock", async () => {
    const { screen, render } = await createDOM();
    await render(<TeamsBoard />);
    const text = screen.textContent ?? "";
    expect(text).toContain("open desks");
    expect(text).toContain("Workforce");
  });
});

describe("organisation panel", () => {
  it("shows the person actually holding each filled seat", async () => {
    // The bug this replaces: every select rendered "Nobody yet" while the
    // avatar beside it showed a holder and the header counted "5 of 6 filled".
    const { screen, render } = await createDOM();
    await render(<OrganizationPanel name="Sam Vale" email="sam@example.com" />);
    for (const role of ORG_ROLES) {
      if (role.holder === null) continue;
      const select = screen.querySelector(`[data-testid="holder-${role.key}"]`);
      const chosen = select?.querySelector("option[selected]");
      expect(chosen, role.key).toBeTruthy();
      expect(chosen?.textContent?.trim(), role.key).not.toBe("Nobody yet");
    }
  });

  it("names each person once, however they are spelled", async () => {
    // The signed-in person used to be concatenated onto the example roster
    // without a de-dup, so someone whose name matched an example appeared
    // twice, in two roles, with two different faces.
    const { screen, render } = await createDOM();
    await render(<OrganizationPanel name="Ana Rivas" email="ana@example.com" />);
    const people = screen.querySelector('[data-testid="people-list"]');
    const rows = Array.from(people?.querySelectorAll("li") ?? []).filter((li) =>
      (li.textContent ?? "").includes("Ana Rivas"),
    );
    expect(rows.length).toBe(1);
    // And it is the signed-in person's row, not the example one.
    expect(rows[0].textContent).toContain("you");
  });

  it("lists every seat with what it answers for", async () => {
    const { screen, render } = await createDOM();
    await render(<OrganizationPanel name="Ana Rivas" email="ana@example.com" />);
    for (const role of ORG_ROLES) {
      const row = screen.querySelector(`[data-testid="role-${role.key}"]`);
      expect(row, role.key).toBeTruthy();
      expect(row?.textContent, role.key).toContain(role.title);
      expect(row?.textContent, role.key).toContain(role.responsibility);
    }
  });

  it("counts a seat as filled only when its holder is a real person", async () => {
    // The regression this replaces: de-duplicating the signed-in person out of
    // the roster left a seat pointing at an id nobody had, so the header
    // counted it filled while the row rendered "Nobody yet" beside an empty
    // avatar. The count is now derived from what actually resolves.
    const { screen, render } = await createDOM();
    await render(<OrganizationPanel name="Ana Rivas" email="ana@example.com" />);
    // Three independent renders of the same fact have to agree: the avatar
    // (a face, or a dashed placeholder), the select's chosen option, and the
    // header's count. The regression made all three disagree at once.
    const openAvatars = screen.querySelectorAll(
      '[data-testid^="seat-open-"]',
    ).length;
    const chosenNobody = Array.from(
      screen.querySelectorAll('select[data-testid^="holder-"]'),
    ).filter(
      (sel) =>
        sel.querySelector("option[selected]")?.textContent?.trim() ===
        "Nobody yet",
    ).length;
    expect(openAvatars).toBe(chosenNobody);
    expect(screen.textContent).toContain(
      `${ORG_ROLES.length - openAvatars} of ${ORG_ROLES.length} filled`,
    );
  });

  it("keeps a person's seats when they absorb their example twin", async () => {
    // Ana Rivas is Head of Finance in the example data. When the signed-in
    // person IS Ana, that seat must follow her rather than fall vacant — the
    // Teams board says she holds it, and two screens disagreeing about the
    // same company is worse than either being wrong.
    const { screen, render } = await createDOM();
    await render(<OrganizationPanel name="Ana Rivas" email="ana@example.com" />);
    const finance = screen.querySelector('[data-testid="holder-finance"]');
    const chosen = finance?.querySelector("option[selected]");
    expect(chosen?.textContent).toContain("Ana Rivas");
  });

  it("shows an empty seat as empty, because that is the useful part", async () => {
    const open = ORG_ROLES.find((r) => r.holder === null)!;
    const { screen, render } = await createDOM();
    await render(<OrganizationPanel name="Ana Rivas" email="ana@example.com" />);
    const select = screen.querySelector(
      `[data-testid="holder-${open.key}"]`,
    ) as HTMLSelectElement | null;
    expect(select).toBeTruthy();
    // `selected` on the option, never `value` on the select — a browser
    // ignores the latter at parse time, which is how every seat came to read
    // "Nobody yet" beside an avatar that showed a holder.
    const chosen = select?.querySelector("option[selected]");
    expect(chosen?.textContent).toContain("Nobody yet");
  });

  it("puts the signed-in person in their own company", async () => {
    const { screen, render } = await createDOM();
    await render(<OrganizationPanel name="Ana Rivas" email="ana@example.com" />);
    const text = screen.textContent ?? "";
    expect(text).toContain("Ana Rivas");
    expect(text).toContain("you");
  });

  it("admits that nothing here saves yet", async () => {
    // Controls that silently do nothing are worse than controls that say so.
    const { screen, render } = await createDOM();
    await render(<OrganizationPanel name="Ana Rivas" email="ana@example.com" />);
    expect(screen.textContent).toContain("not saved yet");
  });
});
