import { $, component$, useSignal, useStore, type QRL } from "@builder.io/qwik";
import { Button } from "~/components/ui/button/button";
import { createOrganization } from "~/lib/api";
import {
  FORM_ERROR,
  FORM_INPUT,
  FORM_LABEL,
} from "~/components/os/form-classes";

/**
 * OwnboardingForm — the first-run setup form for the unique
 * organization in this install.
 *
 * Reference: `openspec/changes/2026-07-06-ownboarding/specs/ownboarding/spec.md`
 *   R-OW-002 (S-OW-010..016) — minimal field set (full_name + identification).
 *   R-OW-003 (S-OW-020..023) — submit + navigate to /home.
 *   R-OW-004 (S-OW-030..036) — error envelope mapping.
 *
 * Why a fork of OrganizationForm, not a reuse:
 *   - Decision #2 from the proposal locks minimum-viable fields
 *     (full_name + identification). Reusing OrganizationForm
 *     would render the shortname/email/phone inputs in the DOM
 *     and create ambiguity about whether they're required.
 *   - The deferred fields belong on a future settings page, not
 *     in a first-run gate.
 *
 * Aphantasic-friendly (UX-4): no <img>, no <picture>, no <svg>.
 *
 * Auto-derivation: typing in `full_name` auto-fills `identification`
 * with a slugified version (lowercase, hyphens, drop non-allowed
 * chars). The user can still override; once they type in
 * `identification` directly, auto-derivation stops.
 *
 * Submit button: disabled while `submitting.value === true` to
 * prevent double-submit (S-OW-022).
 *
 * Wire format: FormData keys are exactly `full_name` and
 * `identification` (S-OW-021). No other keys are sent.
 */
export type FormActionResult =
  | { ok: true; id: number }
  | { ok: false; field: "full_name" | "identification"; message: string }
  | { ok: false; field: "form"; message: string };

export type FormAction = QRL<(data: FormData) => Promise<FormActionResult>>;

/**
 * Optional success callback. Called with the new organization's
 * id when the action returns ok. The route uses this to call
 * `useNavigate()("/home")` so the form has no direct dependency
 * on the router context (testable in isolation via createDOM).
 */
export type OnSuccess = QRL<(id: number) => void>;

interface FieldErrors {
  fullName: string;
  identification: string;
}

interface FormState {
  fullName: string;
  identification: string;
  userOverrodeIdentification: boolean;
  fieldErrors: FieldErrors;
  topError: string;
  submitting: boolean;
}

/**
 * Slugify a free-form full_name into a URL-safe identification.
 * Mirrors the pattern from OrganizationForm: lowercase, replace
 * any character that is not [a-z0-9] with `-`, collapse consecutive
 * hyphens, trim leading/trailing hyphens.
 */
export function deriveIdentification(fullName: string): string {
  return fullName
    .toLowerCase()
    .normalize("NFKD")
    .replace(/[\u0300-\u036f]/g, "") // strip diacritics
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "")
    .slice(0, 60);
}

const MAX_NAME = 120;
const MAX_SLUG = 60;

export const OwnboardingForm = component$<{
  action: FormAction;
  onSuccess$?: OnSuccess;
}>(({ action, onSuccess$ }) => {
  const state = useStore<FormState>({
    fullName: "",
    identification: "",
    userOverrodeIdentification: false,
    fieldErrors: { fullName: "", identification: "" },
    topError: "",
    submitting: false,
  });
  const formRef = useSignal<HTMLFormElement>();

  const onFullNameInput = $((_event: InputEvent, el: HTMLInputElement) => {
    state.fullName = el.value;
    state.fieldErrors.fullName = "";
    state.topError = "";
    if (!state.userOverrodeIdentification) {
      state.identification = deriveIdentification(el.value);
    }
  });

  const onIdentificationInput = $(
    (_event: InputEvent, el: HTMLInputElement) => {
      state.identification = el.value;
      state.userOverrodeIdentification = true;
      state.fieldErrors.identification = "";
      state.topError = "";
    },
  );

  const onSubmit = $(async () => {
    // Default form submission is prevented via the
    // `preventdefault:submit` attribute on the <form> element.
    // Do NOT call `event.preventDefault()` here — qwik's
    // eslint rule disallows preventDefault in async handlers
    // (R-OW-003 / S-OW-022).
    // Re-validate client-side before invoking the action. The
    // server is the source of truth; this is just to surface
    // obvious problems without a round trip.
    const clientErrors = validateClient(state);
    if (clientErrors.fullName || clientErrors.identification) {
      state.fieldErrors = clientErrors;
      return;
    }
    state.submitting = true;
    state.topError = "";
    state.fieldErrors = { fullName: "", identification: "" };
    const fd = new FormData();
    fd.append("full_name", state.fullName);
    fd.append("identification", state.identification);
    const result = await action(fd);
    state.submitting = false;
    if (result.ok) {
      if (onSuccess$) await onSuccess$(result.id);
      return;
    }
    if (result.field === "full_name") {
      state.fieldErrors.fullName = result.message;
    } else if (result.field === "identification") {
      state.fieldErrors.identification = result.message;
    } else {
      state.topError = result.message;
    }
  });

  return (
    <form
      ref={formRef}
      preventdefault:submit
      onSubmit$={onSubmit}
      data-testid="ownboarding-form"
      class="mx-auto max-w-xl space-y-4"
    >
      {state.topError ? (
        <div
          role="alert"
          data-testid="ownboarding-top-error"
          class="border-fail bg-raise font-human text-body text-fail border px-3 py-2"
        >
          {state.topError}
        </div>
      ) : null}

      <div>
        <label for="fullName" class={FORM_LABEL}>
          What is your organization called?
        </label>
        <input
          id="fullName"
          name="full_name"
          type="text"
          required
          maxLength={MAX_NAME}
          value={state.fullName}
          onInput$={onFullNameInput}
          data-testid="ownboarding-full-name"
          class={FORM_INPUT}
          aria-describedby={
            state.fieldErrors.fullName ? "fullName-error" : undefined
          }
        />
        {state.fieldErrors.fullName ? (
          <p
            id="fullName-error"
            data-testid="ownboarding-full-name-error"
            class={FORM_ERROR}
          >
            {state.fieldErrors.fullName}
          </p>
        ) : null}
      </div>

      <div>
        <label for="identification" class={FORM_LABEL}>
          Pick a short identifier for your organization
        </label>
        <input
          id="identification"
          name="identification"
          type="text"
          required
          maxLength={MAX_SLUG}
          value={state.identification}
          onInput$={onIdentificationInput}
          data-testid="ownboarding-identification"
          class={FORM_INPUT}
          aria-describedby={
            state.fieldErrors.identification
              ? "identification-error"
              : undefined
          }
        />
        {state.fieldErrors.identification ? (
          <p
            id="identification-error"
            data-testid="ownboarding-identification-error"
            class={FORM_ERROR}
          >
            {state.fieldErrors.identification}
          </p>
        ) : null}
      </div>

      <Button
        type="submit"
        variant="primary"
        disabled={state.submitting}
        testId="ownboarding-submit"
      >
        {state.submitting ? "Setting up..." : "Create organization"}
      </Button>
    </form>
  );
});

/**
 * Pure client-side validation. Returns the per-field errors that
 * the form should display BEFORE invoking the action. The server
 * is still the source of truth — these checks exist only to avoid
 * a network round trip for obvious problems.
 */
function validateClient(state: FormState): FieldErrors {
  const errors: FieldErrors = { fullName: "", identification: "" };
  const fullName = state.fullName.trim();
  if (fullName.length === 0) {
    errors.fullName = "Name is required.";
  } else if (fullName.length < 3 || fullName.length > MAX_NAME) {
    errors.fullName = "Name must be 3–120 characters.";
  }
  const identification = state.identification.trim();
  if (identification.length === 0) {
    errors.identification = "Identifier is required.";
  } else if (identification.length < 3 || identification.length > MAX_SLUG) {
    errors.identification =
      "Identifier must be 3–60 characters, lowercase letters, digits, and hyphens; must start and end with a letter or digit.";
  } else if (!/^[a-z0-9][a-z0-9-]{1,58}[a-z0-9]$/.test(identification)) {
    errors.identification =
      "Identifier must be 3–60 characters, lowercase letters, digits, and hyphens; must start and end with a letter or digit.";
  }
  return errors;
}

// Re-export createOrganization so the route's submit action can
// import it without an additional file hop.
export { createOrganization };
