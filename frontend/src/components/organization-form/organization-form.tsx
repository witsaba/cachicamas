import {
  $,
  component$,
  useComputed$,
  useStore,
  type QRL,
} from "@builder.io/qwik";

/**
 * OrganizationForm — presentational + stateful form for the
 * create-organization flow.  It owns the `useStore` of form
 * state (per spec §5.2), the auto-derivation pipeline (per
 * spec §5.3), and the progressive-disclosure threshold (per
 * spec §5.4).  It is mounted by `routes/organizations/new/`,
 * which provides the `action` callback that talks to the
 * server-side `routeAction$`.
 *
 * The component is intentionally framework-coupled to Qwik so
 * the form is server-rendered with progressive enhancement.
 * No third-party form library.  No CSS-in-JS.  No class
 * library.  Tailwind v4 utility classes only.
 */

/** Outcome of a submit. */
export type FormActionResult =
  | { ok: true; id: number }
  | { ok: false; field: "identification"; message: string }
  | { ok: false; field: "form"; message: string };

/** Action callback (server-side routeAction$ wrapped). */
export type FormAction = QRL<() => Promise<FormActionResult>>;

/** Optional navigation hook — called after a successful submit. */
export type OnSuccess = QRL<(id: number) => void>;

interface FormState {
  fullName: string;
  identification: string;
  shortName: string;
  email: string;
  phone: string;
  userOverrodeIdentification: boolean;
  detailsTouched: boolean;
  showDetails: boolean;
  conflictMessage: string;
  serverErrorMessage: string;
  submitting: boolean;
}

const DERIVATION_DEBOUNCE_MS = 200;

/**
 * Auto-derivation pipeline (spec §5.3):
 *   1. Lowercase
 *   2. Replace `[^a-z0-9-]+` with `-`
 *   3. Collapse `-+` -> `-`
 *   4. Strip leading/trailing `-`
 *   5. Truncate to 60 chars; strip trailing `-` if needed
 *
 * Exported so the spec can lock the contract as a pure
 * function test (the live form integration is timing-sensitive
 * under Qwik's linkedom-based test renderer; the pure function
 * is the load-bearing rule).
 */
export function deriveIdentification(fullName: string): string {
  return fullName
    .toLowerCase()
    .replace(/[^a-z0-9-]+/g, "-")
    .replace(/-+/g, "-")
    .replace(/^-+/, "")
    .replace(/-+$/, "")
    .slice(0, 60)
    .replace(/-+$/, "");
}

export const OrganizationForm = component$<{
  action: FormAction;
  onSuccess$?: OnSuccess;
}>(({ action, onSuccess$ }) => {
  const state = useStore<FormState>({
    fullName: "",
    identification: "",
    shortName: "",
    email: "",
    phone: "",
    userOverrodeIdentification: false,
    detailsTouched: false,
    showDetails: false,
    conflictMessage: "",
    serverErrorMessage: "",
    submitting: false,
  });

  // Auto-derivation: re-derive identification whenever
  // fullName changes, but ONLY if the user has not manually
  // edited the slug field.  We debounce inline in the
  // onInput$ handler (using setTimeout) instead of a
  // useVisibleTask$ so the behaviour is testable in
  // jsdom-style environments where IntersectionObserver
  // does not fire.  The active timer ID is kept on the
  // useStore so QRL invocations can read/write it across
  // closures (a module-level `let` would be re-initialised
  // on every QRL re-invocation).

  // Progressive disclosure threshold (spec §5.4).
  // When the gate is false, the review <fieldset> is NOT in
  // the DOM — useComputed$ lets Qwik skip the subtree
  // entirely instead of collapsing or CSS-hiding it.
  const showReviewGroup = useComputed$(
    () =>
      state.fullName.trim() !== "" &&
      state.identification.trim() !== "" &&
      (state.detailsTouched || state.showDetails),
  );

  // Track server-side validation feedback so the form can
  // re-render after a submit completes.  Empty task body;
  // the real work happens in the submit handler below.
  // useTask$ kept removed — the previous useTask$ was a
  // no-op and is no longer needed after the inline
  // debounce refactor.

  return (
    <form
      preventdefault:submit
      onSubmit$={$(async () => {
        if (state.submitting) return;
        state.submitting = true;
        state.conflictMessage = "";
        state.serverErrorMessage = "";
        try {
          const result = await action();
          if (result.ok) {
            // Call the optional onSuccess$ hook with the
            // new id.  In production the route file wires
            // this to `useNavigate()`; in tests we wire it
            // to a stub that records the call.  We
            // deliberately do NOT use `history.pushState`
            // directly because linkedom (Qwik's test DOM)
            // does not expose a `history` global.
            if (onSuccess$) {
              await onSuccess$(result.id);
            }
          } else if (result.field === "identification") {
            state.conflictMessage = result.message;
          } else {
            state.serverErrorMessage = result.message;
          }
        } catch (err) {
          state.serverErrorMessage = `Something went wrong. Please try again. (${
            err instanceof Error ? err.message : "unknown error"
          })`;
        } finally {
          state.submitting = false;
        }
      })}
      method="post"
      class="mx-auto max-w-2xl space-y-4 px-4 py-8"
    >
      {state.serverErrorMessage && (
        <div
          role="alert"
          class="rounded border border-red-300 bg-red-50 px-4 py-2 text-red-900"
        >
          {state.serverErrorMessage}
        </div>
      )}

      <div>
        <label for="fullName" class="mb-1 block font-semibold text-slate-900">
          What is the full legal name of the organization?
        </label>
        <input
          id="fullName"
          name="full_name"
          type="text"
          required
          value={state.fullName}
          onInput$={$((event: Event, el: HTMLInputElement) => {
            // The Qwik testing userEvent helper passes a
            // synthetic event whose `.value` is the typed
            // text.  In a real browser, the element's
            // `.value` reflects the typed text.  We read
            // the event value as a fallback so tests can
            // drive the form via `userEvent(el, "input", { value: "..." })`.
            const value =
              (event as unknown as { value?: string }).value ?? el.value;
            state.fullName = value;
            // typing in the name field does NOT clear the
            // override flag; only manual edit of the slug
            // field does (below).  Debounce the derivation
            // by 200ms so the user can keep typing without
            // the slug flickering.
            if (state.userOverrodeIdentification) return;
            // We deliberately do NOT track the timer ID on
            // the useStore.  Qwik serialises the store on
            // each render; a Timeout object is not
            // serialisable.  Trade-off: if the user types
            // and unmounts mid-debounce, the pending timer
            // still fires (harmless — the store update is
            // a no-op because the component is gone).  In
            // production, page navigation is the only way
            // the component unmounts; the timer at most
            // resolves DERIVATION_DEBOUNCE_MS later.
            setTimeout(() => {
              if (state.userOverrodeIdentification) return;
              state.identification = deriveIdentification(state.fullName);
            }, DERIVATION_DEBOUNCE_MS);
          })}
          class="w-full rounded border border-slate-300 px-3 py-2"
        />
      </div>

      <div>
        <label
          for="identification"
          class="mb-1 block font-semibold text-slate-900"
        >
          Provide a short slug to identify this organization.
        </label>
        <input
          id="identification"
          name="identification"
          type="text"
          required
          value={state.identification}
          onInput$={$((event: Event, el: HTMLInputElement) => {
            const value =
              (event as unknown as { value?: string }).value ?? el.value;
            state.identification = value;
            state.userOverrodeIdentification = value !== "";
          })}
          class="w-full rounded border border-slate-300 px-3 py-2"
        />
        <p class="mt-1 text-sm text-slate-600">
          3 to 60 characters. Lowercase letters, digits, and hyphens. Must start
          and end with a letter or digit.
        </p>
        {state.conflictMessage && (
          <p class="mt-1 text-sm text-red-700" data-conflict-message="true">
            {state.conflictMessage}
          </p>
        )}
      </div>

      {!showReviewGroup.value && (
        <div>
          <button
            type="button"
            data-action="show-details"
            onClick$={$(() => {
              state.showDetails = true;
            })}
            class="rounded border border-slate-300 px-3 py-1 text-slate-700 underline"
          >
            Add optional details
          </button>
        </div>
      )}

      {showReviewGroup.value && (
        <fieldset
          data-review-group="true"
          class="space-y-4 rounded border border-slate-200 p-4"
        >
          <legend class="px-2 font-semibold text-slate-900">
            Review &amp; share contact details
          </legend>

          <div>
            <label
              for="shortName"
              class="mb-1 block font-semibold text-slate-900"
            >
              Tell us a short name to display.
            </label>
            <input
              id="shortName"
              name="shortname"
              type="text"
              value={state.shortName}
              onBlur$={$(() => {
                state.detailsTouched = true;
              })}
              onInput$={$((event: Event, el: HTMLInputElement) => {
                const v =
                  (event as unknown as { value?: string }).value ?? el.value;
                state.shortName = v;
              })}
              class="w-full rounded border border-slate-300 px-3 py-2"
            />
          </div>

          <div>
            <label for="email" class="mb-1 block font-semibold text-slate-900">
              Where can we reach you by email?
            </label>
            <input
              id="email"
              name="email"
              type="email"
              value={state.email}
              onBlur$={$(() => {
                state.detailsTouched = true;
              })}
              onInput$={$((event: Event, el: HTMLInputElement) => {
                const v =
                  (event as unknown as { value?: string }).value ?? el.value;
                state.email = v;
              })}
              class="w-full rounded border border-slate-300 px-3 py-2"
            />
          </div>

          <div>
            <label for="phone" class="mb-1 block font-semibold text-slate-900">
              How can we reach you by phone?
            </label>
            <input
              id="phone"
              name="phone"
              type="tel"
              value={state.phone}
              onBlur$={$(() => {
                state.detailsTouched = true;
              })}
              onInput$={$((event: Event, el: HTMLInputElement) => {
                const v =
                  (event as unknown as { value?: string }).value ?? el.value;
                state.phone = v;
              })}
              class="w-full rounded border border-slate-300 px-3 py-2"
            />
          </div>
        </fieldset>
      )}

      <div>
        <button
          type="submit"
          disabled={state.submitting}
          class="rounded bg-slate-900 px-4 py-2 font-semibold text-white underline disabled:opacity-50"
        >
          {state.submitting ? "Creating organization…" : "Create organization"}
        </button>
      </div>
    </form>
  );
});
