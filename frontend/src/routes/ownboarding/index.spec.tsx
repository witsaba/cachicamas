/**
 * Behavioural spec for `routes/ownboarding/index.tsx`.
 *
 * Reference: `openspec/changes/2026-07-06-ownboarding/specs/ownboarding/spec.md`
 *   R-OW-001 (S-OW-002) — authed renders the form.
 *   R-OW-003 (S-OW-020) — submit success navigates to /home.
 *
 * Mirrors the pattern used by `routes/organizations/new/index.spec.tsx`:
 * mock `@builder.io/qwik-city` so useNavigate returns a QRL no-op
 * (the qc-n router context is absent in createDOM()), then mock
 * `plugin@auth` to control the authed/anon branch.
 */
import { $, type QRL } from "@builder.io/qwik";
import { createDOM } from "@builder.io/qwik/testing";
import { test, expect, vi } from "vitest";
import Index from "./index";

vi.mock("@builder.io/qwik-city", async () => {
  const actual =
    await vi.importActual<typeof import("@builder.io/qwik-city")>(
      "@builder.io/qwik-city",
    );
  return {
    ...actual,
    useNavigate: () => $(async () => undefined),
    useLocation: () => ({ url: new URL("http://localhost/ownboarding") }),
  };
});

vi.mock("~/routes/plugin@auth", () => ({
  useSession: () => ({ value: { user: { name: "Test" } } }),
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

vi.mock("~/lib/api", () => ({
  createOrganization: vi.fn(async () => ({
    ok: true as const,
    value: { id: 1, full_name: "Acme", identification: "acme" },
  })),
  getSetupState: vi.fn(async () => ({ hasOrganization: true })),
}));

test("[routes/ownboarding] authed render shows the form heading and OwnboardingForm (R-OW-001 / S-OW-002)", async () => {
  const { screen, render } = await createDOM();
  await render(<Index />);
  const heading = screen.querySelector("h1");
  expect(heading).toBeTruthy();
  expect(heading?.textContent).toContain("Name your company");
  const formEl = screen.querySelector('[data-testid="ownboarding-form"]');
  expect(formEl).toBeTruthy();
  const submitBtn = screen.querySelector(
    '[data-testid="ownboarding-submit"]',
  );
  expect(submitBtn).toBeTruthy();
  expect(submitBtn?.getAttribute("type")).toBe("submit");
});