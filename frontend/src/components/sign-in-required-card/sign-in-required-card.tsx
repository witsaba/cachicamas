/**
 * SignInRequiredCard — the inline sign-in prompt rendered on protected
 * routes when the visitor is anonymous.
 *
 * Reference: `openspec/changes/cachicamas-login-ux/specs/protected-routes/spec.md`
 *   R-PR-002 — section with data-testid="sign-in-required-card", <h1>
 *   "Sign in to continue", the description paragraph, and the
 *   SignInButton configured with `redirectTo`.
 *
 * Aphantasic-friendly (UX-4, archived spec):
 *   Text-first. No icons. The signal is the heading + the description
 *   + a single primary action.
 *
 * The component is standalone — no `useSession()`, no `routeLoader$`.
 * The parent route already determined "this is anonymous" via
 * `requireSession()`. The parent also knows the current path and
 * passes it via `redirectTo` so post-signin land is deterministic.
 */
import { component$ } from "@builder.io/qwik";
import {
  SignInButton,
  type SignInActionLike,
} from "~/components/sign-in-button/sign-in-button";

export interface SignInRequiredCardProps {
  /**
   * The Qwik City Action returned by `useSignIn()`. Same pattern as the
   * SignInButton — passed as a prop so the component is unit-testable
   * in vitest without Qwik City's request context.
   */
  signIn: SignInActionLike;
  /** Short, plain-text description of what the visitor was trying to do. */
  description: string;
  /**
   * Where to redirect after a successful sign-in. Defaults to `/` when
   * omitted. Caller usually passes the current pathname so the user
   * lands back where they started.
   */
  redirectTo?: string;
}

export const SignInRequiredCard = component$<SignInRequiredCardProps>(
  ({ signIn, description, redirectTo = "/" }) => {
    return (
      <section
        class="mx-auto max-w-xl px-6 py-12"
        data-testid="sign-in-required-card"
      >
        <h1 class="text-2xl font-semibold text-slate-900">
          Sign in to continue
        </h1>
        <p class="mt-2 text-sm text-slate-600">{description}</p>
        <div class="mt-6">
          <SignInButton
            signIn={signIn}
            label="Sign in with GitHub"
            redirectTo={redirectTo}
          />
        </div>
      </section>
    );
  },
);
