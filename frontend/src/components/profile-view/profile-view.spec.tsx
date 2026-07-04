/**
 * Test for `components/profile-view/profile-view.tsx`.
 *
 * Reference: `openspec/changes/cachicamas-github-login/specs/frontend-auth/spec.md`
 *   R-FA-050 (S-FA-050..S-FA-058) — the /profile route renders the
 *   authenticated user's name + email when session is non-null, and a
 *   sign-in CTA when session is null.
 *
 * Test strategy:
 *   - Render ProfileView with various session shapes.
 *   - Assert the rendered text and the data-testid markers.
 */
import { $, type QRL } from "@builder.io/qwik";
import { createDOM } from "@builder.io/qwik/testing";
import { describe, it, expect, vi } from "vitest";
import {
  ProfileView,
  type ProfileSession,
  type ProfileViewProps,
} from "./profile-view";
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

describe("components/profile-view", () => {
  it("renders the user's name in <h1> when session is non-null (R-FA-058 brand-mark)", async () => {
    const session: ProfileSession = {
      user: { name: "Octocat", email: "octo@example.com", image: null },
    };
    const { screen, render } = await createDOM();
    await render(<ProfileView session={session} />);

    const h1 = screen.querySelector("h1");
    expect(h1).toBeTruthy();
    expect(h1?.textContent?.trim()).toBe("Octocat");

    // F-1 (spec §6.2) + R-FA-058: exactly one <h1> on the page.
    expect(screen.querySelectorAll("h1").length).toBe(1);
    expect(screen.querySelector('[data-testid="profile-name"]')).toBeTruthy();
    expect(screen.querySelector('[data-testid="profile-email"]')?.textContent).toBe(
      "octo@example.com",
    );
  });

  it("renders the user's avatar image only when session.user.image is non-empty", async () => {
    const withImage: ProfileSession = {
      user: { name: "Octocat", email: "octo@example.com", image: "https://github.com/avatars/octo.png" },
    };
    const { screen, render } = await createDOM();
    await render(<ProfileView session={withImage} />);
    const img = screen.querySelector('[data-testid="profile-image"]') as HTMLImageElement | null;
    expect(img).toBeTruthy();
    expect(img?.getAttribute("src")).toBe("https://github.com/avatars/octo.png");
  });

  it("omits the <img> when session.user.image is null or empty", async () => {
    const noImage: ProfileSession = {
      user: { name: "Octocat", email: "octo@example.com", image: null },
    };
    const { screen, render } = await createDOM();
    await render(<ProfileView session={noImage} />);
    expect(screen.querySelectorAll("img").length).toBe(0);
  });

  it("renders the signed-out CTA + SignInButton when session is null", async () => {
    const signIn = fakeSignIn();
    const { screen, render } = await createDOM();
    await render(<ProfileView session={null} signIn={signIn} />);

    expect(screen.querySelector('[data-testid="profile-signed-out"]')).toBeTruthy();
    expect(screen.querySelector('[data-testid="profile-signed-in"]')).toBeFalsy();
    // SignInButton's data-testid is on the form element.
    const signInForm = screen.querySelector(
      'form[data-testid="sign-in-button"]',
    );
    expect(signInForm).toBeTruthy();
    const providerIdInput = signInForm?.querySelector(
      'input[name="providerId"]',
    ) as HTMLInputElement | null;
    expect(providerIdInput?.value).toBe("github");
  });

  it("renders a Sign out button when onSignOut$ is provided", async () => {
    const session: ProfileSession = {
      user: { name: "Octocat", email: "octo@example.com", image: null },
    };
    // The onSignOut$ callback is wired at the call-site (routes/profile)
    // and is invoked by Qwik's event handler, which we don't simulate in
    // vitest. We just assert the structural presence + that the
    // button text matches the locked copy.
    const onSignOut$ = $(() => {}) as QRL<() => unknown>;
    const { screen, render } = await createDOM();
    await render(<ProfileView session={session} onSignOut$={onSignOut$} />);

    const signOutBtn = screen.querySelector(
      '[data-testid="profile-sign-out"]',
    ) as HTMLButtonElement | null;
    expect(signOutBtn).toBeTruthy();
    expect(signOutBtn?.textContent?.trim()).toBe("Sign out");
  });

  it("does not render a Sign out button when onSignOut$ is omitted (read-only mode)", async () => {
    const session: ProfileSession = {
      user: { name: "Octocat", email: "octo@example.com", image: null },
    };
    const { screen, render } = await createDOM();
    await render(<ProfileView session={session} />);
    expect(screen.querySelector('[data-testid="profile-sign-out"]')).toBeFalsy();
  });

  it("renders an empty name gracefully when session.user.name is null", async () => {
    const session: ProfileSession = {
      user: { name: null, email: "octo@example.com", image: null },
    };
    const { screen, render } = await createDOM();
    await render(<ProfileView session={session} />);
    // The <h1> renders empty (not undefined, not "undefined"); this
    // matches the locked domain.Identity.Name = string zero-value.
    const h1 = screen.querySelector("h1");
    expect(h1?.textContent ?? "").toBe("");
  });
});