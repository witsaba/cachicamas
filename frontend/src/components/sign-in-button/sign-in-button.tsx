/**
 * SignInButton — renders the canonical "Sign in" CTA for the GitHub provider.
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
 * Aphantasic-friendly (UX-4, amended 2026-07-04):
 *   The original UX-4 carve-out was "text-first, no decorative imagery,
 *   no icons that carry meaning". UAT feedback on 2026-07-04
 *   sharpened the principle: imagery that REQUIRES mental visualization
 *   (abstract illustrations, hero photos, decorative icons that the
 *   user has to picture) is still banned, but RECOGNIZABLE BRAND
 *   MARKS (the GitHub Octocat, a Google G, etc.) ARE aphantasia-
 *   friendly — they are visible visual anchors that substitute for
 *   mental imagery rather than requiring it. The component therefore
 *   renders an inline `<svg>` of the GitHub Octocat (CC-licensed per
 *   GitHub's logo usage guidelines) alongside a SHORT text label
 *   ("Sign in", not "Sign in with GitHub" — the brand mark carries
 *   the provider identification).
 *   See `openspec/specs/app-shell/spec.md` UX-4 glossary entry and
 *   R-AS-002 for the formal amendment.
 */
import { $, component$ } from "@builder.io/qwik";
import { Form, type ActionStore } from "@builder.io/qwik-city";
import { Button } from "~/components/ui/button/button";

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
  /**
   * Optional label override. Defaults to "Sign in" (short, per UX-4
   * amendment 2026-07-04 — the GitHub Octocat brand mark carries
   * the provider identification).
   */
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
  ({ signIn, label = "Sign in", redirectTo = "/profile" }) => {
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
        <Button
          type="submit"
          variant="primary"
          testId="sign-in-button"
          // Tailwind 4 + `not-disabled:hover:*` specificity trap:
          //   - The primary variant has `not-disabled:hover:bg-slate-700`,
          //     which compiles to `:not(:disabled):hover` — specificity
          //     (0,3,0) because `:not()` inherits the specificity of its
          //     argument plus the `:hover` pseudo-class.
          //   - A bare `hover:bg-zinc-800` override compiles to `:hover`
          //     only — specificity (0,2,0).
          //   - The variant WINS regardless of emission order, so the
          //     consumer must use `!important` to override.
          // Without `!` the SignInButton would render `hover:bg-slate-700`
          // (the variant's default), not the intended zinc-800. The
          // difference is subtle (both very dark) but the override loses.
          class="!hover:border-zinc-600 !hover:bg-zinc-800 !focus-visible:ring-zinc-500 border border-zinc-700 !bg-zinc-900 !text-zinc-100 shadow-sm hover:shadow-md"
        >
          {/*
GitHub Octocat brand mark — a recognizable visual anchor
for the OAuth provider (UX-4 amendment, 2026-07-04).
`aria-hidden` because the visible text label "Sign in"
already announces the affordance; the brand mark is for
sighted users as a quick recognition cue, not for screen
readers.  Rendered as inline <svg> (no external asset URL)
and styled with currentColor so it inherits the button's
foreground.  The path data is the official GitHub Mark
(https://github.com/logos, GitHub's logo usage guidelines
permit this use of the mark for sign-in affordances).
          */}
          <svg
            viewBox="0 0 24 24"
            aria-hidden="true"
            focusable="false"
            width="16"
            height="16"
            fill="currentColor"
            data-testid="sign-in-button-github-mark"
          >
            <path d="M12 .297c-6.63 0-12 5.373-12 12 0 5.303 3.438 9.8 8.205 11.385.6.113.82-.258.82-.577 0-.285-.01-1.04-.015-2.04-3.338.724-4.042-1.61-4.042-1.61C4.422 18.07 3.633 17.7 3.633 17.7c-1.087-.744.084-.729.084-.729 1.205.084 1.838 1.236 1.838 1.236 1.07 1.835 2.809 1.305 3.495.998.108-.776.417-1.305.76-1.605-2.665-.3-5.466-1.332-5.466-5.93 0-1.31.465-2.38 1.235-3.22-.135-.303-.54-1.523.105-3.176 0 0 1.005-.322 3.3 1.23.96-.267 1.98-.4 3-.405 1.02.005 2.04.138 3 .405 2.28-1.552 3.285-1.23 3.285-1.23.645 1.653.24 2.873.12 3.176.765.84 1.23 1.91 1.23 3.22 0 4.61-2.805 5.625-5.475 5.92.42.36.81 1.096.81 2.22 0 1.606-.015 2.896-.015 3.286 0 .315.21.69.825.57C20.565 22.092 24 17.592 24 12.297c0-6.627-5.373-12-12-12" />
          </svg>
          <span>{label}</span>
        </Button>
      </Form>
    );
  },
);
