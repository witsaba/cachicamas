/**
 * routes/chat/index.spec.tsx — vitest coverage for the /chat route.
 *
 * Reference: openspec/specs/frontend-chat-layer1/spec.md REQ-3,
 *   REQ-5, REQ-7.
 *
 * Test scope:
 *   - REQ-3 S-3.c (anon branch) — anonymous visitor renders
 *     SignInRequiredCard with redirectTo='/chat'
 *   - REQ-5 S-5.a — offline error renders inline when the
 *     hook-owned session reports kind='offline' (the literal
 *     "backend not wired" phrase surfaces verbatim)
 *   - REQ-1 S-1.a (authed branch) — authed user renders the
 *     ChatWindow surface (the chat-window.spec.tsx spec covers
 *     the bubble + streaming rendering; this route spec asserts
 *     the route composes the window correctly)
 *
 * Auth guard wiring (REQ-3 S-3.a / S-3.b — server-side 302
 * redirects via requireAuthRedirect + requireOwnboarding) is
 * tested structurally in `route-guard.spec.ts` (same pattern as
 * routes/home/); vitest's createDOM does not boot a Qwik City
 * request context so the guard itself is unwritable here.
 *
 * Mock discipline:
 *   - `~/routes/plugin@auth` is mocked at the file level so
 *     useSession / useSignIn resolve without a request context.
 *   - `~/components/chat/use-chat-stream` is mocked the same way
 *     as chat-window.spec.tsx (Qrl alias for Qwik's optimizer).
 *     This isolates the route-level assertion (does the route
 *     mount <ChatWindow /> and pass the session through?) from
 *     the hook's EventSource behavior (covered separately in
 *     use-chat-stream.spec.ts).
 */
import { $, type QRL } from "@builder.io/qwik";
import { createDOM } from "@builder.io/qwik/testing";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import type { ChatSession } from "~/lib/chat-types";

// ---------------------------------------------------------------------------
// plugin@auth mock — same shape as routes/home/index.spec.tsx. The
// vi.doMock pattern (per-test override) is used so the spec can swap
// between anon (SignInRequiredCard branch) and authed (ChatWindow
// branch). Auth tests MUST remain LAST in this file — vi.doMock
// mutates the module registry and would leak into earlier vi.mock
// tests if reversed.
// ---------------------------------------------------------------------------

vi.mock("~/routes/plugin@auth", () => ({
  useSession: () => ({ value: null }),
  useSignIn: () => ({
    submit: $((_fd: FormData) => Promise.resolve()) as QRL<
      (formData: FormData) => unknown
    >,
    actionPath: "/auth/signin",
  }),
  useSignOut: () => ({
    submit: $((_fd: FormData) => Promise.resolve()) as QRL<
      (formData: FormData) => unknown
    >,
    actionPath: "/auth/signout",
  }),
  onRequest: () => Promise.resolve(),
}));

// ---------------------------------------------------------------------------
// use-chat-stream mock — same Qrl-alias pattern as chat-window.spec.tsx.
// The mock factory closes over mockSession so each test can drive the
// session state without re-importing the module.
// ---------------------------------------------------------------------------

const fakeSubmit = $(async (_text: string) => ({
  ok: true as const,
  value: { turnId: "trn_x", streamUrl: "/api/agent/turns/trn_x/events" },
})) as QRL<
  (value: string) => Promise<{
    ok: true;
    value: { turnId: string; streamUrl: string };
  }>
>;
const fakeCancel = $(async () => undefined) as QRL<() => Promise<void>>;
let mockSession: ChatSession = {
  messages: [],
  status: "idle",
};
const useChatStreamMock = () => ({
  session: mockSession,
  submit: fakeSubmit,
  cancel: fakeCancel,
});
vi.mock("~/components/chat/use-chat-stream", () => ({
  useChatStream$: useChatStreamMock,
  useChatStreamQrl: useChatStreamMock,
}));

describe("routes/chat (REQ-3, REQ-5, REQ-7)", () => {
  beforeEach(() => {
    mockSession = { messages: [], status: "idle" };
  });
  afterEach(() => {
    // Nothing to restore — QRLs are module-scoped.
  });

  // ===== Anon tests (vi.mock factory default) =====

  it("anon visitor renders SignInRequiredCard with redirectTo='/chat' (REQ-3 S-3.c anon branch)", async () => {
    const { default: ChatRoute } = await import("./index");
    const { render, screen } = await createDOM();
    await render(<ChatRoute />);
    const card = screen.querySelector('[data-testid="sign-in-required-card"]');
    expect(card).toBeTruthy();
    const form = card?.querySelector(
      'form[data-testid="sign-in-button"]',
    ) as HTMLFormElement | null;
    expect(form).toBeTruthy();
    const redirectTo = form?.querySelector(
      'input[name="redirectTo"]',
    ) as HTMLInputElement | null;
    expect(redirectTo?.value).toBe("/chat");
    const providerId = form?.querySelector(
      'input[name="providerId"]',
    ) as HTMLInputElement | null;
    expect(providerId?.value).toBe("github");
  });

  it("anon card description references 'chat' (REQ-3 S-3.c)", async () => {
    const { default: ChatRoute } = await import("./index");
    const { render, screen } = await createDOM();
    await render(<ChatRoute />);
    const card = screen.querySelector('[data-testid="sign-in-required-card"]');
    const text = (card as HTMLElement | null)?.textContent ?? "";
    expect(text.toLowerCase()).toContain("chat");
    // No spurious surfaces from other routes.
    expect(text).not.toContain("workspace");
    expect(text).not.toContain("billing");
  });

  // ===== Offline-error rendering (REQ-5 S-5.a) =====

  it("renders the literal offline phrase when the session reports an error (REQ-5 S-5.a)", async () => {
    // Drive the mock session into the typed-error shape: the
    // hook flips session.status='idle' on offline AND records the
    // typed error on the last assistant bubble's error field.
    // The ChatMessage.error type is ChatStreamError which only
    // carries the four HTTP-error kinds (validation / conflict /
    // not_found / server). For the offline path, the chat-input
    // accepts a fresh submit because session.status='idle'; the
    // inline alert is what surfaces the literal phrase. We use
    // kind='server' here (the closest typed kind) and override
    // the message with the literal offline phrase — that's the
    // same string the hook flips into the bubble on
    // EventSource.onerror before any message.delta (REQ-5 S-5.b).
    mockSession = {
      messages: [
        {
          id: "a-1",
          role: "assistant",
          text: "",
          status: "error",
          error: {
            kind: "server",
            message: "backend not wired — see PR for backend wire",
          },
        },
      ],
      status: "idle",
    };
    const { default: ChatRoute } = await import("./index");
    // The anon branch must NOT take this path — the spec needs the
    // authed branch to reach ChatWindow. Override plugin@auth for
    // this test only; vi.doMock mutates the registry so we keep it
    // AFTER the anon tests above (see file header).
    vi.resetModules();
    vi.doMock("~/routes/plugin@auth", () => ({
      useSession: () => ({
        value: { user: { name: "Test User", email: "t@e.com" } },
      }),
      useSignIn: () => ({
        submit: $((_fd: FormData) => Promise.resolve()) as QRL<
          (formData: FormData) => unknown
        >,
        actionPath: "/auth/signin",
      }),
      useSignOut: () => ({
        submit: $((_fd: FormData) => Promise.resolve()) as QRL<
          (formData: FormData) => unknown
        >,
        actionPath: "/auth/signout",
      }),
      onRequest: () => Promise.resolve(),
    }));
    const { default: AuthedRoute } = await import("./index");
    const { render, screen } = await createDOM();
    await render(<AuthedRoute />);
    // ChatWindow mounts → empty + error message renders. The
    // error alert contains the literal phrase verbatim (REQ-5
    // S-5.a — greppable in DevTools).
    const alert = screen.querySelector('[data-testid="chat-error-alert"]');
    expect(alert).toBeTruthy();
    const message = screen.querySelector(
      '[data-testid="chat-error-message"]',
    );
    expect(message?.textContent).toContain(
      "backend not wired — see PR for backend wire",
    );
  });

  it("authed visitor renders <ChatWindow /> with the empty affordance (REQ-3 S-3.c authed branch)", async () => {
    vi.resetModules();
    vi.doMock("~/routes/plugin@auth", () => ({
      useSession: () => ({
        value: { user: { name: "Test User", email: "t@e.com" } },
      }),
      useSignIn: () => ({
        submit: $((_fd: FormData) => Promise.resolve()) as QRL<
          (formData: FormData) => unknown
        >,
        actionPath: "/auth/signin",
      }),
      useSignOut: () => ({
        submit: $((_fd: FormData) => Promise.resolve()) as QRL<
          (formData: FormData) => unknown
        >,
        actionPath: "/auth/signout",
      }),
      onRequest: () => Promise.resolve(),
    }));
    const { default: AuthedRoute } = await import("./index");
    const { render, screen } = await createDOM();
    await render(<AuthedRoute />);
    // ChatWindow mounts — empty affordance renders because the
    // session has no messages.
    const window = screen.querySelector('[data-testid="chat-window"]');
    expect(window).toBeTruthy();
    const empty = screen.querySelector('[data-testid="chat-empty"]');
    expect(empty).toBeTruthy();
    // No SignInRequiredCard on the authed branch.
    expect(
      screen.querySelector('[data-testid="sign-in-required-card"]'),
    ).toBeFalsy();
  });

  it("the page exports a DocumentHead for discoverability (REQ-7 — head metadata)", async () => {
    // DocumentHead is either an object OR a function (per Qwik
    // City 1.x's DocumentHead type). Assert the export exists and
    // that calling the function (or unwrapping the object) yields
    // the right title.
    const mod = await import("./index");
    expect(mod.head).toBeDefined();
    const resolved =
      typeof mod.head === "function"
        ? // eslint-disable-next-line @typescript-eslint/no-explicit-any
          (mod.head as (props: any) => { title?: string })({} as any)
        : mod.head;
    expect(resolved.title).toBe("Chat \u2014 Cachicamas");
  });
});