import {
  component$,
  Slot,
  useSignal,
  useTask$,
  useVisibleTask$,
} from "@builder.io/qwik";
import { getCurrentOrganization } from "~/lib/api";

/**
 * StatusRail — the one persistent band across the top of the system.
 *
 * A dealing-room board tells you four things before you read anything else:
 * whose board this is, whose account you are on, whether it is live, and what
 * time it is. That is exactly this rail, in that order, and it never changes
 * between screens.
 *
 * The clock is real and client-only. It is rendered as an empty, reserved slot
 * on the server so the band does not reflow when it arrives, and it is the one
 * piece of live data in the whole mockup — everything else is demonstration
 * data, and the rail says so.
 */
export interface StatusRailProps {
  /**
   * Test-only override. Production leaves this unset and the rail reads the
   * install's single organization itself, so the band is identical on every
   * screen without every route having to fetch and thread it through.
   */
  readonly org?: string | null;
  /** Shown when the interface is running on demonstration data. */
  readonly demo?: boolean;
  /**
   * An anonymous visitor has no organization context to surface, so the rail
   * drops the org reading and the demo marker entirely rather than rendering
   * an empty state that means nothing to them.
   */
  readonly authenticated?: boolean;
  /** Where the wordmark goes. The desk when signed in, the landing when not. */
  readonly brandHref?: string;
}

export const StatusRail = component$<StatusRailProps>((props) => {
  const clock = useSignal("");
  const org = useSignal<string | null>(props.org ?? null);

  useTask$(async () => {
    if (props.org !== undefined || props.authenticated === false) return;
    try {
      const current = await getCurrentOrganization();
      org.value = current?.full_name ?? null;
    } catch {
      // No backend reachable in the mockup. The rail says "No organization"
      // rather than inventing a company name.
      org.value = null;
    }
  });

  // A wall clock has no server-side value: rendering one during SSR would ship
  // a stale time and then correct it, which is worse than arriving a frame late.
  // eslint-disable-next-line qwik/no-use-visible-task
  useVisibleTask$(({ cleanup }) => {
    if (typeof window === "undefined") return;
    const tick = () => {
      const d = new Date();
      const pad = (n: number) => String(n).padStart(2, "0");
      clock.value = `${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`;
    };
    tick();
    const id = setInterval(tick, 1000);
    cleanup(() => clearInterval(id));
  });

  return (
    <header
      data-testid="status-rail"
      class="border-rule bg-panel flex items-center gap-3 border-b px-3 py-1.5"
    >
      <a
        href={props.brandHref ?? "/"}
        data-testid="status-rail-brand"
        class="text-label text-amber font-semibold tracking-[0.22em] uppercase no-underline"
      >
        cachicamas
      </a>

      {props.authenticated === false ? null : (
        <>
          <span aria-hidden="true" class="bg-rule hidden h-3 w-px sm:block" />

          <span
            data-testid="status-rail-org"
            class="text-legend text-fg-mid hidden truncate tracking-[0.14em] uppercase sm:inline"
          >
            {org.value && org.value.length > 0 ? org.value : "No organization"}
          </span>
        </>
      )}

      {/* Shown at every width. The organization name can fall off a phone
          without anyone being misled; the demonstration marker cannot. */}
      {props.demo ? (
        <span
          data-testid="status-rail-demo"
          title="Every figure on this board is demonstration data. No archetype is running."
          class="border-amber-dim text-legend text-amber border px-1.5 py-px tracking-[0.14em] whitespace-nowrap uppercase"
        >
          Demo data
        </span>
      ) : null}

      <span class="flex-1" />

      <span
        data-testid="status-rail-clock"
        class="text-legend text-fg-dim hidden tabular-nums sm:inline"
      >
        {clock.value}
      </span>

      <Slot />
    </header>
  );
});
