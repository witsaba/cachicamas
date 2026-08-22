/**
 * SignInRequiredCard — the inline sign-in prompt rendered on protected
 * routes when the visitor is anonymous.
 *
 * Reference: `openspec/changes/cachicamas-login-ux/specs/protected-routes/spec.md`
 *   R-PR-002 — section with data-testid="sign-in-required-card", <h1>
 *   "Sign in to continue", the description paragraph, and the
 *   SignInButton configured with `redirectTo`.
 *
 * Aphantasic-friendly (UX-4, archived spec, amended 2026-07-04):
 *   Text-first. No imagery that requires mental visualization.
 *   Recognizable brand marks (the GitHub Octocat inline SVG inside
 *   the embedded SignInButton) are aphantasia-friendly visual
 *   anchors per the UX-4 amendment — see the SignInButton docstring
 *   and `openspec/specs/app-shell/spec.md` UX-4 entry.
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
        class="border-rule bg-panel border"
        data-testid="sign-in-required-card"
      >
        <header class="border-rule border-b px-3 py-1.5">
          <h1 class="text-label text-amber tracking-[0.14em] uppercase">
            Sign in to continue
          </h1>
        </header>
        <div class="p-4">
          <p class="font-human text-body text-fg-mid max-w-[60ch] leading-relaxed">
            {description}
          </p>
          <div class="mt-5">
            <SignInButton signIn={signIn} redirectTo={redirectTo} />
          </div>
        </div>
      </section>
    );
  },
);
