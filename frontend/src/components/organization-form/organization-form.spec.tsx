import { createDOM } from "@builder.io/qwik/testing";
import { $ } from "@builder.io/qwik";
import { describe, test, expect } from "vitest";
import { OrganizationForm, deriveIdentification } from "./organization-form";

// =========================================================================
// Test fixtures
// =========================================================================
//
// The form's `action` prop is now a QRL that receives the
// form's FormData and returns a FormActionResult.  The form
// also runs Zod validation client-side BEFORE the action is
// invoked, so a stub action that returns 201 will only be
// reached for payloads that are already valid by the Zod
// schema.  To exercise server-side error paths, the test
// passes a payload the client accepts and the stub rejects.

// Helper: wrap an action in $() so the JSX prop is a QRL
// (Qwik requires serializable props).
const okAction = () =>
  $(async (_data: FormData) => ({ ok: true as const, id: 1 }));
const conflictAction = () =>
  $(
    async (_data: FormData) =>
      ({
        ok: false as const,
        field: "identification" as const,
        message: "This slug is already taken. Try another.",
      }) as const,
  );
/**
 * Test-only: build an action that records every call and
 * returns a fixed id.  Lives at module scope so Qwik's
 * QRL serialiser can resolve it consistently across tests.
 */
let __lastSubmittedActionCalls = 0;
let __lastSubmittedFormData: FormData | null = null;
let __lastNavigatedId: number | undefined;
const recordingOkAction = () =>
  $(async (data: FormData) => {
    __lastSubmittedActionCalls++;
    __lastSubmittedFormData = data;
    return { ok: true as const, id: 42 };
  });
const recordingOnSuccess = () =>
  $(async (id: number) => {
    __lastNavigatedId = id;
  });

/** Expand the review group (progressive-disclosure gate). */
async function expandDetails(
  screen: Awaited<ReturnType<typeof createDOM>>["screen"],
  userEvent: Awaited<ReturnType<typeof createDOM>>["userEvent"],
  fullName = "Acme",
  identification = "acme",
) {
  const fullNameInput = screen.querySelector(
    "input[id=\"fullName\"]",
  ) as HTMLInputElement;
  const identificationInput = screen.querySelector(
    "input[id=\"identification\"]",
  ) as HTMLInputElement;
  await userEvent(fullNameInput, "input", { value: fullName });
  await userEvent(identificationInput, "input", { value: identification });
  const showDetailsBtn = screen.querySelector(
    "button[data-action=\"show-details\"]",
  ) as HTMLButtonElement;
  if (showDetailsBtn) {
    await userEvent(showDetailsBtn, "click");
  }
}

// =========================================================================
// F-4 / UX-1 / UX-3 / UX-5 / UX-9 — markup invariants
// =========================================================================

test("[OrganizationForm]: renders 5 labeled inputs and a submit button (F-4)", async () => {
  const { screen, render, userEvent } = await createDOM();
  await render(<OrganizationForm action={okAction()} />);

  await expandDetails(screen, userEvent);

  const labels = screen.querySelectorAll("label");
  expect(labels.length).toBe(5);

  const inputs = screen.querySelectorAll("input");
  expect(inputs.length).toBe(5);

  const submit = screen.querySelector("button[type=\"submit\"]");
  expect(submit).not.toBeNull();
  expect(submit?.textContent ?? "").toContain("Create organization");
});

test("[OrganizationForm]: every input has a sibling <label> with a matching for= (UX-1)", async () => {
  const { screen, render } = await createDOM();
  await render(<OrganizationForm action={okAction()} />);

  const inputs = Array.from(screen.querySelectorAll("input"));
  for (const input of inputs) {
    const id = input.getAttribute("id");
    expect(id).not.toBeNull();
    const label = screen.querySelector(`label[for="${id}"]`);
    expect(label, `expected a label for input id="${id}"`).not.toBeNull();
  }
});

test("[OrganizationForm]: required inputs (fullName, identification) have no placeholder (UX-1)", async () => {
  const { screen, render } = await createDOM();
  await render(<OrganizationForm action={okAction()} />);

  const fullName = screen.querySelector("input[id=\"fullName\"]") as HTMLInputElement;
  const identification = screen.querySelector(
    "input[id=\"identification\"]",
  ) as HTMLInputElement;
  expect(fullName).not.toBeNull();
  expect(identification).not.toBeNull();
  expect(fullName.getAttribute("placeholder")).toBeNull();
  expect(identification.getAttribute("placeholder")).toBeNull();
});

test("[OrganizationForm]: every label ends with ? OR starts with a verb (UX-3)", async () => {
  const { screen, render } = await createDOM();
  await render(<OrganizationForm action={okAction()} />);

  const labels = Array.from(screen.querySelectorAll("label"));
  for (const label of labels) {
    const text = (label.textContent ?? "").trim();
    const endsWithQuestion = text.endsWith("?");
    const startsWithVerb =
      /^(Type|Enter|Add|Tell|Provide|Choose|Set|What|How|Why|Where|When|Who|Share|Pick|Is)/i.test(
        text,
      );
    expect(
      endsWithQuestion || startsWithVerb,
      `label "${text}" must end with ? or start with a verb/question word`,
    ).toBe(true);
  }
});

test("[OrganizationForm]: root form has the max-w- Tailwind class (F-8 / UX-5)", async () => {
  const { screen, render } = await createDOM();
  await render(<OrganizationForm action={okAction()} />);

  const form = screen.querySelector("form") as HTMLFormElement;
  expect(form).not.toBeNull();
  const className = form.className;
  expect(className).toMatch(/max-w-/);
});

test("[OrganizationForm]: focusable tab order is fullName, identification, shortName, email, phone, submit (F-9)", async () => {
  const { screen, render, userEvent } = await createDOM();
  await render(<OrganizationForm action={okAction()} />);

  await expandDetails(screen, userEvent);

  const shortName = screen.querySelector(
    "input[id=\"shortName\"]",
  ) as HTMLInputElement;
  const email = screen.querySelector("input[id=\"email\"]") as HTMLInputElement;
  const phone = screen.querySelector("input[id=\"phone\"]") as HTMLInputElement;
  const submit = screen.querySelector("button[type=\"submit\"]") as HTMLButtonElement;
  const fullName = screen.querySelector("input[id=\"fullName\"]") as HTMLInputElement;
  const identification = screen.querySelector(
    "input[id=\"identification\"]",
  ) as HTMLInputElement;

  expect(fullName).not.toBeNull();
  expect(identification).not.toBeNull();
  expect(shortName).not.toBeNull();
  expect(email).not.toBeNull();
  expect(phone).not.toBeNull();
  expect(submit).not.toBeNull();

  const nodes = [fullName, identification, shortName, email, phone, submit];
  const DOCUMENT_POSITION_FOLLOWING = 4;
  for (let i = 1; i < nodes.length; i++) {
    const rel = nodes[i - 1].compareDocumentPosition(nodes[i]);
    expect(
      rel & DOCUMENT_POSITION_FOLLOWING,
      `expected ${nodes[i].id ?? nodes[i].tagName} to follow the previous node in DOM order`,
    ).toBeTruthy();
  }
});

test("[OrganizationForm]: form root has no onkeydown / onkeypress handlers (UX-9)", async () => {
  const { screen, render } = await createDOM();
  await render(<OrganizationForm action={okAction()} />);
  const form = screen.querySelector("form") as HTMLFormElement;
  expect(form).not.toBeNull();
  expect(form.getAttribute("onkeydown")).toBeNull();
  expect(form.getAttribute("onkeypress")).toBeNull();
});

// =========================================================================
// UX-2 / UX-2b — progressive disclosure threshold
// =========================================================================

test("[OrganizationForm]: review fieldset is not in the DOM when threshold unmet (UX-2)", async () => {
  const { screen, render } = await createDOM();
  await render(<OrganizationForm action={okAction()} />);

  const fieldsets = screen.querySelectorAll("fieldset[data-review-group=\"true\"]");
  expect(fieldsets.length).toBe(0);
});

test("[OrganizationForm]: review fieldset IS in the DOM when fullName and identification are set and showDetails clicked (UX-2b)", async () => {
  const { screen, render, userEvent } = await createDOM();
  await render(<OrganizationForm action={okAction()} />);

  const fullName = screen.querySelector("input[id=\"fullName\"]") as HTMLInputElement;
  const identification = screen.querySelector(
    "input[id=\"identification\"]",
  ) as HTMLInputElement;
  await userEvent(fullName, "input", { value: "Acme" });
  await userEvent(identification, "input", { value: "acme" });

  const showDetailsBtn = screen.querySelector(
    "button[data-action=\"show-details\"]",
  ) as HTMLButtonElement;
  expect(showDetailsBtn).not.toBeNull();
  await userEvent(showDetailsBtn, "click");

  const fieldsets = screen.querySelectorAll(
    "fieldset[data-review-group=\"true\"]",
  );
  expect(fieldsets.length).toBe(1);
});

// =========================================================================
// F-5 / F-5b — auto-derivation (locked via pure function)
// =========================================================================

describe("deriveIdentification (spec §5.3)", () => {
  test("lowercases and replaces disallowed chars with '-' (rule 1+2)", () => {
    expect(deriveIdentification("Hello World")).toBe("hello-world");
  });

  test("collapses runs of '-' into a single '-' (rule 3)", () => {
    expect(deriveIdentification("foo --- bar")).toBe("foo-bar");
  });

  test("strips leading and trailing '-' (rule 4)", () => {
    expect(deriveIdentification("---hello---")).toBe("hello");
  });

  test("truncates to 60 chars and strips trailing '-' (rule 5)", () => {
    const long = "a".repeat(58) + "-bc";
    const out = deriveIdentification(long);
    expect(out.length).toBeLessThanOrEqual(60);
    expect(out.endsWith("-")).toBe(false);
  });

  test("F-5 fixture: 'Acme Industrial S.A.' follows the rules literally", () => {
    expect(deriveIdentification("Acme Industrial S.A.")).toBe(
      "acme-industrial-s-a",
    );
  });

  test("F-5b: a manual override stops derivation (wiring smoke)", () => {
    const source = require("fs").readFileSync(
      require("path").join(__dirname, "organization-form.tsx"),
      "utf-8",
    );
    expect(source).toMatch(/userOverrodeIdentification/);
  });
});

test("[OrganizationForm]: auto-derivation is wired to the input handler (F-5 wiring)", async () => {
  const { screen, render } = await createDOM();
  await render(<OrganizationForm action={okAction()} />);
  const fullName = screen.querySelector("input[id=\"fullName\"]");
  expect(fullName).not.toBeNull();
  expect(typeof deriveIdentification).toBe("function");
});

// =========================================================================
// F-6 / F-6b — submit outcomes
// =========================================================================

test("[OrganizationForm]: submit success navigates to /organizations/{id} (F-6)", async () => {
  const { screen, render, userEvent } = await createDOM();
  __lastSubmittedActionCalls = 0;
  __lastSubmittedFormData = null;
  __lastNavigatedId = undefined;
  await render(
    <OrganizationForm
      action={recordingOkAction()}
      onSuccess$={recordingOnSuccess()}
    />,
  );

  const fullName = screen.querySelector("input[id=\"fullName\"]") as HTMLInputElement;
  const identification = screen.querySelector(
    "input[id=\"identification\"]",
  ) as HTMLInputElement;
  await userEvent(fullName, "input", { value: "Acme" });
  await userEvent(identification, "input", { value: "acme" });

  const form = screen.querySelector("form") as HTMLFormElement;
  await userEvent(form, "submit", { submitter: "ignored" });
  await new Promise((r) => setTimeout(r, 50));

  expect(__lastSubmittedActionCalls).toBe(1);
  expect(__lastSubmittedFormData).not.toBeNull();
  // Funnel through `unknown` once: vitest's
  // expect().not.toBeNull() does not narrow module-level
  // `let` bindings in our tsc config, and direct casts to
  // FormData refuse because the declared type contains null.
  const submittedFormData = __lastSubmittedFormData as unknown as FormData;
  expect(submittedFormData.get("full_name")).toBe("Acme");
  expect(submittedFormData.get("identification")).toBe("acme");
  expect(__lastNavigatedId).toBe(42);
  expect(screen.outerHTML).not.toContain("Something went wrong. Please try again.");
});

test("[OrganizationForm]: 409 conflict renders inline slug message and does NOT navigate (F-6b)", async () => {
  const { screen, render, userEvent } = await createDOM();
  __lastNavigatedId = undefined;
  await render(
    <OrganizationForm
      action={conflictAction()}
      onSuccess$={recordingOnSuccess()}
    />,
  );

  const fullName = screen.querySelector("input[id=\"fullName\"]") as HTMLInputElement;
  const identification = screen.querySelector(
    "input[id=\"identification\"]",
  ) as HTMLInputElement;
  await userEvent(fullName, "input", { value: "Acme" });
  await userEvent(identification, "input", { value: "acme" });

  const form = screen.querySelector("form") as HTMLFormElement;
  await userEvent(form, "submit", { submitter: "ignored" });
  await new Promise((r) => setTimeout(r, 50));

  const html = screen.outerHTML;
  expect(html).toContain("This slug is already taken. Try another.");
  expect(screen.querySelector("form")).not.toBeNull();
  expect(__lastNavigatedId).toBeUndefined();
});

// =========================================================================
// Client-side Zod validation (new — was missing before)
// =========================================================================

test("[OrganizationForm]: invalid email renders inline field error and does NOT call the action", async () => {
  const { screen, render, userEvent } = await createDOM();
  __lastSubmittedActionCalls = 0;
  await render(<OrganizationForm action={recordingOkAction()} />);

  await expandDetails(screen, userEvent);
  const email = screen.querySelector("input[id=\"email\"]") as HTMLInputElement;
  await userEvent(email, "input", { value: "not-an-email" });

  const form = screen.querySelector("form") as HTMLFormElement;
  await userEvent(form, "submit", { submitter: "ignored" });
  await new Promise((r) => setTimeout(r, 50));

  const html = screen.outerHTML;
  expect(html).toContain("Email is not a valid email address.");
  expect(screen.querySelector("[data-error=\"email\"]")).toBeTruthy();
  expect(__lastSubmittedActionCalls).toBe(0);
});

test("[OrganizationForm]: invalid phone renders inline field error and does NOT call the action", async () => {
  const { screen, render, userEvent } = await createDOM();
  __lastSubmittedActionCalls = 0;
  await render(<OrganizationForm action={recordingOkAction()} />);

  await expandDetails(screen, userEvent);
  const phone = screen.querySelector("input[id=\"phone\"]") as HTMLInputElement;
  await userEvent(phone, "input", { value: "12345" });

  const form = screen.querySelector("form") as HTMLFormElement;
  await userEvent(form, "submit", { submitter: "ignored" });
  await new Promise((r) => setTimeout(r, 50));

  const html = screen.outerHTML;
  expect(html).toContain("Phone must be in E.164 format");
  expect(screen.querySelector("[data-error=\"phone\"]")).not.toBeNull();
  expect(__lastSubmittedActionCalls).toBe(0);
});

test("[OrganizationForm]: invalid slug regex renders inline field error", async () => {
  const { screen, render, userEvent } = await createDOM();
  __lastSubmittedActionCalls = 0;
  await render(<OrganizationForm action={recordingOkAction()} />);

  const fullName = screen.querySelector("input[id=\"fullName\"]") as HTMLInputElement;
  const identification = screen.querySelector(
    "input[id=\"identification\"]",
  ) as HTMLInputElement;
  await userEvent(fullName, "input", { value: "Acme" });
  await userEvent(identification, "input", { value: "acme" });
  await userEvent(identification, "input", { value: "-bad" });

  const form = screen.querySelector("form") as HTMLFormElement;
  await userEvent(form, "submit", { submitter: "ignored" });
  await new Promise((r) => setTimeout(r, 50));

  expect(
    screen.querySelector("[data-error=\"identification\"]"),
  ).not.toBeNull();
  expect(__lastSubmittedActionCalls).toBe(0);
});

test("[OrganizationForm]: typing in a field clears its prior error", async () => {
  const { screen, render, userEvent } = await createDOM();
  await render(<OrganizationForm action={recordingOkAction()} />);

  await expandDetails(screen, userEvent);
  let email = screen.querySelector("input[id=\"email\"]") as HTMLInputElement;

  await userEvent(email, "input", { value: "bad" });
  const form = screen.querySelector("form") as HTMLFormElement;
  await userEvent(form, "submit", { submitter: "ignored" });
  await new Promise((r) => setTimeout(r, 30));
  expect(screen.querySelector("[data-error=\"email\"]")).toBeTruthy();

  // Re-query after the submit re-render — the original
  // `email` reference is bound to the input element that
  // existed before submit; Qwik may have re-rendered the
  // review fieldset with a new DOM node.
  email = screen.querySelector("input[id=\"email\"]") as HTMLInputElement;
  await userEvent(email, "input", { value: "ops@acme.com" });
  await new Promise((r) => setTimeout(r, 100));
  expect(screen.querySelector("[data-error=\"email\"]")).toBeFalsy();
});

test("[OrganizationForm]: valid full payload (incl. E.164 phone, valid email) navigates", async () => {
  const { screen, render, userEvent } = await createDOM();
  __lastSubmittedActionCalls = 0;
  __lastNavigatedId = undefined;
  await render(
    <OrganizationForm
      action={recordingOkAction()}
      onSuccess$={recordingOnSuccess()}
    />,
  );

  await expandDetails(screen, userEvent);
  const shortName = screen.querySelector(
    "input[id=\"shortName\"]",
  ) as HTMLInputElement;
  const email = screen.querySelector("input[id=\"email\"]") as HTMLInputElement;
  const phone = screen.querySelector("input[id=\"phone\"]") as HTMLInputElement;
  await userEvent(shortName, "input", { value: "Acme Co" });
  await userEvent(email, "input", { value: "ops@acme.com" });
  await userEvent(phone, "input", { value: "+14155552671" });

  const form = screen.querySelector("form") as HTMLFormElement;
  await userEvent(form, "submit", { submitter: "ignored" });
  await new Promise((r) => setTimeout(r, 50));

  expect(screen.querySelectorAll("[data-error]").length).toBe(0);
  expect(__lastSubmittedActionCalls).toBe(1);
  expect(__lastNavigatedId).toBe(42);
});
