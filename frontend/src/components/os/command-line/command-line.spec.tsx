/**
 * CommandLine — the launcher.
 *
 * The failure mode this component exists to avoid is the silent one: a command
 * bar that does nothing when you type something it does not know. The refusal
 * path is asserted here, along with the suggestion list actually resolving to
 * the destinations it advertises.
 */
import { createDOM } from "@builder.io/qwik/testing";
import { describe, it, expect } from "vitest";
import { CommandLine } from "./command-line";
import { ARCHETYPES, resolveCommand } from "~/lib/mock/registry";

describe("components/os/command-line", () => {
  it("renders a labelled input with a visible prompt character", async () => {
    const { screen, render } = await createDOM();
    await render(<CommandLine />);
    const input = screen.querySelector(
      '[data-testid="command-line-input"]',
    ) as HTMLInputElement | null;
    expect(input).toBeTruthy();
    expect(input?.getAttribute("aria-label") ?? "").toContain("Command line");
    expect(
      screen.querySelector('[data-testid="command-line-prompt"]')?.textContent,
    ).toBe(">");
  });

  it("tells you what it takes before you have typed anything", async () => {
    const { screen, render } = await createDOM();
    await render(<CommandLine />);
    const placeholder =
      screen
        .querySelector('[data-testid="command-line-input"]')
        ?.getAttribute("placeholder") ?? "";
    // Discoverability is the point: the codes are not lore.
    expect(placeholder).toContain("CHAT");
    expect(placeholder).toContain("Enter");
  });

  it("advertises the keyboard route to itself", async () => {
    const { screen, render } = await createDOM();
    await render(<CommandLine />);
    expect(
      screen.querySelector('[data-testid="command-line"]')?.textContent,
    ).toContain("⌘K");
  });

  it("keeps the suggestion list closed until the line is focused", async () => {
    const { screen, render } = await createDOM();
    await render(<CommandLine />);
    expect(
      screen.querySelector('[data-testid="command-line-suggestions"]'),
    ).toBeFalsy();
  });

  it("shows nothing refused before anything has been typed", async () => {
    const { screen, render } = await createDOM();
    await render(<CommandLine />);
    expect(
      screen.querySelector('[data-testid="command-line-refusal"]'),
    ).toBeFalsy();
  });

  it("resolves every archetype code it advertises — the contract behind the UI", () => {
    // The interaction itself needs a browser; what can be proved here is that
    // nothing the component offers is a dead end.
    for (const a of ARCHETYPES) {
      const result = resolveCommand(a.code);
      expect(result.ok, a.code).toBe(true);
    }
  });

  it("refuses an unknown code with a message that names the alternatives", () => {
    const result = resolveCommand("nope");
    expect(result.ok).toBe(false);
    if (!result.ok) {
      expect(result.message).toContain("nope");
      expect(result.message).toContain("CHAT");
    }
  });
});
