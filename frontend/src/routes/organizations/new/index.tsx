import { $, component$ } from "@builder.io/qwik";
import { type DocumentHead, useNavigate } from "@builder.io/qwik-city";
import {
  OrganizationForm,
  type FormAction,
  type FormActionResult,
} from "~/components/organization-form/organization-form";
import { SignInRequiredCard } from "~/components/sign-in-required-card/sign-in-required-card";
import { createOrganization } from "~/lib/api";
import { requireSession } from "~/lib/require-session";
import { useSession, useSignIn } from "~/routes/plugin@auth";

/**
 * /organizations/new — create-organization form route.
 *
 * The form's `onSuccess$` hook is wired to `useNavigate()` so the
 * URL becomes the breadcrumb (UX-7) after a 201.
 *
 * SUBMIT ACTION
 * -------------
 * The submit action proxies the form to the database_administrator
 * Go binary at `PUBLIC_API_BASE_URL` (default
 * `http://localhost:8080`).  It uses Locked #3 form-encoded
 * bodies — same field names the Qwik form already collects — so
 * the wire shape stays mechanical.
 *
 * The Go binary's locked error envelope
 * (`{ error, fields?, message? }`) is mapped to the form's
 * discriminated `FormActionResult`:
 *
 *   - 201 → { ok: true, id }                     → navigate (UX-7)
 *   - 400 validation → { ok: false, field: "form", message }
 *                                                   → top-level alert
 *   - 400 with identification field error
 *                       → { ok: false, field: "identification", message }
 *                                                   → inline slug error
 *   - 409 conflict    → { ok: false, field: "identification", message }
 *                                                   → inline conflict msg
 *   - 5xx or network → { ok: false, field: "form", message }
 *                                                   → top-level alert
 *
 * For the form's own tests, see
 * `src/components/organization-form/organization-form.spec.tsx`.
 */

const submitAction: FormAction = $(
  async (data: FormData): Promise<FormActionResult> => {
    const fullName = String(data.get("full_name") ?? "");
    const identification = String(data.get("identification") ?? "");
    const shortname = String(data.get("shortname") ?? "");
    const email = String(data.get("email") ?? "");
    const phone = String(data.get("phone") ?? "");

    const result = await createOrganization({
      fullName,
      identification,
      ...(shortname ? { shortName: shortname } : {}),
      ...(email ? { email } : {}),
      ...(phone ? { phone } : {}),
    });

    if (result.ok) {
      return { ok: true, id: result.value.id };
    }

    if (result.kind === "validation") {
      // Find the first field-level message and either attach it
      // to the `identification` slot (which the form already
      // renders inline for slug conflicts) or surface it as a
      // generic form error.  Anything beyond identification is
      // a structural bug caught by client-side Zod first, so
      // the generic bucket is the safe default.
      const identificationMessage = result.fields.identification;
      if (identificationMessage) {
        return {
          ok: false,
          field: "identification",
          message: identificationMessage,
        };
      }
      const firstField = Object.entries(result.fields)[0];
      return {
        ok: false,
        field: "form",
        message: firstField
          ? `${firstField[0]}: ${firstField[1]}`
          : "Invalid form data.",
      };
    }

    if (result.kind === "conflict") {
      return { ok: false, field: "identification", message: result.message };
    }

    // server / offline / not_found — anything we can't map onto
    // a field bubbles up as a top-level form alert.
    return { ok: false, field: "form", message: result.message };
  },
);

export default component$(() => {
  const nav = useNavigate();
  const session = useSession();
  const signIn = useSignIn();
  const guard = requireSession(session.value, "/organizations/new");
  if (guard.kind === "anon") {
    return (
      <SignInRequiredCard
        signIn={signIn}
        description="Sign in to configure a new organization."
        redirectTo={guard.pathname}
      />
    );
  }
  return (
    <OrganizationForm
      action={submitAction}
      onSuccess$={$((id: number) => {
        // UX-7: URL is the breadcrumb.
        return nav(`/organizations/${id}`);
      })}
    />
  );
});

export const head: DocumentHead = {
  title: "Create organization · Cachicamas",
  meta: [
    {
      name: "description",
      content: "Create a new organization in Cachicamas.",
    },
  ],
};
