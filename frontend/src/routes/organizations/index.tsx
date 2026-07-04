import { component$ } from "@builder.io/qwik";
import { routeLoader$, type DocumentHead } from "@builder.io/qwik-city";
import {
  OrganizationList,
  type OrganizationSummary,
} from "~/components/organization-list/organization-list";
import { SignInRequiredCard } from "~/components/sign-in-required-card/sign-in-required-card";
import { listOrganizations } from "~/lib/api";
import { requireSession } from "~/lib/require-session";
import { useSession, useSignIn } from "~/routes/plugin@auth";

/**
 * /organizations — list or empty state.
 *
 * Uses `routeLoader$` (server-side) so the data is fetched during SSR.
 * With the Node SSR adapter, this runs on every request (not at build
 * time like the static SSG adapter). The browser receives fully
 * populated HTML and the client just hydrates.
 *
 * Empty arrays preserve the empty state (spec F-2). Transport failures
 * surface as [] plus a top-level alert with the offline or error message.
 *
 * For the Vitest tests (which render the presentational `OrganizationList`
 * component directly), see `routes/organizations/index.spec.tsx`. The
 * refactor does not affect those tests — the component itself is
 * unchanged.
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
  // cachicamas-login-ux (R-PR-003): gate anonymous visitors with the
  // inline SignInRequiredCard before the loader runs. We avoid touching
  // the database_administrator API on the anon path (less work for
  // anonymous traffic; aligned with the "landing is the only thing
  // visible when not logged in" intent).
  const session = useSession();
  const signIn = useSignIn();
  const guard = requireSession(session.value, "/organizations");
  if (guard.kind === "anon") {
    return (
      <SignInRequiredCard
        signIn={signIn}
        description="Sign in to browse organizations."
        redirectTo={guard.pathname}
      />
    );
  }
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
