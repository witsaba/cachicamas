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
 *   - Assert the GitHub Octocat inline SVG is present (UX-4 amendment
 *     2026-07-04 — recognizable brand marks as visual anchors).
 *   - Click the submit button → the fake action's `submit` is invoked
 *     with the FormData containing providerId=github + redirectTo.
 */
import { $, type QRL } from "@builder.io/qwik";
import { createDOM } from "@builder.io/qwik/testing";
import { describe, it, expect } from "vitest";
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

    // Submit button must carry the default label (short, per UX-4
    // amendment — the brand mark carries the provider identification).
    // We use .includes() because textContent now includes the <path d="...">
    // SVG path commands; the visible text is wrapped in a <span>.
    const button = screen.querySelector('button[type="submit"]');
    expect(button).toBeTruthy();
    const labelSpan = button?.querySelector("span");
    expect(labelSpan).toBeTruthy();
    expect(labelSpan?.textContent?.trim()).toBe("Sign in");
  });

  it("renders the GitHub Octocat inline <svg> as a recognizable brand anchor (UX-4 amendment)", async () => {
    const { action } = makeFakeSignIn();
    const { screen, render } = await createDOM();
    await render(<SignInButton signIn={action} />);

    // The brand mark MUST be present in the button as an inline SVG
    // (no external <img src="..."> — aphantasia-friendly and self-
    // contained). data-testid guards the lookup against future
    // expansion. aria-hidden="true" so screen readers announce the
    // visible "Sign in" text label, not the SVG.
    const form = screen.querySelector("form");
    expect(form).toBeTruthy();
    const mark = form?.querySelector(
      'svg[data-testid="sign-in-button-github-mark"]',
    );
    expect(mark).toBeTruthy();
    expect(mark?.getAttribute("aria-hidden")).toBe("true");
    // The SVG MUST carry at least one <path> child (the Octocat
    // silhouette). Asserting the path-element existence is sufficient
    // — asserting exact path data is brittle and ties the test to the
    // upstream logo file rather than the consumer contract.
    const path = mark?.querySelector("path");
    expect(path).toBeTruthy();
  });

  it("honors a custom label override", async () => {
    const { action } = makeFakeSignIn();
    const { screen, render } = await createDOM();
    await render(<SignInButton signIn={action} label="Iniciar sesión" />);

    const button = screen.querySelector('button[type="submit"]');
    const labelSpan = button?.querySelector("span");
    expect(labelSpan?.textContent?.trim()).toBe("Iniciar sesión");
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

  it("does NOT include any external <img> asset (no asset URL dependency)", async () => {
    // UX-4 amendment (2026-07-04): an inline SVG brand mark is allowed,
    // but loading the GitHub Octocat from an external <img src="...">
    // is still banned — it would couple the affordance to a third-
    // party CDN, break in offline scenarios, and reintroduce the
    // mental-imagery problem for aphantasic users.
    const { action } = makeFakeSignIn();
    const { screen, render } = await createDOM();
    await render(<SignInButton signIn={action} />);

    const imgs = screen.querySelectorAll("form img");
    expect(imgs.length).toBe(0);
  });

  it("renders with cursor-pointer + hover/transition affordances (UAT-3 mouse UX)", async () => {
    // UAT-3 (2026-07-04): the button lacked an explicit pointer
    // cursor on some OSes (where <button> elements don't get one
    // automatically) and the hover state was an instant bg flip
    // with no transition. We now declare cursor-pointer explicitly
    // + transition the chrome (bg / border / shadow) over 150ms
    // + provide a subtle active-state press-down. We assert the
    // classes are present in jsdom (no real CSS is computed in
    // createDOM); visual verification is the UAT burden.
    const { action } = makeFakeSignIn();
    const { screen, render } = await createDOM();
    await render(<SignInButton signIn={action} />);

    const button = screen.querySelector('button[type="submit"]');
    expect(button).toBeTruthy();
    const cls = (button as HTMLElement).className;
    // explicit cursor for cross-OS consistency (some browsers/OSes
    // do NOT apply cursor: pointer to <button> automatically)
    expect(cls).toMatch(/cursor-pointer/);
    // hover bg still there (Tailwind 4 `!important` syntax: `!` AFTER the variant)
    expect(cls).toMatch(/hover:!border-amber/);
    expect(cls).toMatch(/hover:!text-amber/);
    expect(cls).toMatch(/!bg-transparent/);
    // transition (without it the hover state is an instant flip)
    expect(cls).toMatch(/transition-/);
    // The world has no press travel and no lifted surfaces.
    expect(cls).not.toMatch(/translate/);
    expect(cls).not.toMatch(/shadow-/);
    // Regression guards. The SignInButton overrides `variant="primary"` so the
    // GitHub cell reads as a secondary route into the product rather than as
    // the page's committing action. Tailwind 4 emits utilities alphabetically
    // and the variant's `not-disabled:hover:*` tokens outrank a bare
    // `hover:*`, so every colliding override MUST carry `!` — and the prefix
    // goes AFTER the variant (`hover:!border-amber`), never before it, which
    // Tailwind silently drops. Without these the button would render as a
    // solid amber block with the Octocat punched out of it.
    expect(cls).toMatch(/!bg-transparent/);
    expect(cls).toMatch(/!text-fg/);
    expect(cls).toMatch(/hover:!border-amber/);
    expect(cls).not.toMatch(/ring-/);
  });
});
