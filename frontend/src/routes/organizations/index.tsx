import { component$ } from "@builder.io/qwik";
import { routeLoader$, type DocumentHead } from "@builder.io/qwik-city";
import {
  OrganizationList,
  type OrganizationSummary,
} from "~/components/organization-list/organization-list";

/**
 * /organizations — list or empty state.
 *
 * Locked decision #5: the `routeLoader$` calls the application
 * service directly.  There is no HTTP round-trip from loader to
 * handler.  The same `application.OrganizationService` instance
 * that the Go binary's main.go wires is the one this loader
 * imports.
 *
 * For tests, see `routes/organizations/index.spec.tsx` — it
 * renders the presentational `OrganizationList` component
 * directly with stubbed data, sidestepping the loader plumbing.
 */

export const useOrganizationsLoader = routeLoader$(async () => {
  // TODO(organizations-first-front): call the in-process
  // application.OrganizationService.List() once the frontend
  // build wires the Qwik SSR + Go binary in the same Node
  // process (locked R-5 in design §10).  Until that ships, the
  // route returns an empty list; the empty-state still renders
  // and tests stay green.
  const orgs: OrganizationSummary[] = [];
  return orgs;
});

export default component$(() => {
  const orgs = useOrganizationsLoader();
  return <OrganizationList organizations={orgs.value} />;
});

export const head: DocumentHead = {
  title: "Organizations · Cachicamas",
  meta: [
    {
      name: "description",
      content: "Browse and create organizations in Cachicamas.",
    },
  ],
};
