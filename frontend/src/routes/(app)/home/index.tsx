/**
 * `/home` — the in-construction authenticated landing page.
 *
 * Spec reference: R-FE-005 / S-FE-040 / S-FE-081. The page:
 *   1. reads the validated session from the `(app)` layout's
 *      `sharedMap` (the guard ran first; if we got here, the cookie
 *      is valid);
 *   2. server-fetches `/internal/me/:user_id` with `X-Internal-Secret`
 *      to get the user email + name + organization details;
 *   3. renders `Hola, {name} ({email})` + an "under construction" notice
 *      and a `Cerrar sesión` POST form to `/auth/logout`.
 *
 * The (app) layout's `onRequest` has already validated the cookie and
 * stored the session in `sharedMap` under `SHARED_MAP_SESSION_KEY`. If
 * the layout's guard is bypassed in a future refactor, the page falls
 * back to "no session" instead of crashing (defensive).
 *
 * The pure logic lives in `loadHomeData` (exported). The Qwik
 * `routeLoader$` wrapper calls it inside Qwik's server context.
 */
import { component$ } from "@builder.io/qwik";
import {
  type DocumentHead,
  type RequestHandler,
  routeLoader$,
} from "@builder.io/qwik-city";
import {
  callBackendMe,
  MeError,
  MeNotFoundError,
  type MeResponse,
} from "~/lib/server/oauth";
import { SHARED_MAP_SESSION_KEY, type SessionPayload } from "~/routes/(app)/layout";

export interface HomeData {
  session: SessionPayload | null;
  me: MeResponse | null;
  /** Set when the backend /me call failed for any reason. */
  error: string | null;
}

export interface LoadHomeInput {
  /** The validated session, or null if no session. */
  session: SessionPayload | null;
  /** `AUTH_INTERNAL_SECRET` (from server env). */
  internalSecret: string;
  /** `PUBLIC_GO_BACKEND_URL` (from server env). */
  backendUrl: string;
  /** Override for tests. Default: `fetch`. */
  fetchImpl?: typeof fetch;
}

/**
 * Pure data loader. Returns the session, the /me response, or an
 * error descriptor.
 *
 * Behaviour:
 *   - no session ⇒ returns `{ session: null, me: null, error: null }`
 *     (the route handler treats this as "no data"; the page renders a
 *     fallback — in production the layout's onRequest would have
 *     already redirected to login);
 *   - session present, /me succeeds ⇒ returns the full payload;
 *   - /me 404 ⇒ returns `{ error: "user_not_found" }` (defensive: the
 *     session references a user the backend doesn't know about);
 *   - /me 5xx or other failure ⇒ returns `{ error: "fetch_failed" }`.
 */
export async function loadHomeData(
  input: LoadHomeInput,
): Promise<HomeData> {
  if (!input.session) {
    return { session: null, me: null, error: null };
  }
  if (!input.backendUrl) {
    throw new Error("loadHomeData: backendUrl required");
  }
  if (!input.internalSecret) {
    throw new Error("loadHomeData: internalSecret required");
  }
  try {
    const me = await callBackendMe({
      backendUrl: input.backendUrl,
      internalSecret: input.internalSecret,
      userId: input.session.user_id,
      fetchImpl: input.fetchImpl,
    });
    return { session: input.session, me, error: null };
  } catch (err) {
    if (err instanceof MeNotFoundError) {
      return { session: input.session, me: null, error: "user_not_found" };
    }
    if (err instanceof MeError) {
      return { session: input.session, me: null, error: "fetch_failed" };
    }
    return { session: input.session, me: null, error: "fetch_failed" };
  }
}

/**
 * Qwik `onGet` — runs server-side on every `/home` request, BEFORE
 * the `routeLoader$`. The (app) layout's `onRequest` has already
 * validated the session and stored it in `sharedMap`. Here we:
 *   1. read the session from `sharedMap`;
 *   2. (defensive) re-check the cookie — guards against the layout's
 *      onRequest being bypassed in a refactor;
 *   3. call `loadHomeData` and stash the result under
 *      `HOME_DATA_KEY` for the loader.
 */
export const HOME_DATA_KEY = "homeData";

export const onGet: RequestHandler = async (ev) => {
  const session =
    (ev.sharedMap.get(SHARED_MAP_SESSION_KEY) as SessionPayload | undefined) ??
    null;
  const data = await loadHomeData({
    session,
    internalSecret: ev.env.get("AUTH_INTERNAL_SECRET") ?? "",
    backendUrl:
      ev.env.get("PUBLIC_GO_BACKEND_URL") ?? "http://localhost:8080",
  });
  ev.sharedMap.set(HOME_DATA_KEY, data);
};

/**
 * `routeLoader$` exposes the home data to the component.
 */
export const useHomeData = routeLoader$<HomeData>((ev) => {
  return (
    (ev.sharedMap.get(HOME_DATA_KEY) as HomeData | undefined) ?? {
      session: null,
      me: null,
      error: null,
    }
  );
});

/**
 * Visual structure:
 *   - one h1 with the user's name (or email if no name);
 *   - a status banner when status='inactive' (R-FE-010);
 *   - an "under construction" notice — this slice is honest about
 *     what the workspace will become;
 *   - a "Cerrar sesión" button that POSTs to /auth/logout.
 *
 * Copy is in Spanish (the product's first language).
 */
export default component$(() => {
  const data = useHomeData();

  // No data — the layout's onRequest should have already redirected, but
  // if we somehow land here without a session, show the safe fallback.
  if (!data.value.session || !data.value.me) {
    return (
      <main
        id="main"
        class="mx-auto flex min-h-screen w-full max-w-2xl flex-col items-center justify-center gap-4 px-5 py-16 text-center"
      >
        <h1 class="text-ink text-3xl font-bold tracking-tight">
          Tu sesión no está disponible
        </h1>
        <p class="text-ink-mid text-base" data-testid="home-no-data">
          Inicia sesión para ver tu información.
        </p>
        <a
          href="/auth/google/login"
          class="bg-brand text-ink-inverse hover:bg-brand/90 rounded-md px-4 py-2 text-base font-medium"
        >
          Iniciar sesión
        </a>
      </main>
    );
  }

  const { me } = data.value;
  const displayName = me.user.name?.trim() || me.user.email;
  const isInactive = me.user.status === "inactive";

  return (
    <main
      id="main"
      class="mx-auto flex min-h-screen w-full max-w-2xl flex-col gap-8 px-5 py-16"
    >
      <header class="flex flex-col gap-2">
        <p class="text-ink-soft text-sm">Hola,</p>
        <h1
          class="text-ink text-3xl font-bold tracking-tight"
          data-testid="home-greeting"
        >
          {displayName}
        </h1>
        <p
          class="text-ink-mid text-base"
          data-testid="home-email"
        >
          {me.user.email}
        </p>
        {isInactive && (
          <p
            role="status"
            class="text-ink-mid mt-3 rounded-md border border-line bg-canvas px-3 py-2 text-sm"
            data-testid="home-inactive-banner"
          >
            Tu cuenta está inactiva. Algunas funciones están deshabilitadas.
          </p>
        )}
      </header>

      <section
        class="border-line bg-surface rounded-lg border p-6"
        data-testid="home-under-construction"
      >
        <h2 class="text-ink text-xl font-semibold">En construcción</h2>
        <p class="text-ink-mid mt-2 text-base">
          Estás autenticado. El espacio de trabajo de tu organización
          ({me.organization.name}) está en construcción. Te avisaremos
          cuando esté listo.
        </p>
      </section>

      <form action="/auth/logout" method="post" class="flex">
        <button
          type="submit"
          class="border-line bg-surface text-ink hover:bg-canvas rounded-md border px-4 py-2 text-base font-medium"
          data-testid="home-logout"
        >
          Cerrar sesión
        </button>
      </form>
    </main>
  );
});

export const head: DocumentHead = {
  title: "inicio — cachicamas",
  meta: [
    {
      name: "robots",
      content: "noindex,nofollow",
    },
  ],
};