/**
 * Composer — where a turn is opened.
 *
 * The composer's job while a turn is in flight is to stop you typing into the
 * void and to offer the stop control instead. Both directions are asserted,
 * plus the demonstration notice, which is the screen's standing admission that
 * nothing here reaches a model.
 */
import { $ } from "@builder.io/qwik";
import { createDOM } from "@builder.io/qwik/testing";
import { describe, it, expect } from "vitest";
import { Composer } from "./composer";

const noop = $(() => undefined);
const noopPrompt = $((_p: string) => undefined);

describe("components/chat/composer", () => {
  it("offers Send while idle, and no Stop", async () => {
    const { screen, render } = await createDOM();
    await render(
      <Composer agentName="Finance" status="idle" onSubmit$={noopPrompt} onCancel$={noop} />,
    );
    expect(screen.querySelector('[data-testid="composer-send"]')).toBeTruthy();
    expect(screen.querySelector('[data-testid="composer-stop"]')).toBeFalsy();
    const input = screen.querySelector(
      '[data-testid="composer-input"]',
    ) as HTMLTextAreaElement;
    expect(input.hasAttribute("disabled")).toBe(false);
  });

  it("swaps Send for Stop while a turn is running, and locks the field", async () => {
    const { screen, render } = await createDOM();
    await render(
      <Composer agentName="Finance" status="running" onSubmit$={noopPrompt} onCancel$={noop} />,
    );
    expect(screen.querySelector('[data-testid="composer-stop"]')).toBeTruthy();
    expect(screen.querySelector('[data-testid="composer-send"]')).toBeFalsy();
    const input = screen.querySelector(
      '[data-testid="composer-input"]',
    ) as HTMLTextAreaElement;
    expect(input.hasAttribute("disabled")).toBe(true);
  });

  it("explains why it is locked, differently for running and for suspended", async () => {
    const running = await createDOM();
    await running.render(
      <Composer agentName="Finance" status="running" onSubmit$={noopPrompt} onCancel$={noop} />,
    );
    const runningText =
      running.screen
        .querySelector('[data-testid="composer-input"]')
        ?.getAttribute("placeholder") ?? "";
    expect(runningText).toContain("Finance is working");

    const held = await createDOM();
    await held.render(
      <Composer agentName="Finance" status="held" onSubmit$={noopPrompt} onCancel$={noop} />,
    );
    const heldText =
      held.screen
        .querySelector('[data-testid="composer-input"]')
        ?.getAttribute("placeholder") ?? "";
    expect(heldText).toContain("Answer the request above");
    expect(heldText).not.toBe(runningText);
  });

  it("offers no Stop while suspended — the run is already not moving", async () => {
    const { screen, render } = await createDOM();
    await render(
      <Composer agentName="Finance" status="held" onSubmit$={noopPrompt} onCancel$={noop} />,
    );
    expect(screen.querySelector('[data-testid="composer-stop"]')).toBeFalsy();
    const send = screen.querySelector(
      '[data-testid="composer-send"]',
    ) as HTMLButtonElement;
    expect(send.hasAttribute("disabled")).toBe(true);
  });

  it("labels the field for assistive technology", async () => {
    const { screen, render } = await createDOM();
    await render(
      <Composer agentName="Finance" status="idle" onSubmit$={noopPrompt} onCancel$={noop} />,
    );
    expect(
      screen
        .querySelector('[data-testid="composer-input"]')
        ?.getAttribute("aria-label"),
    ).toContain("Message Finance");
  });

  it("says it is a demonstration, on the control a person is about to use", async () => {
    const { screen, render } = await createDOM();
    await render(
      <Composer agentName="Finance" status="idle" onSubmit$={noopPrompt} onCancel$={noop} />,
    );
    const text =
      screen.querySelector('[data-testid="composer"]')?.textContent ?? "";
    // The composer states the product's one standing promise where a person
    // is about to act on it, rather than in a settings page nobody opens.
    expect(text).toContain("Enter to send");
    expect(text).toContain("without you approving it first");
  });
});
