import { component$ } from "@builder.io/qwik";
import { routeLoader$, type DocumentHead } from "@builder.io/qwik-city";
import {
  OrganizationReadback,
  type OrganizationReadbackProps,
} from "~/components/organization-readback/organization-readback";
import { getOrganization } from "~/lib/api";

/**
 * /organizations/{id} — read-back of a single organization.
 *
 * The `routeLoader$` proxies `GET /organizations/{id}` on the
 * database_administrator Go binary.  A 404 surfaces an honest
 * "not found" alert instead of the previous fake `Organization
 * #N` placeholder; a transport failure surfaces the offline
 * message.
 *
 * For tests, see `routes/organizations/[id]/index.spec.tsx` —
 * it renders the `OrganizationReadback` component directly with
 * stubbed data, sidestepping the loader plumbing.
 *
 * The TODO(organizations-first-front): UX-10 marker is inside
 * the readback component (the spec explicitly requires the
 * literal `TODO(organizations-first-front): UX-10` string in
 * the repo, see the grep test in `src/__tests__/ux-todo.spec.ts`).
 */

type LoaderResult = {
  org: OrganizationReadbackProps["organization"] | null;
  error?: string;
};

export const useOrganizationLoader = routeLoader$<LoaderResult>(
  async (event) => {
    const id = Number(event.params.id);
    if (!Number.isFinite(id) || id <= 0) {
      return { org: null, error: "Organization not found." };
    }
    const result = await getOrganization(id);
    if (result.ok) {
      return {
        org: {
          id: result.value.id,
          full_name: result.value.full_name,
          identification: result.value.identification,
        },
      };
    }
    return { org: null, error: result.message };
  },
);

export default component$(() => {
  const data = useOrganizationLoader();
  if (data.value.error || !data.value.org) {
    const message = data.value.error ?? "Organization not found.";
    return (
      <div class="mx-auto max-w-2xl space-y-4 px-4 py-8">
        <h1 class="text-3xl font-bold text-slate-900">Organization</h1>
        <div
          role="alert"
          class="rounded border border-red-300 bg-red-50 px-4 py-2 text-red-900"
          data-organization-error
        >
          {message}
        </div>
        <a href="/organizations" class="inline-block text-slate-700 underline">
          Back to organizations
        </a>
      </div>
    );
  }
  return <OrganizationReadback organization={data.value.org} />;
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
