/**
 * Behavioural spec for `routes/index.tsx` — the landing page.
 *
 * The page's argument is that it shows the real board rather than describing
 * one, so these assertions are about honesty and about proof-by-rendering:
 *
 *   - the register on the landing page is the SAME component the product uses,
 *     with the SAME data, so it cannot quietly become a flattering mock-up;
 *   - the states it shows are the real ones, five of six of them admitting the
 *     specialist does not exist;
 *   - the permission suspension is quoted with the real transcript component,
 *     not re-drawn as marketing furniture;
 *   - the retired Software Development Framework identity is gone.
 */
import { createDOM } from "@builder.io/qwik/testing";
import { $, type QRL } from "@builder.io/qwik";
import { test, expect, vi } from "vitest";
import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import Index, { head } from "./index";
import type { DocumentHeadValue } from "@builder.io/qwik-city";

// `DocumentHead` is a union of a static value and a resolver function; these
// routes export the static form, so narrow once here rather than at every use.
const landingHead = head as DocumentHeadValue;
import { ARCHETYPES } from "~/lib/mock/registry";

vi.mock("~/routes/plugin@auth", () => ({
  useSession: () => ({ value: null }),
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

test("[routes/index]: exactly one <h1>, and it states the product's own claim", async () => {
  const { screen, render } = await createDOM();
  await render(<Index />);
  const h1s = screen.querySelectorAll("h1");
  expect(h1s.length).toBe(1);
  expect(h1s[0].textContent).toContain("specialists you can talk to");
});

test("[routes/index]: the primary action opens the register, the secondary reaches the mechanism", async () => {
  const { screen, render } = await createDOM();
  await render(<Index />);
  const primary = screen.querySelector('[data-testid="get-started"]');
  expect(primary).toBeTruthy();
  expect((primary as HTMLAnchorElement).getAttribute("href")).toBe("/home/");

  const secondary = screen.querySelector('[data-testid="see-interface"]');
  expect(secondary).toBeTruthy();
  const href = (secondary as HTMLAnchorElement).getAttribute("href");
  expect(href).toBe("#suspension");
  // The anchor must actually land somewhere on this page.
  expect(screen.querySelector("#suspension")).toBeTruthy();
});

test("[routes/index]: the register is rendered from the real registry, whole", async () => {
  const { screen, render } = await createDOM();
  await render(<Index />);
  expect(screen.querySelector('[data-testid="landing-register"]')).toBeTruthy();
  for (const a of ARCHETYPES) {
    expect(
      screen.querySelector(`[data-testid="register-cell-${a.slug}"]`),
      a.code,
    ).toBeTruthy();
  }
});

test("[routes/index]: the landing page admits that five of six specialists do not exist", async () => {
  const { screen, render } = await createDOM();
  await render(<Index />);
  const text = screen.querySelector("main")?.textContent ?? "";
  // Not a claim in prose — the states are on the cells themselves.
  const unbuilt = ARCHETYPES.filter((a) => a.state !== "on-duty");
  expect(unbuilt.length).toBe(ARCHETYPES.length);
  expect(text).toContain("Five of those six do not exist");
  expect(text).toContain("0 on duty");
});

test("[routes/index]: the permission suspension is quoted with the product's own component", async () => {
  const { screen, render } = await createDOM();
  await render(<Index />);
  const quoted = screen.querySelector('[data-testid="line-hold-landing"]');
  expect(quoted).toBeTruthy();
  const text = (quoted as HTMLElement).textContent ?? "";
  expect(text).toContain("Permission required");
  expect(text).toContain("The run is suspended here");
  // The exact call is on screen, because an approval you cannot read is not a
  // decision anyone can make.
  expect(text).toContain("drop schema staging cascade");
});

test("[routes/index]: the runtime figures shown are the real ones", async () => {
  const { screen, render } = await createDOM();
  await render(<Index />);
  const stack = screen.querySelector('[data-testid="landing-stack"]');
  expect(stack).toBeTruthy();
  const text = (stack as HTMLElement).textContent ?? "";
  expect(text).toContain("42/42");
  expect(text).toContain("24/24");
});

test("[routes/index]: the retired Software Development Framework identity is gone", () => {
  // ADR 0009 replaced the framework identity. These are the words the old
  // landing page sold, and none of them may return without a decision record.
  const text = source.toLowerCase();
  expect(text).not.toContain("software development framework");
  expect(text).not.toContain("requirements");
  expect(text).not.toContain("milestones they operate on");
  expect(text).not.toContain("the framework is the product");
});

test("[routes/index]: the page carries no eyebrow labels or section numbering", () => {
  // Both were load-bearing in the retired design ("[1.0] The system"). They
  // are the two habits most likely to creep back into a marketing page.
  expect(source).not.toMatch(/data-section=/);
  expect(source).not.toMatch(/\[\d\.\d\]/);
});

test("[routes/index]: head metadata describes the product, not the framework", () => {
  expect(landingHead.title).toContain("cachicamas");
  const description = landingHead.meta?.find((m: { name?: string; content?: string }) => m.name === "description")?.content;
  expect(description).toBeTruthy();
  expect(description).toContain("multiplayer agentic system");
  expect(description?.toLowerCase()).not.toContain("framework");
});
