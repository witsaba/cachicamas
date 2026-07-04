/**
 * SignInButton — renders the canonical "Sign in with GitHub" CTA.
 *
 * Reference: `openspec/changes/cachicamas-github-login/specs/frontend-auth/spec.md`
 *   R-FA-040 (S-FA-040..S-FA-046) — the landing CTA is a `<Form>` whose
 *   action is the Auth.js `useSignIn` action, with a hidden `providerId`
 *   field set to `"github"`. The button MUST be reachable from the
 *   landing page (`/`) without a logged-in session.
 *
 * Why `signIn` is a prop (not imported directly):
 *   Qwik City's `Action` types depend on a request context that is
 *   not always available in the vitest `createDOM()` harness (used by
 *   every other component spec in this project). Passing the action
 *   as a prop lets the parent route call `useSignIn()` in its own
 *   Qwik City request context, and lets the test inject a fake.
 *   The shape of the prop matches the Action's `submit` method, so
 *   any Qwik City Action is drop-in compatible.
 *
 * Aphantasic-friendly (UX-4, spec §6.2):
 *   Text-first. No icons that carry meaning. The "Sign in with GitHub"
 *   label is the entire visual content of the button.
 */
import { $, component$ } from "@builder.io/qwik";
import { Form, type ActionStore } from "@builder.io/qwik-city";

/**
 * Loose alias for the Qwik City Action shape. The `unknown` / `any`
 * generics let us accept `useSignIn()` (which returns a tightly-typed
 * ActionStore with provider_id / redirect_to / etc.) and any test
 * stub (which returns a wider ActionStore with FormData). The Form
 * component itself enforces the runtime contract via `action.submit`.
 */
export type SignInActionLike = ActionStore<any, any, any>;

/** Props for the SignInButton component. */
export interface SignInButtonProps {
  /**
   * The Qwik City Action returned by `useSignIn()`. Passed as a prop
   * so the component is testable in vitest without Qwik City's
   * request context.
   */
  signIn: SignInActionLike;
  /** Optional label override. Defaults to "Sign in with GitHub". */
  label?: string;
  /** Where to redirect after a successful sign-in. Default `/profile`. */
  redirectTo?: string;
}

/**
 * Renders a `<Form action={signIn}>` with a hidden `providerId` field
 * and a submit button. Clicking the button posts the form to the
 * Auth.js signIn action; Auth.js then redirects to the GitHub OAuth
 * consent screen.
 *
 * Strict TDD:
 *   T2.8 (RED) — the stub rendered only the label text (no form yet).
 *   T2.9 (GREEN) — this implementation. The test asserts the form
 *   shape, the hidden providerId, the label, and the redirect target.
 */
export const SignInButton = component$<SignInButtonProps>(
  ({ signIn, label = "Sign in with GitHub", redirectTo = "/profile" }) => {
    // Pre-build the form-data map so the Form action's submit sees the
    // hidden fields exactly. Auth.js looks up providerId from the form
    // data to pick which OAuth provider to start.
    const buildFormData = $((event: Event) => {
      const form = event.target as HTMLFormElement;
      const fd = new FormData(form);
      fd.set("providerId", "github");
      fd.set("redirectTo", redirectTo);
      return fd;
    });
    return (
      <Form
        action={signIn}
        data-testid="sign-in-button"
        onSubmit$={buildFormData}
      >
        <input type="hidden" name="providerId" value="github" />
        <input type="hidden" name="redirectTo" value={redirectTo} />
        <button
          type="submit"
          class="rounded-md border border-zinc-700 bg-zinc-900 px-4 py-2 text-sm font-medium text-zinc-100 hover:bg-zinc-800 focus:outline-none focus-visible:ring-2 focus-visible:ring-zinc-500"
        >
          {label}
        </button>
      </Form>
    );
  },
);
