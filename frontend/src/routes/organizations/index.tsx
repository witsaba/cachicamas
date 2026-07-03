import { component$ } from "@builder.io/qwik";
import { routeLoader$, type DocumentHead } from "@builder.io/qwik-city";
import {
  OrganizationList,
  type OrganizationSummary,
} from "~/components/organization-list/organization-list";
import { listOrganizations } from "~/lib/api";

/**
 * /organizations — list or empty state.
 *
 * The `routeLoader$` proxies to the database_administrator Go
 * binary's `GET /organizations` endpoint.  Empty arrays are
 * preserved so the EmptyState still renders (spec F-2); transport
 * failures surface as `[]` plus a top-level banner so the
 * list/empty tests stay green and the user gets a hint instead
 * of a silent failure.
 *
 * For tests, see `routes/organizations/index.spec.tsx` — it
 * renders the presentational `OrganizationList` component
 * directly with stubbed data, sidestepping the loader plumbing.
 */

export const useOrganizationsLoader = routeLoader$(async () => {
  const result = await listOrganizations();
  if (result.ok) {
    const orgs: OrganizationSummary[] = result.value.map((o) => ({
      id: o.id,
      full_name: o.full_name,
      identification: o.identification,
    }));
    return { orgs };
  }
  return { orgs: [] as OrganizationSummary[], error: result.message };
});

export default component$(() => {
  const data = useOrganizationsLoader();
  return (
    <>
      {data.value.error && (
        <div
          role="alert"
          class="mx-auto max-w-2xl border-b border-red-300 bg-red-50 px-4 py-2 text-sm text-red-900"
          data-organization-list-error
        >
          {data.value.error}
        </div>
      )}
      <OrganizationList organizations={data.value.orgs} />
    </>
  );
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
