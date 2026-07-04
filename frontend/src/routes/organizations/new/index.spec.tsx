/**
 * Test for `routes/organizations/new/index.tsx` — the create-organization
 * route after the cachicamas-login-ux protected-route guard (R-PR-003).
 *
 * Reference: `openspec/changes/cachicamas-login-ux/specs/protected-routes/spec.md`
 *   S-PR-020 — anonymous renders SignInRequiredCard, NOT the form.
 *   S-PR-021 — authenticated renders OrganizationForm, NOT the card.
 *
 * Why we mock `~qwik-city`:
 *   The route uses `useNavigate()` (Qwik City request context). In
 *   vitest's `createDOM()` the qc-n context is absent, so we mock the
 *   whole module and replace `useNavigate` with a no-op QRL. This
 *   pattern is what the project's existing route-level specs avoid by
 *   testing only the presentational component — but we need the
 *   auth branch to be proven at the route level for this change.
 */
import { $, type QRL } from "@builder.io/qwik";
import { createDOM } from "@builder.io/qwik/testing";
import { test, expect, vi } from "vitest";

vi.mock("@builder.io/qwik-city", async () => {
  const actual =
    await vi.importActual<typeof import("@builder.io/qwik-city")>(
      "@builder.io/qwik-city",
    );
  return {
    ...actual,
    useNavigate: () => {
      // Qwik requires the navigate handle to be a QRL. We wrap a no-op
      // AsyncFunction in $() so the optimizer keeps it as a serializable
      // QRL. Returning a raw function triggers Qwik's "captured variable
      // can not be serialized" error.
      return $(async () => undefined);
    },
  };
});

type MockSessionValue = unknown;

const mockSession: { value: MockSessionValue } = { value: null };

vi.mock("~/routes/plugin@auth", () => {
  const makeAction = () => ({
    submit: $((_fd: FormData) => Promise.resolve()) as QRL<
      (formData: FormData) => unknown
    >,
    actionPath: "/auth/signin",
    isRunning: false,
    formData: undefined,
    value: undefined,
    submitted: false,
    status: undefined,
  });
  return {
    useSession: () => mockSession,
    useSignIn: () => makeAction(),
    useSignOut: () => makeAction(),
    onRequest: () => Promise.resolve(),
  };
});

import NewOrgRoute from "./index";

test("[/organizations/new] anonymous → SignInRequiredCard (S-PR-020)", async () => {
  mockSession.value = null;
  const { screen, render } = await createDOM();
  await render(<NewOrgRoute />);

  const card = screen.querySelector(
    '[data-testid="sign-in-required-card"]',
  );
  expect(card, "expected sign-in card for anonymous").toBeTruthy();

  // The OrganizationForm's <form> contains a `full_name` input — the
  // SignInRequiredCard's <form> (SignInButton) does not. We use that
  // discriminator so the assertion tolerates the card's own form.
  const orgForm = screen.querySelector('form input[name="full_name"]');
  expect(
    orgForm,
    "expected NO OrganizationForm rendered when anonymous",
  ).toBeFalsy();
});

test("[/organizations/new] authenticated → OrganizationForm, no card (S-PR-021)", async () => {
  mockSession.value = {
    user: { name: "Braejan", email: "braejan@example.com", image: null },
  };
  const { screen, render } = await createDOM();
  await render(<NewOrgRoute />);

  const card = screen.querySelector(
    '[data-testid="sign-in-required-card"]',
  );
  expect(
    card,
    "expected no card when authenticated",
  ).toBeFalsy();

  // OrganizationForm has the `full_name` input — its presence proves
  // the form rendered.
  const fullName = screen.querySelector('input[name="full_name"]');
  expect(
    fullName,
    "expected OrganizationForm when authenticated",
  ).toBeTruthy();
});