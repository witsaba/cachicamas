/**
 * `/settings` — account, company, and the limits that stay put.
 */
import { component$ } from "@builder.io/qwik";
import { type DocumentHead } from "@builder.io/qwik-city";

import { SettingsPanel } from "~/components/workspace/screens/settings-panel";
import { useSession } from "~/routes/plugin@auth";

export default component$(() => {
  const session = useSession();
  return <SettingsPanel user={session.value?.user ?? null} />;
});

export const head: DocumentHead = {
  title: "Settings — cachicamas",
};
