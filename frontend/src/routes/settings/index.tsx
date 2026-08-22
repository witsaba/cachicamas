/**
 * `/settings` — System.
 *
 * The route owns the guard chain and the session; the screen is
 * `<SystemPanel>`, which takes the person and knows about neither.
 */
import { component$ } from "@builder.io/qwik";
import { type DocumentHead, type RequestHandler } from "@builder.io/qwik-city";

import { SystemPanel } from "~/components/os/system-panel/system-panel";
import { requireAuthRedirect } from "~/lib/require-auth-redirect";
import { requireOwnboarding } from "~/lib/require-ownboarding";
import { setSsrCookieHeader } from "~/lib/ssr-cookie-context";
import { useSession } from "~/routes/plugin@auth";

export const onRequest: RequestHandler = async (event) => {
  setSsrCookieHeader(event.request.headers.get("cookie") ?? "");
  requireAuthRedirect(event);
  await requireOwnboarding(event);
};

export default component$(() => {
  const session = useSession();
  return <SystemPanel user={session.value?.user ?? null} />;
});

export const head: DocumentHead = {
  title: "System — cachicamas",
};
