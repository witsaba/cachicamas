/**
 * Test for `components/avatar-dropdown/avatar-dropdown.tsx`.
 *
 * Reference: `openspec/changes/cachicamas-login-ux/specs/app-shell/spec.md`
 *   R-AS-003..005 — the AvatarDropdown renders the avatar trigger
 *   when authenticated and a panel with three entries when open.
 *
 * Reference: `openspec/changes/cachicamas-login-ux/design.md` §3
 *   Sequence diagram + Qwik lifecycle; §7 ADR-0009 (avatar URL
 *   sanitization); §8 Tailwind patterns.
 *
 * **Test scope.** Same caveat as `use-click-outside.spec.tsx`:
 * QRL-driven click behavior is tested structurally here (the
 * presence of the trigger button, the panel entries, and the
 * signout form). End-to-end click behavior — toggle, outside-click,
 * Escape — is asserted in Playwright (`e2e/avatar-menu.spec.ts`,
 * to be added in a follow-up) and through the AvatarDropdown's
 * structural composition with `useClickOutside` (proven by
 * `use-click-outside.spec.tsx`).
 */
import { $, type QRL } from "@builder.io/qwik";
import { createDOM } from "@builder.io/qwik/testing";
import { describe, it, expect } from "vitest";
import { AvatarDropdown } from "./avatar-dropdown";
import type { SignInActionLike } from "~/components/sign-in-button/sign-in-button";

interface SessionShape {
  user?: {
    name?: string | null;
    email?: string | null;
    image?: string | null;
  } | null;
}

function fakeSignOut(): SignInActionLike {
  return {
    submit: $((_fd: FormData) => Promise.resolve()) as QRL<
      (formData: FormData) => unknown
    >,
    actionPath: "/auth/signout",
    isRunning: false,
    formData: undefined,
    value: undefined,
    submitted: false,
    status: undefined,
  } as unknown as SignInActionLike;
}

describe("components/avatar-dropdown/avatar-dropdown", () => {
  it("renders nothing when session.user is null (R-AS-003)", async () => {
    const { render, screen } = await createDOM();
    const session: SessionShape = { user: null };
    await render(<AvatarDropdown session={session} signOut={fakeSignOut()} />);
    expect(screen.querySelector('[data-testid="avatar-dropdown"]')).toBeFalsy();
  });

  it("carries no entry for the retired workspaces surface", async () => {
    // The GitHub-workspaces surface belonged to the Software Development
    // Framework identity that ADR 0009 retired. Its menu entry is gone with
    // it; this assertion exists so it cannot creep back in.
    const { render, screen } = await createDOM();
    const session: SessionShape = {
      user: { name: "Braejan Jan", email: "braejan@example.com", image: null },
    };
    await render(
      <AvatarDropdown session={session} signOut={fakeSignOut()} forceOpen />,
    );
    expect(
      screen.querySelector('[data-testid="avatar-menu-workspaces"]'),
    ).toBeFalsy();
  });

  it("renders the trigger button with aria-label and avatar image (S-AS-020)", async () => {
    const { render, screen } = await createDOM();
    const session: SessionShape = {
      user: {
        name: "Braejan Jan",
        email: "braejan@example.com",
        image: "https://avatars.githubusercontent.com/u/12345?v=4",
      },
    };
    await render(<AvatarDropdown session={session} signOut={fakeSignOut()} />);
    const trigger = screen.querySelector(
      '[data-testid="avatar-dropdown"]',
    ) as HTMLButtonElement | null;
    expect(trigger).toBeTruthy();
    expect(trigger?.getAttribute("aria-label")).toBe("Braejan Jan menu");
    expect(trigger?.getAttribute("aria-haspopup")).toBe("menu");

    // Avatar image is rendered with the sanitized src.
    const img = trigger?.querySelector("img") as HTMLImageElement | null;
    expect(img).toBeTruthy();
    expect(img?.getAttribute("src")).toBe(
      "https://avatars.githubusercontent.com/u/12345?v=4",
    );
    // aria-expanded starts false (panel closed).
    expect(trigger?.getAttribute("aria-expanded")).toBe("false");
  });

  it("omits the avatar image when session.user.image is null/empty (R-AS-003 graceful)", async () => {
    const { render, screen } = await createDOM();
    const session: SessionShape = {
      user: { name: "Bravo", email: "bravo@example.com", image: null },
    };
    await render(<AvatarDropdown session={session} signOut={fakeSignOut()} />);
    const trigger = screen.querySelector('[data-testid="avatar-dropdown"]');
    expect(trigger).toBeTruthy();
    expect(trigger?.querySelectorAll("img").length).toBe(0);
  });

  it("rejects non-https avatar URLs (S-AS-022 / ADR-0009)", async () => {
    const { render, screen } = await createDOM();
    const session: SessionShape = {
      user: {
        name: "X",
        email: "x@example.com",
        image: "javascript:alert(1)",
      },
    };
    await render(<AvatarDropdown session={session} signOut={fakeSignOut()} />);
    const trigger = screen.querySelector('[data-testid="avatar-dropdown"]');
    // The trigger is still rendered; the avatar <img> is omitted.
    expect(trigger).toBeTruthy();
    expect(trigger?.querySelectorAll("img").length).toBe(0);
  });

  it("avatar trigger is a full-width row that tints on hover", async () => {
    // The trigger overrides three of `<Button>`'s primary-variant tokens: the
    // amber fill, the label padding, and the rule colour. Tailwind 4 emits
    // utilities alphabetically, so an override that collides with a variant
    // token only wins with the `!` prefix — and the prefix goes AFTER the
    // variant (`hover:!border-amber`), never before it, which Tailwind
    // silently drops. These assertions pin that, because the visual
    // regressions are invisible in review: an amber block bleeding around the
    // avatar, and the image squeezed into a strip by the label padding.
    const { render, screen } = await createDOM();
    const session: SessionShape = {
      user: {
        name: "Braejan Jan",
        email: "braejan@example.com",
        image: "https://avatars.githubusercontent.com/u/12345?v=4",
      },
    };
    await render(<AvatarDropdown session={session} signOut={fakeSignOut()} />);
    const trigger = screen.querySelector(
      'button[data-testid="avatar-dropdown"]',
    ) as HTMLButtonElement | null;
    expect(trigger).toBeTruthy();
    const cls = trigger?.className ?? "";
    expect(cls).toMatch(/cursor-pointer/);
    expect(cls).toMatch(/transition-/);
    // A full-width row in the rail's footer, tinting on hover like every
    // other row in the product rather than filling with brand.
    expect(cls).toMatch(/w-full/);
    expect(cls).toMatch(/hover:bg-sunken/);
    expect(cls).toMatch(/rounded-md/);
    // Nothing travels or scales under the pointer.
    expect(cls).not.toMatch(/scale-/);
    expect(cls).not.toMatch(/translate/);
  });

  it("panel lists Profile, Chat, Settings and Sign out", async () => {
    const { render, screen } = await createDOM();
    const session: SessionShape = {
      user: {
        name: "Braejan Jan",
        email: "braejan@example.com",
        image: "https://avatars.githubusercontent.com/u/12345?v=4",
      },
    };
    await render(
      <AvatarDropdown session={session} signOut={fakeSignOut()} forceOpen />,
    );
    // The `forceOpen` prop is a test-only escape hatch to render
    // the panel unconditionally. The production consumer toggles it
    // via `useClickOutside`. The behavior is asserted in
    // `use-click-outside.spec.tsx` and the Playwright e2e.
    const profile = screen.querySelector('[data-testid="avatar-menu-profile"]');
    expect(profile).toBeTruthy();
    expect((profile as HTMLAnchorElement).getAttribute("href")).toBe(
      "/profile/",
    );
    expect((profile as HTMLElement).textContent ?? "").toContain("Profile");

    const orgs = screen.querySelector('[data-testid="avatar-menu-orgs"]');
    expect(orgs).toBeFalsy();

    // 2026-08-06 cachicamas-frontend-chat-layer1 (REQ-3 / T-09):
    // The "Chat" entry is the live authenticated navigation
    // discoverability hook for /chat. Auth gating is inherited
    // from the dropdown's anon-render-nothing branch + the /chat
    // route's onRequest guard.
    const chat = screen.querySelector('[data-testid="avatar-menu-chat"]');
    expect(chat).toBeTruthy();
    expect((chat as HTMLAnchorElement).getAttribute("href")).toBe("/chat/");
    expect((chat as HTMLElement).textContent ?? "").toContain("Chat");

    // The System entry is the dock's F8 cell, reachable from the identity
    // menu too so the two navigations never disagree about what exists.
    const settings = screen.querySelector(
      '[data-testid="avatar-menu-settings"]',
    );
    expect(settings).toBeTruthy();
    expect((settings as HTMLAnchorElement).getAttribute("href")).toBe(
      "/settings/",
    );
    expect((settings as HTMLElement).textContent ?? "").toContain("Settings");

    const form = screen.querySelector(
      '[data-testid="avatar-menu-signout-form"]',
    );
    expect(form).toBeTruthy();
    const button = screen.querySelector('[data-testid="avatar-menu-signout"]');
    expect(button).toBeTruthy();
    // Label is wrapped in a <span> (UX-4 amendment 2026-07-04).
    const labelSpan = button?.querySelector("span");
    expect(labelSpan?.textContent?.trim()).toBe("Sign out");

    const redirectTo = form?.querySelector(
      'input[name="redirectTo"]',
    ) as HTMLInputElement | null;
    expect(redirectTo?.value).toBe("/auth/signin");
  });

  it("anon visitor has no Chat link in DOM (auth-gated navigation per REQ-3)", async () => {
    const { render, screen } = await createDOM();
    const session: SessionShape = { user: null };
    await render(<AvatarDropdown session={session} signOut={fakeSignOut()} />);
    // The whole dropdown is null for anon (defensive). The Chat
    // nav is auth-only via the dropdown menu (the /chat route
    // itself enforces requireAuthRedirect).
    expect(screen.querySelector('[data-testid="avatar-menu-chat"]')).toBeFalsy();
  });

  it("Chat link uses the MenuItem affordance (text-first, no imagery) — UX-4", async () => {
    const { render, screen } = await createDOM();
    const session: SessionShape = {
      user: {
        name: "Braejan Jan",
        email: "braejan@example.com",
        image: "https://avatars.githubusercontent.com/u/12345?v=4",
      },
    };
    await render(
      <AvatarDropdown session={session} signOut={fakeSignOut()} forceOpen />,
    );
    const chat = screen.querySelector(
      '[data-testid="avatar-menu-chat"]',
    ) as HTMLAnchorElement | null;
    expect(chat).toBeTruthy();
    expect(chat?.tagName.toLowerCase()).toBe("a");
    // UX-4 (aphantasic-friendly): text-only — no <img>, no <svg>,
    // no decorative content. The Chat affordance is a single
    // text label inside the menu item.
    expect(chat?.querySelectorAll("img").length).toBe(0);
    expect(chat?.querySelectorAll("svg").length).toBe(0);
  });

  it("Sign out menu item renders the Lucide log-out icon + cursor-pointer + hover/transition affordances (UAT-4)", async () => {
    // UAT-4 (2026-07-04): the Sign out menu item had the same UX
    // issues as the SignInButton before UAT-2/3 — no icon (so the
    // functional meaning is text-only and weaker for aphantasic
    // users), no explicit cursor-pointer, and an instant bg hover
    // with no transition. We now:
    //   - render the Lucide "log-out" inline SVG (functional visual
    //     anchor per the UX-4 amendment that now covers both brand
    //     marks and functional icons)
    //   - add cursor-pointer + transition-* + active:translate-y-px
    const { render, screen } = await createDOM();
    const session: SessionShape = {
      user: {
        name: "Braejan Jan",
        email: "braejan@example.com",
        image: "https://avatars.githubusercontent.com/u/12345?v=4",
      },
    };
    await render(
      <AvatarDropdown session={session} signOut={fakeSignOut()} forceOpen />,
    );
    const button = screen.querySelector('[data-testid="avatar-menu-signout"]');
    expect(button).toBeTruthy();
    // Lucide icon present + aria-hidden (label does the talking)
    const icon = button?.querySelector('svg[data-testid="sign-out-icon"]');
    expect(icon).toBeTruthy();
    expect(icon?.getAttribute("aria-hidden")).toBe("true");
    // cursor-pointer + transition + active press-down
    const cls = (button as HTMLElement).className;
    expect(cls).toMatch(/cursor-pointer/);
    expect(cls).toMatch(/transition-/);
  });

  it("panel displays the user's name and email as a header (S-AS-040)", async () => {
    const { render, screen } = await createDOM();
    const session: SessionShape = {
      user: {
        name: "Braejan Jan",
        email: "braejan@example.com",
        image: "https://avatars.githubusercontent.com/u/12345",
      },
    };
    await render(
      <AvatarDropdown session={session} signOut={fakeSignOut()} forceOpen />,
    );
    const panel = screen.querySelector('[role="menu"]') as HTMLElement | null;
    expect(panel).toBeTruthy();
    expect(panel?.textContent).toContain("Braejan Jan");
    expect(panel?.textContent).toContain("braejan@example.com");
  });

  it("panel is closed by default (S-AS-030 panel not rendered)", async () => {
    const { render, screen } = await createDOM();
    const session: SessionShape = {
      user: { name: "Braejan", email: "b@e.com", image: null },
    };
    await render(<AvatarDropdown session={session} signOut={fakeSignOut()} />);
    expect(screen.querySelector('[role="menu"]')).toBeFalsy();
    const trigger = screen.querySelector(
      '[data-testid="avatar-dropdown"]',
    ) as HTMLButtonElement | null;
    expect(trigger?.getAttribute("aria-expanded")).toBe("false");
  });

  it("is text-first inside the panel except for the Sign out icon (UX-4 / aphantasic-friendly)", async () => {
    const { render, screen } = await createDOM();
    const session: SessionShape = {
      user: {
        name: "Braejan",
        email: "b@e.com",
        image: "https://avatars.githubusercontent.com/u/1",
      },
    };
    await render(
      <AvatarDropdown session={session} signOut={fakeSignOut()} forceOpen />,
    );
    const panel = screen.querySelector('[role="menu"]') as HTMLElement | null;
    expect(panel).toBeTruthy();
    // UX-4 (amended 2026-07-04): no decorative imagery except
    // functional icons as visual anchors. Exactly ONE <svg> is
    // allowed in the panel — the Sign out Lucide log-out icon.
    // (The avatar <img> lives on the trigger, NOT in the panel.)
    const svgs = panel?.querySelectorAll("svg") ?? [];
    expect(svgs.length).toBe(1);
    expect(svgs[0]?.getAttribute("data-testid")).toBe("sign-out-icon");
    // No <img> anywhere in the panel.
    expect(panel?.querySelectorAll("img").length).toBe(0);
  });
});
