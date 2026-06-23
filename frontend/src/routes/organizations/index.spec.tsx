import { createDOM } from "@builder.io/qwik/testing";
import { test, expect } from "vitest";
import { OrganizationList } from "~/components/organization-list/organization-list";

// F-2 (spec §6.2) + UX-4 (spec §6.3): the /organizations route,
// when the loader returns an empty array, renders the empty state
// with an <h1>, one explanatory <p>, one CTA <a href="/organizations/new">,
// and zero <img> elements.
test("[routes/organizations]: empty state has h1, one p, one a, zero imgs", async () => {
  const { screen, render } = await createDOM();
  await render(<OrganizationList organizations={[]} />);

  const h1s = screen.querySelectorAll("h1");
  expect(h1s.length).toBe(1);
  expect(h1s[0].textContent).toContain("No organizations yet");

  const ps = screen.querySelectorAll("p");
  expect(ps.length).toBe(1);

  const ctas = screen.querySelectorAll('a[href="/organizations/new"]');
  expect(ctas.length).toBe(1);

  const imgs = screen.querySelectorAll("img");
  expect(imgs.length).toBe(0);
});

// F-3 (spec §6.2): the populated list renders one anchor per
// organization linking to /organizations/{id}, plus a single
// "Create another" CTA pointing to /organizations/new.
test("[routes/organizations]: populated list has one anchor per org plus a Create another CTA", async () => {
  const { screen, render } = await createDOM();
  await render(
    <OrganizationList
      organizations={[
        { id: 1, full_name: "Acme", identification: "acme" },
        { id: 2, full_name: "Beta", identification: "beta" },
      ]}
    />,
  );

  const anchors = screen.querySelectorAll("a");
  const hrefs = Array.from(anchors).map((a) => a.getAttribute("href") ?? "");

  expect(hrefs).toContain("/organizations/1");
  expect(hrefs).toContain("/organizations/2");
  expect(hrefs).toContain("/organizations/new");

  const imgs = screen.querySelectorAll("img");
  expect(imgs.length).toBe(0);
});

test("[routes/organizations]: populated list shows the full_name and identification for each org", async () => {
  const { screen, render } = await createDOM();
  await render(
    <OrganizationList
      organizations={[
        { id: 1, full_name: "Acme Industrial S.A.", identification: "acme" },
        { id: 2, full_name: "Beta Co.", identification: "beta" },
      ]}
    />,
  );

  const html = screen.outerHTML;
  expect(html).toContain("Acme Industrial S.A.");
  expect(html).toContain("acme");
  expect(html).toContain("Beta Co.");
  expect(html).toContain("beta");
});
