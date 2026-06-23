import { createDOM } from "@builder.io/qwik/testing";
import { test, expect } from "vitest";
import { EmptyState } from "./empty-state";

// UX-4 (locked from spec §6.3): the empty state is an
// instruction, not a mood board.  Exactly one <h1>, one
// explanatory <p>, one CTA <a>, and ZERO <img> elements.
// F-2 (spec §6.2): the <a> points to "/organizations/new".

test("[EmptyState Component]: renders exactly one h1, one p, and one a", async () => {
  const { screen, render } = await createDOM();
  await render(
    <EmptyState
      heading="No organizations yet"
      body="You haven't created any organizations yet. Create your first one to start tracking projects and milestones."
      ctaHref="/organizations/new"
      ctaLabel="Create your first organization"
    />,
  );

  const h1s = screen.querySelectorAll("h1");
  expect(h1s.length).toBe(1);
  expect(h1s[0].textContent).toContain("No organizations yet");

  const ps = screen.querySelectorAll("p");
  expect(ps.length).toBe(1);
  expect(ps[0].textContent?.length ?? 0).toBeGreaterThan(0);

  const anchors = screen.querySelectorAll("a");
  expect(anchors.length).toBe(1);
  expect(anchors[0].getAttribute("href")).toBe("/organizations/new");
  expect(anchors[0].textContent?.toLowerCase()).toContain("create");
});

test("[EmptyState Component]: renders zero img elements (UX-4 + F-2)", async () => {
  const { screen, render } = await createDOM();
  await render(
    <EmptyState
      heading="No organizations yet"
      body="You haven't created any organizations yet."
      ctaHref="/organizations/new"
      ctaLabel="Create your first organization"
    />,
  );
  const images = screen.querySelectorAll("img");
  expect(images.length).toBe(0);
});

test("[EmptyState Component]: CTA accessible name contains the ctaLabel", async () => {
  const { screen, render } = await createDOM();
  await render(
    <EmptyState
      heading="No projects yet"
      body="You haven't created any projects yet."
      ctaHref="/projects/new"
      ctaLabel="Create your first project"
    />,
  );
  const anchor = screen.querySelector("a") as HTMLAnchorElement;
  expect(anchor).not.toBeNull();
  expect(anchor.textContent).toContain("Create your first project");
  expect(anchor.getAttribute("href")).toBe("/projects/new");
});

test("[EmptyState Component]: heading is rendered as h1 (UX-4)", async () => {
  const { screen, render } = await createDOM();
  await render(
    <EmptyState
      heading="Nothing here yet"
      body="Get started by adding the first item."
      ctaHref="/items/new"
      ctaLabel="Add the first item"
    />,
  );
  const h1 = screen.querySelector("h1");
  expect(h1).not.toBeNull();
  expect(h1?.textContent).toBe("Nothing here yet");
});
