/**
 * The one rule in this product that carries meaning through form.
 *
 * A PERSON IS A CIRCLE. AN AGENT IS A ROUNDED SQUARE.
 *
 * A directory where the two are indistinguishable would be dishonest in a way
 * no amount of copy fixes — and a shape that carries meaning alone would be
 * inaccessible in a way no amount of shape fixes. So both halves are pinned
 * here: the shapes must differ, and the word must always be available.
 */
import { createDOM } from "@builder.io/qwik/testing";
import { describe, it, expect } from "vitest";
import {
  AgentAvatar,
  PersonAvatar,
  SpeciesLabel,
  departmentFill,
  departmentInk,
} from "./avatar";
import { AGENTS } from "~/lib/mock/staff";

describe("components/workspace/avatar", () => {
  it("draws an agent as a rounded square, never a circle", async () => {
    for (const agent of AGENTS) {
      const { screen, render } = await createDOM();
      await render(<AgentAvatar agent={agent} />);
      const tile = screen.querySelector(
        `[data-testid="agent-avatar-${agent.slug}"]`,
      );
      expect(tile, agent.slug).toBeTruthy();
      const cls = tile?.className ?? "";
      expect(cls, agent.slug).not.toMatch(/rounded-full/);
      expect(cls, agent.slug).toMatch(/rounded-\[/);
      expect(tile?.getAttribute("data-species"), agent.slug).toBe("agent");
    }
  });

  it("draws a person as a circle, never a rounded square", async () => {
    const { screen, render } = await createDOM();
    await render(<PersonAvatar name="Ana Rivas" initials="AR" />);
    const tile = screen.querySelector('[data-species="person"]');
    expect(tile).toBeTruthy();
    expect(tile?.className ?? "").toContain("rounded-full");
  });

  it("keeps a person's photo circular too", async () => {
    const { screen, render } = await createDOM();
    await render(
      <PersonAvatar
        name="Ana Rivas"
        initials="AR"
        image="https://example.com/a.png"
      />,
    );
    const img = screen.querySelector("img");
    expect(img).toBeTruthy();
    expect(img?.className ?? "").toContain("rounded-full");
    // Decorative: the name is always beside it, so the image announces nothing.
    expect(img?.getAttribute("alt")).toBe("");
  });

  it("hides every avatar from assistive technology", async () => {
    // The avatar is recognition for sighted users. The name beside it is what
    // is announced; an avatar that also announced would say everything twice.
    const agent = await createDOM();
    await agent.render(<AgentAvatar agent={AGENTS[0]} />);
    expect(
      agent.screen
        .querySelector('[data-species="agent"]')
        ?.getAttribute("aria-hidden"),
    ).toBe("true");

    const person = await createDOM();
    await person.render(<PersonAvatar name="Ana" initials="AN" />);
    expect(
      person.screen
        .querySelector('[data-species="person"]')
        ?.getAttribute("aria-hidden"),
    ).toBe("true");
  });

  it("spells the species out in a word", async () => {
    const agent = await createDOM();
    await agent.render(<SpeciesLabel species="agent" />);
    expect(agent.screen.textContent).toContain("Agent");

    const person = await createDOM();
    await person.render(<SpeciesLabel species="person" />);
    expect(person.screen.textContent).toContain("Person");
  });

  it("maps every department to a token, and to a different one each time", () => {
    const fills = AGENTS.map((a) => departmentFill(a.department));
    const inks = AGENTS.map((a) => departmentInk(a.department));
    expect(new Set(fills).size).toBe(fills.length);
    expect(new Set(inks).size).toBe(inks.length);
    for (const f of fills) expect(f).toMatch(/^bg-dept-/);
    for (const i of inks) expect(i).toMatch(/^text-dept-/);
  });
});
