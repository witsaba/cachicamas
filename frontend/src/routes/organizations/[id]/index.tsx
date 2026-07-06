import { component$ } from "@builder.io/qwik";
import {
  routeLoader$,
  type DocumentHead,
  useLocation,
} from "@builder.io/qwik-city";
import {
  OrganizationReadback,
  type OrganizationReadbackProps,
} from "~/components/organization-readback/organization-readback";
import { SignInRequiredCard } from "~/components/sign-in-required-card/sign-in-required-card";
import { getOrganization } from "~/lib/api";
import { requireAuthRedirect } from "~/lib/require-auth-redirect";
import { requireSession } from "~/lib/require-session";
import { useSession, useSignIn } from "~/routes/plugin@auth";

// 2026-07-06 native-auth-UI: anon visitors are redirected to the
// native /auth/signin page with `?callbackUrl=/organizations/{id}`
// (including any query string, e.g. `?tab=members`). The inline
// SignInRequiredCard below is defence-in-depth.
export { requireAuthRedirect as onRequest };

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
  // cachicamas-login-ux (R-PR-003): anon visitors see the sign-in card
  // BEFORE the loader hits the database_administrator API. Same pattern
  // as the list route — guard first, fetch second.
  const loc = useLocation();
  const session = useSession();
  const signIn = useSignIn();
  // The current path (e.g. /organizations/123) is read from the request
  // location so the card's `redirectTo` round-trips the user back to
  // this specific org after sign-in.
  const guard = requireSession(
    session.value,
    loc.url.pathname || "/organizations",
  );
  if (guard.kind === "anon") {
    return (
      <SignInRequiredCard
        signIn={signIn}
        description="Sign in to view this organization."
        redirectTo={guard.pathname}
      />
    );
  }
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
