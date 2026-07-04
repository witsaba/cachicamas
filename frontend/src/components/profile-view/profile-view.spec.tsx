/**
 * Test for `components/profile-view/profile-view.tsx`.
 *
 * Reference: `openspec/changes/cachicamas-login-ux/specs/profile-home/spec.md`
 *   R-PH-008 — the page opens with a personalized welcome heading.
 *   R-PH-003 — renders a github.com anchor when github_login is set.
 *   R-PH-005 — renders a "Manage organizations" link to /organizations/new.
 *
 * Reference: `openspec/changes/cachicamas-login-ux/specs/app-shell/spec.md`
 *   R-AS-005 — the shell's AvatarDropdown panel header renders name +
 *     email. ProfileView does NOT duplicate that (per UAT-9).
 *
 * Test strategy:
 *   - Render ProfileView with various session shapes.
 *   - Assert the welcome heading, body, and action links.
 *   - Assert the avatar / email / standalone-name testids are
 *     RETIRED (UAT-9) — the shell's AvatarDropdown owns identity
 *     rendering; /profile is a welcome + actions surface.
 */
import { createDOM } from "@builder.io/qwik/testing";
import { describe, it, expect } from "vitest";
import { ProfileView, type ProfileSession } from "./profile-view";

describe("components/profile-view", () => {
  // ============================================================
  // UAT-9 (2026-07-04) — welcome heading + identity retirement
  // R-PH-008 — welcome UX replaces the identity dump
  // ============================================================

  it("R-PH-008: renders a 'Welcome back, <first-name>!' heading personalized to the user", async () => {
    const session: ProfileSession = {
      user: { name: "Braejan Arias", email: "braejan@proton.me", image: null },
    };
    const { screen, render } = await createDOM();
    await render(<ProfileView session={session} />);

    // The welcome heading uses the FIRST name only (a personalization
    // cue), not the full name (which is already shown in the avatar
    // dropdown's panel header).
    const heading = screen.querySelector(
      '[data-testid="profile-welcome"]',
    ) as HTMLElement | null;
    expect(heading).toBeTruthy();
    expect(heading?.textContent?.trim()).toBe("Welcome back, Braejan!");
  });

  it("R-PH-008: welcome heading falls back to 'Welcome back!' when name is missing", async () => {
    const session: ProfileSession = {
      user: { name: null, email: "anon@example.com", image: null },
    };
    const { screen, render } = await createDOM();
    await render(<ProfileView session={session} />);

    const heading = screen.querySelector(
      '[data-testid="profile-welcome"]',
    ) as HTMLElement | null;
    expect(heading).toBeTruthy();
    expect(heading?.textContent?.trim()).toBe("Welcome back!");
  });

  it("UAT-9: does NOT render an avatar <img> on /profile (shell owns identity)", async () => {
    // The avatar <img> lives in the shell's AvatarDropdown trigger
    // (the one allowed <img> per UX-4 — identity, not decoration).
    // ProfileView MUST NOT duplicate it. This test is a regression
    // guard for the original T2.10/T2.11 implementation that had a
    // 64px avatar circle on /profile.
    const session: ProfileSession = {
      user: {
        name: "Braejan Arias",
        email: "braejan@proton.me",
        image: "https://avatars.githubusercontent.com/u/12345",
      },
    };
    const { screen, render } = await createDOM();
    await render(<ProfileView session={session} />);
    // No <img> elements on /profile — even when session.user.image
    // is present, the profile does not render it.
    expect(screen.querySelectorAll("img").length).toBe(0);
    // The data-testid was retired.
    expect(
      screen.querySelector('[data-testid="profile-image"]'),
    ).toBeFalsy();
  });

  it("UAT-9: does NOT render the user's email as a separate line on /profile", async () => {
    // The email is rendered by the avatar dropdown's panel header
    // (reachable from any signed-in surface via the shell). The
    // profile does NOT duplicate it as a separate <p>.
    const session: ProfileSession = {
      user: {
        name: "Braejan Arias",
        email: "braejan@proton.me",
        image: null,
      },
    };
    const { screen, render } = await createDOM();
    await render(<ProfileView session={session} />);
    // The email string MUST NOT appear anywhere in the rendered tree.
    const html = screen.querySelector('[data-testid="profile-signed-in"]');
    expect(html?.textContent ?? "").not.toContain("braejan@proton.me");
    // The retired data-testid is gone.
    expect(
      screen.querySelector('[data-testid="profile-email"]'),
    ).toBeFalsy();
  });

  it("UAT-9: does NOT render the user's full name as a separate line on /profile", async () => {
    // The full name (e.g. "Braejan Arias") appears in the avatar
    // dropdown's panel header. The profile's welcome heading uses
    // the FIRST name as a personalization cue, but the FULL name
    // does not appear as a separate rendered element.
    const session: ProfileSession = {
      user: {
        name: "Braejan Arias",
        email: "braejan@proton.me",
        image: null,
      },
    };
    const { screen, render } = await createDOM();
    await render(<ProfileView session={session} />);
    const html = screen.querySelector('[data-testid="profile-signed-in"]');
    // The full name string does NOT appear as standalone text
    // (only "Braejan" — the first name — appears as part of the
    // welcome heading).
    expect(html?.textContent ?? "").not.toContain("Braejan Arias");
    // The retired data-testid is gone.
    expect(
      screen.querySelector('[data-testid="profile-name"]'),
    ).toBeFalsy();
  });

  it("renders a body paragraph explaining what cachicamas is", async () => {
    const session: ProfileSession = {
      user: { name: "Braejan", email: "b@e.com", image: null },
    };
    const { screen, render } = await createDOM();
    await render(<ProfileView session={session} />);
    const body = screen.querySelector('[data-testid="profile-body"]');
    expect(body).toBeTruthy();
    expect(body?.textContent ?? "").toContain("cachicamas");
    expect(body?.textContent ?? "").toContain("organizations");
  });

  it("uses first-name extraction on multi-word names", async () => {
    // Three-word name -> take the first word.
    const s1: ProfileSession = {
      user: { name: "Braejan Jan Arias", email: "b@e.com", image: null },
    };
    const r1 = await createDOM();
    await r1.render(<ProfileView session={s1} />);
    expect(
      r1.screen.querySelector('[data-testid="profile-welcome"]')
        ?.textContent?.trim(),
    ).toBe("Welcome back, Braejan!");

    // Single-word name -> use as-is.
    const s2: ProfileSession = {
      user: { name: "octocat", email: "o@c.com", image: null },
    };
    const r2 = await createDOM();
    await r2.render(<ProfileView session={s2} />);
    expect(
      r2.screen.querySelector('[data-testid="profile-welcome"]')
        ?.textContent?.trim(),
    ).toBe("Welcome back, octocat!");

    // Whitespace-only name -> falls back to generic greeting.
    const s3: ProfileSession = {
      user: { name: "   ", email: "w@x.com", image: null },
    };
    const r3 = await createDOM();
    await r3.render(<ProfileView session={s3} />);
    expect(
      r3.screen.querySelector('[data-testid="profile-welcome"]')
        ?.textContent?.trim(),
    ).toBe("Welcome back!");
  });

  // ============================================================
  // R-PH-005 — Manage organizations link (preserved from PR-4)
  // ============================================================

  it("R-PH-005: renders a 'Manage organizations' link pointing at /organizations/new", async () => {
    const session: ProfileSession = {
      user: { name: "Braejan", email: "b@e.com", image: null },
    };
    const { screen, render } = await createDOM();
    await render(<ProfileView session={session} />);

    const link = screen.querySelector(
      'a[data-testid="profile-manage-orgs"]',
    ) as HTMLAnchorElement | null;
    expect(link).toBeTruthy();
    expect(link?.getAttribute("href")).toBe("/organizations/new");
    expect(link?.textContent ?? "").toContain("Manage organizations");
    // UX polish (UAT-3 pattern): cursor-pointer + transition chrome
    expect(link?.className ?? "").toMatch(/cursor-pointer/);
    expect(link?.className ?? "").toMatch(/transition-/);
  });

  // ============================================================
  // R-PH-003 — github_login external link (preserved from PR-4)
  // ============================================================

  it("R-PH-003: renders a github.com link when github_login is set", async () => {
    const session: ProfileSession = {
      user: {
        name: "Braejan",
        email: "b@e.com",
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
    expect(link?.getAttribute("target")).toBe("_blank");
    expect(link?.getAttribute("rel")).toBe("noopener noreferrer");
    // The label includes the handle for clarity (UX-4 — text
    // announces the affordance).
    expect(link?.textContent ?? "").toContain("github.com/witsaba");
  });

  it("R-PH-003: omits the github_login link when github_login is null/empty", async () => {
    const session: ProfileSession = {
      user: {
        name: "Braejan",
        email: "b@e.com",
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
  // T4.3 — null session renders nothing (anon path is the route's
  // SignInRequiredCard, not this component)
  // ============================================================

  it("renders nothing when session is null (T4.3)", async () => {
    const { screen, render } = await createDOM();
    await render(<ProfileView session={null} />);
    expect(
      screen.querySelector('[data-testid="profile-signed-in"]'),
    ).toBeFalsy();
  });

  it("renders nothing when session.user is null (T4.3)", async () => {
    const { screen, render } = await createDOM();
    await render(<ProfileView session={{ user: null }} />);
    expect(
      screen.querySelector('[data-testid="profile-signed-in"]'),
    ).toBeFalsy();
  });

  it("UAT-5: never renders a Sign out button (avatar dropdown is the sole sign-out affordance)", async () => {
    const session: ProfileSession = {
      user: { name: "Braejan", email: "b@e.com", image: null },
    };
    const { screen, render } = await createDOM();
    await render(<ProfileView session={session} />);
    expect(
      screen.querySelector('[data-testid="profile-sign-out"]'),
    ).toBeFalsy();
    expect(
      screen.querySelector('svg[data-testid="sign-out-icon"]'),
    ).toBeFalsy();
  });
});