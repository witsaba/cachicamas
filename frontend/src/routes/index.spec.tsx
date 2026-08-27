/**
 * Behavioural spec for `routes/index.tsx` — the public page.
 *
 * A Persuade surface, so the assertions are about whether it does its job and
 * whether it is honest doing it:
 *
 *   - the offer, the specialists and the primary action are all present;
 *   - the proof is the product's own moment (a colleague stopping to ask),
 *     not a claim about it;
 *   - every specialist named in the product appears, so the page cannot
 *     quietly advertise a roster the workspace does not have;
 *   - the pricing says, in words, that its figures are a preview;
 *   - nothing on the page mentions how any of it is built, and the retired
 *     Software Development Framework identity is gone.
 */
import { createDOM } from "@builder.io/qwik/testing";
import { $, type QRL } from "@builder.io/qwik";
import { test, expect, vi } from "vitest";
import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import Index, { head } from "./index";
import type { DocumentHeadValue } from "@builder.io/qwik-city";
import { AGENTS } from "~/lib/mock/staff";
import { PLANS } from "~/lib/mock/plans";

// `DocumentHead` is a union of a static value and a resolver function; these
// routes export the static form, so narrow once here rather than at every use.
const landingHead = head as DocumentHeadValue;

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
  fileURLToPath(new URL("./index.tsx", import.meta.url)),
  "utf8",
);

async function renderLanding() {
  const { screen, render } = await createDOM();
  await render(<Index />);
  return screen;
}

test("[routes/index]: leads with the offer and one primary action", async () => {
  const screen = await renderLanding();
  const h1 = screen.querySelector("h1");
  expect(h1?.textContent).toContain("Hire the specialists");
  // The action is a real submit into the sign-in flow, not a link to nowhere,
  // and it is the same affordance the rest of the product uses.
  const forms = screen.querySelectorAll('form[data-testid="sign-in-button"]');
  expect(forms.length).toBeGreaterThan(0);
  const first = forms[0] as HTMLElement;
  expect(
    first.querySelector('input[name="providerId"]')?.getAttribute("value"),
  ).toBe("github");
  expect(
    first.querySelector('input[name="redirectTo"]')?.getAttribute("value"),
  ).toBe("/home");
});

test("[routes/index]: opens on the product's own moment, mid-conversation", async () => {
  // The proof PLAYS (see components/marketing/hero-proof), so first paint is
  // the person's request with the colleague about to answer. What is asserted
  // here is that the page mounts the real thing, seeded and named; that the
  // exchange actually reaches the permission is asserted where it can be
  // driven a tick at a time, in hero-proof.spec.tsx.
  const screen = await renderLanding();
  const proof = screen.querySelector('[data-testid="hero-proof"]');
  expect(proof).toBeTruthy();
  const text = proof?.textContent ?? "";
  expect(text).toContain("Assistant");
  expect(text).toContain("Agent");
  expect(text).toContain("Order 4471 arrived damaged");
  expect(text).toContain("Nothing is sent until a person answers");
});

test("[routes/index]: names every specialist the product actually has", async () => {
  // A landing page listing a roster the workspace cannot deliver is the
  // cheapest lie available; this is the guard against it drifting.
  const screen = await renderLanding();
  const text = screen.textContent ?? "";
  for (const agent of AGENTS) {
    expect(text, agent.slug).toContain(agent.name);
  }
});

test("[routes/index]: shows every plan, including the one with open desks", async () => {
  const screen = await renderLanding();
  for (const plan of PLANS) {
    expect(
      screen.querySelector(`[data-testid="plan-${plan.slug}"]`),
      plan.slug,
    ).toBeTruthy();
  }
  const workforce = screen.querySelector('[data-testid="plan-workforce"]');
  const text = workforce?.textContent ?? "";
  expect(text).toContain("open desks");
  expect(text).toContain("paired duo");
});

test("[routes/index]: exactly one plan is recommended", () => {
  expect(PLANS.filter((p) => p.recommended)).toHaveLength(1);
});

test("[routes/index]: says its prices are a preview, in words", async () => {
  // Nobody should be able to read this page and believe they were quoted.
  const screen = await renderLanding();
  const note = screen.querySelector('[data-testid="pricing-disclaimer"]');
  expect(note).toBeTruthy();
  expect(note?.textContent).toMatch(/preview pricing/i);
  expect(note?.textContent).toMatch(/nothing here is a quote/i);
});

test("[routes/index]: offers both billing periods, annual by default", async () => {
  const screen = await renderLanding();
  const monthly = screen.querySelector('[data-testid="billing-monthly"]');
  const annual = screen.querySelector('[data-testid="billing-annual"]');
  expect(monthly).toBeTruthy();
  expect(annual).toBeTruthy();
  expect(annual?.getAttribute("aria-pressed")).toBe("true");
  expect(monthly?.getAttribute("aria-pressed")).toBe("false");
});

test("[routes/index]: mentions nothing about how the product is built", async () => {
  const screen = await renderLanding();
  const text = screen.textContent ?? "";
  expect(text).not.toMatch(
    /\b(archetype|runtime|MCP|schema|Layer [123]|SSE|Qwik|token|endpoint|milestone|ADR)\b/i,
  );
});

test("[routes/index]: the retired Framework identity is gone", () => {
  expect(source).not.toMatch(/Software Development Framework/i);
  expect(source).not.toMatch(/\bworkspaces?\b.*\bskills?\b/i);
});

test("[routes/index]: the document head sells the product, not the stack", () => {
  expect(landingHead.title).toContain("cachicamas");
  const description = (landingHead.meta ?? []).find(
    (m) => m.name === "description",
  );
  expect(description?.content).toBeTruthy();
  expect(description?.content).not.toMatch(/archetype|runtime|Qwik/i);
});

test("[routes/index]: has exactly one <main> and one <h1>", async () => {
  // Two h1s on a marketing page is the tell that a section was pasted in.
  const screen = await renderLanding();
  expect(screen.querySelectorAll("main").length).toBe(1);
  expect(screen.querySelectorAll("h1").length).toBe(1);
});
