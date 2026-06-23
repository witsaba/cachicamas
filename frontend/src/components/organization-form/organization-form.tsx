import {
  $,
  component$,
  useComputed$,
  useStore,
  type QRL,
} from "@builder.io/qwik";
import {
  organizationInputSchema,
  type OrganizationInput,
} from "~/lib/organization-schema";

/**
 * OrganizationForm — presentational + stateful form for the
 * create-organization flow.  It owns the `useStore` of form
 * state (per spec §5.2), the auto-derivation pipeline (per
 * spec §5.3), the progressive-disclosure threshold (per
 * spec §5.4), AND the client-side Zod validation that runs
 * before the action is invoked.
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

/**
 * Action callback.  Receives the form's FormData so the route
 * can parse and validate it server-side (defence in depth —
 * the client already ran Zod, but the server is the source of
 * truth once R-5 lands).
 */
export type FormAction = QRL<(data: FormData) => Promise<FormActionResult>>;

/** Optional navigation hook — called after a successful submit. */
export type OnSuccess = QRL<(id: number) => void>;

/** Per-field validation feedback. */
interface FieldErrors {
  fullName?: string;
  identification?: string;
  shortName?: string;
  email?: string;
  phone?: string;
}

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
  fieldErrors: FieldErrors;
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
    fieldErrors: {},
    submitting: false,
  });

  const showReviewGroup = useComputed$(
    () =>
      state.fullName.trim() !== "" &&
      state.identification.trim() !== "" &&
      (state.detailsTouched || state.showDetails),
  );

  return (
    <form
      preventdefault:submit
      onSubmit$={$(async () => {
        if (state.submitting) return;
        state.fieldErrors = {};
        state.conflictMessage = "";
        state.serverErrorMessage = "";

        const candidate: OrganizationInput = {
          fullName: state.fullName,
          identification: state.identification,
          shortName: state.shortName,
          email: state.email,
          phone: state.phone,
        };

        const parsed = organizationInputSchema.safeParse(candidate);
        if (!parsed.success) {
          const next: FieldErrors = {};
          for (const issue of parsed.error.issues) {
            const field = String(issue.path[0] ?? "") as keyof FieldErrors;
            if (field && !next[field]) {
              next[field] = issue.message;
            }
          }
          state.fieldErrors = next;
          return;
        }

        state.submitting = true;
        try {
          // Build FormData from the live state (NOT from the
          // form element).  linkedom (Qwik's test DOM) leaves
          // both `event.target` and Qwik's second handler arg
          // undefined for submit events, so the canonical
          // path of constructing FormData from the form fails
          // under vitest.  The state we hold is the canonical
          // source anyway (auto-derivation, debounced typing,
          // and progressive disclosure all flow through it),
          // so this is also the right behaviour in production.
          const formData = new FormData();
          formData.append("full_name", state.fullName);
          formData.append("identification", state.identification);
          formData.append("shortname", state.shortName);
          formData.append("email", state.email);
          formData.append("phone", state.phone);
          const result = await action(formData);
          if (result.ok) {
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
      noValidate
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
          aria-invalid={state.fieldErrors.fullName ? "true" : undefined}
          onInput$={$((event: Event, el: HTMLInputElement) => {
            const value =
              (event as unknown as { value?: string }).value ?? el.value;
            state.fullName = value;
            state.fieldErrors = { ...state.fieldErrors, fullName: undefined };
            if (state.userOverrodeIdentification) return;
            setTimeout(() => {
              if (state.userOverrodeIdentification) return;
              state.identification = deriveIdentification(state.fullName);
            }, DERIVATION_DEBOUNCE_MS);
          })}
          class="w-full rounded border border-slate-300 px-3 py-2 aria-invalid:border-red-500"
        />
        {state.fieldErrors.fullName && (
          <p class="mt-1 text-sm text-red-700" data-error="fullName">
            {state.fieldErrors.fullName}
          </p>
        )}
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
          aria-invalid={
            state.fieldErrors.identification || state.conflictMessage
              ? "true"
              : undefined
          }
          onInput$={$((event: Event, el: HTMLInputElement) => {
            const value =
              (event as unknown as { value?: string }).value ?? el.value;
            state.identification = value;
            state.userOverrodeIdentification = value !== "";
            state.fieldErrors = {
              ...state.fieldErrors,
              identification: undefined,
            };
            if (state.conflictMessage) {
              state.conflictMessage = "";
            }
          })}
          class="w-full rounded border border-slate-300 px-3 py-2 aria-invalid:border-red-500"
        />
        <p class="mt-1 text-sm text-slate-600">
          3 to 60 characters. Lowercase letters, digits, and hyphens. Must start
          and end with a letter or digit.
        </p>
        {state.fieldErrors.identification && (
          <p class="mt-1 text-sm text-red-700" data-error="identification">
            {state.fieldErrors.identification}
          </p>
        )}
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
              aria-invalid={state.fieldErrors.shortName ? "true" : undefined}
              onBlur$={$(() => {
                state.detailsTouched = true;
              })}
              onInput$={$((event: Event, el: HTMLInputElement) => {
                const v =
                  (event as unknown as { value?: string }).value ?? el.value;
                state.shortName = v;
                state.fieldErrors = {
                  ...state.fieldErrors,
                  shortName: undefined,
                };
              })}
              class="w-full rounded border border-slate-300 px-3 py-2 aria-invalid:border-red-500"
            />
            {state.fieldErrors.shortName && (
              <p class="mt-1 text-sm text-red-700" data-error="shortName">
                {state.fieldErrors.shortName}
              </p>
            )}
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
              aria-invalid={state.fieldErrors.email ? "true" : undefined}
              onBlur$={$(() => {
                state.detailsTouched = true;
              })}
              onInput$={$((event: Event, el: HTMLInputElement) => {
                const v =
                  (event as unknown as { value?: string }).value ?? el.value;
                state.email = v;
                state.fieldErrors = { ...state.fieldErrors, email: undefined };
              })}
              class="w-full rounded border border-slate-300 px-3 py-2 aria-invalid:border-red-500"
            />
            {state.fieldErrors.email && (
              <p class="mt-1 text-sm text-red-700" data-error="email">
                {state.fieldErrors.email}
              </p>
            )}
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
              aria-invalid={state.fieldErrors.phone ? "true" : undefined}
              onBlur$={$(() => {
                state.detailsTouched = true;
              })}
              onInput$={$((event: Event, el: HTMLInputElement) => {
                const v =
                  (event as unknown as { value?: string }).value ?? el.value;
                state.phone = v;
                state.fieldErrors = { ...state.fieldErrors, phone: undefined };
              })}
              class="w-full rounded border border-slate-300 px-3 py-2 aria-invalid:border-red-500"
            />
            {state.fieldErrors.phone && (
              <p class="mt-1 text-sm text-red-700" data-error="phone">
                {state.fieldErrors.phone}
              </p>
            )}
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
