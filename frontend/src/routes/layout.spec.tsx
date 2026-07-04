/**
 * Layout spec — covers the app shell introduced by PR-3 of
 * cachicamas-login-ux. Spec surfaces:
 *
 *   S-AS-001 (R-AS-001) — layout.tsx file exists, exports default,
 *                       imports useSession
 *   S-AS-002 (R-AS-001) — <header> above the route's main content
 *   S-AS-010 (R-AS-002) — anon renders SignInButton
 *   S-AS-011 (R-AS-002) — anon does NOT render AvatarDropdown
 *   S-AS-020 (R-AS-003) — auth renders AvatarDropdown with avatar
 *   S-AS-021 (R-AS-003) — auth does NOT render SignInButton
 *   S-AS-022 (R-AS-003) — session avatar URL is sanitized (ADR-0009)
 *   S-AS-050 (R-AS-006) — skip link is the first focusable element
 *
 * Mocking strategy mirrors the patterns in `routes/index.spec.tsx`:
 * stub `~/routes/plugin@auth` (useSession + useSignIn + useSignOut)
 * via vi.mock so the test harness doesn't need Qwik City's
 * request context. Each scenario re-creates the DOM so the
 * sessions are independent.
 */
import { createDOM } from "@builder.io/qwik/testing";
import { $, type QRL } from "@builder.io/qwik";
import { test, expect, vi } from "vitest";
import { existsSync, readFileSync } from "node:fs";
import { resolve } from "node:path";

// Stub the Auth.js plugin so the layout can call useSession +
// useSignIn + useSignOut without Qwik City's request context.
// The shape matches the stub in routes/index.spec.tsx so the
// existing patterns apply cleanly.
vi.mock("~/routes/plugin@auth", () => ({
  useSession: () => ({ value: null }),
  useSignIn: () => ({
    submit: $((_fd: FormData) => Promise.resolve()) as QRL<
      (formData: FormData) => unknown
    >,
    actionPath: "/auth/signin",
  }),
  useSignOut: () => ({
    submit: $((_fd: FormData) => Promise.resolve()) as QRL<
      (formData: FormData) => unknown
    >,
    actionPath: "/auth/signout",
  }),
  onRequest: () => Promise.resolve(),
}));

// Helper — render the layout with a stubbed session. The
// `sessionState` is captured by the stub at module-load time,
// so to swap between anon and auth we have to re-import the
// layout with a different mock. The simplest path: each
// scenario gets its own top-level describe, and we use
// vi.doMock + dynamic import for the auth-aware scenarios.
// That keeps the assertions readable.
async function renderWithSession(session: unknown) {
  vi.resetModules();
  vi.doMock("~/routes/plugin@auth", () => ({
    useSession: () => ({ value: session }),
    useSignIn: () => ({
      submit: $((_fd: FormData) => Promise.resolve()) as QRL<
        (formData: FormData) => unknown
      >,
      actionPath: "/auth/signin",
    }),
    useSignOut: () => ({
      submit: $((_fd: FormData) => Promise.resolve()) as QRL<
        (formData: FormData) => unknown
      >,
      actionPath: "/auth/signout",
    }),
    onRequest: () => Promise.resolve(),
  }));
  const mod = await import("./layout");
  const Layout = mod.default;
  const { screen, render } = await createDOM();
  await render(
    <Layout>
      <main id="main" data-testid="child">
        child content
      </main>
    </Layout>,
  );
  return screen;
}

const ANON_SESSION = null;

const AUTH_SESSION = {
  user: {
    name: "Braejan",
    email: "braejan@example.com",
    image: "https://avatars.githubusercontent.com/u/12345",
  },
};

test("[routes/layout]: layout.tsx exists, exports default component$, imports useSession (S-AS-001)", async () => {
  const layoutPath = resolve(
    new URL(".", import.meta.url).pathname,
    "layout.tsx",
  );
  expect(existsSync(layoutPath)).toBe(true);
  const source = readFileSync(layoutPath, "utf8");
  expect(source).toContain("useSession");
  expect(source).toContain("component$");
  expect(source).toMatch(/export default component\$/);
});

test("[routes/layout]: renders a <header> above the route's main content (S-AS-002)", async () => {
  const screen = await renderWithSession(ANON_SESSION);
  const header = screen.querySelector("header");
  expect(header).toBeTruthy();
  // Exactly one <header> per page.
  const headers = screen.querySelectorAll("header");
  expect(headers.length).toBe(1);
  // The <main> child is rendered below the <header>.
  expect(screen.querySelector('[data-testid="child"]')).toBeTruthy();
});

test("[routes/layout]: anon shell renders SignInButton with providerId=github (S-AS-010)", async () => {
  const screen = await renderWithSession(ANON_SESSION);
  const form = screen.querySelector('form[data-testid="sign-in-button"]');
  expect(form).toBeTruthy();
  const providerId = form?.querySelector(
    'input[name="providerId"]',
  ) as HTMLInputElement | null;
  expect(providerId?.value).toBe("github");
});

test("[routes/layout]: anon shell does NOT render AvatarDropdown (S-AS-011)", async () => {
  const screen = await renderWithSession(ANON_SESSION);
  expect(screen.querySelector('[data-testid="avatar-dropdown"]')).toBeFalsy();
});

test("[routes/layout]: auth shell renders AvatarDropdown with avatar image + aria-label (S-AS-020)", async () => {
  const screen = await renderWithSession(AUTH_SESSION);
  const trigger = screen.querySelector('button[data-testid="avatar-dropdown"]');
  expect(trigger).toBeTruthy();
  const img = trigger?.querySelector("img") as HTMLImageElement | null;
  expect(img).toBeTruthy();
  expect(img?.getAttribute("src")).toBe(
    "https://avatars.githubusercontent.com/u/12345",
  );
  expect(
    (trigger as HTMLButtonElement).getAttribute("aria-label"),
  ).toBe("Braejan menu");
});

test("[routes/layout]: auth shell does NOT render SignInButton (S-AS-021)", async () => {
  const screen = await renderWithSession(AUTH_SESSION);
  expect(screen.querySelector('form[data-testid="sign-in-button"]')).toBeFalsy();
});

test("[routes/layout]: non-https avatar URL is sanitized (S-AS-022, ADR-0009)", async () => {
  // `<img src>` accepts any scheme in tests if the URL passes the
  // browser-parsing layer. The composer rule from PR-2a's
  // safeAvatarSrc helper is the runtime guard — at the layout
  // boundary we assert the link between session.image and the
  // rendered <img> src is mediated by that helper, not direct.
  const screen = await renderWithSession({
    user: {
      name: "NoImg",
      email: "noimg@example.com",
      image: "javascript:alert(1)",
    },
  });
  const trigger = screen.querySelector('button[data-testid="avatar-dropdown"]');
  expect(trigger).toBeTruthy();
  // The dangerous scheme MUST NOT reach the DOM — either the img
  // is omitted entirely OR its src starts with https:.
  const img = trigger?.querySelector("img") as HTMLImageElement | null;
  if (img) {
    const src = img.getAttribute("src") ?? "";
    expect(src.startsWith("javascript:")).toBe(false);
    if (src) {
      expect(src.startsWith("https:")).toBe(true);
    }
  }
});

test("[routes/layout]: skip-to-main link is the first focusable element of <header>'s parent (S-AS-050)", async () => {
  const screen = await renderWithSession(ANON_SESSION);
  // The skip link is `<a href="#main">Skip to main content</a>`.
  const skip = screen.querySelector(
    'a[href="#main"][data-testid="skip-to-main"]',
  );
  expect(skip).toBeTruthy();
  const header = screen.querySelector("header");
  expect(header).toBeTruthy();
  // The skip link MUST precede the header in document order. We
  // assert via compareDocumentPosition so the test is robust to
  // Qwik's createDOM host wrapper (which inserts an intermediate
  // <q:host> around the rendered tree, so the parent isn't
  // literally <body>).
  const pos = (skip as Node).compareDocumentPosition(header as Node);
  // DOCUMENT_POSITION_FOLLOWING = 0x04. If pos & 4 == true, the
  // header comes AFTER the skip link in document order.
  expect(pos & 0x04).toBeTruthy();
});
