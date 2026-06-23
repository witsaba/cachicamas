import { component$ } from "@builder.io/qwik";
import { routeLoader$, type DocumentHead } from "@builder.io/qwik-city";
import {
  OrganizationReadback,
  type OrganizationReadbackProps,
} from "~/components/organization-readback/organization-readback";

/**
 * /organizations/{id} — read-back of a single organization.
 *
 * Locked decision #5: `routeLoader$` calls the application
 * service directly.  No HTTP round-trip.  For tests, the
 * `OrganizationReadback` component renders with stubbed
 * data — see `routes/organizations/[id]/index.spec.tsx`.
 *
 * The TODO(organizations-first-front): UX-10 marker is
 * inside the readback component (the spec explicitly
 * requires the literal `TODO(organizations-first-front): UX-10`
 * string in the repo, see the grep test in
 * `src/__tests__/ux-todo.spec.ts`).
 */

export const useOrganizationLoader = routeLoader$(async (event) => {
  // TODO(organizations-first-front): call the in-process
  // application.OrganizationService.Get(id) once the
  // Qwik SSR + Go binary single-process wiring ships
  // (locked R-5 in design §10).  Until then, this loader
  // returns a deterministic placeholder so the route
  // renders and the readback component tests pass.
  const id = Number(event.params.id);
  return {
    id,
    full_name: `Organization #${id}`,
    identification: `org-${id}`,
  } satisfies OrganizationReadbackProps["organization"];
});

export default component$(() => {
  const org = useOrganizationLoader();
  return <OrganizationReadback organization={org.value} />;
});

export const head: DocumentHead = {
  title: "Organization · Cachicamas",
  meta: [
    {
      name: "description",
      content: "View a single organization in Cachicamas.",
    },
  ],
};
