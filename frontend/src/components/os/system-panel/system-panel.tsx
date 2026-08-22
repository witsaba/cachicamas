import { component$ } from "@builder.io/qwik";

import { Field } from "~/components/os/field/field";
import { Panel } from "~/components/os/panel/panel";
import { ScreenTitle } from "~/components/os/screen/screen";
import { Button } from "~/components/ui/button/button";
import { RUNTIME } from "~/lib/mock/registry";

/**
 * SystemPanel — the smallest application in the product.
 *
 * It holds the two things a person actually needs from a system panel — who am
 * I signed in as, and how do I leave — plus an honest inventory of what this
 * install can and cannot be configured to do.
 *
 * That inventory is not filler. A settings screen with three toggles that do
 * nothing is worse than one that says plainly which decisions have not been
 * made yet, and where they will be made when they are.
 */

/** Settings that exist, and settings that deliberately do not. */
const UNDECIDED: readonly { label: string; note: string }[] = [
  {
    label: "Theme",
    note: "One. This system is dark because it is watched, not because it is fashionable.",
  },
  {
    label: "Model and provider",
    note: "Chosen by the backend. The browser never holds a key.",
  },
  {
    label: "Organizations",
    note: "One per install. Multi-tenancy is an open product decision.",
  },
  {
    label: "Notifications",
    note: "Undecided — there is nothing running to be notified about.",
  },
];

export interface SystemPanelProps {
  readonly user?: { name?: string | null; email?: string | null } | null;
}

export const SystemPanel = component$<SystemPanelProps>((props) => {
  const user = props.user;

  return (
    <main
      id="main"
      class="mx-auto w-full max-w-[1100px] flex-1 px-3 py-4 sm:px-4"
    >
      <ScreenTitle
        code="SYSTEM"
        title="System"
        lead="Who you are on this install, and what it can be configured to do."
      />

      <div class="mt-4 grid grid-cols-1 gap-3 lg:grid-cols-2">
        <Panel label="Account" note="F8" testId="account-panel">
          <Field label="Name">
            <span>{user?.name ?? "—"}</span>
          </Field>
          <Field label="Email">
            <span>{user?.email ?? "—"}</span>
          </Field>
          <Field label="Signed in with">
            <span class="text-fg-mid">GitHub</span>
          </Field>
          <div class="border-rule mt-4 flex flex-wrap gap-2 border-t pt-3">
            <Button as="a" href="/profile/" variant="secondary">
              Your profile
            </Button>
            <Button as="a" href="/auth/signout/" variant="destructive">
              Sign out
            </Button>
          </div>
        </Panel>

        <Panel label="Runtime" note="Read-only" testId="system-runtime-panel">
          {RUNTIME.map((l) => (
            <Field key={l.code} label={`${l.code} ${l.name}`}>
              <span class={l.state === "open" ? "text-amber" : "text-live"}>
                {l.stateWord} · {l.done}/{l.total}
              </span>
            </Field>
          ))}
          <p class="border-rule font-human text-data text-fg-mid mt-3 border-t pt-3 leading-relaxed">
            Nothing on this row is configurable from a browser. The stack is
            composed in exactly one place per archetype, and that place is the
            only one permitted to read the environment.
          </p>
        </Panel>

        <Panel
          label="Not configurable yet"
          note={`${UNDECIDED.length} open decisions`}
          testId="undecided-panel"
          class="lg:col-span-2"
        >
          <ul class="grid grid-cols-1 gap-3 md:grid-cols-2">
            {UNDECIDED.map((row) => (
              <li key={row.label} class="border-rule bg-raise border p-3">
                <p class="text-label text-fg-dim tracking-[0.14em] uppercase">
                  {row.label}
                </p>
                <p class="font-human text-data text-fg-mid mt-1 leading-snug">
                  {row.note}
                </p>
              </li>
            ))}
          </ul>
        </Panel>
      </div>
    </main>
  );
});
