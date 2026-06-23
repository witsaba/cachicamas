import { $, component$ } from "@builder.io/qwik";
import { type DocumentHead, useNavigate } from "@builder.io/qwik-city";
import {
  OrganizationForm,
  type FormAction,
  type FormActionResult,
} from "~/components/organization-form/organization-form";

/**
 * /organizations/new — create-organization form route.
 *
 * Per locked decision #5, the `routeAction$` body calls the
 * application service directly (no HTTP round-trip from
 * action to handler).  For tests, see
 * `src/components/organization-form/organization-form.spec.tsx`
 * — the form component is testable in isolation with a
 * stubbed action.
 *
 * The form's `onSuccess$` hook is wired to `useNavigate()`
 * so the URL becomes the breadcrumb (UX-7) after a 201.
 */

const submitAction: FormAction = $(async (): Promise<FormActionResult> => {
  // TODO(organizations-first-front): parse the form-encoded
  // body via the shared Zod schema (src/lib/organization-schema.ts),
  // then call the in-process application.OrganizationService.Create.
  // Until the Qwik SSR + Go binary single-process wiring ships
  // (locked R-5 in design §10), this action returns the
  // locked 409 to exercise the inline error path in tests.
  // The form component tests cover the success + 409 branches
  // via stubbed actions.
  return {
    ok: false,
    field: "identification",
    message: "This slug is already taken. Try another.",
  };
});

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
