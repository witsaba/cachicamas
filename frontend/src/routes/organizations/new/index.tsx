import { $, component$ } from "@builder.io/qwik";
import { type DocumentHead, useNavigate } from "@builder.io/qwik-city";
import {
  OrganizationForm,
  type FormAction,
  type FormActionResult,
} from "~/components/organization-form/organization-form";
import { organizationInputSchema } from "~/lib/organization-schema";

/**
 * /organizations/new — create-organization form route.
 *
 * The form's `onSuccess$` hook is wired to `useNavigate()` so
 * the URL becomes the breadcrumb (UX-7) after a 201.
 *
 * SUBMIT ACTION (smart stub, dev mode)
 * ------------------------------------
 * Until the Qwik SSR + Go binary single-process wiring ships
 * (locked R-5 in design §10), this action stands in for the
 * real `application.OrganizationService.Create` call.  It
 * parses the form body with the same Zod schema the client
 * uses, so:
 *
 *   - Invalid payload  → 400 with the field error
 *   - Valid payload    → 201 with a deterministic fake id
 *
 * The 409 (slug conflict) path is intentionally NOT simulated
 * here; it will be exercised by the real backend the moment
 * R-5 lands.  The form component still handles a 409 result
 * type correctly (see the locked F-6b test) so the contract
 * is forward-compatible.
 *
 * For the form's own tests, see
 * `src/components/organization-form/organization-form.spec.tsx`.
 */

const submitAction: FormAction = $(
  async (data: FormData): Promise<FormActionResult> => {
    // Re-validate server-side.  Defence in depth: the client
    // already ran the same schema, but a malicious or buggy
    // client could bypass it.  The Go binary will own this
    // check post-R-5; for now we mirror it here so dev-mode
    // behaviour is honest.
    const raw = {
      fullName: String(data.get("full_name") ?? ""),
      identification: String(data.get("identification") ?? ""),
      shortName: String(data.get("shortname") ?? ""),
      email: String(data.get("email") ?? ""),
      phone: String(data.get("phone") ?? ""),
    };
    const parsed = organizationInputSchema.safeParse(raw);
    if (!parsed.success) {
      const first = parsed.error.issues[0];
      const field = String(first?.path[0] ?? "");
      const message = first?.message ?? "Invalid form data.";
      if (field === "identification") {
        return { ok: false, field: "identification", message };
      }
      return { ok: false, field: "form", message };
    }
    // Dev-mode success.  Deterministic-ish fake id: a small
    // positive integer so the URL `/organizations/{id}` is
    // human-readable.  Real ids are BIGSERIAL.
    const id =
      Math.abs(
        Array.from(parsed.data.fullName).reduce(
          (acc, ch) => (acc * 31 + ch.charCodeAt(0)) | 0,
          7,
        ),
      ) || 1;
    return { ok: true, id };
  },
);

export default component$(() => {
  const nav = useNavigate();
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
