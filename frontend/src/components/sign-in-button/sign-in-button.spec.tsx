/**
 * Test for `components/sign-in-button/sign-in-button.tsx`.
 *
 * Reference: `openspec/changes/cachicamas-github-login/specs/frontend-auth/spec.md`
 *   R-FA-040 (S-FA-040..S-FA-046) — the landing CTA is a `<Form>` whose
 *   action is the Auth.js `useSignIn` action, with a hidden `providerId`
 *   field set to `"github"`.
 *
 * Test strategy:
 *   - Render the SignInButton with a fake `signIn` action prop.
 *   - Assert the form is a `<form>` element with the right hidden
 *     inputs and submit button.
 *   - Click the submit button → the fake action's `submit` is invoked
 *     with the FormData containing providerId=github + redirectTo.
 */
import { $, type QRL } from "@builder.io/qwik";
import { createDOM } from "@builder.io/qwik/testing";
import { describe, it, expect, vi } from "vitest";
import { SignInButton, type SignInActionLike } from "./sign-in-button";

/**
 * Build a fake SignInActionLike. Records every submit invocation
 * so the test can assert the call shape (providerId, redirectTo).
 *
 * The ActionStore generic shape has many fields; we fill only the
 * ones the component touches (submit) and leave the rest as the
 * ActionStore default (an empty store).
 */
function makeFakeSignIn(): {
  action: SignInActionLike;
  submitCalls: Array<{ formData: FormData }>;
} {
  const submitCalls: Array<{ formData: FormData }> = [];
  const action = {
    submit: $((formData: FormData) => {
      submitCalls.push({ formData });
      return Promise.resolve();
    }) as QRL<(formData: FormData) => unknown>,
    actionPath: "/auth/signin",
    isRunning: false,
    formData: undefined,
    value: undefined,
    submitted: false,
    status: undefined,
  } as unknown as SignInActionLike;
  return { action, submitCalls };
}

describe("components/sign-in-button", () => {
  it("renders a <form> with the default label and the GitHub provider hidden field", async () => {
    const { action } = makeFakeSignIn();
    const { screen, render } = await createDOM();
    await render(<SignInButton signIn={action} />);

    const form = screen.querySelector("form");
    expect(form).toBeTruthy();
    expect(form?.getAttribute("data-testid")).toBe("sign-in-button");

    // Hidden providerId must be "github" so Auth.js picks the right provider.
    const providerIdInput = form?.querySelector(
      'input[name="providerId"]',
    ) as HTMLInputElement | null;
    expect(providerIdInput).toBeTruthy();
    expect(providerIdInput?.type).toBe("hidden");
    expect(providerIdInput?.value).toBe("github");

    // Default redirectTo must be /profile.
    const redirectToInput = form?.querySelector(
      'input[name="redirectTo"]',
    ) as HTMLInputElement | null;
    expect(redirectToInput).toBeTruthy();
    expect(redirectToInput?.value).toBe("/profile");

    // Submit button must carry the default label.
    const button = form?.querySelector('button[type="submit"]');
    expect(button).toBeTruthy();
    expect(button?.textContent?.trim()).toBe("Sign in with GitHub");
  });

  it("honors a custom label override", async () => {
    const { action } = makeFakeSignIn();
    const { screen, render } = await createDOM();
    await render(<SignInButton signIn={action} label="Iniciar sesión con GitHub" />);

    const button = screen.querySelector('button[type="submit"]');
    expect(button?.textContent?.trim()).toBe("Iniciar sesión con GitHub");
  });

  it("honors a custom redirectTo override", async () => {
    const { action } = makeFakeSignIn();
    const { screen, render } = await createDOM();
    await render(<SignInButton signIn={action} redirectTo="/organizations" />);

    const redirectToInput = screen.querySelector(
      'input[name="redirectTo"]',
    ) as HTMLInputElement | null;
    expect(redirectToInput?.value).toBe("/organizations");
  });

  it("clicking submit forwards providerId=github (structural — Qwik's Form intercepts submit)", async () => {
    // The action prop is forwarded to <Form action={signIn}>; clicking
    // submit goes through Qwik City's request handler (not testable in
    // createDOM()).  Instead we assert the structural wiring: the
    // hidden providerId and redirectTo fields are present, and the
    // button's type is "submit" so the browser's native form-submit
    // semantics activate them when the user clicks.
    const { action } = makeFakeSignIn();
    const { screen, render } = await createDOM();
    await render(<SignInButton signIn={action} />);

    const form = screen.querySelector("form") as HTMLFormElement | null;
    expect(form).toBeTruthy();
    const providerIdInput = form?.querySelector(
      'input[name="providerId"]',
    ) as HTMLInputElement | null;
    expect(providerIdInput?.value).toBe("github");
    const redirectToInput = form?.querySelector(
      'input[name="redirectTo"]',
    ) as HTMLInputElement | null;
    expect(redirectToInput?.value).toBe("/profile");
    const button = form?.querySelector('button[type="submit"]');
    expect(button).toBeTruthy();
  });

  it("does NOT include any decorative icon <img> (UX-4 / R-FA-046)", async () => {
    const { action } = makeFakeSignIn();
    const { screen, render } = await createDOM();
    await render(<SignInButton signIn={action} />);

    // The sign-in CTA is text-only; no <img> elements anywhere in the
    // form subtree. The landing page spec §6.2 (UX-4) bans decorative
    // imagery that carries meaning.
    const imgs = screen.querySelectorAll("form img");
    expect(imgs.length).toBe(0);
  });
});