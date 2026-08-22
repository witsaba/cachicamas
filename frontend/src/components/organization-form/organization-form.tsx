import {
  $,
  component$,
  useComputed$,
  useStore,
  type QRL,
} from "@builder.io/qwik";
import { Button } from "~/components/ui/button/button";
import {
  organizationInputSchema,
  type OrganizationInput,
} from "~/lib/organization-schema";
import {
  FORM_ERROR,
  FORM_FIELDSET,
  FORM_HELP,
  FORM_INPUT,
  FORM_LABEL,
  FORM_LEGEND,
} from "~/components/ui/form/classes";

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
  /** E.164 dial code with the leading `+`, e.g. "+1" or "+57". */
  phoneCountryCode: string;
  /** National number, digits only, no spaces or formatting. */
  phoneNational: string;
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
 * Curated country codes for the phone input.  Text-only on
 * purpose — no flag icons, no decorative imagery (aphantasia-
 * friendly constraint, UX-4).  Ordered by call volume: North
 * America first, then Latin America, then Europe, then a few
 * high-volume Asia-Pacific entries.  The dial code is the
 * E.164 prefix including the leading `+`.
 */
const COUNTRY_CODES = [
  { name: "United States", dial: "+1" },
  { name: "Canada", dial: "+1" },
  { name: "Mexico", dial: "+52" },
  { name: "Colombia", dial: "+57" },
  { name: "Peru", dial: "+51" },
  { name: "Chile", dial: "+56" },
  { name: "Argentina", dial: "+54" },
  { name: "Brazil", dial: "+55" },
  { name: "Spain", dial: "+34" },
  { name: "United Kingdom", dial: "+44" },
  { name: "Germany", dial: "+49" },
  { name: "France", dial: "+33" },
] as const;

const DEFAULT_DIAL = "+1";

/**
 * Parse a free-form phone paste into a {dialCode, national}
 * pair, falling back to the default dial code if the paste
 * has no `+` prefix.  Strips whitespace, parens, dashes —
 * only digits and a single leading `+` survive.  Exported
 * for unit testing.
 */
export function extractPhoneParts(
  text: string,
  fallbackDial: string = DEFAULT_DIAL,
): { dialCode: string; national: string } {
  const cleaned = text.replace(/[^\d+]/g, "");
  // Collapse multiple `+` to the first one and drop it from
  // the digit run (the `+` belongs to the dial code, not the
  // national number).
  const plusIdx = cleaned.indexOf("+");
  const digitsOnly = cleaned.replace(/\+/g, "");
  if (plusIdx < 0 || digitsOnly.length === 0) {
    return { dialCode: fallbackDial, national: digitsOnly };
  }
  // Match against the curated list of country codes first
  // (longest match wins so "+57" beats "+5").  This avoids
  // the false-positive where a 1-digit walk picks "+5"
  // before the real "+57" is even considered.
  const knownDials = COUNTRY_CODES.map((c) => c.dial).sort(
    (a, b) => b.length - a.length,
  );
  for (const dial of knownDials) {
    const dialDigits = dial.slice(1);
    if (digitsOnly.startsWith(dialDigits)) {
      return {
        dialCode: dial,
        national: digitsOnly.slice(dialDigits.length),
      };
    }
  }
  // Fallback: walk 1-3 digit dial codes for unknown regions.
  for (const len of [3, 2, 1] as const) {
    if (digitsOnly.length <= len + 4) continue;
    const candidateDial = `+${digitsOnly.slice(0, len)}`;
    if (/^[1-9]\d*$/.test(candidateDial.slice(1))) {
      return {
        dialCode: candidateDial,
        national: digitsOnly.slice(len),
      };
    }
  }
  // Last resort: first digit is the dial code.
  return {
    dialCode: `+${digitsOnly.slice(0, 1)}`,
    national: digitsOnly.slice(1),
  };
}

/**
 * Format a national number (digits only) for display in
 * the input: group digits in 3s from the right, separated
 * by single spaces.  Exported for unit testing.
 *
 *   "4155552671"   -> "415 555 2671"
 *   "1234"         -> "1 234"
 *   "1"            -> "1"
 *   ""             -> ""
 */
export function formatNational(national: string): string {
  return national.replace(/\B(?=(\d{3})+(?!\d))/g, " ");
}

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

/**
 * Validate a single field against its Zod schema. Returns
 * the first error message (matching the locked spec strings
 * in `lib/organization-schema.ts`) or undefined if valid.
 *
 * Used by the form's onBlur handlers to give the user
 * instant feedback as they move between fields, without
 * firing the full submit path.  This is the client-side
 * counterpart to the `submitAction` server-side re-validation
 * — both run the same Zod schema so the messages cannot
 * diverge.
 */
export function validateField(
  field: "fullName" | "identification" | "shortName" | "email" | "phone",
  value: string,
): string | undefined {
  const fieldSchema = organizationInputSchema.shape[field];
  const result = fieldSchema.safeParse(value);
  if (result.success) return undefined;
  return result.error.issues[0]?.message;
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
    phoneCountryCode: DEFAULT_DIAL,
    phoneNational: "",
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
          // Compose the E.164 phone from the country-code
          // selector and the digits-only national number.
          // Empty national number means "no phone" — the
          // Zod schema accepts `""` as a valid optional
          // value, and we must not send the bare dial code
          // (e.g. "+1") which would fail the regex.
          phone: state.phoneNational
            ? `${state.phoneCountryCode}${state.phoneNational}`
            : "",
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
          // Compose the E.164 phone from the country-code
          // selector and the digits-only national number.
          // Empty national number means "no phone" — see
          // the matching comment in the candidate builder
          // above.
          formData.append(
            "phone",
            state.phoneNational
              ? `${state.phoneCountryCode}${state.phoneNational}`
              : "",
          );
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
          class="border-stop/30 bg-stop/[0.05] text-stop rounded-md border px-3 py-2 text-base font-medium"
        >
          {state.serverErrorMessage}
        </div>
      )}

      <div>
        <label for="fullName" class={`mb-1 ${FORM_LABEL}`}>
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
          onBlur$={$(() => {
            // Instant feedback on blur.  If the user emptied
            // the required field we surface the locked
            // "Name is required." message immediately,
            // instead of waiting for submit.
            const error = validateField("fullName", state.fullName);
            state.fieldErrors = { ...state.fieldErrors, fullName: error };
          })}
          class={FORM_INPUT}
        />
        {state.fieldErrors.fullName && (
          <p class={FORM_ERROR} data-error="fullName">
            {state.fieldErrors.fullName}
          </p>
        )}
      </div>

      <div>
        <label for="identification" class={`mb-1 ${FORM_LABEL}`}>
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
          onBlur$={$(() => {
            // Instant feedback on blur.  Surfaces the
            // locked "Slug is required." or
            // "Slug must be 3–60 characters, …" message
            // as soon as the user leaves the field.
            const error = validateField("identification", state.identification);
            state.fieldErrors = {
              ...state.fieldErrors,
              identification: error,
            };
          })}
          class={FORM_INPUT}
        />
        <p class={FORM_HELP}>
          3 to 60 characters. Lowercase letters, digits, and hyphens. Must start
          and end with a letter or digit.
        </p>
        {state.fieldErrors.identification && (
          <p class={FORM_ERROR} data-error="identification">
            {state.fieldErrors.identification}
          </p>
        )}
        {state.conflictMessage && (
          <p class={FORM_ERROR} data-conflict-message="true">
            {state.conflictMessage}
          </p>
        )}
      </div>

      {!showReviewGroup.value && (
        <div>
          <Button
            type="button"
            variant="secondary"
            size="md"
            data-action="show-details"
            onClick$={$(() => {
              state.showDetails = true;
            })}
            class="px-3 py-1"
          >
            Add optional details
          </Button>
        </div>
      )}

      {showReviewGroup.value && (
        <fieldset data-review-group="true" class={`space-y-4 ${FORM_FIELDSET}`}>
          <legend class={FORM_LEGEND}>
            Review &amp; share contact details
          </legend>

          <div>
            <label for="shortName" class={`mb-1 ${FORM_LABEL}`}>
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
                // Instant feedback on blur.  Empty is valid
                // (the field is optional); we only flag
                // when the user typed something that fails
                // the schema (e.g. > 40 chars).
                const error = validateField("shortName", state.shortName);
                state.fieldErrors = {
                  ...state.fieldErrors,
                  shortName: error,
                };
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
              class={FORM_INPUT}
            />
            {state.fieldErrors.shortName && (
              <p class={FORM_ERROR} data-error="shortName">
                {state.fieldErrors.shortName}
              </p>
            )}
          </div>

          <div>
            <label for="email" class={`mb-1 ${FORM_LABEL}`}>
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
                // Instant feedback on blur.  Empty is valid;
                // non-empty must parse as an email.
                const error = validateField("email", state.email);
                state.fieldErrors = {
                  ...state.fieldErrors,
                  email: error,
                };
              })}
              onInput$={$((event: Event, el: HTMLInputElement) => {
                const v =
                  (event as unknown as { value?: string }).value ?? el.value;
                state.email = v;
                state.fieldErrors = { ...state.fieldErrors, email: undefined };
              })}
              class={FORM_INPUT}
            />
            {state.fieldErrors.email && (
              <p class={FORM_ERROR} data-error="email">
                {state.fieldErrors.email}
              </p>
            )}
          </div>

          <div>
            <label for="phone" class={`mb-1 ${FORM_LABEL}`}>
              How can we reach you by phone?
            </label>
            {/* Expert-UX phone input: country-code selector
                (text-only) + national-number input with
                format-as-you-type + live E.164 hint.  See
                the module-level helpers (extractPhoneParts,
                formatNational) for the contract. */}
            <div class="flex" data-phone-input>
              <select
                aria-label="Country code"
                value={state.phoneCountryCode}
                onChange$={$((event: Event, el: HTMLSelectElement) => {
                  state.phoneCountryCode = el.value;
                  if (state.fieldErrors.phone) {
                    state.fieldErrors = {
                      ...state.fieldErrors,
                      phone: undefined,
                    };
                  }
                })}
                class="border-line-control bg-surface text-ink mr-2 rounded-md border px-2.5 py-2 text-base"
                data-phone-country
              >
                {COUNTRY_CODES.map((c) => (
                  <option
                    key={c.dial}
                    value={c.dial}
                    selected={c.dial === state.phoneCountryCode}
                  >
                    {`${c.name} (${c.dial})`}
                  </option>
                ))}
              </select>
              <input
                id="phone"
                name="phone"
                type="tel"
                inputMode="tel"
                autoComplete="tel"
                placeholder="415 555 2671"
                value={formatNational(state.phoneNational)}
                aria-invalid={state.fieldErrors.phone ? "true" : undefined}
                onBlur$={$(() => {
                  state.detailsTouched = true;
                  // Validate the COMPOSED phone
                  // (dial + national).  Empty national is
                  // valid (phone is optional); non-empty
                  // must match E.164 once composed.
                  const composed = state.phoneNational
                    ? `${state.phoneCountryCode}${state.phoneNational}`
                    : "";
                  const error = validateField("phone", composed);
                  state.fieldErrors = {
                    ...state.fieldErrors,
                    phone: error,
                  };
                })}
                onInput$={$((event: Event, el: HTMLInputElement) => {
                  // Accept the user's keystrokes
                  // verbatim, then on paste (or after a
                  // short debounce) split into dial + national.
                  // The simple path here strips non-digits
                  // on every input so the user can't type
                  // letters or spaces — the display is
                  // formatted by formatNational(value).
                  const raw =
                    (event as unknown as { value?: string }).value ?? el.value;
                  const digits = raw.replace(/\D/g, "");
                  state.phoneNational = digits;
                  if (state.fieldErrors.phone) {
                    state.fieldErrors = {
                      ...state.fieldErrors,
                      phone: undefined,
                    };
                  }
                })}
                onPaste$={$((event: ClipboardEvent) => {
                  // Smart paste: if the user pastes a full
                  // international number like
                  // "+57 315 555 2671", split it into
                  // dial code + national and update both.
                  // The default onInput$ runs after and
                  // keeps the formatted display in sync.
                  const text = event.clipboardData?.getData("text") ?? "";
                  const { dialCode, national } = extractPhoneParts(
                    text,
                    state.phoneCountryCode,
                  );
                  state.phoneCountryCode = dialCode;
                  state.phoneNational = national;
                  if (state.fieldErrors.phone) {
                    state.fieldErrors = {
                      ...state.fieldErrors,
                      phone: undefined,
                    };
                  }
                })}
                class={`flex-1 ${FORM_INPUT}`}
                data-phone-national
              />
            </div>
            <p class="text-ink-soft mt-1 text-xs" data-phone-e164>
              E.164: {state.phoneCountryCode}{" "}
              {formatNational(state.phoneNational) || (
                <span class="text-ink-soft">{"<number>"}</span>
              )}
            </p>
            {state.fieldErrors.phone && (
              <p class={FORM_ERROR} data-error="phone">
                {state.fieldErrors.phone}
              </p>
            )}
          </div>
        </fieldset>
      )}

      <div>
        <Button type="submit" variant="primary" disabled={state.submitting}>
          {state.submitting ? "Creating organization…" : "Create organization"}
        </Button>
      </div>
    </form>
  );
});
