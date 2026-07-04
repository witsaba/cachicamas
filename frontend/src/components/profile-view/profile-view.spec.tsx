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
import { createDOM } from "@builder.io/qwik/testing";
import { describe, it, expect } from "vitest";
import { ProfileView, type ProfileSession } from "./profile-view";

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
    expect(
      screen.querySelector('[data-testid="profile-email"]')?.textContent,
    ).toBe("octo@example.com");
  });

  it("renders the user's avatar image only when session.user.image is non-empty", async () => {
    const withImage: ProfileSession = {
      user: {
        name: "Octocat",
        email: "octo@example.com",
        image: "https://github.com/avatars/octo.png",
      },
    };
    const { screen, render } = await createDOM();
    await render(<ProfileView session={withImage} />);
    const img = screen.querySelector(
      '[data-testid="profile-image"]',
    ) as HTMLImageElement | null;
    expect(img).toBeTruthy();
    expect(img?.getAttribute("src")).toBe(
      "https://github.com/avatars/octo.png",
    );
  });

  it("omits the <img> when session.user.image is null or empty", async () => {
    const noImage: ProfileSession = {
      user: { name: "Octocat", email: "octo@example.com", image: null },
    };
    const { screen, render } = await createDOM();
    await render(<ProfileView session={noImage} />);
    expect(screen.querySelectorAll("img").length).toBe(0);
  });

  it("renders nothing when session is null (T4.3 — anon path is the route's SignInRequiredCard, not this component)", async () => {
    const { screen, render } = await createDOM();
    await render(<ProfileView session={null} />);
    // The component renders null when there's no authenticated user.
    // The /profile route renders <SignInRequiredCard> in this case
    // (PR-1b #33), so ProfileView doesn't compete with its own CTA.
    expect(
      screen.querySelector('[data-testid="profile-signed-out"]'),
    ).toBeFalsy();
    expect(
      screen.querySelector('[data-testid="profile-signed-in"]'),
    ).toBeFalsy();
    // And no SignInButton form either — the parent route owns the
    // sign-in affordance.
    expect(
      screen.querySelector('form[data-testid="sign-in-button"]'),
    ).toBeFalsy();
  });

it("does NOT render a Sign out button (UAT-5 — avatar dropdown is the sole sign-out affordance)", async () => {
    // UAT-5 (2026-07-04): the profile previously rendered its own
    // Sign out button. That affordance is now duplicated by the
    // avatar dropdown's Sign out entry (data-testid="avatar-menu-signout"),
    // which is reachable from any signed-in surface via the shell.
    // Keeping two sign-out affordances was duplicative chrome; the
    // profile's button was retired. The avatar dropdown's chrome
    // is asserted in components/avatar-dropdown/avatar-dropdown.spec.tsx
    // (S-AS-042).
    const session: ProfileSession = {
      user: { name: "Octocat", email: "octo@example.com", image: null },
    };
    const { screen, render } = await createDOM();
    await render(<ProfileView session={session} />);
    expect(
      screen.querySelector('[data-testid="profile-sign-out"]'),
    ).toBeFalsy();
    // The sign-out-icon SVG was retired with the button.
    expect(
      screen.querySelector('svg[data-testid="sign-out-icon"]'),
    ).toBeFalsy();
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

  // ============================================================
  // PR-4 of cachicamas-login-ux — /profile enrichment
  // R-PH-003 — github_login link
  // ============================================================

  it("R-PH-003: renders a github.com link when session.user.github_login is set", async () => {
    const session: ProfileSession = {
      user: {
        name: "Octocat",
        email: "octo@example.com",
        image: null,
        github_login: "witsaba",
      },
    };
    const { screen, render } = await createDOM();
    await render(<ProfileView session={session} />);

    const link = screen.querySelector(
      'a[data-testid="profile-github-login"]',
    ) as HTMLAnchorElement | null;
    expect(link).toBeTruthy();
    expect(link?.getAttribute("href")).toBe("https://github.com/witsaba");
    // The anchor opens in a new tab and is hardened against
    // tabnabbing (target=_blank without rel="noopener" lets the
    // opened page access window.opener).
    expect(link?.getAttribute("target")).toBe("_blank");
    expect(link?.getAttribute("rel")).toBe("noopener noreferrer");
  });

  it("R-PH-003: omits the github_login link when session.user.github_login is null/empty", async () => {
    const session: ProfileSession = {
      user: {
        name: "Octocat",
        email: "octo@example.com",
        image: null,
        github_login: null,
      },
    };
    const { screen, render } = await createDOM();
    await render(<ProfileView session={session} />);
    expect(
      screen.querySelector('a[data-testid="profile-github-login"]'),
    ).toBeFalsy();
  });

  // ============================================================
  // PR-4 of cachicamas-login-ux — /profile enrichment
  // R-PH-005 — Manage organizations link
  // ============================================================

  it("R-PH-005: renders a 'Manage organizations' link pointing at /organizations/new", async () => {
    const session: ProfileSession = {
      user: {
        name: "Octocat",
        email: "octo@example.com",
        image: null,
      },
    };
    const { screen, render } = await createDOM();
    await render(<ProfileView session={session} />);

    const link = screen.querySelector(
      'a[data-testid="profile-manage-orgs"]',
    ) as HTMLAnchorElement | null;
    expect(link).toBeTruthy();
    expect(link?.getAttribute("href")).toBe("/organizations/new");
    expect((link?.textContent ?? "").trim()).toContain("Manage organizations");
  });

  // ============================================================
  // PR-4 of cachicamas-login-ux — /profile enrichment
  // R-AS-022 / ADR-0009 — non-https avatar URLs are sanitized
  // ============================================================

  it("R-AS-022: omits the avatar <img> when the URL is non-https (ADR-0009 / safeAvatarSrc)", async () => {
    const session: ProfileSession = {
      user: {
        name: "Octocat",
        email: "octo@example.com",
        image: "javascript:alert(1)",
      },
    };
    const { screen, render } = await createDOM();
    await render(<ProfileView session={session} />);
    // The dangerous scheme MUST NOT reach the DOM. The component
    // reuses safeAvatarSrc() from PR-2a (#34) so the shape is the
    // same as AvatarDropdown's avatar handling.
    expect(screen.querySelectorAll("img").length).toBe(0);
  });

  // ============================================================
  // PR-4 of cachicamas-login-ux — /profile enrichment
  // T4.3 — ProfileView no longer renders the signed-out CTA.
  // The /profile route now uses SignInRequiredCard for anon visitors
  // (PR-1b), so the component's signed-out branch is unreachable.
  // ============================================================

  it("T4.3: never renders the signed-out CTA under any session shape", async () => {
    // Null session — the parent route handles this via
    // SignInRequiredCard. The component should treat the session as
    // "no authenticated content to show" and render nothing,
    // rather than its own CTA competing with the card.
    const { screen: screen1, render: render1 } = await createDOM();
    await render1(<ProfileView session={null} />);
    expect(
      screen1.querySelector('[data-testid="profile-signed-out"]'),
    ).toBeFalsy();

    // Empty user object — same rule: render nothing.
    const empty: ProfileSession = { user: null };
    const { screen: screen2, render: render2 } = await createDOM();
    await render2(<ProfileView session={empty} />);
    expect(
      screen2.querySelector('[data-testid="profile-signed-out"]'),
    ).toBeFalsy();

    // Populated user object — also never renders signed-out.
    const populated: ProfileSession = {
      user: { name: "Octocat", email: "octo@example.com", image: null },
    };
    const { screen: screen3, render: render3 } = await createDOM();
    await render3(<ProfileView session={populated} />);
    expect(
      screen3.querySelector('[data-testid="profile-signed-out"]'),
    ).toBeFalsy();
  });
});
