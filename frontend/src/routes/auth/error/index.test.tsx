/**
 * `/auth/error` — pure mapping test + render smoke.
 *
 * The reason → message mapping is the testable surface (the component
 * itself is a thin presentational shell that re-uses the same mapping).
 * We also render the component once to lock its structure (heading,
 * retry link).
 */
import { describe, expect, test } from "vitest";
import { createDOM } from "@builder.io/qwik/testing";
import AuthError, {
  isKnownReason,
  KNOWN_REASONS,
  reasonMessage,
  head,
} from "./index";

describe("reasonMessage", () => {
  test("renders the canonical generic message when reason is null", () => {
    expect(reasonMessage(null)).toContain("Algo salió mal");
  });

  test("renders the canonical generic message when reason is unknown", () => {
    expect(reasonMessage("garbage-not-a-known-reason")).toContain(
      "Algo salió mal",
    );
  });

  test("renders 'invalid_state' as a reason-specific copy", () => {
    const msg = reasonMessage("invalid_state");
    expect(msg).toBeTruthy();
    expect(msg).not.toContain("Algo salió mal");
    // The copy is Spanish; assert it contains a recognisable verb.
    expect(msg.toLowerCase()).toMatch(/(verificación|seguridad)/);
  });

  test("renders 'token_exchange_failed' as a reason-specific copy", () => {
    const msg = reasonMessage("token_exchange_failed");
    expect(msg).not.toContain("Algo salió mal");
    expect(msg.toLowerCase()).toMatch(/(google|código)/);
  });

  test("renders 'userinfo_failed' as a reason-specific copy", () => {
    const msg = reasonMessage("userinfo_failed");
    expect(msg).not.toContain("Algo salió mal");
    expect(msg.toLowerCase()).toMatch(/(perfil|google)/);
  });

  test("renders 'blocked' as a reason-specific copy (CRITICAL — never generic)", () => {
    const msg = reasonMessage("blocked");
    expect(msg).not.toContain("Algo salió mal");
    // The blocked message MUST mention soporte / soporte so the user
    // knows to escalate. This is the assertion that locks R-FE-009.
    expect(msg.toLowerCase()).toContain("soporte");
  });

  test("renders 'access_denied' as a reason-specific copy", () => {
    const msg = reasonMessage("access_denied");
    expect(msg).not.toContain("Algo salió mal");
    expect(msg.toLowerCase()).toContain("cancelaste");
  });

  test("renders 'internal_error' as the generic copy", () => {
    // Internal errors are intentionally generic (no leaked detail).
    expect(reasonMessage("internal_error")).toContain("Algo salió mal");
  });

  test("renders 'missing_code' as a reason-specific copy", () => {
    const msg = reasonMessage("missing_code");
    expect(msg).not.toContain("Algo salió mal");
    expect(msg.toLowerCase()).toContain("código");
  });
});

describe("isKnownReason", () => {
  test("accepts every locked reason", () => {
    for (const r of KNOWN_REASONS) {
      expect(isKnownReason(r)).toBe(true);
    }
  });

  test("rejects null and unknown strings", () => {
    expect(isKnownReason(null)).toBe(false);
    expect(isKnownReason("")).toBe(false);
    expect(isKnownReason("not-a-reason")).toBe(false);
    expect(isKnownReason("invalid_state ")).toBe(false); // trailing whitespace
  });
});

describe("AuthError component (render smoke)", () => {
  test("renders the heading, the generic message, and the retry link", async () => {
    const { screen, render } = await createDOM();
    await render(<AuthError />);
    const heading = screen.querySelector("h1");
    expect(heading?.textContent).toContain("No pudimos iniciar tu sesión");
    const message = screen.querySelector('[data-testid="auth-error-message"]');
    expect(message?.textContent).toContain("Algo salió mal");
    const retry = screen.querySelector('[data-testid="auth-error-retry"]');
    expect(retry?.getAttribute("href")).toBe("/auth/google/login");
    expect(retry?.textContent).toContain("Volver a intentar");
  });

  test("is marked noindex,nofollow on the document head", async () => {
    // The document head is set via `export const head` and rendered by
    // Qwik City's SSR pipeline — `createDOM` only renders the component
    // body. We assert the head object directly here; the SSR HTML
    // output is verified end-to-end in T3.17.
    // `head` may be exported as a value or as a function; handle both.
    const headValue =
      typeof head === "function" ? (head as unknown as () => unknown)() : head;
    expect(headValue).toBeTruthy();
    const robotsMeta = (
      headValue as { meta?: { name?: string; content?: string }[] }
    ).meta?.find((m) => m.name === "robots");
    expect(robotsMeta?.content).toContain("noindex");
    expect(robotsMeta?.content).toContain("nofollow");
  });
});
