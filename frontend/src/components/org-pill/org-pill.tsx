/**
 * OrgPill — the header chrome that surfaces the current organization
 * context. Single-tenant model: NOT a switcher (there is only one
 * org). The pill is a passive display of the current org's
 * full_name + a monogram of its first letter; clicking it opens a
 * panel with the org's identification and a placeholder for the
 * future settings page.
 *
 * Reference: post-ownboarding-fixes change, R-FIX-002 (2026-07-06).
 * Caught in UAT of ownboarding PR3: a signed-in user landed on /home
 * after creating an org but could not tell which org they were in,
 * because the only chrome (header) showed only the user avatar.
 *
 * Aphantasic-friendly (UX-4, archived spec, amended 2026-07-04):
 *   - The monogram is functional identity (a single initial), not
 *     decorative imagery, so it is allowed under UX-4's "no imagery"
 *     carve-out for identity affordances (same carve-out as the user
 *     avatar image in AvatarDropdown).
 *   - The pill and panel are text-first otherwise.
 *   - The empty-state pill ("No organization yet") uses a single "?"
 *     placeholder, again functional, not decorative.
 *
 * Pattern: mirrors AvatarDropdown's use of `useClickOutside` for
 * open/close state. The `forceOpen` prop is the documented test
 * escape hatch for the panel assertions (the vitest spec renders
 * the panel unconditionally without driving QRL click handlers,
 * which createDOM cannot fire synchronously).
 *
 * Data loading: `useResource$` fetches GET /organization from
 * `~/lib/api#getCurrentOrganization` on first render. The fetch
 * runs both during SSR (so the initial server-rendered HTML
 * already includes the pill data) and on the client when the
 * component resumes (refetch after navigation). Optimistic
 * fallback on fetch failure: treat the same as 404 (no org yet).
 */
import { component$, useSignal, useTask$ } from "@builder.io/qwik";
import { useLocation } from "@builder.io/qwik-city";
import { useClickOutside } from "~/components/avatar-dropdown/use-click-outside";
import { getCurrentOrganization, type OrganizationReadModel } from "~/lib/api";

export interface OrgPillProps {
  /**
   * Test-only escape hatch. When `true`, the panel renders
   * unconditionally (the click-outside hook is bypassed). The
   * production consumer never sets this; it lives so the vitest
   * spec can assert the panel's content without driving QRL clicks.
   */
  forceOpen?: boolean;
  /**
   * Test-only escape hatch. When supplied, the component renders
   * the pill from this prop directly and skips the useResource$
   * fetch. The vitest spec uses this to assert on the rendered
   * output without standing up a fetch mock.
   *
   * Production code never sets this; the loader runs as usual.
   */
  organization?: OrganizationReadModel | null;
}

/**
 * The first letter of the org's full_name, uppercased. Falls back
 * to "?" for the empty state. Exposed as a pure helper so the spec
 * can exercise the edge cases (empty string, unicode, whitespace).
 */
export function orgMonogram(org: OrganizationReadModel | null): string {
  if (!org) return "?";
  const trimmed = org.full_name.trim();
  if (trimmed.length === 0) return "?";
  return trimmed.charAt(0).toUpperCase();
}

export const OrgPill = component$<OrgPillProps>(
  ({ forceOpen = false, organization: orgOverride }) => {
    // Qwik component$ props are exposed as the value directly (the
    // TypeScript prop type matches the runtime value). The signal
    // wrapping is an internal optimization that Qwik unwraps when
    // the prop is read inside the component$ body.
    const overrideValue = orgOverride;

    // useSignal holds the resolved org. Starts as undefined
    // (loading). useTask$ below populates it — runs on the server
    // during SSR AND on the client when the component resumes (the
    // task body re-runs because the `track` subscription fires
    // when the signal is set, triggering a re-render of the
    // component).
    //
    // Why useSignal + useTask$ instead of useResource$:
    //   useResource$ is supposed to be awaited during SSR, but in
    //   practice with this Qwik 1.20 build the resource sometimes
    //   serializes as unresolved (the SSR HTML shows the loading
    //   state forever — the client never receives a resolved value
    //   because the server-side resolution never made it into the
    //   serialized state). useTask$ populates a plain signal, which
    //   Qwik serializes into the SSR HTML the same way it does any
    //   other signal — no async-resource serialization layer to
    //   misbehave.
    const org = useSignal<OrganizationReadModel | null | undefined>(undefined);

    // Track the current URL so the task re-fires on SPA navigation.
    // The pill lives in the layout (which does NOT re-mount when the
    // route changes), so useTask$ would otherwise only run once on
    // the initial mount. Tracking `loc.url.pathname` makes the task
    // re-run after `nav('/home')` from OwnboardingForm's onSuccess$,
    // so the pill picks up the freshly-created org without a manual
    // refresh. (R-FIX-004, 2026-07-06 follow-up to R-FIX-002.)
    const loc = useLocation();
    // eslint-disable-next-line qwik/no-use-visible-task
    useTask$(async ({ track }) => {
      // track the override so a future change to the prop re-runs
      // the task. Track the URL so SPA navigation re-runs the task.
      track(() => overrideValue);
      track(() => loc.url.pathname);
      if (overrideValue !== undefined) {
        // Test escape hatch: skip the fetch entirely.
        org.value = overrideValue;
        return;
      }
      try {
        const fetched = await getCurrentOrganization();
        org.value = fetched;
      } catch {
        // Optimistic fallback: a transient backend hiccup must not
        // break the chrome. Treat as "no org yet" — the pill renders
        // its empty state.
        org.value = null;
      }
    });

    const organization: OrganizationReadModel | null | undefined = org.value;

    // While loading (first render, no override), render a neutral
    // pill that takes the same space so the chrome doesn't reflow.
    if (organization === undefined) {
      return (
        <div
          data-testid="org-pill-loading"
          aria-label="Loading organization"
          class="inline-flex cursor-default items-center gap-2 rounded-md border border-slate-200 bg-white px-3 py-1.5 text-sm font-medium text-slate-300"
        >
          <span
            aria-hidden="true"
            data-testid="org-pill-monogram"
            class="inline-flex h-6 w-6 items-center justify-center rounded bg-slate-200 text-xs font-semibold text-slate-400"
          >
            ·
          </span>
          <span>Loading…</span>
        </div>
      );
    }

    // Empty state: render a non-interactive pill. No button, no click
    // handler — the user has nothing to inspect, and the disabled
    // look makes it obvious why.
    if (organization === null) {
      return (
        <div
          data-testid="org-pill-empty"
          aria-label="No organization yet"
          class="inline-flex cursor-default items-center gap-2 rounded-md border border-slate-200 bg-slate-50 px-3 py-1.5 text-sm font-medium text-slate-400"
        >
          <span
            aria-hidden="true"
            data-testid="org-pill-monogram"
            class="inline-flex h-6 w-6 items-center justify-center rounded bg-slate-300 text-xs font-semibold text-slate-500"
          >
            ?
          </span>
          <span>No organization yet</span>
        </div>
      );
    }

    return (
      <OrgPillInteractive organization={organization} forceOpen={forceOpen} />
    );
  },
);

/**
 * The interactive variant of OrgPill — only rendered once the
 * org data is loaded (non-null). Splitting this out lets the
 * `useClickOutside` hook live inside a component$ that always has
 * the org available, so the JSX is straight-line.
 */
const OrgPillInteractive = component$<{
  organization: OrganizationReadModel;
  forceOpen: boolean;
}>(({ organization, forceOpen }) => {
  const { open, setOpen$, triggerRef, panelRef } = useClickOutside();
  const isOpen = forceOpen || open.value;
  const monogram = orgMonogram(organization);

  return (
    <div class="relative inline-block">
      <button
        ref={triggerRef}
        type="button"
        data-testid="org-pill"
        aria-label={`${organization.full_name} menu`}
        aria-haspopup="menu"
        aria-expanded={isOpen}
        onClick$={() => setOpen$(!open.value)}
        class="inline-flex cursor-pointer items-center gap-2 rounded-md border border-slate-200 bg-white px-3 py-1.5 text-sm font-medium text-slate-900 transition-[background-color,box-shadow] duration-150 hover:bg-slate-50 focus:outline-none focus-visible:ring-2 focus-visible:ring-indigo-500 active:translate-y-px"
      >
        <span
          aria-hidden="true"
          data-testid="org-pill-monogram"
          class="inline-flex h-6 w-6 items-center justify-center rounded bg-slate-900 text-xs font-semibold text-white"
        >
          {monogram}
        </span>
        <span data-testid="org-pill-name">{organization.full_name}</span>
        <span aria-hidden="true" class="text-slate-400">
          ▾
        </span>
      </button>

      {isOpen ? (
        <div
          ref={panelRef}
          role="menu"
          data-testid="org-pill-panel"
          class="absolute top-12 right-0 z-50 w-72 rounded-lg border border-slate-200 bg-white p-4 text-slate-900 shadow-lg ring-1 ring-black/5"
        >
          <p class="font-semibold" data-testid="org-pill-panel-name">
            {organization.full_name}
          </p>
          <p
            class="font-mono text-sm text-slate-500"
            data-testid="org-pill-panel-identification"
          >
            id: {organization.identification}
          </p>
          <hr class="my-3 border-slate-200" />
          {/*
            Settings link is a future-deferred affordance (the
            ownboarding PR2 form intentionally stripped shortname /
            email / phone). Rendered as a visually-disabled row so
            the empty slot is honest: the panel reserves space for
            the link rather than hiding it.
          */}
          <span
            role="menuitem"
            aria-disabled="true"
            data-testid="org-pill-panel-settings"
            class="block cursor-not-allowed rounded px-2 py-1.5 text-sm text-slate-400 select-none"
          >
            Settings (coming soon)
          </span>
        </div>
      ) : null}
    </div>
  );
});
