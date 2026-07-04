/**
 * `useClickOutside` — small Qwik hook for the AvatarDropdown.
 *
 * Reference: `openspec/changes/cachicamas-login-ux/specs/app-shell/spec.md`
 *   R-AS-004 (S-AS-030..S-AS-032) — the dropdown panel closes on
 *   outside-mousedown, on Escape, and on trigger re-click.
 *
 * Reference: `openspec/changes/cachicamas-login-ux/design.md` §3.2
 *   Sequence diagram and Qwik lifecycle.
 *
 * Reference: `openspec/changes/cachicamas-login-ux/design.md` §5
 *   ADR-0007 — in-tree hook, no new top-level dep.
 *
 * Returns:
 *   - `open` — `useSignal<boolean>` the consumer reads to render the panel.
 *   - `setOpen$` — `QRL<(next: boolean) => void>` to flip the signal.
 *   - `triggerRef` — `Signal<Element | undefined>` to attach to the
 *     trigger button (used by the mousedown discriminator).
 *   - `panelRef` — same, for the panel.
 *
 * Behavior:
 *   - The trigger button's `onClick$` flips the open signal.
 *   - The `keydown` listener on the document closes the panel on Escape.
 *   - The `mousedown` listener on the document closes the panel if
 *     the event target is outside both the trigger and the panel.
 *   - Listeners are auto-cleaned when the component unmounts (Qwik's
 *     `useOnDocument` / `useOnWindow` semantics).
 */
import {
  $,
  type QRL,
  type Signal,
  useOnDocument,
  useSignal,
} from "@builder.io/qwik";

export interface UseClickOutside {
  open: Signal<boolean>;
  setOpen$: QRL<(next: boolean) => void>;
  triggerRef: Signal<Element | undefined>;
  panelRef: Signal<Element | undefined>;
}

export function useClickOutside(): UseClickOutside {
  const open = useSignal(false);
  const triggerRef = useSignal<Element | undefined>();
  const panelRef = useSignal<Element | undefined>();

  // Escape closes the panel (S-AS-031). The listener is only active
  // when the panel is open — we short-circuit early when `open.value`
  // is false so the panel doesn't get hijacked by stray Esc presses
  // on the page.
  useOnDocument(
    "keydown",
    $((event) => {
      if (!open.value) return;
      const e = event as KeyboardEvent;
      if (e.key === "Escape") {
        open.value = false;
      }
    }),
  );

  // Outside-mousedown closes the panel (S-AS-032). Same gating.
  useOnDocument(
    "mousedown",
    $((event) => {
      if (!open.value) return;
      const e = event as MouseEvent;
      const target = e.target as Node | null;
      if (!target) return;
      const trigger = triggerRef.value;
      const panel = panelRef.value;
      if (trigger?.contains(target)) return;
      if (panel?.contains(target)) return;
      open.value = false;
    }),
  );

  const setOpen$ = $((next: boolean) => {
    open.value = next;
  });

  return { open, setOpen$, triggerRef, panelRef };
}
