/**
 * ScreenTitle — the title block at the head of an application.
 *
 * The code chip is not decoration: it is the command you would have typed to
 * arrive here, so the title block teaches the command line by repetition. The
 * assertions guard that, and that a screen has exactly one <h1>.
 */
import { createDOM } from "@builder.io/qwik/testing";
import { describe, it, expect } from "vitest";
import { ScreenTitle } from "./screen";

describe("components/os/screen", () => {
  it("renders one h1 carrying the title", async () => {
    const { screen, render } = await createDOM();
    await render(<ScreenTitle code="CHAT" title="Chat" />);
    const h1s = screen.querySelectorAll("h1");
    expect(h1s.length).toBe(1);
    expect(h1s[0].textContent).toBe("Chat");
  });

  it("shows the command that reaches this screen", async () => {
    const { screen, render } = await createDOM();
    await render(<ScreenTitle code="SYSTEM" title="System" testId="t" />);
    expect(screen.querySelector('[data-testid="t"]')?.textContent).toContain(
      "SYSTEM",
    );
  });

  it("renders the lead only when there is one", async () => {
    const withLead = await createDOM();
    await withLead.render(
      <ScreenTitle code="DESK" title="The register" lead="Who works here." />,
    );
    expect(withLead.screen.querySelectorAll("p").length).toBe(1);

    const without = await createDOM();
    await without.render(<ScreenTitle code="DESK" title="The register" />);
    expect(without.screen.querySelectorAll("p").length).toBe(0);
  });

  it("slots a status beside the title without disturbing the heading", async () => {
    const { screen, render } = await createDOM();
    await render(
      <ScreenTitle code="CHAT" title="Chat">
        <span data-testid="status">In build</span>
      </ScreenTitle>,
    );
    expect(screen.querySelector('[data-testid="status"]')).toBeTruthy();
    expect(screen.querySelectorAll("h1").length).toBe(1);
  });
});
