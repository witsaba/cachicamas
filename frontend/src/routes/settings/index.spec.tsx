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

// The guard chain moved to the section layout when the workspace shell was
// introduced, so every screen under it inherits one copy. Read the body of
// `onRequest`, not the whole file: the import block is alphabetical and says
// nothing about the order things run in.
const guardBody = (
  readFileSync(
    fileURLToPath(import.meta.url).replace(/index\.spec\.tsx$/, "layout.tsx"),
    "utf8",
  ).split("export const onRequest")[1] ?? ""
);

describe("routes/settings", () => {
  it("captures the SSR cookie before either guard can throw", () => {
    const cookieAt = guardBody.indexOf("setSsrCookieHeader(");
    const authAt = guardBody.indexOf("requireAuthRedirect(event)");
    const onboardAt = guardBody.indexOf("await requireOwnboarding(event)");
    expect(cookieAt).toBeGreaterThan(-1);
    expect(authAt).toBeGreaterThan(cookieAt);
    expect(onboardAt).toBeGreaterThan(authAt);
  });

  it("renders exactly one <h1>, titled Settings", async () => {
    const { screen, render } = await createDOM();
    await render(<Index />);
    const h1s = screen.querySelectorAll("h1");
    expect(h1s.length).toBe(1);
    expect(h1s[0].textContent).toContain("Settings");
  });

  it("shows who is signed in", async () => {
    const { screen, render } = await createDOM();
    await render(<Index />);
    const account = screen.querySelector('[data-testid="account-panel"]');
    expect(account).toBeTruthy();
    const text = (account as HTMLElement).textContent ?? "";
    expect(text).toContain("Alice");
    expect(text).toContain("alice@example.com");
  });

  it("names the company and the plan it is on", async () => {
    const { screen, render } = await createDOM();
    await render(<Index />);
    const company = screen.querySelector('[data-testid="company-panel"]');
    expect(company).toBeTruthy();
    const text = (company as HTMLElement).textContent ?? "";
    expect(text).toContain("Witsaba");
    expect(text).toContain("Plan");
  });

  it("states what a customer may NOT switch off", async () => {
    // A product that lets agents act inside a company has to be explicit about
    // which limits cannot be loosened. A limit you cannot find is a limit
    // nobody trusts.
    const { screen, render } = await createDOM();
    await render(<Index />);
    const panel = screen.querySelector('[data-testid="fixed-limits-panel"]');
    expect(panel).toBeTruthy();
    const items = (panel as HTMLElement).querySelectorAll("li");
    expect(items.length).toBeGreaterThan(0);
    // Every limit names itself AND says what it means, so the section reads as
    // a record rather than as an apology.
    for (const li of Array.from(items)) {
      expect((li.textContent ?? "").trim().length).toBeGreaterThan(40);
    }
    expect((panel as HTMLElement).textContent).toContain(
      "Nothing leaves the building unapproved",
    );
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

  it("says nothing about how the product is built", async () => {
    const { screen, render } = await createDOM();
    await render(<Index />);
    expect(screen.textContent ?? "").not.toMatch(
      /\b(archetype|runtime|MCP|Layer [123]|SSE|endpoint|milestone|ADR)\b/i,
    );
  });

  it("exports head metadata naming the product in lowercase", () => {
    expect(systemHead.title).toBe("Settings — cachicamas");
  });
});
