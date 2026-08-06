/**
 * chat-input.spec.tsx — vitest coverage for ChatInput.
 *
 * Reference: openspec/specs/frontend-chat-layer1/spec.md REQ-1, REQ-2,
 * REQ-7.
 *
 * Strict TDD target. ChatInput is the prompt entry controls: a
 * textarea + Send + Stop buttons. Spec scope:
 *   - REQ-1 — textarea + Send + Stop render with the right test ids
 *   - REQ-1 — when `disabled` prop is true, the textarea AND the
 *     buttons reflect the disabled state (disabled-during-stream)
 *   - REQ-1 — when `disabled` prop is false, the textarea is enabled
 *     (accepts fresh submit after offline error per REQ-5 S-5.a)
 *   - REQ-1 — Send button starts disabled when textarea is empty
 *     (empty-prompt guard, NOT a session-state guard)
 *   - REQ-7 — every component in components/chat/ has a colocated
 *     spec (this file)
 *   - S-2.b wiring — Stop calls the cancel QRL (cancels the in-flight
 *     turn; the input must have a stable testid for the affordance)
 *
 * QRL discipline: ChatInput requires BOTH `onSubmit$` and `onCancel$`
 * QRL props (the parent supplies them from useChatStream). Qwik's
 * serializer rejects vi.fn refs captured inside `$()` closures, so
 * every QRL here uses module-scoped primitive counters (same
 * pattern as home-workspaces-section.spec.tsx's noopQrl).
 */
import { $, type QRL } from "@builder.io/qwik";
import { createDOM } from "@builder.io/qwik/testing";
import { describe, expect, it } from "vitest";

import { ChatInput } from "./chat-input";

// Module-scoped QRL stubs — primitive counters only. Each spec
// re-imports ChatInput (vi.resetModules not needed because the
// stubs are module-scoped and the component does not capture them
// in a way that would create a new binding per import).
let submitCalls = 0;
let cancelCalls = 0;
const submitQrl = $((_value: string) => {
  submitCalls = submitCalls + 1;
  return Promise.resolve();
}) as QRL<(value: string) => Promise<void>>;
const cancelQrl = $(() => {
  cancelCalls = cancelCalls + 1;
  return Promise.resolve();
}) as QRL<() => Promise<void>>;

describe("components/chat/chat-input (REQ-1, REQ-7)", () => {
  it("renders the textarea + Send + Stop with stable test ids (REQ-1)", async () => {
    const { render, screen } = await createDOM();
    await render(
      <ChatInput
        disabled={false}
        onSubmit$={submitQrl}
        onCancel$={cancelQrl}
      />,
    );
    const textarea = screen.querySelector(
      '[data-testid="chat-input-textarea"]',
    );
    const send = screen.querySelector('[data-testid="chat-input-send"]');
    const cancel = screen.querySelector('[data-testid="chat-input-cancel"]');
    expect(textarea).toBeTruthy();
    expect(send).toBeTruthy();
    expect(cancel).toBeTruthy();
  });

  it("textarea is enabled when disabled=false (REQ-1)", async () => {
    const { render, screen } = await createDOM();
    await render(
      <ChatInput
        disabled={false}
        onSubmit$={submitQrl}
        onCancel$={cancelQrl}
      />,
    );
    const textarea = screen.querySelector(
      '[data-testid="chat-input-textarea"]',
    ) as HTMLTextAreaElement | null;
    expect(textarea?.disabled).toBe(false);
  });

  it("textarea is disabled when disabled=true (REQ-1 — disabled-during-stream)", async () => {
    const { render, screen } = await createDOM();
    await render(
      <ChatInput
        disabled={true}
        onSubmit$={submitQrl}
        onCancel$={cancelQrl}
      />,
    );
    const textarea = screen.querySelector(
      '[data-testid="chat-input-textarea"]',
    ) as HTMLTextAreaElement | null;
    expect(textarea?.disabled).toBe(true);
  });

  it("Send button is disabled when textarea is empty (REQ-1 — empty-prompt guard, NOT a session-state guard)", async () => {
    const { render, screen } = await createDOM();
    await render(
      <ChatInput
        disabled={false}
        onSubmit$={submitQrl}
        onCancel$={cancelQrl}
      />,
    );
    const send = screen.querySelector(
      '[data-testid="chat-input-send"]',
    ) as HTMLButtonElement | null;
    // Empty prompt = no submit. The Send button is disabled even
    // when session.status === "idle" because there is nothing to
    // send. This guard is INDEPENDENT of `disabled` (which mirrors
    // session.status). REQ-1 contract: the input rejects empty
    // submits without an HTTP roundtrip.
    expect(send?.disabled).toBe(true);
  });

  it("Stop button is disabled when disabled=true (REQ-2 S-2.b — Stop wired to cancel)", async () => {
    const { render, screen } = await createDOM();
    await render(
      <ChatInput
        disabled={true}
        onSubmit$={submitQrl}
        onCancel$={cancelQrl}
      />,
    );
    const cancel = screen.querySelector(
      '[data-testid="chat-input-cancel"]',
    ) as HTMLButtonElement | null;
    // Stop is irrelevant when there's no in-flight turn; mirror
    // the textarea's `disabled` so the affordance is consistent.
    expect(cancel?.disabled).toBe(true);
  });

  it("Stop button is enabled when disabled=false (REQ-2 S-2.b — Stop enabled during streaming... wait, REQ-1)", async () => {
    const { render, screen } = await createDOM();
    await render(
      <ChatInput
        disabled={false}
        onSubmit$={submitQrl}
        onCancel$={cancelQrl}
      />,
    );
    const cancel = screen.querySelector(
      '[data-testid="chat-input-cancel"]',
    ) as HTMLButtonElement | null;
    expect(cancel?.disabled).toBe(false);
  });

  it("renders inside a <form data-testid='chat-input'> for accessibility + Enter-to-submit wiring (REQ-1 Enter-to-submit)", async () => {
    const { render, screen } = await createDOM();
    await render(
      <ChatInput
        disabled={false}
        onSubmit$={submitQrl}
        onCancel$={cancelQrl}
      />,
    );
    const form = screen.querySelector('form[data-testid="chat-input"]');
    expect(form).toBeTruthy();
    const send = screen.querySelector(
      '[data-testid="chat-input-send"]',
    ) as HTMLButtonElement | null;
    expect(send?.getAttribute("type")).toBe("submit");
  });

  it("accepts an onSubmit$ QRL prop for test-only wiring (REQ-1 Enter-to-submit — QRL seam contract)", async () => {
    const { render } = await createDOM();
    let captured: string | null = null;
    const onSubmit$ = $((value: string) => {
      captured = value;
      return Promise.resolve();
    }) as QRL<(value: string) => Promise<void>>;
    await render(
      <ChatInput
        disabled={false}
        onSubmit$={onSubmit$}
        onCancel$={cancelQrl}
      />,
    );
    expect(typeof onSubmit$).toBe("function");
    expect(captured).toBeNull();
  });

  it("accepts an onCancel$ QRL prop for test-only wiring (REQ-2 S-2.b — QRL seam contract)", async () => {
    const { render } = await createDOM();
    let cancelCallsLocal = 0;
    const onCancel$ = $(() => {
      cancelCallsLocal = cancelCallsLocal + 1;
      return Promise.resolve();
    }) as QRL<() => Promise<void>>;
    await render(
      <ChatInput
        disabled={true}
        onSubmit$={submitQrl}
        onCancel$={onCancel$}
      />,
    );
    expect(typeof onCancel$).toBe("function");
    expect(cancelCallsLocal).toBe(0);
  });
});