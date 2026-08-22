/**
 * Behavioural spec for `routes/settings/index.tsx` — System.
 *
 * The screen this replaced was a Launchpad grid whose tiles opened the Prompts
 * and Skills surfaces; both belonged to the retired Software Development
 * Framework identity and went with it (ADR 0009). What is left is deliberately
 * small, and the assertion worth having is that it stays honest: the panel of
 * things that are NOT configurable is the point of the screen, not filler.
 */
import { createDOM } from "@builder.io/qwik/testing";
import { $, type QRL } from "@builder.io/qwik";
import { describe, it, expect, vi } from "vitest";
import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import Index, { head } from "./index";
import type { DocumentHeadValue } from "@builder.io/qwik-city";

// `DocumentHead` is a union of a static value and a resolver function; these
// routes export the static form, so narrow once here rather than at every use.
const systemHead = head as DocumentHeadValue;

vi.mock("~/routes/plugin@auth", () => ({
  useSession: () => ({
    value: { user: { name: "Alice", email: "alice@example.com" } },
  }),
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

const source = readFileSync(
  fileURLToPath(import.meta.url).replace(/\.spec\.tsx$/, ".tsx"),
  "utf8",
);

describe("routes/settings — System", () => {
  it("captures the SSR cookie before either guard can throw", () => {
    const cookieAt = source.indexOf("setSsrCookieHeader(");
    const authAt = source.indexOf("requireAuthRedirect(event)");
    const onboardAt = source.indexOf("await requireOwnboarding(event)");
    expect(cookieAt).toBeGreaterThan(-1);
    expect(authAt).toBeGreaterThan(cookieAt);
    expect(onboardAt).toBeGreaterThan(authAt);
  });

  it("renders exactly one <h1>, titled System", async () => {
    const { screen, render } = await createDOM();
    await render(<Index />);
    const h1s = screen.querySelectorAll("h1");
    expect(h1s.length).toBe(1);
    expect(h1s[0].textContent).toContain("System");
  });

  it("shows who is signed in, and how to leave", async () => {
    const { screen, render } = await createDOM();
    await render(<Index />);
    const account = screen.querySelector('[data-testid="account-panel"]');
    expect(account).toBeTruthy();
    const text = (account as HTMLElement).textContent ?? "";
    expect(text).toContain("Alice");
    expect(text).toContain("alice@example.com");
    const signOut = Array.from(
      (account as HTMLElement).querySelectorAll("a"),
    ).find((a) => a.getAttribute("href") === "/auth/signout/");
    expect(signOut).toBeTruthy();
  });

  it("states what is NOT configurable rather than shipping dead toggles", async () => {
    const { screen, render } = await createDOM();
    await render(<Index />);
    const panel = screen.querySelector('[data-testid="undecided-panel"]');
    expect(panel).toBeTruthy();
    const items = (panel as HTMLElement).querySelectorAll("li");
    expect(items.length).toBeGreaterThan(0);
    // Every open decision names itself AND says why, so the screen reads as a
    // record rather than as an apology.
    for (const li of Array.from(items)) {
      expect((li.textContent ?? "").trim().length).toBeGreaterThan(20);
    }
  });

  it("ships no form control at all — nothing here is settable yet", async () => {
    const { screen, render } = await createDOM();
    await render(<Index />);
    const main = screen.querySelector("main") as HTMLElement;
    expect(main.querySelectorAll("input").length).toBe(0);
    expect(main.querySelectorAll("select").length).toBe(0);
    expect(main.querySelectorAll('[role="switch"]').length).toBe(0);
  });

  it("carries no link into the retired framework surfaces", async () => {
    const { screen, render } = await createDOM();
    await render(<Index />);
    const hrefs = Array.from(screen.querySelectorAll("a")).map((a) =>
      a.getAttribute("href"),
    );
    for (const dead of [
      "/settings/prompts",
      "/settings/skills",
      "/workspaces",
    ]) {
      expect(hrefs).not.toContain(dead);
    }
  });

  it("exports head metadata naming the product in lowercase", () => {
    expect(systemHead.title).toBe("System — cachicamas");
  });
});
