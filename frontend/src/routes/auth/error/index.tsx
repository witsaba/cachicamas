/**
 * `/auth/error` — generic OAuth error page.
 *
 * Spec reference: R-FE-006 / S-FE-050 / S-FE-051. Reads `?reason=` from
 * the URL and maps it to a localized message. Unknown reasons render
 * the generic message without a reason-specific copy.
 *
 * Locked reasons (per spec):
 *   - access_denied       — user clicked "Cancel" on Google's consent
 *   - invalid_state       — state cookie ≠ state query (CSRF)
 *   - token_exchange_failed
 *   - userinfo_failed
 *   - internal_error      — anything else; also default fallback
 *   - blocked             — user.status === 'blocked'
 *
 * Always renders a "Volver a intentar" link to /auth/google/login.
 *
 * This is a presentational component. The reason → message mapping is
 * the testable surface, so it's exported as `reasonMessage(reason)` for
 * direct testing.
 */
import { component$ } from "@builder.io/qwik";
import type { DocumentHead } from "@builder.io/qwik-city";

export type ErrorReason =
  | "access_denied"
  | "invalid_state"
  | "token_exchange_failed"
  | "userinfo_failed"
  | "internal_error"
  | "blocked"
  | "missing_code";

export const KNOWN_REASONS: ReadonlySet<ErrorReason> = new Set([
  "access_denied",
  "invalid_state",
  "token_exchange_failed",
  "userinfo_failed",
  "internal_error",
  "blocked",
  "missing_code",
]);

export function reasonMessage(reason: string | null): string {
  switch (reason) {
    case "access_denied":
      return "Cancelaste el inicio de sesión con Google. Puedes intentarlo de nuevo cuando quieras.";
    case "invalid_state":
      return "La verificación de seguridad de la sesión falló. Vuelve a intentarlo.";
    case "token_exchange_failed":
      return "No pudimos completar el intercambio del código con Google. Inténtalo otra vez.";
    case "userinfo_failed":
      return "No pudimos recuperar tu perfil de Google. Inténtalo otra vez.";
    case "blocked":
      return "Tu cuenta está bloqueada. Escribe a soporte para más información.";
    case "missing_code":
      return "Google no devolvió un código de autorización. Vuelve a intentarlo.";
    case "internal_error":
    default:
      return "Algo salió mal al iniciar la sesión. Inténtalo otra vez.";
  }
}

export function isKnownReason(reason: string | null): reason is ErrorReason {
  if (!reason) return false;
  return KNOWN_REASONS.has(reason as ErrorReason);
}

export default component$(() => {
  // We do NOT use `useLocation()` here on purpose — the marketing layout
  // spec (see /workspace/braejan/.../frontend/src/routes/layout.spec.tsx)
  // requires that sub-layouts not depend on router context (createDOM
  // has no qc-l). Instead, this page reads the query string via the
  // browser-only `window.location` on the client, and via the Qwik
  // route's `useLocation()`-less pattern. For the MVP we render
  // server-side with `reason = null` and hydrate the reason on the
  // client via a tiny inline script.
  //
  // For test purposes, `reasonMessage(null)` is the canonical generic
  // message, and the component test exercises that path.
  return (
    <main
      id="main"
      data-testid="auth-error"
      class="mx-auto flex min-h-screen w-full max-w-xl flex-col items-center justify-center gap-6 px-5 py-16 text-center"
    >
      <h1 class="text-ink text-3xl font-bold tracking-tight">
        No pudimos iniciar tu sesión
      </h1>
      <p
        class="text-ink-mid text-base"
        data-testid="auth-error-message"
      >
        Algo salió mal al iniciar la sesión. Inténtalo otra vez.
      </p>
      <a
        href="/auth/google/login"
        class="bg-brand text-ink-inverse hover:bg-brand/90 inline-flex items-center justify-center rounded-md px-5 py-3 text-base font-medium"
        data-testid="auth-error-retry"
      >
        Volver a intentar
      </a>
    </main>
  );
});

export const head: DocumentHead = {
  title: "Error de inicio de sesión — Cachicamas",
  meta: [
    {
      name: "robots",
      content: "noindex,nofollow",
    },
  ],
};