import { $, component$ } from "@builder.io/qwik";
import { type DocumentHead, useNavigate } from "@builder.io/qwik-city";
import { ScreenTitle } from "~/components/os/screen/screen";
import { OwnboardingForm } from "~/components/ownboarding-form/ownboarding-form";
import type {
  FormAction,
  FormActionResult,
} from "~/components/ownboarding-form/ownboarding-form";
import { requireAuthRedirect } from "~/lib/require-auth-redirect";
import { createOrganization } from "~/lib/api";
import { useSession, useSignIn } from "~/routes/plugin@auth";

/**
 * /ownboarding — first-run setup form route.
 *
 * Reference: `openspec/changes/2026-07-06-ownboarding/specs/ownboarding/spec.md`
 *   R-OW-001 (S-OW-001..004) — authed-only with native redirect.
 *   R-OW-003 (S-OW-020) — successful submit navigates to /home.
 *
 * Native auth (ADR-0011): anonymous visitors are redirected to
 * `/auth/signin?callbackUrl=/ownboarding` via the requireAuthRedirect
 * onRequest handler. The component$ body is for the authed render
 * only; no inline SignInRequiredCard (matches the new pattern;
 * defence-in-depth is delegated to the route-level fallback).
 */
export { requireAuthRedirect as onRequest };

/**
 * Submit action for the ownboarding form. Maps the createOrganization
 * envelope to the FormActionResult discriminated union:
 *   - 201     → { ok: true, id }                            → navigate to /home
 *   - 400 with fields.identification → { field: "identification" }
 *   - 400 with fields.full_name     → { field: "full_name" }
 *   - 400 other (top-level)         → { field: "form" }
 *   - 409 conflict on slug          → { field: "identification" } (slug taken)
 *   - 5xx / network / not_found     → { field: "form" }
 */
const submitAction: FormAction = $(
  async (data: FormData): Promise<FormActionResult> => {
    const fullName = String(data.get("full_name") ?? "");
    const identification = String(data.get("identification") ?? "");

    const result = await createOrganization({ fullName, identification });

    if (result.ok) {
      return { ok: true as const, id: result.value.id };
    }
    if (result.kind === "validation") {
      const identificationMessage = result.fields.identification;
      if (identificationMessage) {
        return {
          ok: false as const,
          field: "identification" as const,
          message: identificationMessage,
        };
      }
      const fullNameMessage = result.fields.full_name;
      if (fullNameMessage) {
        return {
          ok: false as const,
          field: "full_name" as const,
          message: fullNameMessage,
        };
      }
      const firstField = Object.entries(result.fields)[0];
      return {
        ok: false as const,
        field: "form" as const,
        message: firstField
          ? `${firstField[0]}: ${firstField[1]}`
          : "Invalid form data.",
      };
    }
    if (result.kind === "conflict") {
      return {
        ok: false as const,
        field: "identification" as const,
        message: result.message,
      };
    }
    return {
      ok: false as const,
      field: "form" as const,
      message: result.message,
    };
  },
);

export default component$(() => {
  const nav = useNavigate();
  const session = useSession();
  const signIn = useSignIn();
  const guard = requireAuthRedirect; // unused at render — the onRequest
  // above already redirected anon visitors. We still need the
  // symbol in scope for tooling that statically analyses the
  // route export.
  void guard;
  const sessionSig = useSession();
  void sessionSig;
  // Defence-in-depth: if createDOM test or production runtime ever
  // renders an anon branch without the onRequest running, fall back
  // to a SignInRequiredCard-equivalent inline branch. Per ADR-0011
  // the canonical UX is the native /auth/signin page; we don't
  // re-render the inline card here, but we DO return null so a
  // misconfigured render path doesn't leak protected JSX.
  if (session.value === null) {
    void signIn;
    return null;
  }
  return (
    <main id="main" class="mx-auto w-full max-w-2xl flex-1 px-4 py-12">
      <ScreenTitle
        code="SETUP"
        title="Name the company"
        lead="Every specialist on the register works for exactly one organization. Name it once and the board is yours; you can change these details later from System."
      />
      <div class="mt-6">
        <OwnboardingForm
          action={submitAction}
          onSuccess$={$(async () => {
            await nav("/home/");
          })}
        />
      </div>
    </main>
  );
});

export const head: DocumentHead = {
  title: "Set up your organization \u2014 Cachicamas",
  meta: [
    {
      name: "description",
      content:
        "First-run setup form for the unique organization in this Cachicamas install.",
    },
  ],
};
