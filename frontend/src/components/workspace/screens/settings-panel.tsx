/**
 * Settings — your account, your company, and the one section that exists to
 * say what cannot be changed.
 *
 * Most settings screens are a list of switches. This one carries a section
 * called "Not yours to change", because a product that lets agents act inside
 * a company has to be explicit about which limits a customer is not allowed to
 * loosen — and a limit you cannot find is a limit nobody trusts.
 */
import { component$ } from "@builder.io/qwik";
import { Icon } from "~/components/icon/icon";
import { Button } from "~/components/ui/button/button";
import { PersonAvatar } from "~/components/workspace/avatar/avatar";
import {
  PAGE_WELL,
  PageHeader,
} from "~/components/workspace/page-header/page-header";
import { initialsOf } from "~/lib/initials";
import { COMPANY } from "~/lib/mock/company";

const CARD =
  "rounded-md border border-line bg-surface shadow-[var(--shadow-raised)]";
const ROW = "flex flex-wrap items-center gap-4 px-5 py-4";

export interface SettingsPanelProps {
  readonly user: {
    readonly name?: string | null;
    readonly email?: string | null;
    readonly image?: string | null;
  } | null;
}

const FIXED_LIMITS: readonly { title: string; detail: string }[] = [
  {
    title: "Nothing leaves the building unapproved",
    detail:
      "An email, a payment or a message to a customer always waits for a person. This cannot be switched off.",
  },
  {
    title: "One colleague owns your data's shape",
    detail:
      "Only the Database Administrator may change how your data is structured. Everyone else asks it.",
  },
  {
    title: "Every action is written down",
    detail:
      "Which colleague did what, which tool they used, and who approved it. Kept whether you look at it or not.",
  },
];

export const SettingsPanel = component$<SettingsPanelProps>(({ user }) => {
  const name = user?.name ?? "";
  const email = user?.email ?? "";

  return (
    <div class={PAGE_WELL}>
      <PageHeader
        title="Settings"
        lede="Your account, your company, and the limits that stay put."
      />

      <section
        class={CARD}
        aria-labelledby="account"
        data-testid="account-panel"
      >
        <h2
          id="account"
          class="border-line text-ink border-b px-5 py-4 text-base font-semibold"
        >
          Your account
        </h2>
        <div class={ROW}>
          <PersonAvatar
            name={name}
            initials={initialsOf(name, email)}
            image={user?.image ?? null}
            size="xl"
          />
          <div class="min-w-0 flex-1">
            <p
              class="text-ink text-lg font-semibold"
              data-testid="settings-name"
            >
              {name || "Not set"}
            </p>
            <p class="text-ink-soft text-base" data-testid="settings-email">
              {email || "Not set"}
            </p>
          </div>
          <Button as="a" href="/profile/" size="md" variant="secondary">
            Edit profile
          </Button>
        </div>
      </section>

      <section
        class={`${CARD} mt-4`}
        aria-labelledby="company"
        data-testid="company-panel"
      >
        <h2
          id="company"
          class="border-line text-ink border-b px-5 py-4 text-base font-semibold"
        >
          Your company
        </h2>
        <div class="divide-line divide-y">
          <div class={ROW}>
            <div class="min-w-[12rem] flex-1">
              <p class="text-ink text-base font-medium">Name</p>
              <p class="text-ink-soft text-sm">
                What your colleagues call this company.
              </p>
            </div>
            <p class="text-ink-mid text-base">{COMPANY.name}</p>
          </div>
          <div class={ROW}>
            <div class="min-w-[12rem] flex-1">
              <p class="text-ink text-base font-medium">Plan</p>
              <p class="text-ink-soft text-sm">
                Decides how many specialists you can have on staff.
              </p>
            </div>
            <p class="flex items-center gap-3">
              <span class="text-ink-mid text-base">{COMPANY.plan}</span>
              <Button as="a" href="/#plans" size="sm" variant="secondary">
                Change plan
              </Button>
            </p>
          </div>
          <div class={ROW}>
            <div class="min-w-[12rem] flex-1">
              <p class="text-ink text-base font-medium">People</p>
              <p class="text-ink-soft text-sm">
                Everyone who can talk to your colleagues.
              </p>
            </div>
            <p class="text-ink-mid text-base" data-numeric>
              {COMPANY.people}
            </p>
          </div>
        </div>
      </section>

      <section
        class={`${CARD} mt-4`}
        aria-labelledby="fixed"
        data-testid="fixed-limits-panel"
      >
        <h2
          id="fixed"
          class="border-line text-ink border-b px-5 py-4 text-base font-semibold"
        >
          Not yours to change
        </h2>
        <ul class="divide-line divide-y">
          {FIXED_LIMITS.map((limit) => (
            <li key={limit.title} class="flex gap-3 px-5 py-4">
              <Icon
                name="shield"
                size={18}
                class="text-ink-soft mt-0.5 shrink-0"
              />
              <span>
                <span class="text-ink block text-base font-medium">
                  {limit.title}
                </span>
                <span class="text-ink-soft block text-sm">{limit.detail}</span>
              </span>
            </li>
          ))}
        </ul>
      </section>
    </div>
  );
});
