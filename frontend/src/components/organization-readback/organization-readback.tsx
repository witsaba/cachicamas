import { component$ } from "@builder.io/qwik";

/**
 * OrganizationReadback — presentational component used by
 * `/organizations/{id}` (F-7 + UX-7 + UX-8).
 *
 * The route file (`routes/organizations/[id]/index.tsx`)
 * is responsible for fetching the org via `routeLoader$`
 * and passing it as a prop.  This split keeps the markup
 * testable with a plain `createDOM()` render and no loader
 * plumbing.
 */

export interface OrganizationReadbackProps {
  organization: {
    id: number;
    full_name: string;
    identification: string;
  };
}

export const OrganizationReadback = component$<OrganizationReadbackProps>(
  ({ organization }) => {
    return (
      <main class="mx-auto max-w-2xl space-y-4 px-4 py-8">
        <h1 class="text-3xl font-bold text-slate-900">
          {organization.full_name}
        </h1>
        <p class="text-slate-700">
          <span class="font-semibold">Slug:</span>{" "}
          <code class="rounded bg-slate-100 px-1 py-0.5">
            {organization.identification}
          </code>
        </p>
        {/* UX-7: the URL is the breadcrumb.  We do NOT show
            a modal or toast on success; the URL change to
            /organizations/{id} IS the confirmation. */}
        {/* TODO(organizations-first-front): UX-10 — when AI
            pre-fill is implemented, this is where the
            suggested "First requirements" content will
            render.  See the UX-10 spec scenario. */}
        <div>
          <a
            href="#"
            class="inline-block rounded border border-slate-300 px-4 py-2 text-slate-700 underline"
            data-placeholder="create-first-project"
          >
            Create your first project
          </a>
        </div>
        <div>
          <a
            href="/organizations"
            class="inline-block text-slate-700 underline"
          >
            Back to organizations
          </a>
        </div>
      </main>
    );
  },
);
