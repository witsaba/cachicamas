import { createDOM } from "@builder.io/qwik/testing";
import { test, expect } from "vitest";
import { OrganizationReadback } from "~/components/organization-readback/organization-readback";

// F-7 + UX-8 (spec §6.2 + §6.3): the /organizations/{id}
// route renders the org's full_name and identification,
// with zero <dialog> elements and zero <img> elements.

test("[routes/organizations/[id]]: renders the org's full_name and identification", async () => {
  const { screen, render } = await createDOM();
  await render(
    <OrganizationReadback
      organization={{
        id: 42,
        full_name: "Acme Industrial S.A.",
        identification: "acme-industrial",
      }}
    />,
  );

  const html = screen.outerHTML;
  expect(html).toContain("Acme Industrial S.A.");
  expect(html).toContain("acme-industrial");
});

test("[routes/organizations/[id]]: renders zero <dialog> elements (F-7 + UX-8)", async () => {
  const { screen, render } = await createDOM();
  await render(
    <OrganizationReadback
      organization={{
        id: 42,
        full_name: "Acme Industrial S.A.",
        identification: "acme-industrial",
      }}
    />,
  );
  const dialogs = screen.querySelectorAll("dialog");
  expect(dialogs.length).toBe(0);
});

test("[routes/organizations/[id]]: renders zero <img> elements (F-7)", async () => {
  const { screen, render } = await createDOM();
  await render(
    <OrganizationReadback
      organization={{
        id: 42,
        full_name: "Acme Industrial S.A.",
        identification: "acme-industrial",
      }}
    />,
  );
  const imgs = screen.querySelectorAll("img");
  expect(imgs.length).toBe(0);
});

test("[routes/organizations/[id]]: shows a 'Back to organizations' link to /organizations", async () => {
  const { screen, render } = await createDOM();
  await render(
    <OrganizationReadback
      organization={{
        id: 42,
        full_name: "Acme Industrial S.A.",
        identification: "acme-industrial",
      }}
    />,
  );
  const back = screen.querySelector('a[href="/organizations"]');
  expect(back).not.toBeNull();
  expect((back as HTMLAnchorElement).textContent ?? "").toContain(
    "Back to organizations",
  );
});
