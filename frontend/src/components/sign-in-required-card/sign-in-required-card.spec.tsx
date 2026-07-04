/**
 * Test for `components/sign-in-required-card/sign-in-required-card.tsx`.
 *
 * Reference: `openspec/changes/cachicamas-login-ux/specs/protected-routes/spec.md`
 *   R-PR-002 — the card renders a heading, a description, and a SignInButton
 *   whose hidden `redirectTo` equals the `redirectTo` prop. The card carries
 *   `data-testid="sign-in-required-card"`.
 *
 * Why we pass `signIn` as a prop (not import useSignIn directly):
 *   Same pattern as SignInButton and ProfileView — keeps the component
 *   testable in vitest `createDOM()` without the Qwik City request
 *   context.
 */
import { $, type QRL } from "@builder.io/qwik";
import { createDOM } from "@builder.io/qwik/testing";
import { describe, it, expect } from "vitest";
import { SignInRequiredCard } from "./sign-in-required-card";
import type { SignInActionLike } from "~/components/sign-in-button/sign-in-button";

function fakeSignIn(): SignInActionLike {
  return {
    submit: $((_fd: FormData) => Promise.resolve()) as QRL<
      (formData: FormData) => unknown
    >,
    actionPath: "/auth/signin",
    isRunning: false,
    formData: undefined,
    value: undefined,
    submitted: false,
    status: undefined,
  } as unknown as SignInActionLike;
}

describe("components/sign-in-required-card", () => {
  it("renders heading, description, and SignInButton (S-PR-010)", async () => {
    const signIn = fakeSignIn();
    const { screen, render } = await createDOM();
    await render(
      <SignInRequiredCard
        signIn={signIn}
        description="Sign in to configure a new organization."
        redirectTo="/organizations/new"
      />,
    );

    const card = screen.querySelector(
      '[data-testid="sign-in-required-card"]',
    );
    expect(card).toBeTruthy();

    const h1 = screen.querySelector("h1");
    expect(h1?.textContent?.trim()).toBe("Sign in to continue");

    const cardText = (card as HTMLElement).textContent ?? "";
    expect(cardText).toContain("Sign in to configure a new organization.");

    const form = screen.querySelector(
      'form[data-testid="sign-in-button"]',
    );
    expect(form).toBeTruthy();

    const providerIdInput = form?.querySelector(
      'input[name="providerId"]',
    ) as HTMLInputElement | null;
    expect(providerIdInput?.value).toBe("github");
  });

  it("uses the redirectTo prop verbatim on the SignInButton (S-PR-011)", async () => {
    const signIn = fakeSignIn();
    const { screen, render } = await createDOM();
    await render(
      <SignInRequiredCard
        signIn={signIn}
        description="Sign in to view your profile."
        redirectTo="/organizations/new"
      />,
    );
    const form = screen.querySelector(
      'form[data-testid="sign-in-button"]',
    );
    const redirectToInput = form?.querySelector(
      'input[name="redirectTo"]',
    ) as HTMLInputElement | null;
    expect(redirectToInput?.value).toBe("/organizations/new");
  });

  it("uses / as the default redirectTo when omitted", async () => {
    const signIn = fakeSignIn();
    const { screen, render } = await createDOM();
    await render(
      <SignInRequiredCard
        signIn={signIn}
        description="Sign in to continue."
      />,
    );
    const form = screen.querySelector(
      'form[data-testid="sign-in-button"]',
    );
    const redirectToInput = form?.querySelector(
      'input[name="redirectTo"]',
    ) as HTMLInputElement | null;
    expect(redirectToInput?.value).toBe("/");
  });

  it("is text-first — no decorative icons except the SignInButton brand mark (UX-4 / aphantasic-friendly)", async () => {
    // UX-4 was amended on 2026-07-04 (UAT-2): recognizable brand marks
    // as visible visual anchors are aphantasia-friendly. The card
    // embeds a SignInButton which renders the GitHub Octocat inline
    // SVG; that ONE <svg> is allowed. No other SVG/icon imagery
    // should appear in the card chrome (heading, description, layout).
    const signIn = fakeSignIn();
    const { screen, render } = await createDOM();
    await render(
      <SignInRequiredCard
        signIn={signIn}
        description="Sign in to continue."
        redirectTo="/foo"
      />,
    );
    const card = screen.querySelector(
      '[data-testid="sign-in-required-card"]',
    );
    // No <img> elements anywhere — brand marks are inline SVG.
    expect(card?.querySelectorAll("img").length).toBe(0);
    // Exactly ONE <svg>: the GitHub Octocat brand mark inside the
    // embedded SignInButton. Any other SVG would be decoration and
    // violate UX-4.
    const svgs = card?.querySelectorAll("svg") ?? [];
    expect(svgs.length).toBe(1);
    expect(svgs[0]?.getAttribute("data-testid")).toBe(
      "sign-in-button-github-mark",
    );
  });
});
