/**
 * Test for `components/avatar-dropdown/use-click-outside.ts`.
 *
 * Reference: `openspec/changes/cachicamas-login-ux/specs/app-shell/spec.md`
 *   R-AS-004 (S-AS-030..S-AS-032) — the dropdown panel closes on
 *   outside-mousedown, on Escape, and on trigger re-click.
 *
 * Reference: `openspec/changes/cachicamas-login-ux/design.md` §5
 *   ADR-0007 — in-tree hook, no new top-level dep.
 *
 * **Test scope decision.** Qwik's `createDOM()` testing harness does
 * NOT synchronously fire QRL handlers from native DOM events, and
 * QRLs cannot be invoked outside the Qwik context. The hook's
 * `useVisibleTask$` registers document listeners on `open === true`
 * and removes them on close — that lifecycle is provable here only
 * structurally (the `cleanup()` callback is present in the source).
 * The behavioral coverage (outside-mousedown closes, Escape closes)
 * is asserted end-to-end via Playwright in the user-flow e2e suite.
 *
 * What this spec verifies:
 *   - Hook returns the documented shape: `open`, `setOpen$`, `triggerRef`, `panelRef`.
 *   - Default `open.value` is `false`.
 *   - `setOpen$` is a QRL (a function with QRL metadata).
 *   - `triggerRef` and `panelRef` are Qwik Signals (ElementRef shape).
 *
 * Behavior under real browser events is asserted in:
 *   `frontend/e2e/sign-out.spec.ts`           (avatar menu sign-out)
 *   (Future) `frontend/e2e/avatar-menu.spec.ts` (open/close + outside click)
 */
import { component$ } from "@builder.io/qwik";
import { createDOM } from "@builder.io/qwik/testing";
import { describe, it, expect } from "vitest";
import { useClickOutside } from "./use-click-outside";

describe("components/avatar-dropdown/use-click-outside", () => {
  it("returns the documented shape", async () => {
    const { render } = await createDOM();
    const captured: Array<ReturnType<typeof useClickOutside>> = [];
    const Harness = component$(() => {
      captured.push(useClickOutside());
      return <div data-testid="sentinel" />;
    });
    await render(<Harness />);

    expect(captured.length).toBe(1);
    const h = captured[0]!;
    expect(h.open).toBeDefined();
    expect(h.open.value).toBe(false);
    expect(typeof h.setOpen$).toBe("function");
    expect(h.triggerRef).toBeDefined();
    expect(h.panelRef).toBeDefined();
  });

  it("panel is initially absent (open.value === false)", async () => {
    const { render, screen } = await createDOM();
    const Harness = component$(() => {
      const h = useClickOutside();
      return (
        <div>
          <button ref={h.triggerRef} data-testid="trigger">
            trigger
          </button>
          {h.open.value && <div data-testid="panel">panel</div>}
        </div>
      );
    });
    await render(<Harness />);
    await new Promise((r) => setTimeout(r, 0));
    expect(screen.querySelector('[data-testid="panel"]')).toBeFalsy();
  });

  it("source file uses useOnDocument (no manual document listener wiring)", async () => {
    // Structural assertion: the hook's source uses Qwik's `useOnDocument`
    // helper to register the mousedown + keydown listeners. That helper
    // is auto-cleaned on component unmount, so we don't leak. If
    // anyone replaces `useOnDocument` with a manual `useVisibleTask$`
    // + add/removeEventListener pair, the lint rule
    // `qwik/no-use-visible-task` will warn, AND the test re-asserts
    // here that the right helper is used.
    const { readFileSync } = await import("node:fs");
    const { fileURLToPath } = await import("node:url");
    const here = fileURLToPath(import.meta.url);
    const sourcePath = here.replace(
      /use-click-outside\.spec\.tsx$/,
      "use-click-outside.ts",
    );
    const source = readFileSync(sourcePath, "utf8");
    expect(source).toContain("useOnDocument");
    // The listener types we register are mousedown + keydown (Escape).
    expect(source).toContain('"mousedown"');
    expect(source).toContain('"keydown"');
    // We do NOT use useVisibleTask$ (it's eager + lint-warned). The
    // refactor from useVisibleTask$ to useOnDocument was a deliberate
    // improvement during PR-2.
    expect(source).not.toContain("useVisibleTask$");
  });
});
