import { component$ } from "@builder.io/qwik";
import { EmptyState } from "~/components/empty-state/empty-state";

/**
 * OrganizationList — presentational component used by
 * `/organizations` (F-2, F-3, UX-4).
 *
 * It is a pure component: it does NOT call the application
 * service itself.  The route file (`routes/organizations/index.tsx`)
 * is responsible for fetching the list via `routeLoader$` and
 * passing it as a prop.  This split keeps the markup testable
 * with a plain `createDOM()` render and no loader plumbing.
 */
export interface OrganizationSummary {
  id: number;
  full_name: string;
  identification: string;
}

export interface OrganizationListProps {
  organizations: OrganizationSummary[];
}

export const OrganizationList = component$<OrganizationListProps>(
  ({ organizations }) => {
    if (organizations.length === 0) {
      return (
        <EmptyState
          heading="No organizations yet"
          body="You haven't created any organizations yet. Create your first one to start tracking projects and milestones."
          ctaHref="/organizations/new"
          ctaLabel="Create your first organization"
        />
      );
    }

    return (
      <main class="mx-auto max-w-2xl px-4 py-8">
        <h1 class="text-3xl font-bold text-slate-900">Organizations</h1>
        <ul class="mt-4 space-y-2">
          {organizations.map((org) => (
            <li key={org.id}>
              <a
                href={`/organizations/${org.id}`}
                class="block rounded border border-slate-200 px-4 py-3 text-slate-900 underline"
              >
                <span class="font-semibold">{org.full_name}</span>
                <span class="ml-2 text-slate-500">({org.identification})</span>
              </a>
            </li>
          ))}
        </ul>
        <div class="mt-6">
          <a
            href="/organizations/new"
            class="inline-block rounded bg-slate-900 px-4 py-2 font-semibold text-white underline"
          >
            Create another
          </a>
        </div>
      </main>
    );
  },
);
