import {
  $,
  component$,
  useOnDocument,
  useSignal,
  useVisibleTask$,
} from "@builder.io/qwik";
import { ARCHETYPES, archetypeHref } from "~/lib/mock/registry";

/**
 * FunctionRail — the dock, as a function-key legend.
 *
 * The terminal's dock is the strip of key labels along the bottom edge: one
 * cell per destination, each showing the key that opens it and the name of
 * what it opens. It is a dock in every way that matters — always present,
 * one cell per application, current one marked — and it needs no icon set,
 * which is the reason it fits this product: every entry is already a word,
 * so nothing depends on recognising a picture (PRODUCT.md § Accessibility).
 *
 * The function keys genuinely work. F1 opens Chat.
 */
export const FunctionRail = component$(() => {
  const here = useSignal("");

  // The active cell is a property of the browser's current URL, and the layout
  // deliberately takes no router context, so it is read once on the client.
  // eslint-disable-next-line qwik/no-use-visible-task
  useVisibleTask$(() => {
    // `createDOM()` renders in Node with no `window`, and a task that throws
    // there takes the whole run down rather than failing one assertion.
    if (typeof window === "undefined") return;
    here.value = window.location.pathname;
  });

  useOnDocument(
    "keydown",
    $((event: Event) => {
      const e = event as KeyboardEvent;
      if (typeof window === "undefined") return;
      if (!/^F[1-8]$/.test(e.key)) return;
      // Never steal a function key from a modifier combination, and never from
      // someone who is typing: F1 fired mid-draft would navigate away and
      // discard the composer's contents with no confirmation.
      if (e.metaKey || e.ctrlKey || e.altKey || e.shiftKey) return;
      const focused = e.target as HTMLElement | null;
      const tag = focused?.tagName?.toLowerCase();
      if (tag === "input" || tag === "textarea" || tag === "select") return;
      if (focused?.isContentEditable) return;
      const system = e.key === "F8" ? "/settings/" : null;
      const archetype = ARCHETYPES.find((a) => a.fkey === e.key);
      const href = system ?? (archetype ? archetypeHref(archetype) : null);
      if (!href) return;
      // A document-level key handler has no `preventdefault:` attribute to
      // declare, so this is the only place the browser's own F-key behaviour
      // can be suppressed.
      // eslint-disable-next-line qwik/no-async-prevent-default
      e.preventDefault();
      window.location.assign(href);
    }),
  );

  const cells = [
    ...ARCHETYPES.map((a) => ({
      fkey: a.fkey,
      // The dock uses the short name: "Database Administrator" across a cell
      // pushes every other destination off a laptop screen.
      name: a.dockName,
      href: archetypeHref(a),
      dimmed: a.state === "unplanned",
    })),
    { fkey: "F8", name: "System", href: "/settings/", dimmed: false },
  ];

  return (
    <nav
      data-testid="function-rail"
      aria-label="Applications"
      class="border-rule bg-panel sticky bottom-0 z-20 flex items-stretch gap-px overflow-x-auto border-t"
    >
      {cells.map((cell, i) => {
        const active = here.value === cell.href;
        return (
          <a
            key={cell.fkey}
            href={cell.href}
            data-testid={`dock-${cell.fkey}`}
            aria-current={active ? "page" : undefined}
            class={[
              "flex min-w-0 shrink-0 items-baseline gap-1.5 px-3 py-1.5 no-underline transition-colors duration-150",
              i === cells.length - 1 ? "ml-auto" : "",
              active
                ? "bg-fg text-void"
                : cell.dimmed
                  ? "text-fg-dim hover:bg-raise hover:text-fg-mid"
                  : "text-fg-mid hover:bg-raise hover:text-fg",
            ].join(" ")}
          >
            <span
              class={`text-legend tracking-[0.12em] ${active ? "text-void" : "text-fg-dim"}`}
            >
              {cell.fkey}
            </span>
            <span class="text-label tracking-[0.1em] uppercase">
              {cell.name}
            </span>
          </a>
        );
      })}
    </nav>
  );
});
