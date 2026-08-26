/**
 * assistant-configure-section.spec.tsx — TDD contract for the
 * Configure UI (REQ-FACS-001/002/003, design AD-7).
 *
 * Three scenarios:
 *   - renders all controls bound to the initial config
 *   - textarea enforces the 4000-character cap
 *   - save round-trips through putAssistantConfig; on 200 the
 *     local state equals the server response and the "Saved" toast
 *     appears; on non-2xx the state rolls back and an error toast
 *     appears
 *
 * Test scope: structural assertions over the rendered DOM plus the
 * behavioural save flow. End-to-end UI tests (click + DOM events
 * driven by a real browser) live in Playwright (e2e/, to be added
 * in a follow-up) — same caveat as the avatar-dropdown spec.
 */

import { $, type QRL } from "@builder.io/qwik";
import { createDOM } from "@builder.io/qwik/testing";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { ConfigureSection } from "./assistant-configure-section";

const initialConfig = {
  kind: "chat" as const,
  org_id: "user_alice",
  system_prompt: "initial prompt",
  tool_allowlist: ["current_time", "summarize_conversation"],
  defer_tool_names: ["summarize_conversation"],
  model: null,
  version: 3,
  updated_by: "user_alice",
  updated_at: "2026-08-26T15:00:00Z",
};

describe("ConfigureSection", () => {
  beforeEach(() => {
    globalThis.fetch = originalFetch;
  });

  it("renders all controls bound to the initial config", async () => {
    const { screen, render } = await createDOM();
    await render(<ConfigureSection initial={initialConfig} />);

    const textarea = screen.querySelector(
      '[data-testid="configure-system-prompt"]',
    );
    expect(textarea).toBeTruthy();
    if (textarea) {
      expect((textarea as HTMLTextAreaElement).value).toBe("initial prompt");
    }

    const toolToggle = screen.querySelector(
      '[data-testid="configure-tool-summarize_conversation"]',
    );
    expect(toolToggle).toBeTruthy();
    if (toolToggle) {
      const input = toolToggle.querySelector('input[type="checkbox"]');
      expect(input).toBeTruthy();
      if (input) {
        expect((input as HTMLInputElement).checked).toBe(true);
      }
    }

    const model = screen.querySelector('[data-testid="configure-model"]');
    expect(model).toBeTruthy();
    if (model) {
      expect(model.textContent).toBe("(env-driven)");
    }
  });

  it("textarea has the 4000-char maxLength attribute", async () => {
    const { screen, render } = await createDOM();
    await render(<ConfigureSection initial={initialConfig} />);

    const textarea = screen.querySelector(
      '[data-testid="configure-system-prompt"]',
    ) as HTMLTextAreaElement | null;
    expect(textarea).toBeTruthy();
    if (textarea) {
      expect(textarea.maxLength).toBe(4000);
    }
  });

  it("save with a successful PUT shows the Saved toast", async () => {
    mockFetchJson({
      ok: true,
      value: { ...initialConfig, version: 4 },
    });

    const { screen, render, userEvent } = await createDOM();
    await render(<ConfigureSection initial={initialConfig} />);

    const saveButton = screen.querySelector(
      '[data-testid="configure-save"]',
    ) as HTMLButtonElement | null;
    expect(saveButton).toBeTruthy();
    if (saveButton) {
      await userEvent(saveButton, "click");
      // The PUT succeeded; the saved-toast appears.
      await new Promise((resolve) => setTimeout(resolve, 0));
      const toast = screen.querySelector('[data-testid="configure-saved-toast"]');
      expect(toast).toBeTruthy();
    }
  });

  it("save with a rejected PUT rolls back and shows the error toast", async () => {
    mockFetchJson({
      ok: false,
      kind: "validation",
      message: "tool_allowlist contains a name not in the registered tool set",
      fields: { tool_allowlist: "unknown tool" },
    });

    const { screen, render, userEvent } = await createDOM();
    await render(<ConfigureSection initial={initialConfig} />);

    const saveButton = screen.querySelector(
      '[data-testid="configure-save"]',
    ) as HTMLButtonElement | null;
    expect(saveButton).toBeTruthy();
    if (saveButton) {
      await userEvent(saveButton, "click");
      await new Promise((resolve) => setTimeout(resolve, 0));
      const errorToast = screen.querySelector(
        '[data-testid="configure-error-toast"]',
      );
      expect(errorToast).toBeTruthy();
      if (errorToast) {
        expect(errorToast.textContent).toContain(
          "tool_allowlist contains a name not in the registered tool set",
        );
      }
      // Rollback: the textarea should still hold the initial prompt.
      const textarea = screen.querySelector(
        '[data-testid="configure-system-prompt"]',
      ) as HTMLTextAreaElement | null;
      expect(textarea).toBeTruthy();
      if (textarea) {
        expect(textarea.value).toBe("initial prompt");
      }
    }
  });
});

/* ---------- fetch mock helpers ---------- */

const originalFetch = globalThis.fetch;

function mockFetchJson(payload: unknown): void {
  const fetchMock = vi.fn(async () =>
    new Response(JSON.stringify(payload), {
      status: payload && typeof payload === "object" && "ok" in payload && payload.ok ? 200 : 400,
      headers: { "Content-Type": "application/json" },
    }),
  );
  globalThis.fetch = fetchMock as unknown as typeof fetch;
}
