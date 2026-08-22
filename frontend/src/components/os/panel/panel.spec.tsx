/**
 * Panel — the only container in the system.
 *
 * Two things are worth pinning: that a panel is a real landmark with a real
 * heading (a screen built from `<div>`s with amber text is not navigable), and
 * that the header's right-hand note stays a readout rather than growing
 * controls, which is how a header band turns into a toolbar.
 */
import { createDOM } from "@builder.io/qwik/testing";
import { describe, it, expect } from "vitest";
import { Panel } from "./panel";

describe("components/os/panel", () => {
  it("is a section with a real heading, not a styled div", async () => {
    const { screen, render } = await createDOM();
    await render(<Panel label="Archetypes">body</Panel>);
    const section = screen.querySelector("section");
    expect(section).toBeTruthy();
    const heading = section?.querySelector("h2");
    expect(heading?.textContent).toBe("Archetypes");
  });

  it("can drop to h3 so a screen's panels form a real outline", async () => {
    const { screen, render } = await createDOM();
    await render(
      <Panel label="Nested" as="h3">
        body
      </Panel>,
    );
    expect(screen.querySelector("h3")?.textContent).toBe("Nested");
    expect(screen.querySelector("h2")).toBeFalsy();
  });

  it("renders the note as text only", async () => {
    const { screen, render } = await createDOM();
    await render(
      <Panel label="Runtime" note="3 layers" testId="p">
        body
      </Panel>,
    );
    const note = screen.querySelector('[data-testid="p-note"]');
    expect(note?.textContent).toBe("3 layers");
    // A header note that can be clicked is a toolbar wearing a readout's
    // clothes; the panel deliberately gives it no way to become one.
    expect(note?.querySelector("a")).toBeFalsy();
    expect(note?.querySelector("button")).toBeFalsy();
  });

  it("omits the note element entirely when there is nothing to report", async () => {
    const { screen, render } = await createDOM();
    await render(
      <Panel label="Bare" testId="p">
        body
      </Panel>,
    );
    expect(screen.querySelector('[data-testid="p-note"]')).toBeFalsy();
  });

  it("hands padding to the body by default, and yields it on request", async () => {
    const { screen, render } = await createDOM();
    await render(
      <Panel label="Padded" testId="a">
        <span data-testid="child">x</span>
      </Panel>,
    );
    const child = screen.querySelector('[data-testid="child"]');
    expect(child?.parentElement?.className).toContain("p-3");

    const second = await createDOM();
    await second.render(
      <Panel label="Flush" padded={false} testId="b">
        <span data-testid="child2">x</span>
      </Panel>,
    );
    const child2 = second.screen.querySelector('[data-testid="child2"]');
    expect(child2?.parentElement?.className).not.toContain("p-3");
  });

  it("carries the world's material: a rule, a panel ground, no radius", async () => {
    const { screen, render } = await createDOM();
    await render(<Panel label="X">body</Panel>);
    const cls = screen.querySelector("section")?.className ?? "";
    expect(cls).toContain("border-rule");
    expect(cls).toContain("bg-panel");
    expect(cls).not.toMatch(/rounded|shadow-/);
  });
});
