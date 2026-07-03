import { component$ } from "@builder.io/qwik";
import { routeLoader$, type DocumentHead } from "@builder.io/qwik-city";
import {
  OrganizationReadback,
  type OrganizationReadbackProps,
} from "~/components/organization-readback/organization-readback";
import { getOrganization } from "~/lib/api";

/**
 * /organizations/{id} — read-back de una sola organización.
 *
 * Uses `routeLoader$` (server-side) so the data is fetched during SSR.
 * With the Node SSR adapter, this runs on every request. The browser
 * receives fully populated HTML (no client-side fetch needed for the
 * initial render). Dynamic IDs work because the server is in front —
 * there's no SSG prerender step that needs a fixed ID list.
 *
 * - 200 → renderiza el readback con la org.
 * - 404 → renderiza un alert "Organization not found" + "Back" link.
 * - Offline / error → renderiza un alert con el mensaje de error.
 * - id inválido (no numérico o <= 0) → alert inmediato, sin fetch.
 *
 * El TODO(organizations-first-front): UX-10 marker está adentro del
 * componente readback (ver el grep test en `src/__tests__/ux-todo.spec.ts`).
 */
export const useOrganizationLoader = routeLoader$<{
  org: OrganizationReadbackProps["organization"] | null;
  error?: string;
}>(async (event) => {
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
});

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
