/**
 * The workspace shell.
 *
 * These carry forward the identity assertions that used to live on the root
 * layout (S-AS-020..022): the signed-in person's avatar, the sanitisation of
 * the URL behind it (ADR-0009), and the fact that an anonymous visitor never
 * sees a company rail belonging to nobody.
 *
 * They also pin the two things the shell owns that no screen can forget:
 * exactly one `<main id="main">` for the skip link to land on, and the
 * standing demonstration notice.
 */
import { $, type QRL } from "@builder.io/qwik";
import { createDOM } from "@builder.io/qwik/testing";
import { describe, it, expect } from "vitest";
import type { SignInActionLike } from "~/components/sign-in-button/sign-in-button";
import { Workspace } from "./workspace";
import { AGENTS } from "~/lib/mock/staff";

// The real action carries Qwik City's full ActionStore shape; the shell only
// ever reads `submit` and `actionPath`, so the stub is narrowed at the cast.
const signOut = {
  submit: $((_fd: FormData) => Promise.resolve()) as QRL<
    (formData: FormData) => unknown
  >,
  actionPath: "/auth/signout",
} as unknown as SignInActionLike;

const SESSION = {
  user: {
    name: "Braejan",
    email: "braejan@example.com",
    image: "https://avatars.githubusercontent.com/u/12345",
  },
};

async function renderShell(session: unknown = SESSION) {
  const { screen, render } = await createDOM();
  await render(
    <Workspace
      section="home"
      session={session as never}
      signOut={signOut}
    >
      <p data-testid="child">child content</p>
    </Workspace>,
  );
  return screen;
}

describe("components/workspace", () => {
  it("renders exactly one <main id='main'> for the skip link to land on", async () => {
    const screen = await renderShell();
    const mains = screen.querySelectorAll("main");
    expect(mains.length).toBe(1);
    expect((mains[0] as HTMLElement).getAttribute("id")).toBe("main");
    expect(screen.querySelector('[data-testid="child"]')).toBeTruthy();
  });

  it("keeps the rail on screen and marks the current section", async () => {
    const screen = await renderShell();
    expect(screen.querySelector('[data-testid="workspace-rail"]')).toBeTruthy();
    const current = screen.querySelector('[data-testid="nav-home"]');
    expect(current?.getAttribute("aria-current")).toBe("page");
    // Exactly one destination may be current.
    expect(screen.querySelectorAll('[aria-current="page"]').length).toBe(1);
  });

  it("lists the colleagues on staff, and only those", async () => {
    // Someone you have not hired is not in your rail; that is what the
    // directory is for.
    const screen = await renderShell();
    for (const agent of AGENTS) {
      const row = screen.querySelector(
        `[data-testid="sidebar-agent-${agent.slug}"]`,
      );
      if (agent.status === "available") {
        expect(row, agent.slug).toBeFalsy();
      } else {
        expect(row, agent.slug).toBeTruthy();
      }
    }
  });

  it("renders the signed-in person's avatar (S-AS-020)", async () => {
    const screen = await renderShell();
    const trigger = screen.querySelector('button[data-testid="avatar-dropdown"]');
    expect(trigger).toBeTruthy();
    expect(trigger?.getAttribute("aria-label")).toContain("Braejan");
    const img = trigger?.querySelector("img") as HTMLImageElement | null;
    expect(img).toBeTruthy();
    expect(img?.getAttribute("src")).toContain("avatars.githubusercontent.com");
  });

  it("sanitizes a non-https avatar URL out of the DOM (S-AS-022, ADR-0009)", async () => {
    const screen = await renderShell({
      user: {
        name: "Braejan",
        email: "braejan@example.com",
        // eslint-disable-next-line no-script-url
        image: "javascript:alert(1)",
      },
    });
    const trigger = screen.querySelector('button[data-testid="avatar-dropdown"]');
    expect(trigger).toBeTruthy();
    expect(trigger?.querySelector("img")).toBeFalsy();
  });

  it("renders no rail at all for an anonymous visitor", async () => {
    // A company rail belonging to nobody is worse than no rail: it implies a
    // signed-in state that is not there.
    const screen = await renderShell(null);
    expect(screen.querySelector('[data-testid="workspace-rail"]')).toBeFalsy();
    expect(screen.querySelector('[data-testid="avatar-dropdown"]')).toBeFalsy();
  });

  it("says once, on every screen, that the data is a demonstration", async () => {
    const screen = await renderShell();
    const strip = screen.querySelector('[data-testid="demo-strip"]');
    expect(strip).toBeTruthy();
    expect(strip?.textContent).toContain("Demonstration workspace");
  });
});
