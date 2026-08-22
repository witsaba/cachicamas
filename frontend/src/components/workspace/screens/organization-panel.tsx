/**
 * Organisation — who holds which seat.
 *
 * A company's officer roles, and the people in them. The seats are fixed and
 * the holders are not, which is why this is a list of roles with a person
 * beside each rather than a list of people with a title beside each: an empty
 * seat is information, and a list of people cannot show one.
 *
 * The controls are a mockup. Changing a holder does not persist yet, and the
 * screen says so once rather than disabling everything into a dead page.
 */
import { component$, useSignal } from "@builder.io/qwik";
import { Icon } from "~/components/icon/icon";
import { PersonAvatar } from "~/components/workspace/avatar/avatar";
import {
  PAGE_WELL,
  PageHeader,
} from "~/components/workspace/page-header/page-header";
import { COMPANY } from "~/lib/mock/company";
import { ORG_ROLES, PEOPLE } from "~/lib/mock/staff";

const CARD =
  "rounded-md border border-line bg-surface shadow-[var(--shadow-raised)]";

export interface OrganizationPanelProps {
  readonly name: string;
  readonly email: string;
}

export const OrganizationPanel = component$<OrganizationPanelProps>(
  ({ name, email }) => {
    // The signed-in person sits at the top of their own company's list.
    const you = {
      id: "you",
      name: name || email || "You",
      initials: (name || email || "?").slice(0, 2).toUpperCase(),
      title: null as string | null,
    };
    const roster = [you, ...PEOPLE];
    const holders = useSignal<Record<string, string>>(
      Object.fromEntries(ORG_ROLES.map((r) => [r.key, r.holder ?? ""])),
    );

    return (
      <div class={PAGE_WELL}>
        <PageHeader
          title="Organisation"
          lede={`Who answers for what at ${COMPANY.name}. An empty seat is worth seeing — it is usually the reason something keeps not getting decided.`}
        />

        <section class={CARD} aria-labelledby="officers">
          <div class="border-line flex items-center justify-between gap-3 border-b px-5 py-4">
            <h2 id="officers" class="text-ink text-base font-semibold">
              Officer roles
            </h2>
            <span class="text-ink-soft text-xs">
              {ORG_ROLES.filter((r) => holders.value[r.key]).length} of{" "}
              {ORG_ROLES.length} filled
            </span>
          </div>

          <ul class="divide-line divide-y">
            {ORG_ROLES.map((role) => {
              const holderId = holders.value[role.key];
              const holder = roster.find((p) => p.id === holderId);
              return (
                <li
                  key={role.key}
                  class="flex flex-wrap items-center gap-4 px-5 py-4"
                  data-testid={`role-${role.key}`}
                >
                  <div class="min-w-[13rem] flex-1">
                    <p class="text-ink text-base font-medium">{role.title}</p>
                    <p class="text-ink-soft text-sm">{role.responsibility}</p>
                  </div>

                  <div class="flex shrink-0 items-center gap-2.5">
                    {holder ? (
                      <PersonAvatar
                        name={holder.name}
                        initials={holder.initials}
                        size="md"
                      />
                    ) : (
                      <span
                        aria-hidden="true"
                        class="border-line-control text-ink-soft inline-flex h-8 w-8 items-center justify-center rounded-full border border-dashed"
                      >
                        <Icon name="plus" size={14} />
                      </span>
                    )}
                    <label class="sr-only" for={`holder-${role.key}`}>
                      Who holds {role.title}
                    </label>
                    <select
                      id={`holder-${role.key}`}
                      data-testid={`holder-${role.key}`}
                      value={holderId}
                      onChange$={(_, el) => {
                        holders.value = {
                          ...holders.value,
                          [role.key]: el.value,
                        };
                      }}
                      class="border-line-control bg-surface text-ink h-9 min-w-[12rem] cursor-pointer rounded-md border px-2.5 text-base"
                    >
                      <option value="">Nobody yet</option>
                      {roster.map((p) => (
                        <option key={p.id} value={p.id}>
                          {p.name}
                        </option>
                      ))}
                    </select>
                  </div>
                </li>
              );
            })}
          </ul>

          <p class="border-line text-ink-soft border-t px-5 py-3 text-xs">
            Changes here are not saved yet — this part of the workspace is a
            mockup.
          </p>
        </section>

        <section class={`${CARD} mt-4`} aria-labelledby="people">
          <h2
            id="people"
            class="border-line text-ink border-b px-5 py-4 text-base font-semibold"
          >
            People · {roster.length}
          </h2>
          <ul class="divide-line divide-y">
            {roster.map((p) => (
              <li key={p.id} class="flex items-center gap-3 px-5 py-3">
                <PersonAvatar name={p.name} initials={p.initials} size="md" />
                <span class="min-w-0 flex-1">
                  <span class="text-ink block truncate text-base font-medium">
                    {p.name}
                    {p.id === "you" ? (
                      <span class="text-ink-soft pl-2 text-xs font-normal">
                        you
                      </span>
                    ) : null}
                  </span>
                  <span class="text-ink-soft block truncate text-xs">
                    {ORG_ROLES.filter((r) => holders.value[r.key] === p.id)
                      .map((r) => r.title)
                      .join(" · ") || "No officer role"}
                  </span>
                </span>
                <span class="border-line bg-sunken text-2xs text-ink-soft shrink-0 rounded-sm border px-1.5 py-px font-semibold tracking-wide uppercase">
                  Person
                </span>
              </li>
            ))}
          </ul>
        </section>
      </div>
    );
  },
);
